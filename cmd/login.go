package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/loickal/newsletter-cli/internal/imap"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to your email account via IMAP",
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Email: ")
		email, _ := reader.ReadString('\n')
		fmt.Print("Password: ")
		pass, _ := reader.ReadString('\n')

		email = strings.TrimSpace(email)
		pass = strings.TrimSpace(pass)

		fmt.Println("🔐 Testing IMAP connection...")
		if err := imap.ConnectIMAP(email, pass); err != nil {
			fmt.Printf("❌ Connection failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Logged in successfully as %s\n", email)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
