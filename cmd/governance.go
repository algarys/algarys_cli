package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const mainSourceCheck = "Enforce dev as main source / only-dev-source"

// commandExecutor permite testar a integração com git, gh e uv sem executar
// processos reais.
type commandExecutor func(dir, stdin, name string, args ...string) (string, error)

var runCommand commandExecutor = executeCommand

func executeCommand(dir, stdin, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err == nil {
		return string(output), nil
	}

	command := strings.Join(append([]string{name}, args...), " ")
	detail := strings.TrimSpace(stderr.String())
	if detail == "" {
		return "", fmt.Errorf("%s: %w", command, err)
	}
	return "", fmt.Errorf("%s: %w: %s", command, err, detail)
}

func initLocalGit(projectName string) error {
	commands := [][]string{
		{"init", "-q", "-b", "main"},
		{"add", "."},
		{"commit", "-q", "-m", "Initial commit - Algarys project structure"},
		{"branch", "dev"},
	}

	for _, args := range commands {
		if _, err := runCommand(projectName, "", "git", args...); err != nil {
			return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		}
	}
	return nil
}

func initUV(projectName string) error {
	if _, err := runCommand(projectName, "", "uv", "sync"); err != nil {
		return fmt.Errorf("uv sync: %w", err)
	}
	return nil
}

func createGovernanceFiles(projectName string) error {
	workflowDir := filepath.Join(projectName, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return fmt.Errorf("criar diretório de workflows: %w", err)
	}

	files := map[string]string{
		filepath.Join(workflowDir, "enforce-dev-source-to-main.yml"):      devSourceWorkflow,
		filepath.Join(projectName, ".github", "pull_request_template.md"): pullRequestTemplate,
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("escrever %s: %w", path, err)
		}
	}
	return nil
}

const devSourceWorkflow = `name: Enforce dev as main source

on:
  pull_request_target:
    branches:
      - main
    types:
      - opened
      - synchronize
      - reopened
      - ready_for_review

permissions:
  contents: read

jobs:
  only-dev-source:
    name: only-dev-source
    runs-on: ubuntu-latest
    timeout-minutes: 5
    steps:
      - name: Validate source branch
        env:
          BASE_REF: ${{ github.event.pull_request.base.ref }}
          HEAD_REF: ${{ github.event.pull_request.head.ref }}
          HEAD_REPOSITORY: ${{ github.event.pull_request.head.repo.full_name }}
          CURRENT_REPOSITORY: ${{ github.repository }}
        run: |
          set -euo pipefail

          if [[ "${BASE_REF}" != "main" ]]; then
            echo "Unexpected base branch: ${BASE_REF}"
            exit 1
          fi

          if [[ "${HEAD_REF}" != "dev" || "${HEAD_REPOSITORY}" != "${CURRENT_REPOSITORY}" ]]; then
            echo "Only the dev branch from this repository may be merged into main."
            echo "Received: ${HEAD_REPOSITORY}:${HEAD_REF}"
            exit 1
          fi

          echo "Validated source: ${HEAD_REPOSITORY}:${HEAD_REF} -> ${CURRENT_REPOSITORY}:main"
`

const pullRequestTemplate = `## O que foi feito

<!-- Descreva objetivamente a mudança. -->

## Por que

<!-- Explique o contexto, a motivação e a issue/tarefa relacionada. -->

## Como testar

<!-- Liste os comandos e cenários usados na validação. -->

## Checklist

- [ ] Testes unitários passando
- [ ] Testes de integração executados, quando aplicável
- [ ] Lint e formatação sem erros
- [ ] Documentação atualizada, quando aplicável
- [ ] PR com menos de 1000 linhas, desconsiderando testes e Markdown
`

func createGitHubRepo(projectName, description, org string) error {
	if _, err := runCommand(projectName, "", "gh", "auth", "status"); err != nil {
		return fmt.Errorf("autenticação do GitHub: %w", err)
	}

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

	if _, err := runCommand(projectName, "", "gh", args...); err != nil {
		return fmt.Errorf("criar repositório %s/%s: %w", org, repoName, err)
	}
	if _, err := runCommand(projectName, "", "git", "push", "origin", "dev"); err != nil {
		return fmt.Errorf("publicar branch dev: %w", err)
	}
	return nil
}

func configureRulesets(repoName, org string) error {
	definitions := []struct {
		name           string
		branch         string
		approvals      int
		requiredStatus bool
		linearHistory  bool
	}{
		{
			name: "Protect dev", branch: "dev", approvals: 1,
		},
		{
			name: "Protect main", branch: "main", requiredStatus: true, linearHistory: true,
		},
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/rulesets", org, repoName)
	for _, definition := range definitions {
		payload, err := buildRulesetPayload(
			definition.name,
			definition.branch,
			definition.approvals,
			definition.requiredStatus,
			definition.linearHistory,
		)
		if err != nil {
			return fmt.Errorf("gerar payload do ruleset %s: %w", definition.name, err)
		}

		if _, err := runCommand(
			"",
			payload,
			"gh",
			"api", endpoint,
			"-X", "POST",
			"-H", "Accept: application/vnd.github+json",
			"--input", "-",
		); err != nil {
			return fmt.Errorf("criar ruleset %s: %w", definition.name, err)
		}
	}
	return nil
}

func buildRulesetPayload(name, branch string, approvals int, requiredStatus, linearHistory bool) (string, error) {
	pullRequestParameters := map[string]any{
		"required_approving_review_count":   approvals,
		"dismiss_stale_reviews_on_push":     false,
		"require_code_owner_review":         false,
		"require_last_push_approval":        false,
		"required_review_thread_resolution": false,
	}
	rules := []map[string]any{
		{
			"type":       "pull_request",
			"parameters": pullRequestParameters,
		},
	}
	if requiredStatus {
		rules = append(rules, map[string]any{
			"type": "required_status_checks",
			"parameters": map[string]any{
				"do_not_enforce_on_create":             false,
				"strict_required_status_checks_policy": true,
				"required_status_checks": []map[string]string{
					{"context": mainSourceCheck},
				},
			},
		})
	}
	if linearHistory {
		rules = append(rules, map[string]any{"type": "required_linear_history"})
	}

	payload := map[string]any{
		"name":        name,
		"target":      "branch",
		"enforcement": "active",
		"conditions": map[string]any{
			"ref_name": map[string]any{
				"include": []string{fmt.Sprintf("refs/heads/%s", branch)},
				"exclude": []string{},
			},
		},
		"rules":         rules,
		"bypass_actors": []any{},
	}

	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
