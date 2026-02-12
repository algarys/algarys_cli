package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/algarys/algarys_cli/cmd/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const (
	repoOwner       = "algarys"
	repoName        = "algarys_cli"
	checkInterval   = 24 * time.Hour
	cacheFile       = ".algarys_update_check"
)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Atualiza o Algarys CLI para a última versão",
	Run:   runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) {
	fmt.Println()
	fmt.Println(ui.RenderBanner())
	fmt.Println()

	spinner := ui.NewSpinner(ui.IconGear + "  Verificando última versão...")
	spinner.Start()

	latest, err := getLatestVersion()
	if err != nil {
		spinner.Error("Erro ao verificar versão")
		fmt.Println(ui.RenderError(fmt.Sprintf("Não foi possível verificar: %v", err)))
		return
	}

	currentVersion := strings.TrimPrefix(Version, "v")
	latestVersion := strings.TrimPrefix(latest.TagName, "v")

	if currentVersion == latestVersion {
		spinner.Success("Você já está na última versão!")
		fmt.Println()

		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ui.Primary).
			Padding(1, 2).
			Render(
				lipgloss.NewStyle().Foreground(ui.Primary).Render(
					fmt.Sprintf("%s Algarys CLI v%s (atual)", ui.IconCheck, currentVersion),
				),
			)
		fmt.Println(box)
		fmt.Println()
		return
	}

	spinner.Success(fmt.Sprintf("Nova versão disponível: v%s → v%s", currentVersion, latestVersion))
	fmt.Println()

	// Perguntar se quer atualizar
	fmt.Print(lipgloss.NewStyle().Foreground(ui.Primary).Render("  Deseja atualizar agora? [S/n] "))

	var response string
	fmt.Scanln(&response)

	if response != "" && strings.ToLower(response) != "s" && strings.ToLower(response) != "sim" {
		fmt.Println()
		fmt.Println(ui.RenderInfo("Atualização cancelada"))
		fmt.Println()
		fmt.Println(lipgloss.NewStyle().Foreground(ui.Muted).PaddingLeft(2).Render(
			"Para atualizar manualmente, execute:",
		))
		fmt.Println(lipgloss.NewStyle().Foreground(ui.Primary).PaddingLeft(4).Render(
			"curl -fsSL https://raw.githubusercontent.com/algarys/algarys_cli/main/install.sh | bash",
		))
		fmt.Println()
		return
	}

	// Executar atualização
	fmt.Println()
	spinnerUpdate := ui.NewSpinner(ui.IconRocket + "  Baixando e instalando...")
	spinnerUpdate.Start()

	err = runInstallScript()
	if err != nil {
		spinnerUpdate.Error("Erro na atualização")
		fmt.Println(ui.RenderError(fmt.Sprintf("Falha: %v", err)))
		fmt.Println()
		fmt.Println(lipgloss.NewStyle().Foreground(ui.Muted).PaddingLeft(2).Render(
			"Tente manualmente:",
		))
		fmt.Println(lipgloss.NewStyle().Foreground(ui.Primary).PaddingLeft(4).Render(
			"curl -fsSL https://raw.githubusercontent.com/algarys/algarys_cli/main/install.sh | bash",
		))
		return
	}

	spinnerUpdate.Success("Atualização concluída!")
	fmt.Println()

	successBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.Primary).
		Padding(1, 2).
		Render(
			lipgloss.NewStyle().Foreground(ui.Primary).Bold(true).Render(
				fmt.Sprintf("%s Algarys CLI atualizado para v%s!", ui.IconDone, latestVersion),
			),
		)
	fmt.Println(successBox)
	fmt.Println()
}

func getLatestVersion() (*GitHubRelease, error) {
	// Usar gh CLI para acessar repo privado
	cmd := exec.Command("gh", "api",
		fmt.Sprintf("/repos/%s/%s/releases/latest", repoOwner, repoName),
	)

	output, err := cmd.Output()
	if err != nil {
		// Fallback para HTTP (repos públicos)
		return getLatestVersionHTTP()
	}

	var release GitHubRelease
	if err := json.Unmarshal(output, &release); err != nil {
		return nil, err
	}

	return &release, nil
}

