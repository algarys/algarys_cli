package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type ProjectConfig struct {
	Name          string
	Description   string
	PythonVersion string
	CreateGitHub  bool
	GitHubOrg     string
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Inicializa um novo projeto Python com estrutura SOLID",
	Long: `Cria um novo projeto Python seguindo os padrões da Algarys:
- Repositório privado no GitHub (github.com/algarys)
- Estrutura de pastas SOLID (domain, application, infrastructure, interfaces)
- Estrutura para AI (agents, tools, prompts, models, notebooks)
- Integração com Temporal (activities, workflows, worker)
- Gerenciamento de dependências com UV`,
	Run: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) {
	// Banner
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Println("\n  █████╗ ██╗      ██████╗  █████╗ ██████╗ ██╗   ██╗███████╗")
	cyan.Println(" ██╔══██╗██║     ██╔════╝ ██╔══██╗██╔══██╗╚██╗ ██╔╝██╔════╝")
	cyan.Println(" ███████║██║     ██║  ███╗███████║██████╔╝ ╚████╔╝ ███████╗")
	cyan.Println(" ██╔══██║██║     ██║   ██║██╔══██║██╔══██╗  ╚██╔╝  ╚════██║")
	cyan.Println(" ██║  ██║███████╗╚██████╔╝██║  ██║██║  ██║   ██║   ███████║")
	cyan.Println(" ╚═╝  ╚═╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝   ╚══════╝")
	fmt.Println()
	color.New(color.FgWhite).Println(" Criando novo projeto Python com estrutura SOLID + AI\n")

	config := ProjectConfig{
		GitHubOrg: "algarys",
	}

	// Formulário interativo
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Nome do projeto").
				Description("Use kebab-case (ex: meu-projeto)").
				Placeholder("meu-projeto").
				Value(&config.Name).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("nome é obrigatório")
					}
					if strings.Contains(s, " ") {
						return fmt.Errorf("use hífen ao invés de espaços")
					}
					return nil
				}),

			huh.NewInput().
				Title("Descrição").
				Description("Uma breve descrição do projeto").
				Placeholder("Agente de IA para...").
				Value(&config.Description),

			huh.NewSelect[string]().
				Title("Versão do Python").
				Options(
					huh.NewOption("Python 3.12 (Recomendado)", "3.12"),
					huh.NewOption("Python 3.11", "3.11"),
					huh.NewOption("Python 3.10", "3.10"),
				).
				Value(&config.PythonVersion),

			huh.NewConfirm().
				Title("Criar repositório no GitHub?").
				Description("Será criado em github.com/algarys (requer gh auth login)").
				Affirmative("Sim").
				Negative("Não").
				Value(&config.CreateGitHub),
		),
	)

	err := form.Run()
	if err != nil {
		if err.Error() == "user aborted" {
			fmt.Println("\n Cancelado pelo usuário.")
			return
		}
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}

	// Normalizar nome do projeto
	config.Name = strings.ToLower(strings.ReplaceAll(config.Name, " ", "-"))
	moduleName := strings.ReplaceAll(config.Name, "-", "_")

	fmt.Println()
	printStep("Criando projeto: " + config.Name)

	// Verificar se diretório já existe
	if _, err := os.Stat(config.Name); !os.IsNotExist(err) {
		color.Red("✗ Erro: diretório '%s' já existe", config.Name)
		os.Exit(1)
	}

	// Criar diretório do projeto
	if err := os.MkdirAll(config.Name, 0755); err != nil {
		color.Red("✗ Erro ao criar diretório: %v", err)
		os.Exit(1)
	}

	// Criar estrutura completa
	printStep("Criando estrutura SOLID + AI + Temporal...")
	createProjectStructure(config.Name, moduleName)
	printSuccess("Estrutura de pastas criada")

	// Criar arquivos de configuração
	printStep("Criando arquivos de configuração...")
	createConfigFiles(config.Name, moduleName, config.Description, config.PythonVersion)
	printSuccess("pyproject.toml, .gitignore, README.md criados")

	// Inicializar git
	printStep("Inicializando Git...")
	initLocalGit(config.Name)
	printSuccess("Repositório Git inicializado")

	// Inicializar UV
	printStep("Configurando UV...")
	if initUV(config.Name) {
		printSuccess("Ambiente virtual criado e dependências instaladas")
	}

	// Criar repositório no GitHub
	if config.CreateGitHub {
		repoName := fmt.Sprintf("algarys_%s", config.Name)
		printStep("Criando repositório no GitHub...")
		if createGitHubRepo(config.Name, config.Description, config.GitHubOrg) {
			printSuccess(fmt.Sprintf("Repositório criado: github.com/%s/%s", config.GitHubOrg, repoName))

			// Configurar ruleset
			printStep("Configurando regras de proteção...")
			if configureRuleset(repoName, config.GitHubOrg) {
				printSuccess("Ruleset configurado: PR obrigatório + linear history")
			} else {
				printWarning("Não foi possível configurar ruleset automaticamente")
			}
		}
	}

	// Resumo final
	fmt.Println()
	color.Green("✓ Projeto %s criado com sucesso!", config.Name)
	fmt.Println()
	color.White("  Próximos passos:")
	fmt.Println()
	color.Cyan("    cd %s", config.Name)
	color.Cyan("    uv sync")
	color.Cyan("    uv run python -m %s", moduleName)
	fmt.Println()
}

