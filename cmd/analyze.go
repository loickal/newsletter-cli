package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/loickal/newsletter-cli/internal/config"
	"github.com/loickal/newsletter-cli/internal/imap"
	"github.com/loickal/newsletter-cli/internal/ui"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze newsletters in your inbox",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _ := config.Load()

		email := cfg.Email
		pass := config.Decrypt(cfg.Password)
		server := cfg.Server

		if email == "" || pass == "" || server == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Email: ")
			email, _ = reader.ReadString('\n')
			fmt.Print("Password: ")
			pass, _ = reader.ReadString('\n')
			fmt.Print("IMAP server (e.g. imap.gmail.com:993): ")
			server, _ = reader.ReadString('\n')

			email = strings.TrimSpace(email)
			pass = strings.TrimSpace(pass)
			server = strings.TrimSpace(server)
		} else {
			fmt.Printf("🔐 Using saved account %s @ %s\n", email, server)
		}

		fmt.Print("Analyze last how many days? (default 30): ")
		reader := bufio.NewReader(os.Stdin)
		daysStr, _ := reader.ReadString('\n')
		daysStr = strings.TrimSpace(daysStr)
		if daysStr == "" {
			daysStr = "30"
		}
		days, _ := time.ParseDuration(daysStr + "24h")
		since := time.Now().Add(-days)

		fmt.Printf("🔍 Fetching newsletters since %s...\n", since.Format("2006-01-02"))

		stats, err := imap.FetchNewsletterStats(server, email, pass, since)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		if err := ui.Run(stats); err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
}
