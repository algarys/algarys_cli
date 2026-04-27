package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/algarys/algarys_cli/cmd/ui"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
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
	fmt.Println()
	fmt.Println(ui.RenderBanner())
	fmt.Println()

	// Subtítulo
	subtitle := lipgloss.NewStyle().
		Foreground(ui.TextDim).
		Italic(true).
		Render("  Criando novo projeto Python com estrutura SOLID + AI")
	fmt.Println(subtitle)
	fmt.Println()

	config := ProjectConfig{
		GitHubOrg: "algarys",
	}

	// Tema customizado para o formulário
	theme := huh.ThemeBase()
	theme.Focused.Title = theme.Focused.Title.Foreground(ui.Primary)
	theme.Focused.SelectedOption = theme.Focused.SelectedOption.Foreground(ui.Primary)
	theme.Focused.SelectSelector = theme.Focused.SelectSelector.Foreground(ui.Primary)
	theme.Blurred.Title = theme.Blurred.Title.Foreground(ui.TextDim)

	// Formulário interativo
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("📦 Nome do projeto").
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
				Title("📝 Descrição").
				Description("Uma breve descrição do projeto").
				Placeholder("Agente de IA para...").
				Value(&config.Description),

			huh.NewSelect[string]().
				Title("🐍 Versão do Python").
				Options(
					huh.NewOption("Python 3.12 (Recomendado)", "3.12"),
					huh.NewOption("Python 3.11", "3.11"),
					huh.NewOption("Python 3.10", "3.10"),
				).
				Value(&config.PythonVersion),

			huh.NewConfirm().
				Title("🐙 Criar repositório no GitHub?").
				Description("Será criado em github.com/algarys (requer gh auth login)").
				Affirmative("Sim").
				Negative("Não").
				Value(&config.CreateGitHub),
		),
	).WithTheme(theme)

	err := form.Run()
	if err != nil {
		if err.Error() == "user aborted" {
			fmt.Println()
			fmt.Println(ui.RenderWarning("Cancelado pelo usuário"))
			return
		}
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}

	// Normalizar nome do projeto
	config.Name = strings.ToLower(strings.ReplaceAll(config.Name, " ", "-"))
	moduleName := strings.ReplaceAll(config.Name, "-", "_")

	fmt.Println()

	// Header do projeto
	projectHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.Primary).
		Render(fmt.Sprintf("  %s Criando projeto: %s", ui.IconRocket, config.Name))
	fmt.Println(projectHeader)
	fmt.Println()

	// Verificar se diretório já existe
	if _, err := os.Stat(config.Name); !os.IsNotExist(err) {
		fmt.Println(ui.RenderError(fmt.Sprintf("Diretório '%s' já existe", config.Name)))
		os.Exit(1)
	}

	// Criar diretório do projeto
	if err := os.MkdirAll(config.Name, 0755); err != nil {
		fmt.Println(ui.RenderError(fmt.Sprintf("Erro ao criar diretório: %v", err)))
		os.Exit(1)
	}

	// Executar etapas com spinners
	steps := []struct {
		icon    string
		message string
		action  func() bool
	}{
		{ui.IconFolder, "Criando estrutura de diretórios", func() bool {
			createProjectStructure(config.Name, moduleName)
			return true
		}},
		{ui.IconFile, "Gerando arquivos de configuração", func() bool {
			createConfigFiles(config.Name, moduleName, config.Description, config.PythonVersion)
			return true
		}},
		{ui.IconGit, "Inicializando repositório Git", func() bool {
			initLocalGit(config.Name)
			return true
		}},
		{ui.IconPython, "Configurando ambiente UV", func() bool {
			return initUV(config.Name)
		}},
	}

	for _, step := range steps {
		spinner := ui.NewSpinner(step.icon + "  " + step.message)
		spinner.Start()
		time.Sleep(300 * time.Millisecond) // Pequeno delay para visual

		success := step.action()

		if success {
			spinner.Success(step.message)
		} else {
			spinner.Warning(step.message + " (pulado)")
		}
	}

	// Criar repositório no GitHub
	if config.CreateGitHub {
		// Verificar se está autenticado
		if !IsLoggedIn() {
			fmt.Println()
			fmt.Println(ui.RenderWarning("Você precisa estar autenticado para criar repos na org."))
			fmt.Println(lipgloss.NewStyle().Foreground(ui.Primary).PaddingLeft(4).Render("Execute: algarys login"))
			fmt.Println()
		} else {
			repoName := fmt.Sprintf("algarys_%s", config.Name)

			spinner := ui.NewSpinner(ui.IconGitHub + "  Criando repositório no GitHub")
			spinner.Start()
			time.Sleep(300 * time.Millisecond)

			if createGitHubRepo(config.Name, config.Description, config.GitHubOrg) {
				spinner.Success(fmt.Sprintf("Repositório criado: github.com/%s/%s", config.GitHubOrg, repoName))

				// Configurar ruleset
				spinner2 := ui.NewSpinner(ui.IconLock + "  Configurando regras de proteção")
				spinner2.Start()
				time.Sleep(300 * time.Millisecond)

				if configureRuleset(repoName, config.GitHubOrg) {
					spinner2.Success("Ruleset configurado (PR + linear history)")
				} else {
					spinner2.Warning("Ruleset não configurado automaticamente")
				}
			} else {
				spinner.Warning("Repositório não criado (verifique acesso)")
			}
		}
	}

	// Resumo final
	fmt.Println()

	// Success box
	successBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.Primary).
		Padding(1, 2).
		Render(
			lipgloss.NewStyle().Foreground(ui.Primary).Bold(true).Render(
				fmt.Sprintf("%s Projeto %s criado com sucesso!", ui.IconDone, config.Name),
			),
		)
	fmt.Println(successBox)
	fmt.Println()

	// Próximos passos
	nextStepsTitle := lipgloss.NewStyle().
		Foreground(ui.TextDim).
		Bold(true).
		Render("  Próximos passos:")
	fmt.Println(nextStepsTitle)
	fmt.Println()

	cmdStyle := lipgloss.NewStyle().
		Foreground(ui.Primary).
		PaddingLeft(4)

	fmt.Println(cmdStyle.Render(fmt.Sprintf("cd %s", config.Name)))
	fmt.Println(cmdStyle.Render("uv sync --all-extras"))
	fmt.Println(cmdStyle.Render("uv run python -m app"))
	fmt.Println()

	// Dica
	tipStyle := lipgloss.NewStyle().
		Foreground(ui.Muted).
		Italic(true).
		PaddingLeft(2)
	fmt.Println(tipStyle.Render(fmt.Sprintf("%s Dica: use 'algarys --help' para ver outros comandos", ui.IconMagic)))
	fmt.Println()
}