func printStep(msg string) {
	color.New(color.FgYellow).Printf("→ %s\n", msg)
}

func printSuccess(msg string) {
	color.New(color.FgGreen).Printf("  ✓ %s\n", msg)
}

func printWarning(msg string) {
	color.New(color.FgYellow).Printf("  ⚠ %s\n", msg)
}

func createProjectStructure(projectName, moduleName string) {
	basePath := filepath.Join(projectName, moduleName)

	// Estrutura de pastas
	dirs := []string{
		// Domain - Entidades e regras de negócio
		filepath.Join(basePath, "domain", "entities"),
		filepath.Join(basePath, "domain", "repositories"),
		filepath.Join(basePath, "domain", "value_objects"),

		// Application - Casos de uso e serviços
		filepath.Join(basePath, "application", "services"),
		filepath.Join(basePath, "application", "use_cases"),
		filepath.Join(basePath, "application", "dtos"),

		// Infrastructure - Implementações concretas
		filepath.Join(basePath, "infrastructure", "database"),
		filepath.Join(basePath, "infrastructure", "external"),
		filepath.Join(basePath, "infrastructure", "repositories"),

		// Interfaces - Controllers, API, CLI
		filepath.Join(basePath, "interfaces", "api"),
		filepath.Join(basePath, "interfaces", "cli"),

		// AI - Agentes de IA
		filepath.Join(basePath, "ai", "agents"),
		filepath.Join(basePath, "ai", "tools"),
		filepath.Join(basePath, "ai", "prompts"),
		filepath.Join(basePath, "ai", "models"),
		filepath.Join(basePath, "ai", "notebooks"),
		filepath.Join(basePath, "ai", "workflows"),
		filepath.Join(basePath, "ai", "evaluations"),
		filepath.Join(basePath, "ai", "data"),
		filepath.Join(basePath, "ai", "configs"),

		// Temporal - Orquestração de workflows
		filepath.Join(basePath, "temporal", "activities"),
		filepath.Join(basePath, "temporal", "workflows"),
		filepath.Join(basePath, "temporal", "worker"),

		// Tests
		filepath.Join(projectName, "tests", "unit"),
		filepath.Join(projectName, "tests", "integration"),
	}

	for _, dir := range dirs {
		os.MkdirAll(dir, 0755)
	}

	// Criar __init__.py em todos os pacotes Python
	initFiles := []string{
		// Root
		filepath.Join(basePath, "__init__.py"),
		filepath.Join(basePath, "__main__.py"),

		// Domain
		filepath.Join(basePath, "domain", "__init__.py"),
		filepath.Join(basePath, "domain", "entities", "__init__.py"),
		filepath.Join(basePath, "domain", "repositories", "__init__.py"),
		filepath.Join(basePath, "domain", "value_objects", "__init__.py"),

		// Application
		filepath.Join(basePath, "application", "__init__.py"),
		filepath.Join(basePath, "application", "services", "__init__.py"),
		filepath.Join(basePath, "application", "use_cases", "__init__.py"),
		filepath.Join(basePath, "application", "dtos", "__init__.py"),

		// Infrastructure
		filepath.Join(basePath, "infrastructure", "__init__.py"),
		filepath.Join(basePath, "infrastructure", "database", "__init__.py"),
		filepath.Join(basePath, "infrastructure", "external", "__init__.py"),
		filepath.Join(basePath, "infrastructure", "repositories", "__init__.py"),

		// Interfaces
		filepath.Join(basePath, "interfaces", "__init__.py"),
		filepath.Join(basePath, "interfaces", "api", "__init__.py"),
		filepath.Join(basePath, "interfaces", "cli", "__init__.py"),

		// AI
		filepath.Join(basePath, "ai", "__init__.py"),
		filepath.Join(basePath, "ai", "agents", "__init__.py"),
		filepath.Join(basePath, "ai", "tools", "__init__.py"),
		filepath.Join(basePath, "ai", "prompts", "__init__.py"),
		filepath.Join(basePath, "ai", "models", "__init__.py"),
		filepath.Join(basePath, "ai", "workflows", "__init__.py"),
		filepath.Join(basePath, "ai", "evaluations", "__init__.py"),
		filepath.Join(basePath, "ai", "configs", "__init__.py"),

		// Temporal
		filepath.Join(basePath, "temporal", "__init__.py"),
		filepath.Join(basePath, "temporal", "activities", "__init__.py"),
		filepath.Join(basePath, "temporal", "workflows", "__init__.py"),
		filepath.Join(basePath, "temporal", "worker", "__init__.py"),

		// Tests
		filepath.Join(projectName, "tests", "__init__.py"),
		filepath.Join(projectName, "tests", "unit", "__init__.py"),
		filepath.Join(projectName, "tests", "integration", "__init__.py"),
	}

	for _, f := range initFiles {
		os.WriteFile(f, []byte(""), 0644)
	}

	// Criar __main__.py com conteúdo
	mainContent := fmt.Sprintf(`"""Ponto de entrada do módulo %s."""


def main() -> None:
    """Função principal."""
    print("Hello from %s!")


if __name__ == "__main__":
    main()
`, moduleName, moduleName)
	os.WriteFile(filepath.Join(basePath, "__main__.py"), []byte(mainContent), 0644)

	// Criar exemplo de entity
	entityExample := `"""Entidade base do domínio."""
from dataclasses import dataclass, field
from datetime import datetime
from uuid import UUID, uuid4


@dataclass
class BaseEntity:
    """Classe base para entidades."""

    id: UUID = field(default_factory=uuid4)
    created_at: datetime = field(default_factory=datetime.utcnow)
    updated_at: datetime = field(default_factory=datetime.utcnow)
`
	os.WriteFile(filepath.Join(basePath, "domain", "entities", "base.py"), []byte(entityExample), 0644)

	// Criar exemplo de repository interface
	repoExample := `"""Interfaces de repositório (abstrações)."""
from abc import ABC, abstractmethod
from typing import Generic, TypeVar
from uuid import UUID

T = TypeVar("T")


class Repository(ABC, Generic[T]):
    """Interface base para repositórios."""

    @abstractmethod
    async def get_by_id(self, id: UUID) -> T | None:
        """Busca entidade por ID."""
        ...

    @abstractmethod
    async def save(self, entity: T) -> T:
        """Salva ou atualiza entidade."""
        ...

    @abstractmethod
    async def delete(self, id: UUID) -> bool:
        """Remove entidade por ID."""
        ...
`
	os.WriteFile(filepath.Join(basePath, "domain", "repositories", "base.py"), []byte(repoExample), 0644)

	// Criar exemplo de agent base
	agentExample := `"""Base para agentes de IA."""
from abc import ABC, abstractmethod
from typing import Any


class BaseAgent(ABC):
    """Classe base para agentes de IA."""

    def __init__(self, name: str, model: str = "gpt-4"):
        self.name = name
        self.model = model

    @abstractmethod
    async def run(self, input: str, **kwargs) -> Any:
        """Executa o agente com o input fornecido."""
        ...

    @abstractmethod
    def get_tools(self) -> list:
        """Retorna as ferramentas disponíveis para o agente."""
        ...
`
	os.WriteFile(filepath.Join(basePath, "ai", "agents", "base.py"), []byte(agentExample), 0644)

	// Criar exemplo de tool base
	toolExample := `"""Base para ferramentas de agentes."""
from abc import ABC, abstractmethod
from typing import Any
from pydantic import BaseModel


class BaseTool(ABC):
    """Classe base para ferramentas de agentes."""

    name: str
    description: str

    @abstractmethod
    async def execute(self, **kwargs) -> Any:
        """Executa a ferramenta."""
        ...

    def to_openai_function(self) -> dict:
        """Converte para formato de função OpenAI."""
        return {
            "name": self.name,
            "description": self.description,
            "parameters": self.get_parameters(),
        }

    @abstractmethod
    def get_parameters(self) -> dict:
        """Retorna o schema de parâmetros da ferramenta."""
        ...
`
	os.WriteFile(filepath.Join(basePath, "ai", "tools", "base.py"), []byte(toolExample), 0644)

	// Criar exemplo de prompt template
	promptExample := `"""Templates de prompts."""

SYSTEM_PROMPT = """Você é um assistente útil da Algarys.
Responda de forma clara e objetiva.
"""

AGENT_PROMPT = """Você é um agente especializado em {domain}.

Contexto:
{context}

Ferramentas disponíveis:
{tools}

Tarefa: {task}
"""
`
	os.WriteFile(filepath.Join(basePath, "ai", "prompts", "templates.py"), []byte(promptExample), 0644)

	// Criar exemplo de Temporal activity
	activityExample := `"""Activities do Temporal."""
from temporalio import activity


@activity.defn
async def process_data(data: dict) -> dict:
    """Activity para processar dados."""
    # Implementar lógica de processamento
    return {"status": "processed", "data": data}


@activity.defn
async def call_ai_agent(prompt: str, agent_name: str) -> str:
    """Activity para chamar um agente de IA."""
    # Implementar chamada ao agente
    return f"Response from {agent_name}"
`
	os.WriteFile(filepath.Join(basePath, "temporal", "activities", "ai_activities.py"), []byte(activityExample), 0644)

	// Criar exemplo de Temporal workflow
	workflowExample := `"""Workflows do Temporal."""
from datetime import timedelta
from temporalio import workflow

with workflow.unsafe.imports_passed_through():
    from %s.temporal.activities.ai_activities import process_data, call_ai_agent


@workflow.defn
class AIProcessingWorkflow:
    """Workflow para processamento com IA."""

    @workflow.run
    async def run(self, input_data: dict) -> dict:
        """Executa o workflow."""
        # Processar dados
        processed = await workflow.execute_activity(
            process_data,
            input_data,
            start_to_close_timeout=timedelta(minutes=5),
        )

        # Chamar agente
        response = await workflow.execute_activity(
            call_ai_agent,
            args=["Analyze this data", "analyst"],
            start_to_close_timeout=timedelta(minutes=10),
        )

        return {"processed": processed, "ai_response": response}
`
	os.WriteFile(filepath.Join(basePath, "temporal", "workflows", "ai_workflow.py"), []byte(fmt.Sprintf(workflowExample, moduleName)), 0644)

	// Criar exemplo de Temporal worker
	workerExample := `"""Worker do Temporal."""
import asyncio
from temporalio.client import Client
from temporalio.worker import Worker

from %s.temporal.activities.ai_activities import process_data, call_ai_agent
from %s.temporal.workflows.ai_workflow import AIProcessingWorkflow


async def main():
    """Inicia o worker do Temporal."""
    client = await Client.connect("localhost:7233")

    worker = Worker(
        client,
        task_queue="ai-processing-queue",
        workflows=[AIProcessingWorkflow],
        activities=[process_data, call_ai_agent],
    )

    print("Worker iniciado. Aguardando tarefas...")
    await worker.run()


if __name__ == "__main__":
    asyncio.run(main())
`
	os.WriteFile(filepath.Join(basePath, "temporal", "worker", "main.py"), []byte(fmt.Sprintf(workerExample, moduleName, moduleName)), 0644)

	// Criar .gitkeep em pastas que podem ficar vazias
	gitkeepDirs := []string{
		filepath.Join(basePath, "ai", "notebooks"),
		filepath.Join(basePath, "ai", "data"),
		filepath.Join(basePath, "ai", "configs"),
		filepath.Join(basePath, "ai", "evaluations"),
	}

	for _, dir := range gitkeepDirs {
		os.WriteFile(filepath.Join(dir, ".gitkeep"), []byte(""), 0644)
	}
}

