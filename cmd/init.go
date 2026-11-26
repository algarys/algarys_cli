package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/algarys/algarys_cli/cmd/ui"
	"github.com/algarys/algarys_cli/templates"
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
	initCmd.Flags().StringVar(&aiTeamFlag, "ai-team", defaultAITeamSlug,
		"slug do time da org com permissão admin no repositório criado (vazio desativa; env "+aiTeamEnvVar+")")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) {
	aiTeamFlagSet = cmd.Flags().Changed("ai-team")
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
	steps := projectSteps(config, moduleName)
	for _, step := range steps {
		spinner := ui.NewSpinner(step.icon + "  " + step.message)
		spinner.Start()
		time.Sleep(300 * time.Millisecond) // Pequeno delay para visual

		err := step.action()

		if err == nil {
			spinner.Success(step.message)
		} else {
			spinner.Warning(fmt.Sprintf("%s: %v", step.message, err))
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

			if err := createGitHubRepo(config.Name, config.Description, config.GitHubOrg); err == nil {
				spinner.Success(fmt.Sprintf("Repositório criado: github.com/%s/%s", config.GitHubOrg, repoName))

				if teamSlug := resolveAITeamSlug(); teamSlug != "" {
					spinnerTeam := ui.NewSpinner(ui.IconLock + "  Dando admin ao time " + teamSlug)
					spinnerTeam.Start()
					if err := grantTeamAdmin(config.GitHubOrg, repoName, teamSlug); err == nil {
						spinnerTeam.Success(fmt.Sprintf("Time %s com admin no repositório", teamSlug))
					} else {
						spinnerTeam.Warning(fmt.Sprintf("Time não recebeu admin (siga sem ele; peça a um owner da org): %v", err))
					}
				}

				spinner2 := ui.NewSpinner(ui.IconLock + "  Configurando governança de branches")
				spinner2.Start()
				time.Sleep(300 * time.Millisecond)

				if err := configureRulesets(repoName, config.GitHubOrg); err == nil {
					spinner2.Success("Rulesets dev e main configurados")
				} else {
					spinner2.Warning(fmt.Sprintf("Governança não configurada: %v", err))
				}
			} else {
				spinner.Warning(fmt.Sprintf("Repositório não criado: %v", err))
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

type initStep struct {
	icon    string
	message string
	action  func() error
}

func projectSteps(config ProjectConfig, moduleName string) []initStep {
	return []initStep{
		{ui.IconFolder, "Criando estrutura de diretórios", func() error {
			createProjectStructure(config.Name, moduleName)
			return nil
		}},
		{ui.IconFile, "Gerando arquivos de configuração", func() error {
			createConfigFiles(config.Name, moduleName, config.Description, config.PythonVersion)
			data := templates.NewData(config.Name, false)
			data.PythonVersion = config.PythonVersion
			return templates.RenderProject(config.Name, data)
		}},
		// Antes do Git, para o uv.lock entrar no commit inicial.
		{ui.IconPython, "Configurando ambiente UV", func() error {
			return initUV(config.Name)
		}},
		{ui.IconGit, "Inicializando repositório Git", func() error {
			return initLocalGit(config.Name)
		}},
	}
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

import uvicorn


def main() -> None:
    uvicorn.run(
        "app.api.presentation.http.app:app", host="0.0.0.0", port=8000, reload=True
    )


if __name__ == "__main__":
    main()
`, moduleName)
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


class Repository(ABC, Generic[T]):  # noqa: UP046 (aceita Python < 3.12)
    @abstractmethod
    async def get_by_id(self, id: UUID) -> T | None: ...

    @abstractmethod
    async def save(self, entity: T) -> T: ...

    @abstractmethod
    async def delete(self, id: UUID) -> bool: ...
`
	os.WriteFile(filepath.Join(basePath, "api", "application", "interfaces", "repository.py"), []byte(repoInterfaceExample), 0644)

	coreExample := `"""Exceções e configurações globais."""


class AppError(Exception):
    def __init__(self, message: str, code: str = "INTERNAL_ERROR") -> None:
        self.message = message
        self.code = code
        super().__init__(message)


class NotFoundError(AppError):
    def __init__(self, resource: str, id: str) -> None:
        super().__init__(f"{resource} '{id}' não encontrado.", code="NOT_FOUND")


class ValidationError(AppError):
    def __init__(self, message: str) -> None:
        super().__init__(message, code="VALIDATION_ERROR")
`
	os.WriteFile(filepath.Join(basePath, "core", "exceptions.py"), []byte(coreExample), 0644)

	settingsModule := `"""Configurações da aplicação via pydantic-settings."""

from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    # API
    cors_origins: str = "*"
    git_sha: str = "dev"

    # Logging
    log_level: str = "INFO"

    # OpenAI
    openai_api_key: str = ""

    # Anthropic
    anthropic_api_key: str = ""

    # Temporal
    temporal_host: str = "localhost:7233"
    temporal_namespace: str = "default"

    # Database
    database_url: str = ""


@lru_cache
def get_settings() -> Settings:
    return Settings()


settings = get_settings()
`
	os.WriteFile(filepath.Join(basePath, "core", "settings.py"), []byte(settingsModule), 0644)

	loggingSetup := `"""Configuração centralizada de logging."""

import sys
from collections.abc import Callable
from pathlib import Path
from typing import TYPE_CHECKING

from loguru import logger

from app.core.settings import settings

if TYPE_CHECKING:
    from loguru import Record

LOGS_DIR = Path("logs")
_SUBDIRS = ["debug", "info", "warning", "error", "exception"]
_FMT_FILE = "{time:YYYY-MM-DD HH:mm:ss} | {level: <9} | {name}:{line} - {message}"
_FMT_CONSOLE = (
    "<green>{time:HH:mm:ss}</green> | <level>{level: <9}</level> | "
    "<cyan>{name}</cyan>:<cyan>{line}</cyan> - <level>{message}</level>"
)


def _level_filter(name: str) -> Callable[["Record"], bool]:
    def _filter(record: "Record") -> bool:
        return record["level"].name == name.upper()

    return _filter


def _exception_filter(record: "Record") -> bool:
    return record["exception"] is not None


def setup_logging() -> None:
    for sub in _SUBDIRS:
        (LOGS_DIR / sub).mkdir(parents=True, exist_ok=True)

    logger.remove()

    logger.add(
        sys.stdout, level=settings.log_level.upper(), colorize=True, format=_FMT_CONSOLE
    )

    for lvl in ("debug", "info", "warning", "error"):
        logger.add(
            LOGS_DIR / lvl / "app.log",
            level=lvl.upper(),
            filter=_level_filter(lvl),
            rotation="10 MB",
            retention="30 days",
            encoding="utf-8",
            format=_FMT_FILE,
        )

    logger.add(
        LOGS_DIR / "exception" / "app.log",
        level="ERROR",
        filter=_exception_filter,
        rotation="10 MB",
        retention="30 days",
        encoding="utf-8",
        format=_FMT_FILE,
    )
`
	os.WriteFile(filepath.Join(basePath, "core", "logging.py"), []byte(loggingSetup), 0644)

	agentExample := `"""Base para agentes de IA."""

from abc import ABC, abstractmethod
from typing import Any


class BaseAgent(ABC):
    def __init__(self, name: str, model: str = "gpt-4o"):
        self.name = name
        self.model = model

    @abstractmethod
    async def run(self, input: str, **kwargs: Any) -> Any: ...

    @abstractmethod
    def get_tools(self) -> list[Any]: ...
`
	os.WriteFile(filepath.Join(basePath, "ia", "agents", "base.py"), []byte(agentExample), 0644)

	toolExample := `"""Base para ferramentas de agentes."""

from abc import ABC, abstractmethod
from typing import Any


class BaseTool(ABC):
    name: str
    description: str

    @abstractmethod
    async def execute(self, **kwargs: Any) -> Any: ...

    @abstractmethod
    def get_parameters(self) -> dict[str, Any]: ...

    def to_openai_function(self) -> dict[str, Any]:
        return {
            "name": self.name,
            "description": self.description,
            "parameters": self.get_parameters(),
        }
`
	os.WriteFile(filepath.Join(basePath, "ia", "agents", "tools", "_shared", "base.py"), []byte(toolExample), 0644)

	contractExample := `"""Contratos de mensagens entre serviços."""

from typing import Any, Protocol, runtime_checkable


@runtime_checkable
class Runnable(Protocol):
    async def run(self, *args: Any, **kwargs: Any) -> Any: ...
`
	os.WriteFile(filepath.Join(basePath, "shared", "contracts", "base.py"), []byte(contractExample), 0644)

	healthRoute := `"""Rota de healthcheck."""

from typing import Annotated

from fastapi import APIRouter, Depends
from pydantic import BaseModel

from app.core.settings import Settings, get_settings

router = APIRouter(prefix="/health", tags=["health"])


class HealthResponse(BaseModel):
    status: str
    version: str


@router.get("", response_model=HealthResponse)
async def health(
    settings: Annotated[Settings, Depends(get_settings)],
) -> HealthResponse:
    return HealthResponse(status="ok", version=settings.git_sha)
`
	os.WriteFile(filepath.Join(basePath, "api", "presentation", "http", "routes", "health.py"), []byte(healthRoute), 0644)

	httpApp := fmt.Sprintf(`"""FastAPI application factory."""

from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.api.presentation.http.routes.health import router as health_router
from app.core.logging import setup_logging
from app.core.settings import settings


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    setup_logging()
    yield


app = FastAPI(title="%s", version="0.1.0", lifespan=lifespan)

_raw = settings.cors_origins
origins = [o.strip() for o in _raw.split(",")] if _raw != "*" else ["*"]

app.add_middleware(
    CORSMiddleware,
    allow_origins=origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(health_router)
`, projectName)
	os.WriteFile(filepath.Join(basePath, "api", "presentation", "http", "app.py"), []byte(httpApp), 0644)

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
    "pydantic-settings>=2.0.0",
    "httpx>=0.25.0",
    "fastapi>=0.110.0",
    "uvicorn[standard]>=0.27.0",
    "loguru>=0.7.0",
    "sqlalchemy[asyncio]>=2.0",
    "asyncpg>=0.31.0",
    "alembic>=1.16.0",
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

[tool.ruff.lint.isort]
known-first-party = ["app"]

[tool.mypy]
python_version = "%s"
strict = true
warn_return_any = true
warn_unused_ignores = true

[tool.pytest.ini_options]
testpaths = ["tests"]
pythonpath = ["."]
asyncio_mode = "auto"

[tool.coverage.run]
concurrency = ["greenlet", "thread"]
source = ["app"]
omit = [
    "app/__main__.py",
    "app/ia/*",
    "app/shared/contracts/base.py",
    "app/api/application/interfaces/repository.py",
    "app/api/domain/entities/base.py",
]

[tool.coverage.report]
fail_under = 100
show_missing = true
exclude_also = ['if __name__ == "__main__":']
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

# Logs
logs/

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
!/openapi.json
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
DATABASE_URL=postgresql+asyncpg://user:pass@localhost:5432/db

# CORS — use "*" para liberar tudo (dev) ou lista separada por vírgula (prod)
# Exemplo prod: CORS_ORIGINS=https://app.algarys.com.br,https://admin.algarys.com.br
CORS_ORIGINS=*
`
	os.WriteFile(filepath.Join(projectName, ".env.example"), []byte(envExample), 0644)

	// README.md
	readme := fmt.Sprintf(`# %s

%s

## Governança de branches

Este projeto utiliza o fluxo de desenvolvimento:

feature/* → dev → main

- dev é a branch de integração e requer Pull Request com aprovação.
- main é a fonte de verdade e de release.
- Pull Requests para main devem partir exclusivamente de dev.

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
