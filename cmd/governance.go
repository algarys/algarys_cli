package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

const mainSourceCheck = "only-dev-source"

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
	if _, err := runCommand(projectName, "", "uv", "run", "python", "scripts/gerar_contrato.py"); err != nil {
		return fmt.Errorf("gerar openapi.json: %w", err)
	}
	return nil
}

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

type rulesetSpec struct {
	name         string
	branch       string
	checks       []string
	strict       bool
	mergeMethods []string
}

// O context do check é o nome do job, não "<workflow> / <job>".
func rulesetSpecs(temFrontend bool) []rulesetSpec {
	ciChecks := []string{"back", "stack", "imagem"}
	if temFrontend {
		ciChecks = append(ciChecks, "front")
	}
	return []rulesetSpec{
		{
			name: "Protect dev", branch: "dev",
			checks: ciChecks, strict: true, mergeMethods: []string{"squash"},
		},
		{
			// strict desligado: cada promoção por merge commit cria um commit na main que a dev não tem.
			name: "Protect main", branch: "main",
			checks: append([]string{mainSourceCheck}, ciChecks...), strict: false, mergeMethods: []string{"merge"},
		},
	}
}

func configureRulesets(repoName, org string, temFrontend bool) error {
	endpoint := fmt.Sprintf("/repos/%s/%s/rulesets", org, repoName)
	for _, spec := range rulesetSpecs(temFrontend) {
		payload, err := buildRulesetPayload(spec)
		if err != nil {
			return fmt.Errorf("gerar payload do ruleset %s: %w", spec.name, err)
		}
		if err := ghAPI("POST", endpoint, payload); err != nil {
			return fmt.Errorf("criar ruleset %s: %w", spec.name, err)
		}
	}
	return nil
}

func ghAPI(method, endpoint, payload string) error {
	_, err := runCommand(
		"",
		payload,
		"gh",
		"api", endpoint,
		"-X", method,
		"-H", "Accept: application/vnd.github+json",
		"--input", "-",
	)
	return err
}

func buildRulesetPayload(spec rulesetSpec) (string, error) {
	checks := make([]map[string]string, 0, len(spec.checks))
	for _, check := range spec.checks {
		checks = append(checks, map[string]string{"context": check})
	}
	rules := []map[string]any{
		{
			"type": "pull_request",
			"parameters": map[string]any{
				"required_approving_review_count":   1,
				"dismiss_stale_reviews_on_push":     false,
				"require_code_owner_review":         false,
				"require_last_push_approval":        false,
				"required_review_thread_resolution": false,
				"allowed_merge_methods":             spec.mergeMethods,
			},
		},
		{
			"type": "required_status_checks",
			"parameters": map[string]any{
				"do_not_enforce_on_create":             false,
				"strict_required_status_checks_policy": spec.strict,
				"required_status_checks":               checks,
			},
		},
	}

	payload := map[string]any{
		"name":        spec.name,
		"target":      "branch",
		"enforcement": "active",
		"conditions": map[string]any{
			"ref_name": map[string]any{
				"include": []string{fmt.Sprintf("refs/heads/%s", spec.branch)},
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