func createConfigFiles(projectName, moduleName, description, pythonVersion string) {
	// pyproject.toml
	pyproject := fmt.Sprintf(`[project]
name = "%s"
version = "0.1.0"
description = "%s"
readme = "README.md"
requires-python = ">=%s"
dependencies = [
    "pydantic>=2.0.0",
    "httpx>=0.25.0",
]

[project.optional-dependencies]
ai = [
    "openai>=1.0.0",
    "anthropic>=0.18.0",
    "langchain>=0.1.0",
    "langsmith>=0.1.0",
]
temporal = [
    "temporalio>=1.4.0",
]
dev = [
    "pytest>=8.0.0",
    "pytest-cov>=4.1.0",
    "pytest-asyncio>=0.23.0",
    "ruff>=0.1.0",
    "mypy>=1.8.0",
]
all = [
    "%s[ai,temporal,dev]",
]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
packages = ["%s"]

[tool.ruff]
target-version = "py312"
line-length = 88
src = ["%s", "tests"]

[tool.ruff.lint]
select = ["E", "F", "I", "N", "W", "UP", "B", "C4", "SIM"]

[tool.mypy]
python_version = "%s"
strict = true
warn_return_any = true
warn_unused_ignores = true

[tool.pytest.ini_options]
testpaths = ["tests"]
pythonpath = ["."]
asyncio_mode = "auto"
`, projectName, description, pythonVersion, projectName, moduleName, moduleName, pythonVersion)

	os.WriteFile(filepath.Join(projectName, "pyproject.toml"), []byte(pyproject), 0644)

	// .python-version
	os.WriteFile(filepath.Join(projectName, ".python-version"), []byte(pythonVersion+"\n"), 0644)

	// .gitignore
	gitignore := `# Python
__pycache__/
*.py[cod]
*$py.class
*.so
.Python
build/
develop-eggs/
dist/
downloads/
eggs/
.eggs/
lib/
lib64/
parts/
sdist/
var/
wheels/
*.egg-info/
.installed.cfg
*.egg

# Virtual environments
.venv/
venv/
ENV/

# UV
.uv/

# IDE
.idea/
.vscode/
*.swp
*.swo

# Testing
.coverage
htmlcov/
.pytest_cache/
.mypy_cache/

# Environment
.env
.env.local
*.env

# OS
.DS_Store
Thumbs.db

# Jupyter
.ipynb_checkpoints/
*.ipynb_checkpoints

# AI/ML
*.pt
*.pth
*.onnx
*.safetensors
mlruns/
wandb/

# Data
*.csv
*.parquet
*.json
!ai/configs/*.json
`
	os.WriteFile(filepath.Join(projectName, ".gitignore"), []byte(gitignore), 0644)

	// .env.example
	envExample := `# OpenAI
OPENAI_API_KEY=sk-...

# Anthropic
ANTHROPIC_API_KEY=sk-ant-...

# Temporal
TEMPORAL_HOST=localhost:7233
TEMPORAL_NAMESPACE=default

# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/db
`
	os.WriteFile(filepath.Join(projectName, ".env.example"), []byte(envExample), 0644)

	// README.md
	readme := fmt.Sprintf(`# %s

%s

## Estrutura do Projeto

`+"```"+`
%s/
├── domain/              # Camada de domínio
│   ├── entities/        # Entidades e agregados
│   ├── repositories/    # Interfaces de repositório
│   └── value_objects/   # Objetos de valor
├── application/         # Camada de aplicação
│   ├── services/        # Serviços de aplicação
│   ├── use_cases/       # Casos de uso
│   └── dtos/            # Data Transfer Objects
├── infrastructure/      # Camada de infraestrutura
│   ├── database/        # Configuração de banco
│   ├── external/        # Integrações externas
│   └── repositories/    # Implementações de repositório
├── interfaces/          # Camada de interface
│   ├── api/             # Controllers REST
│   └── cli/             # Comandos CLI
├── ai/                  # Módulo de IA
│   ├── agents/          # Agentes de IA
│   ├── tools/           # Ferramentas para agentes
│   ├── prompts/         # Templates de prompts
│   ├── models/          # Modelos e schemas
│   ├── notebooks/       # Jupyter notebooks
│   ├── workflows/       # Pipelines de agentes
│   ├── evaluations/     # Métricas e testes
│   ├── data/            # Datasets
│   └── configs/         # Configurações
└── temporal/            # Orquestração Temporal
    ├── activities/      # Activities
    ├── workflows/       # Workflows
    └── worker/          # Worker
`+"```"+`

## Desenvolvimento

### Requisitos

- Python %s+
- [UV](https://docs.astral.sh/uv/)
- [Temporal](https://temporal.io/) (para workflows)

### Instalação

`+"```bash"+`
# Instalar todas as dependências
uv sync --all-extras

# Ou apenas o necessário
uv sync                    # básico
uv sync --extra ai         # + libs de IA
uv sync --extra temporal   # + Temporal
`+"```"+`

### Executar

`+"```bash"+`
uv run python -m %s
`+"```"+`

### Temporal Worker

`+"```bash"+`
# Iniciar Temporal server (dev)
temporal server start-dev

# Iniciar worker
uv run python -m %s.temporal.worker.main
`+"```"+`

### Testes

`+"```bash"+`
uv run pytest
uv run pytest --cov
`+"```"+`

### Lint e Type Check

`+"```bash"+`
uv run ruff check .
uv run ruff format .
uv run mypy %s/
`+"```"+`

---

Criado com [Algarys CLI](https://github.com/algarys/algarys)
`, projectName, description, moduleName, pythonVersion, moduleName, moduleName, moduleName)

	os.WriteFile(filepath.Join(projectName, "README.md"), []byte(readme), 0644)
}

