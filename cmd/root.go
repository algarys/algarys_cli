package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "algarys",
	Short: "CLI oficial da Algarys",
	Long: `Algarys CLI - Ferramenta de linha de comando para automação interna.

Use este CLI para executar tarefas de automação,
gerenciar recursos e facilitar o dia a dia da equipe.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
