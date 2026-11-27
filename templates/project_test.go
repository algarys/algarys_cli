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

func TestRenderProjectGeneratesHomologDeploy(t *testing.T) {
	dest := renderDefault(t)
	deploy := readFile(t, filepath.Join(dest, ".github", "workflows", "deploy.yml"))

	assertContains(t, "deploy.yml", deploy,
		"on:\n  push:\n    branches: [dev, main]",
		"group: deploy-${{ github.ref_name }}\n  cancel-in-progress: false",
		"REGISTRO: ghcr.io/algarys/algarys_meu-projeto",
		`ANTES: ${{ github.event.before }}`,
		`[[ "$ANTES" =~ ^0+$ ]]`,
		"uses: ./.github/workflows/ci.yml",
		"needs: ci",
		"packages: write",
		"tags: ${{ env.REGISTRO }}-api:sha-${{ env.SHA_CURTO }}",
		"cache-from: type=gha,scope=api",
		"name: homolog",
		`if [[ -z "$PORTAINER_URL" ]]; then`,
		"PORTAINER_ACCESS_TOKEN: ${{ secrets.PORTAINER_ACCESS_TOKEN }}",
		`IMAGE_TAG="sha-${GITHUB_SHA::7}" python3 scripts/deploy_portainer.py`,
		"needs.deploy-homolog.result == 'success' && needs.deploy-homolog.outputs.configurada == 'true'",
	)
	if strings.Contains(deploy, "<%") {
		t.Error("deploy.yml ficou com delimitador de template")
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

func TestRenderProjectGeneratesProductionDeploy(t *testing.T) {
	dest := renderDefault(t)
	deploy := readFile(t, filepath.Join(dest, ".github", "workflows", "deploy.yml"))

	assertContains(t, "deploy.yml", deploy,
		"  IMAGENS: api\n",
		"  ci:\n    needs: [criacao, origem]\n",
		"(github.ref_name == 'dev' || needs.origem.outputs.tipo == 'hotfix')",
		"  build:\n    needs: ci\n    if: ${{ !cancelled() && needs.ci.result == 'success' }}",
		"needs.build.result == 'success' && github.ref_name == 'dev' }}",
		`for imagem in $IMAGENS; do`,
		`-t "$REGISTRO-$imagem:homolog" "$REGISTRO-$imagem:sha-${GITHUB_SHA::7}"`,
		"  origem:\n    needs: criacao\n    if: needs.criacao.outputs.nova == 'false' && github.ref_name == 'main'",
		"  promover:\n    needs: [origem, build]\n",
		"  cobrar-volta-dev:\n    needs: origem\n",
		"  deploy-producao:\n    needs: [origem, promover]\n",
		"if: ${{ !cancelled() && needs.promover.result == 'success' }}",
		"      name: producao\n",
		"Environment producao sem PORTAINER_URL: deploy pulado",
		"IMAGE_TAG: ${{ needs.origem.outputs.tag }}\n        run: python3 scripts/deploy_portainer.py",
	)

	comFront := t.TempDir()
	if err := RenderProject(comFront, NewData("meu-projeto", true)); err != nil {
		t.Fatal(err)
	}
	assertContains(t, "deploy.yml com front",
		readFile(t, filepath.Join(comFront, ".github", "workflows", "deploy.yml")), "  IMAGENS: api frontend\n")
}

func TestRenderProjectGeneratesRollback(t *testing.T) {
	dest := renderDefault(t)
	rollback := readFile(t, filepath.Join(dest, ".github", "workflows", "rollback.yml"))

	assertContains(t, "rollback.yml", rollback,
		"on:\n  workflow_dispatch:\n",
		"options: [homolog, producao]",
		"group: deploy-${{ github.ref_name }}\n  cancel-in-progress: false",
		"REGISTRO: ghcr.io/algarys/algarys_meu-projeto\n",
		"IMAGENS: api\n",
		"AMBIENTE: ${{ inputs.ambiente }}",
		"TAG: ${{ inputs.tag }}",
		"  deploy:\n    needs: validar\n",
		"name: ${{ inputs.ambiente }}",
		`IMAGE_TAG="$TAG" python3 scripts/deploy_portainer.py`,
		"  etiqueta:\n    needs: deploy\n",
	)
	if strings.Contains(rollback, "<%") {
		t.Error("rollback.yml ficou com delimitador de template")
	}
}

func TestRenderProjectGeneratesADRs(t *testing.T) {
	adrs := filepath.Join(renderDefault(t), "docs", "adr")

	assertContains(t, "ADR-0001", readFile(t, filepath.Join(adrs, "0001-deploy-pela-api-do-portainer.md")),
		"# ADR-0001", "IMAGE_TAG", "`pullImage: true`", "PORTAINER_ACCESS_TOKEN",
	)
	adr3 := readFile(t, filepath.Join(adrs, "0003-promover-a-imagem-testada.md"))
	assertContains(t, "ADR-0003", adr3,
		"# ADR-0003",
		"1 aprovação de qualquer membro",
		"sem `CODEOWNERS`",
		"merge commit",
		"histórico linear",
		"**Exceção: hotfix.**",
		"`hotfix/* → main`",
	)
	if strings.Contains(adr3, "code owner") {
		t.Error("ADR-0003 não deveria exigir code owner")
	}
	if _, err := os.Stat(filepath.Join(adrs, "0002-duas-imagens-uma-unidade-de-deploy.md")); err == nil {
		t.Error("ADR-0002 só existe com frontend")
	}

	comFront := t.TempDir()
	if err := RenderProject(comFront, NewData("meu-projeto", true)); err != nil {
		t.Fatal(err)
	}
	assertContains(t, "ADR-0002",
		readFile(t, filepath.Join(comFront, "docs", "adr", "0002-duas-imagens-uma-unidade-de-deploy.md")),
		"# ADR-0002", "`…-api` e `…-frontend`",
	)
}

func renderComFront(t *testing.T) string {
	t.Helper()
	dest := t.TempDir()
	if err := RenderProject(dest, NewData("meu-projeto", true)); err != nil {
		t.Fatalf("RenderProject() error = %v", err)
	}
	return dest
}

func TestRenderProjectPutsOnlyFrontendOnProxyNetwork(t *testing.T) {
	compose := readFile(t, filepath.Join(renderComFront(t), "docker-compose.prod.yml"))

	assertContains(t, "docker-compose.prod.yml com front", compose,
		"image: ghcr.io/algarys/algarys_meu-projeto-frontend:${IMAGE_TAG:?",
		"  frontend:\n",
		"      api:\n        condition: service_healthy",
		`test: ["CMD", "wget", "-q", "--spider", "http://127.0.0.1/"]`,
	)
	if t.Failed() {
		return
	}
	api := compose[strings.Index(compose, "  api:\n"):strings.Index(compose, "  frontend:\n")]
	if strings.Contains(api, "nginx-proxy-manager_default") {
		t.Errorf("com front, só o frontend entra na rede do proxy:\n%s", api)
	}
	frontend := compose[strings.Index(compose, "  frontend:\n"):strings.Index(compose, "\nnetworks:")]
	assertContains(t, "serviço frontend", frontend, "- nginx-proxy-manager_default")
	if strings.Contains(compose, "ports:") {
		t.Error("docker-compose.prod.yml não deveria publicar portas")
	}
}

func TestRenderProjectGeneratesFrontend(t *testing.T) {
	front := filepath.Join(renderComFront(t), "frontend")

	assertContains(t, "package.json", readFile(t, filepath.Join(front, "package.json")),
		`"name": "meu-projeto-frontend"`,
		`"lint": "eslint ."`,
		`"types:api": "openapi-typescript ../openapi.json -o src/types/api.ts"`,
		`"@tanstack/react-query"`, `"react": "^19`, `"react-router"`,
		`"prettier"`, `"vite"`,
	)
	assertContains(t, "Dockerfile do front", readFile(t, filepath.Join(front, "Dockerfile")),
		"FROM node:22-alpine AS build",
		"RUN npm ci",
		"FROM nginx:",
		"ARG GIT_SHA=dev",
		`echo "$GIT_SHA" > /usr/share/nginx/html/version.txt`,
	)
	assertContains(t, "nginx.conf", readFile(t, filepath.Join(front, "nginx.conf")),
		"location /api/ {\n        proxy_pass http://api:8000/;",
		"location = /version.txt {\n        add_header Cache-Control \"no-store\";",
		"try_files $uri $uri/ /index.html;",
	)
	assertContains(t, "index.html", readFile(t, filepath.Join(front, "index.html")),
		"<title>meu-projeto</title>", `<div id="root"></div>`,
	)
	for _, arquivo := range []string{
		".gitignore", ".dockerignore", ".prettierrc", "eslint.config.js", "vite.config.ts",
		"tsconfig.json", "src/main.tsx", "src/app/router.tsx", "src/pages/home/HomePage.tsx",
	} {
		if _, err := os.Stat(filepath.Join(front, filepath.FromSlash(arquivo))); err != nil {
			t.Errorf("frontend/%s deveria existir: %v", arquivo, err)
		}
	}
}
