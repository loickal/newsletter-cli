package imap

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type NewsletterStat struct {
	Sender string
	Count  int
}

// FetchNewsletterStats connects to IMAP, fetches messages and groups newsletters.
func FetchNewsletterStats(server, email, password string, since time.Time) ([]NewsletterStat, error) {
	log.Println("📬 Connecting to IMAP for analysis...")
	c, err := client.DialTLS(server, &tls.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer c.Logout()

	if err := c.Login(email, password); err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	// Select INBOX
	_, err = c.Select("INBOX", false)
	if err != nil {
		return nil, fmt.Errorf("select INBOX failed: %w", err)
	}

	// Build search criteria
	criteria := imap.NewSearchCriteria()
	criteria.Since = since
	ids, err := c.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no emails found since %s", since.Format("2006-01-02"))
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchBodyStructure}, messages)
	}()

	stats := map[string]int{}
	for msg := range messages {
		if msg.Envelope == nil {
			continue
		}

		from := ""
		if len(msg.Envelope.From) > 0 {
			from = msg.Envelope.From[0].Address()
		}

		// Skip personal or reply emails heuristically
		if from == "" || strings.Contains(from, email) {
			continue
		}
		if !isLikelyNewsletter(from, msg.Envelope.Subject) {
			continue
		}

		stats[from]++
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	var results []NewsletterStat
	for sender, count := range stats {
		results = append(results, NewsletterStat{Sender: sender, Count: count})
	}

	return results, nil
}

// crude heuristic to detect newsletters
func isLikelyNewsletter(from, subject string) bool {
	keywords := []string{"newsletter", "digest", "update", "offers", "weekly", "report", "news"}
	for _, k := range keywords {
		if strings.Contains(strings.ToLower(subject), k) {
			return true
		}
	}
	domains := []string{"@news.", "@mailer.", "@updates.", "@notify.", "@mail."}
	for _, d := range domains {
		if strings.Contains(strings.ToLower(from), d) {
			return true
		}
	}
	return false
}
