package imap

import (
	"bufio"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// ConnectIMAP tries to connect and authenticate to an IMAP server.
// If server is provided, it uses that. Otherwise, it tries to guess from the email.
// If the server cannot be guessed, the user is prompted to provide it.
func ConnectIMAP(email, password, server string) error {
	if server == "" {
		var err error
		server, err = guessIMAPServer(email)
		if err != nil {
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("⚠️  Could not determine IMAP server for %s\n", email)
			fmt.Print("Please enter your IMAP server (e.g. imap.yourdomain.com:993): ")
			input, _ := reader.ReadString('\n')
			server = strings.TrimSpace(input)
		}
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

// DiscoverIMAPServer discovers the IMAP server using DNS autodiscover
// This is a public function for use by the UI
func DiscoverIMAPServer(email string) (string, error) {
	// Extract domain from email
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid email address")
	}
	domain := strings.ToLower(parts[1])

	// Try known providers first (faster)
	if server := getKnownProviderServer(domain); server != "" {
		log.Printf("Using known provider server: %s", server)
		return server, nil
	}

	// Try DNS SRV records (RFC 6186)
	if server, err := discoverSRV(domain); err == nil {
		log.Printf("Discovered IMAP server via SRV record: %s", server)
		return server, nil
	}

	// Try autoconfig/autodiscover endpoints
	if server, err := discoverAutoconfig(domain, email); err == nil {
		log.Printf("Discovered IMAP server via autoconfig: %s", server)
		return server, nil
	}

	// Try common hostname patterns
	if server := tryCommonPatterns(domain); server != "" {
		log.Printf("Discovered IMAP server via pattern: %s", server)
		return server, nil
	}

	return "", fmt.Errorf("could not discover IMAP server for domain: %s", domain)
}

// getKnownProviderServer returns server for well-known email providers
func getKnownProviderServer(domain string) string {
	switch {
	case domain == "gmail.com":
		return "imap.gmail.com:993"
	case domain == "outlook.com" || domain == "hotmail.com" || strings.HasSuffix(domain, "live.com") || strings.HasSuffix(domain, "outlook.com"):
		return "outlook.office365.com:993"
	case strings.Contains(domain, "yahoo"):
		return "imap.mail.yahoo.com:993"
	case domain == "icloud.com" || domain == "me.com" || domain == "mac.com":
		return "imap.mail.me.com:993"
	case domain == "protonmail.com" || domain == "proton.me":
		return "127.0.0.1:1143" // ProtonMail uses bridge
	case domain == "fastmail.com":
		return "imap.fastmail.com:993"
	case domain == "mailbox.org":
		return "imap.mailbox.org:993"
	}
	return ""
}

// discoverSRV tries to find IMAP server using DNS SRV records
// RFC 6186 defines _imaps._tcp (IMAP over TLS) and _imap._tcp (IMAP)
func discoverSRV(domain string) (string, error) {
	// Try IMAPS (IMAP over TLS) first - port 993
	_, srvs, err := net.LookupSRV("imaps", "tcp", domain)
	if err == nil && len(srvs) > 0 {
		target := strings.TrimSuffix(srvs[0].Target, ".")
		port := srvs[0].Port
		if port == 0 {
			port = 993 // Default IMAPS port
		}
		return fmt.Sprintf("%s:%d", target, port), nil
	}

	// Try IMAP (non-encrypted) - but we'll use TLS anyway
	_, srvs, err = net.LookupSRV("imap", "tcp", domain)
	if err == nil && len(srvs) > 0 {
		target := strings.TrimSuffix(srvs[0].Target, ".")
		port := srvs[0].Port
		if port == 0 {
			port = 143 // Default IMAP port, but we'll try 993 first
		}
		// Try TLS on standard port first, then fall back to SRV port
		if testConnection(fmt.Sprintf("%s:993", target)) {
			return fmt.Sprintf("%s:993", target), nil
		}
		return fmt.Sprintf("%s:%d", target, port), nil
	}

	return "", fmt.Errorf("no SRV record found")
}

// tryCommonPatterns attempts common IMAP server hostname patterns
func tryCommonPatterns(domain string) string {
	patterns := []string{
		"imap.%s:993",
		"mail.%s:993",
		"imaps.%s:993",
		"imap-secure.%s:993",
		"mailserver.%s:993",
		"mail.%s:143",
		"imap.%s:143",
	}

	for _, pattern := range patterns {
		server := fmt.Sprintf(pattern, domain)
		if testConnection(server) {
			return server
		}
	}

	return ""
}

// testConnection quickly tests if an IMAP server is reachable
func testConnection(server string) bool {
	conn, err := net.DialTimeout("tcp", server, 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// XML structures for autoconfig/autodiscover
type autoconfigResponse struct {
	XMLName xml.Name        `xml:"clientConfig"`
	Email   autoconfigEmail `xml:"emailProvider"`
}

type autoconfigEmail struct {
	IncomingServers []autoconfigIncomingServer `xml:"incomingServer"`
}

type autoconfigIncomingServer struct {
	Type     string `xml:"type,attr"`
	Hostname string `xml:"hostname"`
	Port     int    `xml:"port"`
}

type autodiscoverResponse struct {
	XMLName xml.Name            `xml:"Autodiscover"`
	Account autodiscoverAccount `xml:"Response>Account"`
}

type autodiscoverAccount struct {
	Protocols []autodiscoverProtocol `xml:"Protocol"`
}

type autodiscoverProtocol struct {
	Type   string `xml:"Type"`
	Server string `xml:"Server"`
	Port   int    `xml:"Port"`
}

// discoverAutoconfig tries to discover IMAP server using autoconfig/autodiscover endpoints
func discoverAutoconfig(domain, email string) (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Try autoconfig first (Mozilla Thunderbird format)
	urls := []string{
		fmt.Sprintf("https://autoconfig.%s/mail/config-v1.1.xml?emailaddress=%s", domain, email),
		fmt.Sprintf("https://autoconfig.%s/mail/config-v1.1.xml", domain),
		fmt.Sprintf("http://autoconfig.%s/mail/config-v1.1.xml", domain),
		// Try autodiscover (Microsoft Outlook format)
		fmt.Sprintf("https://autodiscover.%s/autodiscover/autodiscover.xml", domain),
		fmt.Sprintf("http://autodiscover.%s/autodiscover/autodiscover.xml", domain),
	}

	for _, url := range urls {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "newsletter-cli/1.0")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		// Try parsing as autoconfig format
		var autoconfig autoconfigResponse
		if err := xml.Unmarshal(body, &autoconfig); err == nil {
			for _, server := range autoconfig.Email.IncomingServers {
				if server.Type == "imap" {
					port := server.Port
					if port == 0 {
						port = 993 // Default IMAPS port
					}
					return fmt.Sprintf("%s:%d", server.Hostname, port), nil
				}
			}
		}

		// Try parsing as autodiscover format
		var autodiscover autodiscoverResponse
		if err := xml.Unmarshal(body, &autodiscover); err == nil {
			for _, protocol := range autodiscover.Account.Protocols {
				if strings.ToLower(protocol.Type) == "imap" {
					port := protocol.Port
					if port == 0 {
						port = 993 // Default IMAPS port
					}
					return fmt.Sprintf("%s:%d", protocol.Server, port), nil
				}
			}
		}

		// Try simple XML parsing by searching for IMAP-related tags
		bodyStr := string(body)
		if strings.Contains(bodyStr, "<hostname>") || strings.Contains(bodyStr, "<Server>") {
			// Try to extract IMAP server from XML using regex as fallback
			// Look for patterns like <hostname>mail.example.com</hostname> or <Server>mail.example.com</Server>
			if strings.Contains(strings.ToLower(bodyStr), "imap") {
				// Simple pattern matching for IMAP servers
				// This is a fallback if structured parsing fails
				lines := strings.Split(bodyStr, "\n")
				var hostname string
				var port int
				for i, line := range lines {
					lineLower := strings.ToLower(line)
					if strings.Contains(lineLower, "<hostname>") || strings.Contains(lineLower, "<server>") {
						// Extract hostname
						if idx := strings.Index(line, ">"); idx != -1 {
							if endIdx := strings.Index(line[idx+1:], "<"); endIdx != -1 {
								hostname = strings.TrimSpace(line[idx+1 : idx+1+endIdx])
							}
						}
					}
					if strings.Contains(lineLower, "<port>") {
						// Extract port
						if idx := strings.Index(line, ">"); idx != -1 {
							if endIdx := strings.Index(line[idx+1:], "<"); endIdx != -1 {
								fmt.Sscanf(line[idx+1:idx+1+endIdx], "%d", &port)
							}
						}
					}
					// Check if we have enough info and this section is about IMAP
					if i > 0 && strings.Contains(strings.ToLower(lines[i-1]), "imap") && hostname != "" {
						if port == 0 {
							port = 993
						}
						return fmt.Sprintf("%s:%d", hostname, port), nil
					}
				}
				if hostname != "" {
					if port == 0 {
						port = 993
					}
					return fmt.Sprintf("%s:%d", hostname, port), nil
				}
			}
		}
	}

	return "", fmt.Errorf("autoconfig/autodiscover not available")
}

// guessIMAPServer is kept for backward compatibility, now uses autodiscover
func guessIMAPServer(email string) (string, error) {
	return DiscoverIMAPServer(email)
}
