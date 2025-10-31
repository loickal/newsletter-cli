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
		// Load selected account (if any) to pre-fill the form
		account, _ := config.GetSelectedAccount()
		email := ""
		password := ""
		server := ""
		if account != nil {
			email = account.Email
			var err error
			password, err = config.Decrypt(account.Password)
			if err != nil {
				password = "" // Continue with empty password if decryption fails
			}
			server = account.Server
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
