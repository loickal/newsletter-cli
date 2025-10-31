package unsubscribe

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

// UnsubscribeResult represents the result of an unsubscribe attempt
type UnsubscribeResult struct {
	Sender   string
	Link     string
	Success  bool
	ErrorMsg string
}

// Unsubscribe attempts to unsubscribe from a newsletter using the provided link
// Supports both HTTP (GET/POST) and mailto: links
// email, password, and imapServer are required for mailto: links to send via SMTP
func Unsubscribe(sender, unsubscribeLink string, email, password, imapServer string) UnsubscribeResult {
	result := UnsubscribeResult{
		Sender: sender,
		Link:   unsubscribeLink,
	}

	if unsubscribeLink == "" {
		result.ErrorMsg = "No unsubscribe link provided"
		return result
	}

	// Handle mailto: links
	if strings.HasPrefix(unsubscribeLink, "mailto:") {
		if email == "" || password == "" || imapServer == "" {
			result.ErrorMsg = "SMTP credentials required for mailto: links"
			return result
		}
		return unsubscribeMailto(sender, unsubscribeLink, email, password, imapServer)
	}

	// Handle HTTP links
	if !strings.HasPrefix(unsubscribeLink, "http://") && !strings.HasPrefix(unsubscribeLink, "https://") {
		result.ErrorMsg = "Invalid unsubscribe link format"
		return result
	}

	// Try POST first (most common for unsubscribe), then GET
	if err := unsubscribePOST(unsubscribeLink); err == nil {
		result.Success = true
		return result
	}

	// If POST fails, try GET
	if err := unsubscribeGET(unsubscribeLink); err == nil {
		result.Success = true
		return result
	}

	// Both POST and GET failed - try GET one more time to get the error
	err := unsubscribeGET(unsubscribeLink)
	result.ErrorMsg = fmt.Sprintf("Failed to unsubscribe: %v", err)
	return result
}

// unsubscribePOST attempts to unsubscribe via HTTP POST
func unsubscribePOST(link string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Some unsubscribe links use POST with empty body or specific content type
	req, err := http.NewRequest("POST", link, bytes.NewBuffer([]byte{}))
	if err != nil {
		return err
	}

	// Set common headers
	req.Header.Set("User-Agent", "Newsletter-CLI/1.0")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read response body (some servers require it)
	io.Copy(io.Discard, resp.Body)

	// Consider 2xx and 3xx as success
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return nil
	}

	return fmt.Errorf("POST returned status %d", resp.StatusCode)
}

// unsubscribeGET attempts to unsubscribe via HTTP GET
func unsubscribeGET(link string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
		// Don't follow redirects - just check initial response
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Newsletter-CLI/1.0")
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read response body
	io.Copy(io.Discard, resp.Body)

	// Consider 2xx and 3xx as success
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return nil
	}

	return fmt.Errorf("GET returned status %d", resp.StatusCode)
}

// BatchUnsubscribe processes multiple unsubscribe requests concurrently
// email, password, and imapServer are required for mailto: links
func BatchUnsubscribe(requests []struct {
	Sender string
	Link   string
}, email, password, imapServer string) []UnsubscribeResult {
	results := make([]UnsubscribeResult, len(requests))
	resultChan := make(chan UnsubscribeResult, len(requests))

	// Process all requests concurrently
	for _, req := range requests {
		go func(sender, link string) {
			resultChan <- Unsubscribe(sender, link, email, password, imapServer)
		}(req.Sender, req.Link)
	}

	// Collect results
	for i := 0; i < len(requests); i++ {
		results[i] = <-resultChan
	}

	return results
}

// unsubscribeMailto handles mailto: unsubscribe links by sending an email via SMTP
func unsubscribeMailto(sender, mailtoLink, email, password, imapServer string) UnsubscribeResult {
	result := UnsubscribeResult{
		Sender: sender,
		Link:   mailtoLink,
	}

	// Parse mailto link
	u, err := url.Parse(mailtoLink)
	if err != nil {
		result.ErrorMsg = fmt.Sprintf("Invalid mailto link: %v", err)
		return result
	}

	// Extract email address
	toEmail := u.Opaque
	if toEmail == "" {
		toEmail = u.Path
	}
	if toEmail == "" {
		result.ErrorMsg = "No recipient email in mailto link"
		return result
	}

	// Extract subject and body from query parameters
	subject := "Unsubscribe"
	body := "Please unsubscribe me from your mailing list."
	if u.Query().Get("subject") != "" {
		subject = u.Query().Get("subject")
	}
	if u.Query().Get("body") != "" {
		body = u.Query().Get("body")
	}

	// Determine SMTP server from IMAP server
	smtpServer, err := getSMTPServer(imapServer)
	if err != nil {
		result.ErrorMsg = fmt.Sprintf("Could not determine SMTP server: %v", err)
		return result
	}

	// Send email via SMTP
	if err := sendUnsubscribeEmail(email, password, smtpServer, toEmail, subject, body); err != nil {
		result.ErrorMsg = fmt.Sprintf("Failed to send unsubscribe email: %v", err)
		return result
	}

	result.Success = true
	return result
}

