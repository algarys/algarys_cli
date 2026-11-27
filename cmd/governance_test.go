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

type rulesetExpectation struct {
	checks       []string
	strict       bool
	mergeMethods []string
}

func TestRulesetPayloadsFollowSpec(t *testing.T) {
	tests := []struct {
		name     string
		frontend bool
		want     map[string]rulesetExpectation
	}{
		{
			name: "sem front",
			want: map[string]rulesetExpectation{
				"dev":  {checks: []string{"back", "stack", "imagem"}, strict: true, mergeMethods: []string{"squash"}},
				"main": {checks: []string{"only-dev-source", "back", "stack", "imagem"}, strict: false, mergeMethods: []string{"merge"}},
			},
		},
		{
			name:     "com front",
			frontend: true,
			want: map[string]rulesetExpectation{
				"dev":  {checks: []string{"back", "stack", "imagem", "front"}, strict: true, mergeMethods: []string{"squash"}},
				"main": {checks: []string{"only-dev-source", "back", "stack", "imagem", "front"}, strict: false, mergeMethods: []string{"merge"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			specs := rulesetSpecs(tt.frontend)
			if len(specs) != 2 {
				t.Fatalf("quantidade de rulesets = %d, want 2", len(specs))
			}
			for _, spec := range specs {
				payloadText, err := buildRulesetPayload(spec)
				if err != nil {
					t.Fatal(err)
				}
				var payload map[string]any
				if err := json.Unmarshal([]byte(payloadText), &payload); err != nil {
					t.Fatalf("payload inválido: %v", err)
				}

				branch := strings.TrimPrefix(payload["conditions"].(map[string]any)["ref_name"].(map[string]any)["include"].([]any)[0].(string), "refs/heads/")
				want, ok := tt.want[branch]
				if !ok {
					t.Fatalf("branch inesperada %q", branch)
				}
				if payload["name"] != "Protect "+branch || payload["enforcement"] != "active" {
					t.Errorf("metadados inesperados: %v/%v", payload["name"], payload["enforcement"])
				}
				if len(payload["bypass_actors"].([]any)) != 0 {
					t.Error("ruleset não deve possuir bypass actors")
				}

				var pr, status map[string]any
				for _, rawRule := range payload["rules"].([]any) {
					rule := rawRule.(map[string]any)
					switch rule["type"] {
					case "pull_request":
						pr = rule["parameters"].(map[string]any)
					case "required_status_checks":
						status = rule["parameters"].(map[string]any)
					case "required_linear_history":
						t.Errorf("%s não deve exigir histórico linear", branch)
					}
				}
				if pr == nil || status == nil {
					t.Fatalf("%s sem regra de PR ou de checks", branch)
				}
				if pr["required_approving_review_count"] != float64(1) {
					t.Errorf("%s aprovações = %v, want 1", branch, pr["required_approving_review_count"])
				}
				if pr["require_code_owner_review"] != false {
					t.Errorf("%s deve ter require_code_owner_review false", branch)
				}
				if got := toStrings(pr["allowed_merge_methods"]); strings.Join(got, ",") != strings.Join(want.mergeMethods, ",") {
					t.Errorf("%s métodos de merge = %v, want %v", branch, got, want.mergeMethods)
				}
				if status["strict_required_status_checks_policy"] != want.strict {
					t.Errorf("%s strict = %v, want %v", branch, status["strict_required_status_checks_policy"], want.strict)
				}
				var contexts []string
				for _, check := range status["required_status_checks"].([]any) {
					contexts = append(contexts, check.(map[string]any)["context"].(string))
				}
				if strings.Join(contexts, ",") != strings.Join(want.checks, ",") {
					t.Errorf("%s checks = %v, want %v", branch, contexts, want.checks)
				}
			}
		})
	}
}

func toStrings(raw any) []string {
	var out []string
	for _, item := range raw.([]any) {
		out = append(out, item.(string))
	}
	return out
}

func TestConfigureRulesetsUsesDevThenMain(t *testing.T) {
	var calls []recordedCommand
	replaceCommandRunner(t, func(dir, stdin, name string, args ...string) (string, error) {
		calls = append(calls, recordedCommand{dir: dir, stdin: stdin, name: name, args: append([]string(nil), args...)})
		return "", nil
	})

	if err := configureRulesets("algarys_meu-projeto", "algarys", false); err != nil {
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

	err := configureRulesets("algarys_meu-projeto", "algarys", false)
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

func recordCalls(t *testing.T) *[]recordedCommand {
	t.Helper()
	var calls []recordedCommand
	replaceCommandRunner(t, func(dir, stdin, name string, args ...string) (string, error) {
		calls = append(calls, recordedCommand{dir: dir, stdin: stdin, name: name, args: append([]string(nil), args...)})
		return "", nil
	})
	return &calls
}

func TestConfigureRepoSettingsMakesDevDefaultAndDeletesMergedBranches(t *testing.T) {
	calls := recordCalls(t)

	if err := configureRepoSettings("algarys_meu-projeto", "algarys"); err != nil {
		t.Fatalf("configureRepoSettings() error = %v", err)
	}
	if len(*calls) != 1 {
		t.Fatalf("quantidade de chamadas = %d, want 1", len(*calls))
	}
	call := (*calls)[0]
	if got := strings.Join(call.args[:4], " "); got != "api /repos/algarys/algarys_meu-projeto -X PATCH" {
		t.Errorf("chamada = %q", got)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(call.stdin), &payload); err != nil {
		t.Fatalf("payload inválido: %v", err)
	}
	if payload["default_branch"] != "dev" || payload["delete_branch_on_merge"] != true {
		t.Errorf("payload = %v", payload)
	}
}

func TestConfigureEnvironmentsRestrictsEachToItsBranch(t *testing.T) {
	calls := recordCalls(t)

	if err := configureEnvironments("algarys_meu-projeto", "algarys"); err != nil {
		t.Fatalf("configureEnvironments() error = %v", err)
	}

	want := []struct{ method, endpoint, field, value string }{
		{"PUT", "/repos/algarys/algarys_meu-projeto/environments/homolog", "", ""},
		{"POST", "/repos/algarys/algarys_meu-projeto/environments/homolog/deployment-branch-policies", "name", "dev"},
		{"PUT", "/repos/algarys/algarys_meu-projeto/environments/producao", "", ""},
		{"POST", "/repos/algarys/algarys_meu-projeto/environments/producao/deployment-branch-policies", "name", "main"},
	}
	if len(*calls) != len(want) {
		t.Fatalf("quantidade de chamadas = %d, want %d", len(*calls), len(want))
	}
	for i, w := range want {
		call := (*calls)[i]
		if got := strings.Join(call.args[:4], " "); got != "api "+w.endpoint+" -X "+w.method {
			t.Errorf("chamada %d = %q", i, got)
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(call.stdin), &payload); err != nil {
			t.Fatalf("payload %d inválido: %v", i, err)
		}
		if w.method == "PUT" {
			policy := payload["deployment_branch_policy"].(map[string]any)
			if policy["custom_branch_policies"] != true || policy["protected_branches"] != false {
				t.Errorf("policy %d = %v", i, policy)
			}
			continue
		}
		if payload[w.field] != w.value || payload["type"] != "branch" {
			t.Errorf("branch policy %d = %v", i, payload)
		}
	}
}

func TestConfigureEnvironmentsReportsWhichEnvironmentFailed(t *testing.T) {
	replaceCommandRunner(t, func(dir, stdin, name string, args ...string) (string, error) {
		if strings.Contains(strings.Join(args, " "), "environments/producao") {
			return "", errors.New("plano sem environments")
		}
		return "", nil
	})

	err := configureEnvironments("algarys_meu-projeto", "algarys")
	if err == nil || !strings.Contains(err.Error(), "producao") {
		t.Fatalf("erro = %v, deveria identificar o environment", err)
	}
}

func TestEnvironmentChecklistListsWhatToFillPerEnvironment(t *testing.T) {
	lines := environmentChecklist("algarys", "algarys_meu-projeto")
	text := strings.Join(lines, "\n")

	for _, expected := range []string{
		"https://github.com/algarys/algarys_meu-projeto/settings/environments",
		"homolog", "producao",
		"PORTAINER_URL", "PORTAINER_STACK_ID", "PORTAINER_ENDPOINT_ID", "APP_URL",
		"PORTAINER_ACCESS_TOKEN",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("checklist não contém %q:\n%s", expected, text)
		}
	}
}
