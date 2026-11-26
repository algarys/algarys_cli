package templates

import (
	"io/fs"
	"os"
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

func TestRenderProjectGeneratesAsyncAlembic(t *testing.T) {
	dest := renderDefault(t)

	assertContains(t, "alembic.ini", readFile(t, filepath.Join(dest, "alembic.ini")),
		"script_location = %(here)s/alembic",
	)
	assertContains(t, "alembic/env.py", readFile(t, filepath.Join(dest, "alembic", "env.py")),
		"settings.database_url",
		"async_engine_from_config",
		"target_metadata = Base.metadata",
	)
	if info, err := os.Stat(filepath.Join(dest, "alembic", "versions")); err != nil || !info.IsDir() {
		t.Errorf("alembic/versions deveria existir: %v", err)
	}
	assertContains(t, "Dockerfile", readFile(t, filepath.Join(dest, "Dockerfile")),
		"COPY alembic.ini ./", "COPY alembic ./alembic",
	)
}

func TestRenderProjectGeneratesProductionStack(t *testing.T) {
	dest := renderDefault(t)
	compose := readFile(t, filepath.Join(dest, "docker-compose.prod.yml"))

	assertContains(t, "docker-compose.prod.yml", compose,
		"name: meu-projeto",
		"image: ghcr.io/algarys/algarys_meu-projeto-api:${IMAGE_TAG:?",
		"image: postgres:17-alpine",
		`command: ["alembic", "upgrade", "head"]`,
		"condition: service_completed_successfully",
		"driver: json-file",
		"nginx-proxy-manager_default:\n    external: true",
	)
	for _, forbidden := range []string{"ports:", "temporal", "frontend"} {
		if strings.Contains(compose, forbidden) {
			t.Errorf("docker-compose.prod.yml não deveria conter %q", forbidden)
		}
	}

	assertContains(t, "stack.env.example", readFile(t, filepath.Join(dest, "stack.env.example")),
		"IMAGE_TAG=", "POSTGRES_USER=", "POSTGRES_PASSWORD=", "POSTGRES_DB=",
	)
	assertContains(t, "scripts/verificar_stack.py", readFile(t, filepath.Join(dest, "scripts", "verificar_stack.py")),
		"def problemas(", "publica porta no host",
	)
}

func TestRenderProjectGeneratesCI(t *testing.T) {
	dest := renderDefault(t)
	ci := readFile(t, filepath.Join(dest, ".github", "workflows", "ci.yml"))

	assertContains(t, "ci.yml", ci,
		"on:\n  pull_request:\n",
		"  workflow_call:\n",
		"group: ci-${{ github.workflow }}-${{ github.ref }}",
		"permissions:\n  contents: read",
		"actions/checkout@v7",
		"astral-sh/setup-uv@v10.2.0",
		"image: postgres:17-alpine",
		"uv sync --frozen --all-extras",
		"uv run ruff check .",
		"uv run ruff format --check .",
		"uv run mypy app/",
		"uv run alembic upgrade head",
		"uv run alembic check",
		"uv run python scripts/gerar_contrato.py",
		`git status --porcelain -- openapi.json`,
		"uv run pytest tests --cov\n",
		"docker compose -f docker-compose.prod.yml --env-file stack.env.example config -q",
		"uv run --frozen python scripts/verificar_stack.py",
		"needs: [back, stack]",
		`docker build -t "ghcr.io/algarys/algarys_meu-projeto-api:$IMAGE_TAG"`,
		`--build-arg GIT_SHA="$SHA_CURTO"`,
		"docker network create nginx-proxy-manager_default",
		"up -d --wait",
		"exec -T api",
		`jq -e --arg v "$SHA_CURTO" '.status == "ok" and .version == $v'`,
		"if: failure()",
	)
	for _, forbidden := range []string{"front", "<%"} {
		if strings.Contains(ci, forbidden) {
			t.Errorf("ci.yml não deveria conter %q", forbidden)
		}
	}
}

func TestRenderProjectGeneratesPortainerDeployScript(t *testing.T) {
	casos := map[bool][]string{
		false: {`IMAGENS = ("api",)`, `HEALTH = "/health"`},
		true:  {`IMAGENS = ("api", "frontend")`, `HEALTH = "/api/health"`, `"/version.txt"`},
	}
	for temFrontend, esperado := range casos {
		dest := t.TempDir()
		if err := RenderProject(dest, NewData("meu-projeto", temFrontend)); err != nil {
			t.Fatalf("RenderProject() error = %v", err)
		}
		script := readFile(t, filepath.Join(dest, "scripts", "deploy_portainer.py"))
		assertContains(t, "deploy_portainer.py", script, append(esperado,
			`USER_AGENT = "meu-projeto-deploy/1.0"`,
			`"pullImage": True`,
			"PRAZO_SEGUNDOS = 180",
		)...)
		if !temFrontend && strings.Contains(script, "version.txt") {
			t.Error("deploy_portainer.py sem front não deveria conferir /version.txt")
		}
		for _, teste := range []string{
			filepath.Join("tests", "unit", "scripts", "test_deploy_portainer.py"),
			filepath.Join("tests", "integration", "scripts", "test_deploy_portainer_http.py"),
			filepath.Join("tests", "integration", "scripts", "__init__.py"),
		} {
			if _, err := os.Stat(filepath.Join(dest, teste)); err != nil {
				t.Errorf("%s deveria existir: %v", teste, err)
			}
		}
	}
}
