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
