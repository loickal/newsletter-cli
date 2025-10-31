package cmd

import (
	"fmt"
	"os"

	"github.com/loickal/newsletter-cli/internal/config"
	"github.com/loickal/newsletter-cli/internal/ui"
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
		// Load saved credentials
		cfg, _ := config.Load()
		email := ""
		password := ""
		server := ""
		if cfg != nil {
			email = cfg.Email
			password = config.Decrypt(cfg.Password)
			server = cfg.Server
		}

		// Get current version for update check
		currentVersion := getVersion()

		// Show unified UI - it will handle welcome screen and navigation
		if err := ui.RunAppSync(email, password, server, 0, false, "", currentVersion); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var currentVersion string

func getVersion() string {
	if currentVersion != "" {
		return currentVersion
	}
	// Try to get version from main package
	return "dev"
}

func SetVersion(version string) {
	currentVersion = version
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
