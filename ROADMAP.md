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

## 🧩 Completed Features (v0.1.0 - v1.0.0-BETA-1)

### Core Features (v0.1.0 - v0.2.0) ✅
✅ IMAP login & server autodiscovery (DNS SRV, autoconfig/autodiscover)  
✅ Fetch & analyze recent emails  
✅ Heuristic newsletter detection  
✅ Interactive Bubble Tea dashboard  
✅ Real unsubscribe link detection (via `List-Unsubscribe` header)  
✅ Secure encryption using age (ChaCha20Poly1305)  
✅ Config persistence & local encryption  
✅ Multiple account management (add, switch, delete)  
✅ Mass unsubscribe with multiselect  
✅ Automatic mailto: unsubscribe via SMTP  
✅ Persistent tracking of unsubscribed newsletters  
✅ CLI flags for non-interactive usage  
✅ Color-coded counts and visual indicators  
✅ Search/filter functionality  
✅ Auto-discovery of IMAP servers  
✅ GUI launcher (double-click support)  
✅ Cross-platform support (macOS, Linux, Windows)  
✅ Homebrew package distribution  
✅ Winget package (pending Microsoft approval)

### Premium Features (v1.0.0-BETA-1) ✅ **NEW**
✅ **Cloud Sync** - Sync accounts and unsubscribed lists across devices  
✅ **Conflict Resolution** - Three-way merge with automatic conflict detection  
✅ **Stripe Subscriptions** - Full subscription management (Starter, Pro, Enterprise)  
✅ **Analytics Dashboard** - Web-based dashboard with charts and trends  
✅ **Advanced Analytics** - Newsletter categorization & quality scoring  
✅ **Usage Tracking** - API usage statistics and abuse detection  
✅ **Rate Limiting** - Tier-based API rate limits  
✅ **HMAC Signing** - Optional request signing for extra security  
✅ **Server-Side Security** - All premium features secured server-side  
✅ **Account Limit Validation** - Server-side enforced account limits
  - Starter: 3 accounts, Pro: 10 accounts, Enterprise: 50 accounts
  - Validates on every sync attempt, cannot be bypassed

---

## 📈 Phase 2 – Core Enhancements ✅ COMPLETE

### 🔹 Auto-Unsubscribe Mode ✅
- ✅ Mass unsubscribe with multiselect (`Space` to select, `U` to unsubscribe)
- ✅ Handle both HTTP (GET/POST) and `mailto:` unsubscribe links
- ✅ Automatic SMTP server detection for mailto links
- ✅ Display summary after unsubscribe operations

### 🔹 Multiple Account Management ✅
- ✅ Add, switch, and delete email accounts
- ✅ Persistent account selection
- ✅ Account management UI screen

### 🔹 Command-Line Flags ✅
- ✅ Support non-interactive usage:
  ```bash
  newsletter-cli analyze --days 60 --email foo@example.com --server imap.gmail.com:993
  ```

### 🔹 Persistent Unsubscribe Tracking ✅
- ✅ Save unsubscribed newsletters to `unsubscribed.json`
- ✅ Visual indicators for already unsubscribed items
- ✅ Persistent across sessions

---

## 💅 Phase 3 – UI & UX Polish ✅ MOSTLY COMPLETE

### 🔹 TUI Improvements ✅
- ✅ Search / filter bar (`bubbles/textinput`) - `/` key
- ✅ Color-coded counts (green → red scale)
- ✅ Keymap help shown in footer
- ✅ Header summary (total newsletters & emails)
- ✅ Visual indicators for selected and unsubscribed items

### 🔹 Loading & Feedback ✅
- ✅ Spinner animation while fetching (`bubbles/spinner`)
- ✅ Status messages during operations
- ✅ Error handling and user feedback

### 🔹 Remaining
- Keymap help dialog (`?` key)
- Confirmation dialog for unsubscribing (currently direct)
- Persistent selection memory per sender

---

