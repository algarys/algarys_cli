package cmd

import (
	"fmt"
	"os"

	"github.com/algarys/algarys_cli/cmd/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "algarys",
	Short: "CLI oficial da Algarys",
	Long:  "", // Será renderizado customizado
	Run: func(cmd *cobra.Command, args []string) {
		// Mostrar help customizado quando rodar sem argumentos
		showWelcome()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Verificar updates após executar o comando
	CheckForUpdates()
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Customizar template de help
	rootCmd.SetHelpTemplate(customHelpTemplate())
}

func showWelcome() {
	fmt.Println()
	fmt.Println(ui.RenderBanner())
	fmt.Println()

	subtitle := lipgloss.NewStyle().
		Foreground(ui.TextDim).
		Italic(true).
		PaddingLeft(2).
		Render("CLI oficial para criação e gerenciamento de projetos")
	fmt.Println(subtitle)
	fmt.Println()

	// Comandos disponíveis
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.Primary).
		Bold(true).
		PaddingLeft(2)

	cmdStyle := lipgloss.NewStyle().
		Foreground(ui.Text).
		PaddingLeft(4)

	descStyle := lipgloss.NewStyle().
		Foreground(ui.TextDim).
		PaddingLeft(2)

	fmt.Println(titleStyle.Render("Comandos disponíveis:"))
	fmt.Println()

	commands := []struct {
		icon string
		name string
		desc string
	}{
		{ui.IconRocket, "init", "Criar novo projeto Python"},
		{"🎧", "transcribe", "Transcrever áudio para texto"},
		{ui.IconKey, "login", "Autenticar na Algarys"},
		{ui.IconPackage, "update", "Atualizar o CLI"},
		{ui.IconInfo, "version", "Mostrar versão do CLI"},
	}

	for _, c := range commands {
		cmdName := lipgloss.NewStyle().
			Foreground(ui.Primary).
			Bold(true).
			Render(c.name)
		fmt.Println(cmdStyle.Render(fmt.Sprintf("%s  %s", c.icon, cmdName)) + descStyle.Render(c.desc))
	}

	fmt.Println()

	// Quick start
	fmt.Println(titleStyle.Render("Quick start:"))
	fmt.Println()

	quickCmd := lipgloss.NewStyle().
		Foreground(ui.Primary).
		PaddingLeft(4).
		Render("algarys init")
	fmt.Println(quickCmd)
	fmt.Println()

	// Dica
	tipStyle := lipgloss.NewStyle().
		Foreground(ui.Muted).
		Italic(true).
		PaddingLeft(2)
	fmt.Println(tipStyle.Render(fmt.Sprintf("%s Use 'algarys <comando> --help' para mais detalhes", ui.IconMagic)))
	fmt.Println()
}

func customHelpTemplate() string {
	return `{{if .Long}}{{.Long}}{{else}}{{.Short}}{{end}}

{{if .HasAvailableSubCommands}}` + lipgloss.NewStyle().Foreground(ui.Primary).Bold(true).Render("Comandos:") + `{{range .Commands}}{{if .IsAvailableCommand}}
  {{.Name}}{{"\t"}}{{.Short}}{{end}}{{end}}{{end}}

{{if .HasAvailableFlags}}` + lipgloss.NewStyle().Foreground(ui.Primary).Bold(true).Render("Flags:") + `
{{.LocalFlags.FlagUsages}}{{end}}

Use "{{.CommandPath}} [comando] --help" para mais informações.
`
}
