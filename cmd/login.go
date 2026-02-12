package cmd

import (
	"fmt"
	"os/exec"

	"github.com/algarys/algarys_cli/cmd/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Autenticar na Algarys (GitHub)",
	Long: `Autentica no GitHub para acessar funcionalidades da empresa.

Necessário para:
  - Criar repositórios na org (algarys init)
  - Atualizar o CLI (algarys update)

Não necessário para:
  - Transcrever áudio (algarys transcribe)`,
	Run: runLogin,
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Desconectar da Algarys (GitHub)",
	Run:   runLogout,
}

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
}

func runLogin(cmd *cobra.Command, args []string) {
	fmt.Println()
	fmt.Println(ui.RenderBanner())
	fmt.Println()

	// Verificar se gh está instalado
	if _, err := exec.LookPath("gh"); err != nil {
		fmt.Println(ui.RenderError("GitHub CLI (gh) não encontrado"))
		fmt.Println()
		fmt.Println(lipgloss.NewStyle().Foreground(ui.Muted).PaddingLeft(2).Render("Instale com:"))
		fmt.Println(lipgloss.NewStyle().Foreground(ui.Primary).PaddingLeft(4).Render("brew install gh"))
		fmt.Println(lipgloss.NewStyle().Foreground(ui.TextDim).PaddingLeft(4).Render("ou visite: https://cli.github.com"))
		fmt.Println()
		return
	}

	// Verificar se já está autenticado
	if IsLoggedIn() {
		user := getGitHubUser()
		fmt.Println(lipgloss.NewStyle().Foreground(ui.Primary).PaddingLeft(2).Render(
			fmt.Sprintf("%s Você já está autenticado como %s", ui.IconCheck, user),
		))
		fmt.Println()

		// Verificar acesso à org
		if hasOrgAccess() {
			fmt.Println(lipgloss.NewStyle().Foreground(ui.Primary).PaddingLeft(2).Render(
				fmt.Sprintf("%s Acesso à org algarys confirmado", ui.IconCheck),
			))
		} else {
			fmt.Println(ui.RenderWarning("Sem acesso à org algarys. Peça convite ao admin."))
		}
		fmt.Println()
		return
	}

	// Login via gh
	fmt.Println(lipgloss.NewStyle().
		Foreground(ui.TextDim).
		PaddingLeft(2).
		Render("Abrindo autenticação GitHub..."))
	fmt.Println()

	ghCmd := exec.Command("gh", "auth", "login", "-h", "github.com", "-p", "https", "-w")
	ghCmd.Stdin = nil
	ghCmd.Stdout = nil
	ghCmd.Stderr = nil

	// Rodar interativamente
	ghCmd.Stdin = cmd.InOrStdin()
	ghCmd.Stdout = cmd.OutOrStdout()
	ghCmd.Stderr = cmd.ErrOrStderr()

	if err := ghCmd.Run(); err != nil {
		fmt.Println()
		fmt.Println(ui.RenderError("Falha na autenticação"))
		return
	}

	fmt.Println()

	// Verificar se deu certo
	if IsLoggedIn() {
		user := getGitHubUser()

		successBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ui.Primary).
			Padding(1, 2).
			Render(
				lipgloss.NewStyle().Foreground(ui.Primary).Bold(true).Render(
					fmt.Sprintf("%s Autenticado como %s!", ui.IconDone, user),
				),
			)
		fmt.Println(successBox)

		if hasOrgAccess() {
			fmt.Println()
			fmt.Println(lipgloss.NewStyle().Foreground(ui.Primary).PaddingLeft(2).Render(
				fmt.Sprintf("%s Acesso à org algarys confirmado", ui.IconCheck),
			))
		}
	} else {
		fmt.Println(ui.RenderError("Autenticação não concluída"))
	}
	fmt.Println()
}

func runLogout(cmd *cobra.Command, args []string) {
	fmt.Println()
	fmt.Println(ui.RenderBanner())
	fmt.Println()

	if !IsLoggedIn() {
		fmt.Println(lipgloss.NewStyle().Foreground(ui.Muted).PaddingLeft(2).Render(
			"Você não está autenticado.",
		))
		fmt.Println()
		return
	}

	ghCmd := exec.Command("gh", "auth", "logout", "-h", "github.com")
	ghCmd.Stdin = cmd.InOrStdin()
	ghCmd.Stdout = cmd.OutOrStdout()
	ghCmd.Stderr = cmd.ErrOrStderr()

	if err := ghCmd.Run(); err != nil {
		fmt.Println(ui.RenderError("Erro ao desconectar"))
		return
	}

	fmt.Println()
	fmt.Println(ui.RenderSuccess("Desconectado com sucesso"))
	fmt.Println()
}

// IsLoggedIn verifica se o usuário está autenticado no GitHub
func IsLoggedIn() bool {
	cmd := exec.Command("gh", "auth", "status")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

func getGitHubUser() string {
	cmd := exec.Command("gh", "api", "user", "-q", ".login")
	output, err := cmd.Output()
	if err != nil {
		return "desconhecido"
	}
	return string(output[:len(output)-1]) // remover \n
}

func hasOrgAccess() bool {
	cmd := exec.Command("gh", "api", "orgs/algarys/members", "-q", "length")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}
