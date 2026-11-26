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
