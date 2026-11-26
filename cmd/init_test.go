package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectStepsSyncUVAfterFilesAndBeforeInitialCommit(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "meu-projeto")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	var calls []string
	replaceCommandRunner(t, func(dir, stdin, name string, args ...string) (string, error) {
		if name == "uv" {
			for _, generated := range []string{
				"pyproject.toml",
				filepath.Join(".github", "workflows", "enforce-dev-source-to-main.yml"),
				filepath.Join(".github", "pull_request_template.md"),
				filepath.Join("scripts", "gerar_contrato.py"),
			} {
				if _, err := os.Stat(filepath.Join(dir, generated)); err != nil {
					t.Errorf("%s deveria existir antes do uv sync: %v", generated, err)
				}
			}
		}
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))
		return "", nil
	})

	config := ProjectConfig{Name: projectDir, PythonVersion: "3.12"}
	for _, step := range projectSteps(config, "meu_projeto") {
		if err := step.action(); err != nil {
			t.Fatalf("%s: %v", step.message, err)
		}
	}

	want := []string{
		"uv sync",
		"uv run python scripts/gerar_contrato.py",
		"git init -q -b main",
		"git add .",
		"git commit -q -m Initial commit - Algarys project structure",
		"git branch dev",
	}
	if strings.Join(calls, "\n") != strings.Join(want, "\n") {
		t.Errorf("comandos = %q, want %q", calls, want)
	}
}

func generatedConfigFile(t *testing.T, name string) string {
	t.Helper()
	projectDir := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	createConfigFiles(projectDir, "meu_projeto", "Descrição", "3.12")
	content, err := os.ReadFile(filepath.Join(projectDir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func TestGeneratedProjectDependsOnPostgresAndAlembic(t *testing.T) {
	pyproject := generatedConfigFile(t, "pyproject.toml")
	for _, dep := range []string{`"sqlalchemy[asyncio]>=`, `"asyncpg>=`, `"alembic>=`} {
		if !strings.Contains(pyproject, dep) {
			t.Errorf("pyproject.toml não declara %s", dep)
		}
	}

	envExample := generatedConfigFile(t, ".env.example")
	if !strings.Contains(envExample, "DATABASE_URL=postgresql+asyncpg://") {
		t.Errorf(".env.example deveria usar o driver asyncpg:\n%s", envExample)
	}
}

func TestGeneratedProjectRequiresFullCoverage(t *testing.T) {
	pyproject := generatedConfigFile(t, "pyproject.toml")
	for _, expected := range []string{
		"[tool.coverage.report]\nfail_under = 100",
		`source = ["app", "scripts"]`,
		`"scripts/verificar_stack.py"`,
		`"app/__main__.py"`,
		`"app/ia/*"`,
		`"app/shared/contracts/base.py"`,
		`"app/api/application/interfaces/repository.py"`,
		`"app/api/domain/entities/base.py"`,
	} {
		if !strings.Contains(pyproject, expected) {
			t.Errorf("pyproject.toml não contém %q", expected)
		}
	}
}

func TestProjectStepsBuildImageWithChosenPythonVersion(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "meu-projeto")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	replaceCommandRunner(t, func(dir, stdin, name string, args ...string) (string, error) {
		return "", nil
	})

	config := ProjectConfig{Name: projectDir, PythonVersion: "3.11"}
	for _, step := range projectSteps(config, "meu_projeto") {
		if err := step.action(); err != nil {
			t.Fatalf("%s: %v", step.message, err)
		}
	}

	dockerfile, err := os.ReadFile(filepath.Join(projectDir, "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(dockerfile), "FROM python:3.11-slim\n") {
		t.Errorf("Dockerfile deveria usar o Python escolhido:\n%s", dockerfile)
	}
}

func TestGeneratedGitignoreKeepsOpenAPIContract(t *testing.T) {
	gitignore := generatedConfigFile(t, ".gitignore")
	if !strings.Contains(gitignore, "\n!/openapi.json\n") {
		t.Errorf(".gitignore ignora o openapi.json da raiz:\n%s", gitignore)
	}
}