func initUV(projectName string) bool {
	if _, err := exec.LookPath("uv"); err != nil {
		printWarning("UV não encontrado. Instale com: curl -LsSf https://astral.sh/uv/install.sh | sh")
		return false
	}

	cmd := exec.Command("uv", "sync")
	cmd.Dir = projectName
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		printWarning("Erro ao executar uv sync")
		return false
	}

	return true
}

func createGitHubRepo(projectName, description, org string) bool {
	if _, err := exec.LookPath("gh"); err != nil {
		printWarning("GitHub CLI não encontrado. Instale com: brew install gh")
		printWarning("Depois autentique com: gh auth login")
		return false
	}

	authCheck := exec.Command("gh", "auth", "status")
	authCheck.Stdout = nil
	authCheck.Stderr = nil
	if err := authCheck.Run(); err != nil {
		printWarning("GitHub CLI não autenticado. Execute: gh auth login")
		return false
	}

	// Nome do repo segue padrão da org: algarys_nome-do-projeto
	repoName := fmt.Sprintf("algarys_%s", projectName)

	args := []string{
		"repo", "create",
		fmt.Sprintf("%s/%s", org, repoName),
		"--private",
		"--source", ".",
		"--push",
	}

	if description != "" {
		args = append(args, "--description", description)
	}

	cmd := exec.Command("gh", args...)
	cmd.Dir = projectName
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		printWarning(fmt.Sprintf("Erro ao criar repo. Verifique se você tem acesso à org '%s'", org))
		printWarning(fmt.Sprintf("Você pode criar manualmente: gh repo create %s/%s --private --source . --push", org, repoName))
		return false
	}

	return true
}

func initLocalGit(projectName string) {
	cmds := [][]string{
		{"git", "init", "-q", "-b", "main"},
		{"git", "add", "."},
		{"git", "commit", "-q", "-m", "Initial commit - Algarys project structure"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = projectName
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Run()
	}
}

func configureRuleset(repoName, org string) bool {
	// Ruleset JSON: exige PR (1 approval) e linear history na branch main
	rulesetJSON := `{
		"name": "Protect main",
		"target": "branch",
		"enforcement": "active",
		"conditions": {
			"ref_name": {
				"include": ["refs/heads/main"],
				"exclude": []
			}
		},
		"rules": [
			{
				"type": "pull_request",
				"parameters": {
					"required_approving_review_count": 1,
					"dismiss_stale_reviews_on_push": false,
					"require_code_owner_review": false,
					"require_last_push_approval": false,
					"required_review_thread_resolution": false
				}
			},
			{
				"type": "required_linear_history"
			}
		]
	}`

	cmd := exec.Command("gh", "api",
		fmt.Sprintf("/repos/%s/%s/rulesets", org, repoName),
		"-X", "POST",
		"-H", "Accept: application/vnd.github+json",
		"--input", "-",
	)
	cmd.Stdin = strings.NewReader(rulesetJSON)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}