func createProjectStructure(projectName, moduleName string) {
	// Raiz do pacote Python é sempre "app/"
	basePath := filepath.Join(projectName, "app")

	dirs := []string{
		// api/ — módulo principal da API (SOLID)
		filepath.Join(basePath, "api", "application", "dtos"),
		filepath.Join(basePath, "api", "application", "interfaces"),
		filepath.Join(basePath, "api", "application", "services"),
		filepath.Join(basePath, "api", "application", "use_cases"),
		filepath.Join(basePath, "api", "application", "utils"),
		filepath.Join(basePath, "api", "domain", "entities"),
		filepath.Join(basePath, "api", "infrastructure", "cache"),
		filepath.Join(basePath, "api", "infrastructure", "database", "models"),
		filepath.Join(basePath, "api", "infrastructure", "database", "repositories"),
		filepath.Join(basePath, "api", "infrastructure", "external"),
		filepath.Join(basePath, "api", "infrastructure", "messaging"),
		filepath.Join(basePath, "api", "infrastructure", "queue"),
		filepath.Join(basePath, "api", "infrastructure", "storage"),
		filepath.Join(basePath, "api", "presentation", "http", "routes"),
		filepath.Join(basePath, "api", "presentation", "http", "schemas"),
		filepath.Join(basePath, "api", "presentation", "webhooks"),

		// ia/ — módulo de IA e agentes
		filepath.Join(basePath, "ia", "activities"),
		filepath.Join(basePath, "ia", "adapters"),
		filepath.Join(basePath, "ia", "agents", "tools", "catalog"),
		filepath.Join(basePath, "ia", "agents", "tools", "conversational"),
		filepath.Join(basePath, "ia", "agents", "tools", "_shared"),
		filepath.Join(basePath, "ia", "models"),
		filepath.Join(basePath, "ia", "prompts"),
		filepath.Join(basePath, "ia", "services"),
		filepath.Join(basePath, "ia", "utils"),
		filepath.Join(basePath, "ia", "workers"),
		filepath.Join(basePath, "ia", "workflows"),

		// core/, interfaces/, shared/
		filepath.Join(basePath, "core"),
		filepath.Join(basePath, "interfaces", "workers"),
		filepath.Join(basePath, "shared", "contracts"),

		// tests/
		filepath.Join(projectName, "tests", "unit"),
		filepath.Join(projectName, "tests", "integration"),
	}

	for _, dir := range dirs {
		os.MkdirAll(dir, 0755)
	}

	initFiles := []string{
		filepath.Join(basePath, "__init__.py"),
		filepath.Join(basePath, "__main__.py"),

		// api/
		filepath.Join(basePath, "api", "__init__.py"),
		filepath.Join(basePath, "api", "application", "__init__.py"),
		filepath.Join(basePath, "api", "application", "dtos", "__init__.py"),
		filepath.Join(basePath, "api", "application", "interfaces", "__init__.py"),
		filepath.Join(basePath, "api", "application", "services", "__init__.py"),
		filepath.Join(basePath, "api", "application", "use_cases", "__init__.py"),
		filepath.Join(basePath, "api", "application", "utils", "__init__.py"),
		filepath.Join(basePath, "api", "domain", "__init__.py"),
		filepath.Join(basePath, "api", "domain", "entities", "__init__.py"),
		filepath.Join(basePath, "api", "infrastructure", "__init__.py"),
		filepath.Join(basePath, "api", "infrastructure", "cache", "__init__.py"),
		filepath.Join(basePath, "api", "infrastructure", "database", "__init__.py"),
		filepath.Join(basePath, "api", "infrastructure", "database", "models", "__init__.py"),
		filepath.Join(basePath, "api", "infrastructure", "database", "repositories", "__init__.py"),
		filepath.Join(basePath, "api", "infrastructure", "external", "__init__.py"),
		filepath.Join(basePath, "api", "infrastructure", "messaging", "__init__.py"),
		filepath.Join(basePath, "api", "infrastructure", "queue", "__init__.py"),
		filepath.Join(basePath, "api", "infrastructure", "storage", "__init__.py"),
		filepath.Join(basePath, "api", "presentation", "__init__.py"),
		filepath.Join(basePath, "api", "presentation", "http", "__init__.py"),
		filepath.Join(basePath, "api", "presentation", "http", "routes", "__init__.py"),
		filepath.Join(basePath, "api", "presentation", "http", "schemas", "__init__.py"),
		filepath.Join(basePath, "api", "presentation", "webhooks", "__init__.py"),

		// ia/
		filepath.Join(basePath, "ia", "__init__.py"),
		filepath.Join(basePath, "ia", "activities", "__init__.py"),
		filepath.Join(basePath, "ia", "adapters", "__init__.py"),
		filepath.Join(basePath, "ia", "agents", "__init__.py"),
		filepath.Join(basePath, "ia", "agents", "tools", "__init__.py"),
		filepath.Join(basePath, "ia", "agents", "tools", "catalog", "__init__.py"),
		filepath.Join(basePath, "ia", "agents", "tools", "conversational", "__init__.py"),
		filepath.Join(basePath, "ia", "agents", "tools", "_shared", "__init__.py"),
		filepath.Join(basePath, "ia", "models", "__init__.py"),
		filepath.Join(basePath, "ia", "prompts", "__init__.py"),
		filepath.Join(basePath, "ia", "services", "__init__.py"),
		filepath.Join(basePath, "ia", "utils", "__init__.py"),
		filepath.Join(basePath, "ia", "workers", "__init__.py"),
		filepath.Join(basePath, "ia", "workflows", "__init__.py"),

		// core/, interfaces/, shared/
		filepath.Join(basePath, "core", "__init__.py"),
		filepath.Join(basePath, "interfaces", "__init__.py"),
		filepath.Join(basePath, "interfaces", "workers", "__init__.py"),
		filepath.Join(basePath, "shared", "__init__.py"),
		filepath.Join(basePath, "shared", "contracts", "__init__.py"),

		// tests/
		filepath.Join(projectName, "tests", "__init__.py"),
		filepath.Join(projectName, "tests", "unit", "__init__.py"),
		filepath.Join(projectName, "tests", "integration", "__init__.py"),
	}

	for _, f := range initFiles {
		os.WriteFile(f, []byte(""), 0644)
	}

	mainContent := fmt.Sprintf(`"""Ponto de entrada de %s."""


def main() -> None:
    print("Hello from %s!")


if __name__ == "__main__":
    main()
`, moduleName, moduleName)
	os.WriteFile(filepath.Join(basePath, "__main__.py"), []byte(mainContent), 0644)

	entityExample := `"""Entidade base do domínio."""
from dataclasses import dataclass, field
from datetime import datetime
from uuid import UUID, uuid4


@dataclass
class BaseEntity:
    id: UUID = field(default_factory=uuid4)
    created_at: datetime = field(default_factory=datetime.utcnow)
    updated_at: datetime = field(default_factory=datetime.utcnow)
`
	os.WriteFile(filepath.Join(basePath, "api", "domain", "entities", "base.py"), []byte(entityExample), 0644)

	repoInterfaceExample := `"""Interfaces de repositório."""
from abc import ABC, abstractmethod
from typing import Generic, TypeVar
from uuid import UUID

T = TypeVar("T")


class Repository(ABC, Generic[T]):
    @abstractmethod
    async def get_by_id(self, id: UUID) -> T | None: ...

    @abstractmethod
    async def save(self, entity: T) -> T: ...

    @abstractmethod
    async def delete(self, id: UUID) -> bool: ...
`
	os.WriteFile(filepath.Join(basePath, "api", "application", "interfaces", "repository.py"), []byte(repoInterfaceExample), 0644)

	coreExample := `"""Exceções e configurações globais."""


class AppException(Exception):
    def __init__(self, message: str, code: str = "INTERNAL_ERROR"):
        self.message = message
        self.code = code
        super().__init__(message)


class NotFoundError(AppException):
    def __init__(self, resource: str, id: str):
        super().__init__(f"{resource} '{id}' não encontrado.", code="NOT_FOUND")


class ValidationError(AppException):
    def __init__(self, message: str):
        super().__init__(message, code="VALIDATION_ERROR")
`
	os.WriteFile(filepath.Join(basePath, "core", "exceptions.py"), []byte(coreExample), 0644)

	agentExample := `"""Base para agentes de IA."""
from abc import ABC, abstractmethod
from typing import Any


class BaseAgent(ABC):
    def __init__(self, name: str, model: str = "gpt-4o"):
        self.name = name
        self.model = model

    @abstractmethod
    async def run(self, input: str, **kwargs) -> Any: ...

    @abstractmethod
    def get_tools(self) -> list: ...
`
	os.WriteFile(filepath.Join(basePath, "ia", "agents", "base.py"), []byte(agentExample), 0644)

	toolExample := `"""Base para ferramentas de agentes."""
from abc import ABC, abstractmethod
from typing import Any


class BaseTool(ABC):
    name: str
    description: str

    @abstractmethod
    async def execute(self, **kwargs) -> Any: ...

    @abstractmethod
    def get_parameters(self) -> dict: ...

    def to_openai_function(self) -> dict:
        return {
            "name": self.name,
            "description": self.description,
            "parameters": self.get_parameters(),
        }
`
	os.WriteFile(filepath.Join(basePath, "ia", "agents", "tools", "_shared", "base.py"), []byte(toolExample), 0644)

	contractExample := `"""Contratos de mensagens entre serviços."""
from typing import Protocol, runtime_checkable


@runtime_checkable
class Runnable(Protocol):
    async def run(self, *args, **kwargs): ...
`
	os.WriteFile(filepath.Join(basePath, "shared", "contracts", "base.py"), []byte(contractExample), 0644)

	// .gitkeep em pastas de artefatos não-Python
	for _, dir := range []string{
		filepath.Join(basePath, "ia", "prompts"),
		filepath.Join(basePath, "api", "infrastructure", "storage"),
	} {
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
packages = ["app"]

[tool.ruff]
target-version = "py312"
line-length = 88
src = ["app", "tests"]

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
`, projectName, description, pythonVersion, projectName, pythonVersion)

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
app/
├── api/                           # Módulo principal da API
│   ├── application/
│   │   ├── dtos/
│   │   ├── interfaces/
│   │   ├── services/
│   │   ├── use_cases/             # Subpastas por contexto (ex: medware/)
│   │   └── utils/
│   ├── domain/
│   │   └── entities/
│   ├── infrastructure/
│   │   ├── cache/
│   │   ├── database/
│   │   │   ├── models/
│   │   │   └── repositories/
│   │   ├── external/              # Clientes de APIs externas (subpastas por contexto)
│   │   ├── messaging/
│   │   ├── queue/
│   │   └── storage/
│   └── presentation/
│       ├── http/
│       │   ├── routes/            # Rotas agrupadas por contexto
│       │   └── schemas/
│       └── webhooks/
├── ia/                            # Módulo de IA e agentes
│   ├── activities/
│   ├── adapters/
│   ├── agents/
│   │   └── tools/
│   │       ├── catalog/
│   │       ├── conversational/
│   │       └── _shared/
│   ├── models/
│   ├── prompts/
│   ├── services/
│   ├── utils/
│   ├── workers/
│   └── workflows/
├── core/                          # Exceções e configs globais
├── interfaces/
│   └── workers/                   # Workers de mídia e jobs periódicos
└── shared/
    └── contracts/                 # Contratos de mensagens entre serviços
`+"```"+`

## Desenvolvimento

### Requisitos

- Python %s+
- [UV](https://docs.astral.sh/uv/)

### Instalação

`+"```bash"+`
uv sync --all-extras
`+"```"+`

### Executar

`+"```bash"+`
uv run python -m app
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
uv run mypy app/
`+"```"+`

---

Criado com [Algarys CLI](https://github.com/algarys/algarys_cli)
`, projectName, description, pythonVersion)

	os.WriteFile(filepath.Join(projectName, "README.md"), []byte(readme), 0644)
}

func initUV(projectName string) bool {
	if _, err := exec.LookPath("uv"); err != nil {
		return false
	}

	cmd := exec.Command("uv", "sync")
	cmd.Dir = projectName
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

func createGitHubRepo(projectName, description, org string) bool {
	if _, err := exec.LookPath("gh"); err != nil {
		return false
	}

	authCheck := exec.Command("gh", "auth", "status")
	authCheck.Stdout = nil
	authCheck.Stderr = nil
	if err := authCheck.Run(); err != nil {
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
