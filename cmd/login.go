package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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

		fmt.Printf("✅ Saved credentials for %s (mock)\n", email)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