## 🏗️ Phase 4 – Architecture & Quality 🟡 IN PROGRESS

### 🔹 Secure Storage ✅
- ✅ Replace XOR encryption with [`filippo.io/age`](https://pkg.go.dev/filippo.io/age)
- ✅ Store credentials and tokens securely in `$HOME/.config/newsletter-cli/`
- ✅ System-specific key derivation (machine-bound encryption)

### 🔹 Logging & Verbosity ⏳
- Integrate `zerolog` or Go `slog` for structured logging
- `--debug` and `--silent` flags

### 🔹 Configuration ⏳
- Fully integrate `viper` for env/flag/config overrides
- Consistent defaults across OSes
- Account naming/labeling support

### 🔹 Local Cache ⏳
- Use BoltDB for sender cache (faster re-runs)
- Track unsubscribe status per sender
- Cache IMAP server discovery results

---

## 🌍 Phase 5 – Distribution & Release ✅ COMPLETE

### 🔹 Automated Releases ✅
- ✅ [GoReleaser](https://goreleaser.com/) configuration
- ✅ GitHub Actions workflow for tagged releases
- ✅ Automatic release on tag push

### 🔹 Packaging ✅
- ✅ Build binaries for macOS, Linux, Windows (amd64, arm64)
- ✅ Homebrew tap:
  ```bash
  brew install loickal/newsletter-cli/newsletter-cli
  ```
- ✅ Winget package manager (Windows) - **Pending Microsoft approval**:
  ```bash
  winget install Loickal.NewsletterCLI  # Coming soon
  ```
- ✅ Docker image:
  ```bash
  docker run --rm -it -v ~/.config/newsletter-cli:/config loickal/newsletter-cli analyze
  ```
- ✅ GUI launcher (double-click support on macOS, Linux, Windows)

---

## ⚙️ Phase 6 – Testing & CI/CD

- Unit tests for header parsing & IMAP logic
- Mock IMAP backend (`emersion/go-imap/backend/memory`)
- GitHub Action: `go test`, `go vet`, `golangci-lint`
- Integration test with fake inbox dataset

---

## 🧠 Phase 7 – Future Premium Features

| Feature | Description | Status |
|----------|-------------|--------|
| 📊 Trend Dashboard | ✅ Visual charts of newsletters/week (web export) | ✅ **Complete** |
| ☁️ Cloud Sync | ✅ Sync unsubscribed list across devices | ✅ **Complete** |
| 📈 Analytics | ✅ Analytics collection and web dashboard | ✅ **Complete** |
| 🎯 Advanced Analytics | ✅ Newsletter categorization and quality scoring | ✅ **Complete** |
| 🔔 Notifications | Optional Discord / Telegram alerts for new senders | 🔜 Future |
| 🤖 AI Classifier | Use LLM to classify "useful" vs "spammy" newsletters | 🔜 Future |
| 📅 Smart Scheduling | Gradual unsubscribe automation with queue system | 🔜 Future |
| 🔗 Webhooks | Integration webhooks for automation | 🔜 Future |
| 👥 Team Workspaces | Multi-user workspaces (Enterprise) | 🔜 Future |
| 📋 Compliance Reports | GDPR/CCPA audit logs and reports (Enterprise) | 🔜 Future |

---

## 🧩 Milestones

| Version | Scope | Status |
|----------|--------|--------|
| **v0.1.0** | MVP: IMAP connection, newsletter detection, basic TUI | ✅ Complete |
| **v0.2.0** | Mass unsubscribe, mailto support, multiple accounts, age encryption | ✅ Complete |
| **v0.4.0** | CI/CD with GoReleaser, Homebrew (Winget pending) | ✅ Complete |
| **v1.0.0-BETA-1** | Premium features, cloud sync, analytics, subscriptions, account limits | ✅ **Current** |
| **v1.0.0** | Stable release with comprehensive testing + docs | 🔜 Future |

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
**License:** AGPL-3.0  
**Current Version:** `v1.0.0-BETA-1`