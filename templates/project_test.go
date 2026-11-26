package templates

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderProjectGeneratesGovernanceFiles(t *testing.T) {
	dest := t.TempDir()
	if err := RenderProject(dest, NewData("meu-projeto", false)); err != nil {
		t.Fatalf("RenderProject() error = %v", err)
	}

	workflow := readFile(t, filepath.Join(dest, ".github", "workflows", "enforce-dev-source-to-main.yml"))
	for _, expected := range []string{
		"pull_request_target",
		"- edited",
		`"${HEAD_REF}" != hotfix/*`,
		"${{ github.event.pull_request.head.repo.full_name }}",
		"Only dev or hotfix/* branches from this repository may be merged into main.",
	} {
		if !strings.Contains(workflow, expected) {
			t.Errorf("workflow não contém %q", expected)
		}
	}
	if strings.Contains(workflow, "actions/checkout") {
		t.Error("workflow não deve fazer checkout do código da origem")
	}

	prTemplate := readFile(t, filepath.Join(dest, ".github", "pull_request_template.md"))
	for _, expected := range []string{
		"## O que foi feito",
		"## Checklist",
		"até 400 linhas alteradas",
		"`.ipynb`, `.env*`, `pyproject.toml`, `uv.lock`",
		"comentário/docstring",
	} {
		if !strings.Contains(prTemplate, expected) {
			t.Errorf("template de PR não contém %q", expected)
		}
	}
}

func TestProjectTemplatesNeverGenerateCodeowners(t *testing.T) {
	for _, temFrontend := range []bool{false, true} {
		dest := t.TempDir()
		if err := RenderProject(dest, NewData("p", temFrontend)); err != nil {
			t.Fatalf("RenderProject() error = %v", err)
		}
		err := filepath.WalkDir(dest, func(path string, entry fs.DirEntry, err error) error {
			if err == nil && strings.EqualFold(entry.Name(), "CODEOWNERS") {
				t.Errorf("CODEOWNERS gerado em %s", path)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func renderDefault(t *testing.T) string {
	t.Helper()
	dest := t.TempDir()
	if err := RenderProject(dest, NewData("meu-projeto", false)); err != nil {
		t.Fatalf("RenderProject() error = %v", err)
	}
	return dest
}

func assertContains(t *testing.T, name, content string, expected ...string) {
	t.Helper()
	for _, e := range expected {
		if !strings.Contains(content, e) {
			t.Errorf("%s não contém %q", name, e)
		}
	}
}

func TestRenderProjectGeneratesDockerfile(t *testing.T) {
	dest := renderDefault(t)

	assertContains(t, "Dockerfile", readFile(t, filepath.Join(dest, "Dockerfile")),
		"uv sync --frozen --no-dev",
		"USER app",
		"ARG GIT_SHA=dev",
		"ENV GIT_SHA=$GIT_SHA",
		"HEALTHCHECK",
		"http://localhost:8000/health",
		`"uvicorn", "app.api.presentation.http.app:app"`,
	)
	assertContains(t, ".dockerignore", readFile(t, filepath.Join(dest, ".dockerignore")),
		".env", "tests", "stack.env*",
	)
}
