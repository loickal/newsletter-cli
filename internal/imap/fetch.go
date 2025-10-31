package imap

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type NewsletterStat struct {
	Sender      string
	Count       int
	Unsubscribe string
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

	_, err = c.Select("INBOX", false)
	if err != nil {
		return nil, fmt.Errorf("select INBOX failed: %w", err)
	}

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
		section := &imap.BodySectionName{}
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, section.FetchItem()}, messages)
	}()

	type seen struct {
		count int
		link  string
	}
	stats := map[string]seen{}

	for msg := range messages {
		if msg.Envelope == nil || len(msg.Envelope.From) == 0 {
			continue
		}
		from := msg.Envelope.From[0].Address()
		if from == "" || strings.Contains(from, email) {
			continue
		}
		if !isLikelyNewsletter(from, msg.Envelope.Subject) {
			continue
		}

		// Parse raw header for List-Unsubscribe
		var link string
		if r := msg.GetBody(&imap.BodySectionName{}); r != nil {
			buf := new(bytes.Buffer)
			buf.ReadFrom(r)
			m, err := mail.ReadMessage(bytes.NewReader(buf.Bytes()))
			if err == nil {
				lh := m.Header.Get("List-Unsubscribe")
				link = extractUnsubscribeLink(lh)
			}
		}

		entry := stats[from]
		entry.count++
		if entry.link == "" && link != "" {
			entry.link = link
		}
		stats[from] = entry
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	var results []NewsletterStat
	for sender, s := range stats {
		results = append(results, NewsletterStat{Sender: sender, Count: s.count, Unsubscribe: s.link})
	}
	return results, nil
}

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

var reLink = regexp.MustCompile(`<([^>]+)>`)

func extractUnsubscribeLink(header string) string {
	if header == "" {
		return ""
	}
	m := reLink.FindStringSubmatch(header)
	if len(m) > 1 {
		return m[1]
	}
	// Sometimes it’s just a raw URL or mailto
	if strings.HasPrefix(header, "http") || strings.HasPrefix(header, "mailto") {
		return strings.TrimSpace(header)
	}
	return ""
}
