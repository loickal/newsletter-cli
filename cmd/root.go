package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "newsletter-cli",
	Short: "Analyze and manage your newsletters from the terminal",
	Long: `A beautiful TUI-based CLI to analyze, list and unsubscribe 
from newsletters using your IMAP inbox.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use 'newsletter-cli login' or 'newsletter-cli analyze' to get started.")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
