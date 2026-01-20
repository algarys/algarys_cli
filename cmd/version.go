package cmd

import (
	"fmt"

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
		fmt.Printf("Algarys CLI v%s\n", Version)
		fmt.Printf("Build: %s\n", BuildDate)
		fmt.Printf("Commit: %s\n", GitCommit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
