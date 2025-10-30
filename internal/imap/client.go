package imap

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// ConnectIMAP tries to connect and authenticate to an IMAP server.
// If the server cannot be guessed, the user is prompted to provide it.
func ConnectIMAP(email, password string) error {
	server, err := guessIMAPServer(email)
	if err != nil {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("⚠️  Could not determine IMAP server for %s\n", email)
		fmt.Print("Please enter your IMAP server (e.g. imap.yourdomain.com:993): ")
		input, _ := reader.ReadString('\n')
		server = strings.TrimSpace(input)
	}

	log.Printf("Connecting to IMAP server: %s", server)
	c, err := client.DialTLS(server, &tls.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer c.Logout()

	if err := c.Login(email, password); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	count := 0
	for range mailboxes {
		count++
	}
	if err := <-done; err != nil {
		return fmt.Errorf("listing mailboxes failed: %w", err)
	}

	log.Printf("✅ IMAP login successful. Found %d mailboxes.", count)
	return nil
}

func guessIMAPServer(email string) (string, error) {
	switch {
	case contains(email, "gmail.com"):
		return "imap.gmail.com:993", nil
	case contains(email, "outlook.com"), contains(email, "hotmail."), contains(email, "live."):
		return "outlook.office365.com:993", nil
	case contains(email, "yahoo."):
		return "imap.mail.yahoo.com:993", nil
	default:
		return "", fmt.Errorf("unknown domain")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
