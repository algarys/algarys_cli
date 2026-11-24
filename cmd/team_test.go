package cmd

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func useFakeGitHub(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	previousBase := githubAPIBase
	githubAPIBase = server.URL
	t.Cleanup(func() {
		githubAPIBase = previousBase
		server.Close()
	})
	replaceCommandRunner(t, func(dir, stdin, name string, args ...string) (string, error) {
		if name == "gh" && len(args) == 2 && args[0] == "auth" && args[1] == "token" {
			return "token-falso\n", nil
		}
		return "", errors.New("comando inesperado")
	})
}

func TestGrantTeamAdminSendsExpectedRequest(t *testing.T) {
	var method, path, auth string
	var body map[string]string
	useFakeGitHub(t, func(w http.ResponseWriter, r *http.Request) {
		method, path, auth = r.Method, r.URL.Path, r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.WriteHeader(http.StatusNoContent)
	})

	if err := grantTeamAdmin("algarys", "algarys_meu-projeto", "engenheiros-de-ia"); err != nil {
		t.Fatalf("grantTeamAdmin() error = %v", err)
	}
	if method != http.MethodPut {
		t.Errorf("método = %s", method)
	}
	if path != "/orgs/algarys/teams/engenheiros-de-ia/repos/algarys/algarys_meu-projeto" {
		t.Errorf("caminho = %s", path)
	}
	if auth != "Bearer token-falso" {
		t.Errorf("autorização = %q", auth)
	}
	if body["permission"] != "admin" {
		t.Errorf("payload = %v", body)
	}
}

func TestGrantTeamAdminReportsAPIFailure(t *testing.T) {
	useFakeGitHub(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"Must have admin rights"}`))
	})

	err := grantTeamAdmin("algarys", "algarys_x", "engenheiros-de-ia")
	if err == nil || !strings.Contains(err.Error(), "HTTP 403") || !strings.Contains(err.Error(), "Must have admin rights") {
		t.Fatalf("erro = %v", err)
	}
}

func TestGrantTeamAdminReportsTokenFailure(t *testing.T) {
	replaceCommandRunner(t, func(dir, stdin, name string, args ...string) (string, error) {
		return "", errors.New("sem login")
	})
	err := grantTeamAdmin("algarys", "algarys_x", "engenheiros-de-ia")
	if err == nil || !strings.Contains(err.Error(), "token") {
		t.Fatalf("erro = %v", err)
	}
}

func TestGrantTeamAdminSkipsEmptySlug(t *testing.T) {
	replaceCommandRunner(t, func(dir, stdin, name string, args ...string) (string, error) {
		t.Fatal("não deveria executar comandos")
		return "", nil
	})
	if err := grantTeamAdmin("algarys", "algarys_x", ""); err != nil {
		t.Fatal(err)
	}
}

func TestResolveAITeamSlug(t *testing.T) {
	previousFlag, previousSet := aiTeamFlag, aiTeamFlagSet
	t.Cleanup(func() { aiTeamFlag, aiTeamFlagSet = previousFlag, previousSet })

	aiTeamFlagSet = false
	t.Setenv(aiTeamEnvVar, "")
	if got := resolveAITeamSlug(); got != "" {
		t.Errorf("env vazio deve desativar, got %q", got)
	}

	aiTeamFlagSet = false
	t.Setenv(aiTeamEnvVar, "outro-time")
	if got := resolveAITeamSlug(); got != "outro-time" {
		t.Errorf("env, got %q", got)
	}

	aiTeamFlag, aiTeamFlagSet = "time-flag", true
	if got := resolveAITeamSlug(); got != "time-flag" {
		t.Errorf("flag deve vencer, got %q", got)
	}
}

func TestResolveAITeamSlugDefault(t *testing.T) {
	previousSet := aiTeamFlagSet
	t.Cleanup(func() { aiTeamFlagSet = previousSet })
	aiTeamFlagSet = false
	os.Unsetenv(aiTeamEnvVar)
	if got := resolveAITeamSlug(); got != "engenheiros-de-ia" {
		t.Errorf("padrão = %q", got)
	}
}
