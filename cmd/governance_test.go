package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type recordedCommand struct {
	dir   string
	stdin string
	name  string
	args  []string
}

func replaceCommandRunner(t *testing.T, runner commandExecutor) {
	t.Helper()
	previous := runCommand
	runCommand = runner
	t.Cleanup(func() { runCommand = previous })
}

func TestCreateGovernanceFiles(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := createGovernanceFiles(projectDir); err != nil {
		t.Fatalf("createGovernanceFiles() error = %v", err)
	}

	workflowPath := filepath.Join(projectDir, ".github", "workflows", "enforce-dev-source-to-main.yml")
	workflow, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("ler workflow: %v", err)
	}
	workflowText := string(workflow)
	for _, expected := range []string{
		"pull_request_target",
		"HEAD_REPOSITORY",
		"CURRENT_REPOSITORY",
		"Only the dev branch from this repository",
	} {
		if !strings.Contains(workflowText, expected) {
			t.Errorf("workflow não contém %q", expected)
		}
	}
	if strings.Contains(workflowText, "actions/checkout") {
		t.Error("workflow não deve fazer checkout do código da origem")
	}

	templatePath := filepath.Join(projectDir, ".github", "pull_request_template.md")
	template, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("ler template de PR: %v", err)
	}
	for _, expected := range []string{
		"## O que foi feito",
		"## Por que",
		"## Como testar",
		"## Checklist",
	} {
		if !strings.Contains(string(template), expected) {
			t.Errorf("template de PR não contém %q", expected)
		}
	}
}

func TestInitLocalGitCreatesDevAfterInitialCommit(t *testing.T) {
	var calls []recordedCommand
	replaceCommandRunner(t, func(dir, stdin, name string, args ...string) (string, error) {
		calls = append(calls, recordedCommand{dir: dir, stdin: stdin, name: name, args: append([]string(nil), args...)})
		return "", nil
	})

	if err := initLocalGit("project"); err != nil {
		t.Fatalf("initLocalGit() error = %v", err)
	}

	if len(calls) != 4 {
		t.Fatalf("quantidade de comandos = %d, want 4", len(calls))
	}
	want := [][]string{
		{"git", "init", "-q", "-b", "main"},
		{"git", "add", "."},
		{"git", "commit", "-q", "-m", "Initial commit - Algarys project structure"},
		{"git", "branch", "dev"},
	}
	for i, command := range calls {
		got := append([]string{command.name}, command.args...)
		if strings.Join(got, " ") != strings.Join(want[i], " ") {
			t.Errorf("comando %d = %v, want %v", i, got, want[i])
		}
		if command.dir != "project" {
			t.Errorf("diretório do comando %d = %q, want project", i, command.dir)
		}
	}
}

func TestCreateGitHubRepoPushesBothBranches(t *testing.T) {
	var calls []recordedCommand
	replaceCommandRunner(t, func(dir, stdin, name string, args ...string) (string, error) {
		calls = append(calls, recordedCommand{dir: dir, stdin: stdin, name: name, args: append([]string(nil), args...)})
		return "", nil
	})

	if err := createGitHubRepo("meu-projeto", "Descrição", "algarys"); err != nil {
		t.Fatalf("createGitHubRepo() error = %v", err)
	}

	if len(calls) != 3 {
		t.Fatalf("quantidade de comandos = %d, want 3", len(calls))
	}
	if got := strings.Join(append([]string{calls[0].name}, calls[0].args...), " "); got != "gh auth status" {
		t.Errorf("primeiro comando = %q", got)
	}
	if got := strings.Join(append([]string{calls[1].name}, calls[1].args...), " "); !strings.Contains(got, "gh repo create algarys/algarys_meu-projeto") {
		t.Errorf("comando de criação = %q", got)
	}
	if calls[1].dir != "meu-projeto" || calls[2].dir != "meu-projeto" {
		t.Errorf("diretórios = %q e %q, want meu-projeto", calls[1].dir, calls[2].dir)
	}
	if got := strings.Join(append([]string{calls[2].name}, calls[2].args...), " "); got != "git push origin dev" {
		t.Errorf("comando de publicação = %q", got)
	}
}

func TestCreateGitHubRepoReportsCommandFailure(t *testing.T) {
	replaceCommandRunner(t, func(dir, stdin, name string, args ...string) (string, error) {
		if name == "gh" && len(args) > 0 && args[0] == "repo" {
			return "", errors.New("permissão negada")
		}
		return "", nil
	})

	err := createGitHubRepo("meu-projeto", "", "algarys")
	if err == nil || !strings.Contains(err.Error(), "criar repositório algarys/algarys_meu-projeto") {
		t.Fatalf("erro = %v, deveria identificar a etapa de criação", err)
	}
}

