package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/loickal/newsletter-cli/internal/config"
	"github.com/loickal/newsletter-cli/internal/imap"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to your email account via IMAP",
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("📧 Email: ")
		email, _ := reader.ReadString('\n')
		email = strings.TrimSpace(email)

		fmt.Print("🔒 Password: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // New line after password input
		if err != nil {
			fmt.Printf("❌ Error reading password: %v\n", err)
			os.Exit(1)
		}
		pass := strings.TrimSpace(string(bytePassword))

		fmt.Print("🌐 IMAP server (e.g. imap.gmail.com:993): ")
		server, _ := reader.ReadString('\n')
		server = strings.TrimSpace(server)

		fmt.Print("\n🔐 Testing IMAP connection...")
		if err := imap.ConnectIMAP(email, pass, server); err != nil {
			fmt.Printf("\n❌ Connection failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(" ✅")

		cfg := config.Config{
			Email:    email,
			Server:   server,
			Password: config.Encrypt(pass),
		}
		if err := config.Save(cfg); err != nil {
			fmt.Printf("❌ Failed to save config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Logged in and saved credentials for %s\n", email)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
