package cmd

import (
	"fmt"
	"os"

	"github.com/loickal/newsletter-cli/internal/config"
	"github.com/loickal/newsletter-cli/internal/ui"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to your email account via IMAP",
	Run: func(cmd *cobra.Command, args []string) {
		// Load saved credentials (if any) to pre-fill the form
		cfg, _ := config.Load()
		email := ""
		password := ""
		server := ""
		if cfg != nil {
			email = cfg.Email
			password = config.Decrypt(cfg.Password)
			server = cfg.Server
		}

		currentVersion := getVersion()
		if err := ui.RunAppSync(email, password, server, 0, false, "login", currentVersion); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
