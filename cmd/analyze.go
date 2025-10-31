package cmd

import (
	"fmt"
	"os"

	"github.com/loickal/newsletter-cli/internal/config"
	"github.com/loickal/newsletter-cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	daysFlag   int
	emailFlag  string
	serverFlag string
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze newsletters in your inbox",
	Long: `Analyze newsletters in your inbox and display them in an interactive dashboard.

If you have saved credentials, they will be used automatically.
You can also provide credentials via flags for non-interactive use.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _ := config.Load()

		email := emailFlag
		if email == "" {
			email = cfg.Email
		}

		var err error
		pass, err := config.Decrypt(cfg.Password)
		if err != nil {
			pass = "" // Continue with empty password if decryption fails
		}
		server := serverFlag
		if server == "" {
			server = cfg.Server
		}

		flagsProvided := daysFlag > 0 || emailFlag != "" || serverFlag != ""

		currentVersion := getVersion()
		if err := ui.RunAppSync(email, pass, server, daysFlag, flagsProvided, "analyze", currentVersion); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	analyzeCmd.Flags().IntVarP(&daysFlag, "days", "d", 30, "Number of days to analyze (default: 30)")
	analyzeCmd.Flags().StringVarP(&emailFlag, "email", "e", "", "Email address (overrides saved credentials)")
	analyzeCmd.Flags().StringVarP(&serverFlag, "server", "s", "", "IMAP server (overrides saved credentials)")
	rootCmd.AddCommand(analyzeCmd)
}
