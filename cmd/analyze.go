package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/loickal/newsletter-cli/internal/config"
	"github.com/loickal/newsletter-cli/internal/imap"
	"github.com/loickal/newsletter-cli/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
			fmt.Print("📧 Email: ")
			email, _ = reader.ReadString('\n')
			email = strings.TrimSpace(email)

			fmt.Print("🔒 Password: ")
			bytePassword, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println() // New line after password input
			if err != nil {
				fmt.Printf("❌ Error reading password: %v\n", err)
				os.Exit(1)
			}
			pass = strings.TrimSpace(string(bytePassword))

			fmt.Print("🌐 IMAP server (e.g. imap.gmail.com:993): ")
			server, _ = reader.ReadString('\n')
			server = strings.TrimSpace(server)
		} else {
			fmt.Printf("🔐 Using saved account: %s @ %s\n\n", email, server)
		}

		fmt.Print("📅 Analyze last how many days? (default 30): ")
		reader := bufio.NewReader(os.Stdin)
		daysStr, _ := reader.ReadString('\n')
		daysStr = strings.TrimSpace(daysStr)
		if daysStr == "" {
			daysStr = "30"
		}

		daysInt, err := strconv.Atoi(daysStr)
		if err != nil {
			fmt.Printf("❌ Invalid number of days: %v\n", err)
			os.Exit(1)
		}

		days := time.Duration(daysInt) * 24 * time.Hour
		since := time.Now().Add(-days)

		fmt.Printf("\n🔍 Fetching newsletters since %s...\n", since.Format("2006-01-02"))

		stats, err := imap.FetchNewsletterStats(server, email, pass, since)
		if err != nil {
			fmt.Printf("\n❌ Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println() // Empty line before opening TUI

		if err := ui.Run(stats); err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
}
