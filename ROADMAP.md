# 🗺️ Newsletter CLI Roadmap

> **Repository:** [github.com/loickal/newsletter-cli](https://github.com/loickal/newsletter-cli)

A modern, privacy-friendly terminal tool to analyze and clean up newsletters directly from your inbox.

---

## 🚀 Project Vision

`newsletter-cli` helps users reclaim control over their inbox by:
- Analyzing incoming emails via IMAP
- Identifying recurring newsletters
- Displaying sender statistics in an elegant TUI
- Offering one-click or automated unsubscribing

Built with **Go**, **Cobra**, and **Bubble Tea** — focused on simplicity, local privacy, and delightful UX.

---

## 🧩 Current MVP (v0.1.0)

✅ IMAP login & server autodetect  
✅ Fetch & analyze recent emails  
✅ Heuristic newsletter detection  
✅ Interactive Bubble Tea dashboard  
✅ Real unsubscribe link detection (via `List-Unsubscribe`)  
✅ Config persistence & local encryption

---

## 📈 Phase 2 – Core Enhancements

### 🔹 Auto-Unsubscribe Mode
- Add `newsletter-cli unsubscribe --all`
- Handle both HTTP and `mailto:` unsubscribe links
- Display summary after run

### 🔹 Improved Newsletter Recognition
- Detect via headers (`Precedence`, `List-Id`, `Auto-Submitted`)
- Cache results in local BoltDB / SQLite
- Flag suspicious or duplicate senders

### 🔹 Command-Line Flags
- Support non-interactive usage:
  newsletter-cli analyze --days 60 --email foo@example.com --server imap.gmail.com:993

markdown
Copy code

---

## 💅 Phase 3 – UI & UX Polish

### 🔹 TUI Improvements
- Search / filter bar (`bubbles/textinput`)
- Color-coded counts (green → red scale)
- Keymap help (`?` dialog)
- Header summary (total newsletters & emails)

### 🔹 Loading & Feedback
- Spinner animation while fetching (`bubbles/spinner`)
- Confirmation dialog for unsubscribing
- Persistent selection memory per sender

---

## 🏗️ Phase 4 – Architecture & Quality

### 🔹 Secure Storage
- Replace XOR encryption with [`filippo.io/age`](https://pkg.go.dev/filippo.io/age)
- Store credentials and tokens securely in `$HOME/.config/newsletter-cli/`

### 🔹 Logging & Verbosity
- Integrate `zerolog` or Go `slog` for structured logging
- `--debug` and `--silent` flags

### 🔹 Configuration
- Fully integrate `viper` for env/flag/config overrides
- Consistent defaults across OSes

### 🔹 Local Cache
- Use BoltDB for sender cache (faster re-runs)
- Track unsubscribe status per sender

---

## 🌍 Phase 5 – Distribution & Release

### 🔹 Automated Releases
- Add [GoReleaser](https://goreleaser.com/) configuration
- GitHub Actions workflow for tagged releases

### 🔹 Packaging
- Build binaries for macOS, Linux, Windows
- Publish Homebrew tap:
  brew install loickal/newsletter-cli/newsletter-cli

arduino
Copy code
- Optional Docker image for automation:
  docker run --rm -it -v ~/.config/newsletter-cli:/config loickal/newsletter-cli analyze

yaml
Copy code

---

## ⚙️ Phase 6 – Testing & CI/CD

- Unit tests for header parsing & IMAP logic
- Mock IMAP backend (`emersion/go-imap/backend/memory`)
- GitHub Action: `go test`, `go vet`, `golangci-lint`
- Integration test with fake inbox dataset

---

## 🧠 Phase 7 – Future “Pro” Features

| Feature | Description |
|----------|-------------|
| 📊 Trend Dashboard | Visual charts of newsletters/week (ASCII or web export) |
| 🔔 Notifications | Optional Discord / Telegram alerts for new senders |
| 🤖 AI Classifier | Use LLM to classify “useful” vs “spammy” newsletters |
| ☁️ Cloud Sync | Sync unsubscribed list across devices |
| 💌 Smart Inbox | Export non-newsletter messages for backup |

---

## 🧩 Milestones

| Version | Scope | Status |
|----------|--------|--------|
| **v0.2.0** | Auto-unsubscribe, flags, UI polish | 🟡 In progress |
| **v0.3.0** | Config, caching, logging improvements | ⏳ Planned |
| **v0.4.0** | CI/CD with GoReleaser & Docker | ⏳ Planned |
| **v1.0.0** | Full release with docs + Homebrew support | 🔜 Future |

---

## 🧠 Commit Convention

Follow [Conventional Commits](https://www.conventionalcommits.org):

| Type | Example |
|------|----------|
| `feat(ui): add search filter to dashboard` |
| `feat(unsubscribe): support --all flag` |
| `refactor(imap): move parsing to separate package` |
| `fix(config): ensure secure file permissions` |
| `chore(release): bump version to v0.2.0` |

---

## ❤️ Contributing

1. Fork the repo
2. Create a feature branch (`feat/unsubscribe-automation`)
3. Follow semantic commits
4. Submit PR → tagged reviewers

---

## 🧭 Summary

The long-term vision is to make **Newsletter CLI** the “brew install inbox hygiene” tool:
lightweight, cross-platform, and privacy-respecting — no web tracking, no cloud sync by default, just pure terminal control.

---

**Maintainer:** [@loickal](https://github.com/loickal)  
**License:** MIT  
**Version:** `v0.1.0` (MVP)