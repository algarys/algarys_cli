package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultAITeamSlug = "engenheiros-de-ia"
	aiTeamEnvVar      = "ALGARYS_AI_TEAM"
)

var (
	githubAPIBase = "https://api.github.com"
	httpClient    = &http.Client{Timeout: 30 * time.Second}
	aiTeamFlag    string
	aiTeamFlagSet bool
)

func resolveAITeamSlug() string {
	if aiTeamFlagSet {
		return strings.TrimSpace(aiTeamFlag)
	}
	if value, ok := os.LookupEnv(aiTeamEnvVar); ok {
		return strings.TrimSpace(value)
	}
	return defaultAITeamSlug
}

func grantTeamAdmin(org, repoName, teamSlug string) error {
	if teamSlug == "" {
		return nil
	}

	token, err := runCommand("", "", "gh", "auth", "token")
	if err != nil {
		return fmt.Errorf("obter token do GitHub: %w", err)
	}
	token = strings.TrimSpace(token)

	endpoint := fmt.Sprintf(
		"%s/orgs/%s/teams/%s/repos/%s/%s",
		strings.TrimRight(githubAPIBase, "/"),
		url.PathEscape(org),
		url.PathEscape(teamSlug),
		url.PathEscape(org),
		url.PathEscape(repoName),
	)
	body, err := json.Marshal(map[string]string{"permission": "admin"})
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("Content-Type", "application/json")

	response, err := httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("chamar API do GitHub: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	detail, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
	return fmt.Errorf(
		"dar admin ao time %s em %s/%s: HTTP %d: %s",
		teamSlug, org, repoName, response.StatusCode, strings.TrimSpace(string(detail)),
	)
}
