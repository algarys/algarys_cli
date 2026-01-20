package cmd

import (
	"fmt"

	"github.com/algarys/algarys_cli/cmd/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	Version   = "0.1.0"
	BuildDate = "dev"
	GitCommit = "none"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Mostra a versão do CLI",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println()

		// Box com versão
		versionText := lipgloss.NewStyle().
			Foreground(ui.Primary).
			Bold(true).
			Render(fmt.Sprintf("Algarys CLI v%s", Version))

		detailStyle := lipgloss.NewStyle().Foreground(ui.TextDim)

		content := versionText + "\n\n" +
			detailStyle.Render(fmt.Sprintf("Build:  %s", BuildDate)) + "\n" +
			detailStyle.Render(fmt.Sprintf("Commit: %s", GitCommit))

		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ui.Primary).
			Padding(1, 2).
			Render(content)

		fmt.Println(box)
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