// getSMTPServer determines SMTP server from IMAP server
func getSMTPServer(imapServer string) (string, error) {
	// Remove port if present
	server := strings.Split(imapServer, ":")[0]

	// Handle known providers
	if strings.Contains(server, "gmail.com") {
		return "smtp.gmail.com:587", nil
	}
	if strings.Contains(server, "outlook.office365.com") || strings.Contains(server, "outlook.com") {
		return "smtp-mail.outlook.com:587", nil
	}
	if strings.Contains(server, "yahoo") {
		return "smtp.mail.yahoo.com:587", nil
	}
	if strings.Contains(server, "icloud") || strings.Contains(server, "me.com") || strings.Contains(server, "mac.com") {
		return "smtp.mail.me.com:587", nil
	}
	if strings.Contains(server, "fastmail") {
		return "smtp.fastmail.com:587", nil
	}
	if strings.Contains(server, "mailbox.org") {
		return "smtp.mailbox.org:587", nil
	}

	// Try common SMTP patterns based on IMAP server
	patterns := []string{
		"smtp.%s:587",
		"mail.%s:587",
		"smtp.%s:25",
		"mail.%s:25",
	}

	// Extract domain from server (handle subdomains)
	parts := strings.Split(server, ".")
	var domain string
	if len(parts) >= 2 {
		// Get last two parts (e.g., "gmail.com" from "imap.gmail.com")
		domain = strings.Join(parts[len(parts)-2:], ".")
	} else {
		domain = server
	}

	for _, pattern := range patterns {
		smtpServer := fmt.Sprintf(pattern, domain)
		if testSMTPConnection(smtpServer) {
			return smtpServer, nil
		}
	}

	return "", fmt.Errorf("could not determine SMTP server for %s", imapServer)
}

// testSMTPConnection tests if an SMTP server is reachable
func testSMTPConnection(server string) bool {
	conn, err := net.DialTimeout("tcp", server, 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// sendUnsubscribeEmail sends an unsubscribe email via SMTP
func sendUnsubscribeEmail(fromEmail, password, smtpServer, toEmail, subject, body string) error {
	// Parse email addresses
	from, err := mail.ParseAddress(fromEmail)
	if err != nil {
		return fmt.Errorf("invalid from email: %w", err)
	}
	to, err := mail.ParseAddress(toEmail)
	if err != nil {
		return fmt.Errorf("invalid to email: %w", err)
	}

	// Create message
	message := fmt.Sprintf("From: %s\r\n", from.Address)
	message += fmt.Sprintf("To: %s\r\n", to.Address)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "\r\n"
	message += body + "\r\n"

	// Extract hostname and port
	host := strings.Split(smtpServer, ":")[0]
	port := "587"
	if parts := strings.Split(smtpServer, ":"); len(parts) == 2 {
		port = parts[1]
	}

	// Create auth
	auth := smtp.PlainAuth("", fromEmail, password, host)

	// Send email
	if port == "587" || port == "465" {
		// Use TLS for secure SMTP
		tlsConfig := &tls.Config{
			ServerName: host,
		}
		if err := sendMailTLS(host+":"+port, auth, fromEmail, []string{to.Address}, []byte(message), tlsConfig); err != nil {
			return fmt.Errorf("SMTP send failed: %w", err)
		}
	} else {
		// Use plain SMTP
		if err := smtp.SendMail(host+":"+port, auth, fromEmail, []string{to.Address}, []byte(message)); err != nil {
			return fmt.Errorf("SMTP send failed: %w", err)
		}
	}

	return nil
}

// sendMailTLS sends email with TLS support (for ports 587 and 465)
func sendMailTLS(addr string, a smtp.Auth, from string, to []string, msg []byte, tlsConfig *tls.Config) error {
	// Connect to server
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	host := strings.Split(addr, ":")[0]

	// Create client
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	// Start TLS if port is 587
	if strings.HasSuffix(addr, ":587") {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsConfig); err != nil {
				return err
			}
		}
	}

	// Authenticate
	if a != nil {
		if err := client.Auth(a); err != nil {
			return err
		}
	}

	// Set sender and recipients
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}

	// Send message
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	return client.Quit()
}

// FormatMailtoLink formats a mailto link for display
func FormatMailtoLink(mailtoLink string) string {
	if !strings.HasPrefix(mailtoLink, "mailto:") {
		return mailtoLink
	}

	email := strings.TrimPrefix(mailtoLink, "mailto:")
	if idx := strings.Index(email, "?"); idx != -1 {
		email = email[:idx]
	}

	return fmt.Sprintf("mailto:%s (requires manual email)", email)
}
