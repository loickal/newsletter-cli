package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "newsletter-cli",
	Short: "Analyze and manage your newsletters from the terminal",
	Long: `📬 Newsletter CLI

A beautiful TUI-based CLI to analyze, list and unsubscribe 
from newsletters using your IMAP inbox.

Get started:
  newsletter-cli login     Save your IMAP credentials
  newsletter-cli analyze   Analyze and manage newsletters`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("📬 Newsletter CLI")
		fmt.Println()
		fmt.Println("Get started:")
		fmt.Println("  newsletter-cli login     Save your IMAP credentials")
		fmt.Println("  newsletter-cli analyze   Analyze and manage newsletters")
		fmt.Println()
		fmt.Println("Use 'newsletter-cli --help' for more information.")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