func getLatestVersionHTTP() (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API retornou status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func runInstallScript() error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("atualização automática não suportada no Windows")
	}

	// Detectar OS e arquitetura
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Criar diretório temporário
	tmpDir, err := os.MkdirTemp("", "algarys-update-*")
	if err != nil {
		return fmt.Errorf("erro ao criar diretório temporário: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Baixar release via gh CLI (funciona com repo privado)
	pattern := fmt.Sprintf("algarys_%s_%s.tar.gz", goos, goarch)
	dlCmd := exec.Command("gh", "release", "download", "--repo",
		fmt.Sprintf("%s/%s", repoOwner, repoName),
		"--pattern", pattern, "--dir", tmpDir)
	dlCmd.Stdout = nil
	dlCmd.Stderr = nil

	if err := dlCmd.Run(); err != nil {
		return fmt.Errorf("erro ao baixar release: %v", err)
	}

	// Extrair
	tarPath := fmt.Sprintf("%s/%s", tmpDir, pattern)
	extractCmd := exec.Command("tar", "-xzf", tarPath, "-C", tmpDir)
	if err := extractCmd.Run(); err != nil {
		return fmt.Errorf("erro ao extrair: %v", err)
	}

	// Encontrar onde o binário atual está instalado
	currentBin, err := exec.LookPath("algarys")
	if err != nil {
		currentBin = "/usr/local/bin/algarys"
	}

	// Copiar novo binário
	newBin := fmt.Sprintf("%s/algarys", tmpDir)
	var mvCmd *exec.Cmd

	// Verificar se precisa de sudo
	installDir := fmt.Sprintf("%s", currentBin)
	testFile, testErr := os.OpenFile(installDir, os.O_WRONLY, 0)
	if testErr != nil {
		// Precisa de sudo
		mvCmd = exec.Command("sudo", "cp", newBin, currentBin)
	} else {
		testFile.Close()
		mvCmd = exec.Command("cp", newBin, currentBin)
	}
	mvCmd.Stdout = nil
	mvCmd.Stderr = nil

	if err := mvCmd.Run(); err != nil {
		return fmt.Errorf("erro ao instalar binário: %v", err)
	}

	// Garantir permissão de execução
	chmodCmd := exec.Command("chmod", "+x", currentBin)
	chmodCmd.Run()

	return nil
}

// CheckForUpdates verifica se há atualizações disponíveis
// Chamado automaticamente após executar comandos
func CheckForUpdates() {
	// Não checar se estiver executando o próprio comando update
	if len(os.Args) > 1 && os.Args[1] == "update" {
		return
	}

	// Verificar se deve checar (cache de 24h)
	if !shouldCheckForUpdates() {
		return
	}

	// Verificar se gh está disponível
	if _, err := exec.LookPath("gh"); err != nil {
		return
	}

	// Checar usando gh CLI (suporta repo privado)
	release, err := getLatestVersion()
	if err != nil {
		return
	}

	currentVersion := strings.TrimPrefix(Version, "v")
	latestVersion := strings.TrimPrefix(release.TagName, "v")

	// Salvar que já checou
	saveUpdateCheck()

	if currentVersion != latestVersion && latestVersion > currentVersion {
		// Mostrar aviso
		fmt.Println()
		warningBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ui.Warning).
			Padding(0, 2).
			Render(
				lipgloss.NewStyle().Foreground(ui.Warning).Render(
					fmt.Sprintf("%s Nova versão disponível: v%s → v%s", ui.IconWarning, currentVersion, latestVersion),
				) + "\n" +
					lipgloss.NewStyle().Foreground(ui.Muted).Render(
						"   Execute 'algarys update' para atualizar",
					),
			)
		fmt.Println(warningBox)
		fmt.Println()
	}
}

func shouldCheckForUpdates() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return true
	}

	cacheFilePath := fmt.Sprintf("%s/%s", homeDir, cacheFile)
	info, err := os.Stat(cacheFilePath)
	if err != nil {
		return true // Arquivo não existe, deve checar
	}

	// Verificar se passou mais de 24h desde a última checagem
	return time.Since(info.ModTime()) > checkInterval
}

func saveUpdateCheck() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	cacheFilePath := fmt.Sprintf("%s/%s", homeDir, cacheFile)
	os.WriteFile(cacheFilePath, []byte(time.Now().String()), 0644)
}