func TestRulesetPayloads(t *testing.T) {
	tests := []struct {
		name           string
		branch         string
		approvals      int
		requiredStatus bool
		linearHistory  bool
		wantStatus     bool
		wantLinear     bool
	}{
		{name: "Protect dev", branch: "dev", approvals: 1},
		{name: "Protect main", branch: "main", requiredStatus: true, linearHistory: true, wantStatus: true, wantLinear: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadText, err := buildRulesetPayload(tt.name, tt.branch, tt.approvals, tt.requiredStatus, tt.linearHistory)
			if err != nil {
				t.Fatal(err)
			}

			var payload map[string]any
			if err := json.Unmarshal([]byte(payloadText), &payload); err != nil {
				t.Fatalf("payload inválido: %v", err)
			}
			if payload["name"] != tt.name || payload["target"] != "branch" || payload["enforcement"] != "active" {
				t.Errorf("metadados inesperados: %#v", payload)
			}
			if len(payload["bypass_actors"].([]any)) != 0 {
				t.Error("ruleset não deve possuir bypass actors")
			}

			conditions := payload["conditions"].(map[string]any)
			refName := conditions["ref_name"].(map[string]any)
			if got := refName["include"].([]any)[0]; got != "refs/heads/"+tt.branch {
				t.Errorf("branch protegida = %v", got)
			}

			rules := payload["rules"].([]any)
			foundStatus, foundLinear := false, false
			for _, rawRule := range rules {
				rule := rawRule.(map[string]any)
				switch rule["type"] {
				case "required_status_checks":
					foundStatus = true
					parameters := rule["parameters"].(map[string]any)
					checks := parameters["required_status_checks"].([]any)
					if checks[0].(map[string]any)["context"] != mainSourceCheck {
						t.Errorf("check obrigatório = %#v", checks[0])
					}
				case "required_linear_history":
					foundLinear = true
				}
			}
			if foundStatus != tt.wantStatus || foundLinear != tt.wantLinear {
				t.Errorf("regras status/linear = %v/%v, want %v/%v", foundStatus, foundLinear, tt.wantStatus, tt.wantLinear)
			}
		})
	}
}

func TestConfigureRulesetsUsesDevThenMain(t *testing.T) {
	var calls []recordedCommand
	replaceCommandRunner(t, func(dir, stdin, name string, args ...string) (string, error) {
		calls = append(calls, recordedCommand{dir: dir, stdin: stdin, name: name, args: append([]string(nil), args...)})
		return "", nil
	})

	if err := configureRulesets("algarys_meu-projeto", "algarys"); err != nil {
		t.Fatalf("configureRulesets() error = %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("quantidade de chamadas = %d, want 2", len(calls))
	}
	for i, call := range calls {
		if call.name != "gh" || strings.Join(call.args[:2], " ") != "api /repos/algarys/algarys_meu-projeto/rulesets" {
			t.Errorf("chamada %d inesperada: %#v", i, call)
		}
		if call.stdin == "" {
			t.Errorf("chamada %d sem payload", i)
		}
	}
	if !strings.Contains(calls[0].stdin, `"name": "Protect dev"`) {
		t.Error("primeiro ruleset deveria proteger dev")
	}
	if !strings.Contains(calls[1].stdin, `"name": "Protect main"`) {
		t.Error("segundo ruleset deveria proteger main")
	}
}

func TestConfigureRulesetsReportsAPIFailure(t *testing.T) {
	replaceCommandRunner(t, func(dir, stdin, name string, args ...string) (string, error) {
		if name == "gh" && len(args) > 0 && args[0] == "api" && strings.Contains(stdin, `"name": "Protect main"`) {
			return "", errors.New("API indisponível")
		}
		return "", nil
	})

	err := configureRulesets("algarys_meu-projeto", "algarys")
	if err == nil || !strings.Contains(err.Error(), "criar ruleset Protect main") {
		t.Fatalf("erro = %v, deveria identificar o ruleset da falha", err)
	}
}

func TestCreateGitHubRepoReportsAuthFailure(t *testing.T) {
	replaceCommandRunner(t, func(dir, stdin, name string, args ...string) (string, error) {
		if name == "gh" && len(args) > 1 && args[0] == "auth" && args[1] == "status" {
			return "", errors.New("login necessário")
		}
		return "", nil
	})

	err := createGitHubRepo("meu-projeto", "", "algarys")
	if err == nil || !strings.Contains(err.Error(), "autenticação do GitHub") {
		t.Fatalf("erro = %v, deveria identificar a autenticação", err)
	}
}

func TestCreateGitHubRepoReportsPushFailure(t *testing.T) {
	replaceCommandRunner(t, func(dir, stdin, name string, args ...string) (string, error) {
		if name == "git" && len(args) > 0 && args[0] == "push" {
			return "", errors.New("branch rejeitada")
		}
		return "", nil
	})

	err := createGitHubRepo("meu-projeto", "", "algarys")
	if err == nil || !strings.Contains(err.Error(), "publicar branch dev") {
		t.Fatalf("erro = %v, deveria identificar o push da branch dev", err)
	}
}

func TestGeneratedReadmeDocumentsBranchFlow(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	createConfigFiles(projectDir, "meu_projeto", "Descrição", "3.12")
	readme, err := os.ReadFile(filepath.Join(projectDir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"feature/* → dev → main",
		"dev é a branch de integração",
		"Pull Requests para main devem partir exclusivamente de dev",
	} {
		if !strings.Contains(string(readme), expected) {
			t.Errorf("README não contém %q", expected)
		}
	}
}
