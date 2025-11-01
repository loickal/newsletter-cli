# 📬 Newsletter CLI

> A beautiful, privacy-friendly terminal tool to analyze and unsubscribe from newsletters — right from your inbox.

**Current Version:** `v1.0.0-BETA-1` 🚧 **Beta Release** - See [CHANGELOG.md](CHANGELOG.md) for details.

[![Go Version](https://img.shields.io/github/go-mod/go-version/loickal/newsletter-cli)](https://go.dev/)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Build Status](https://img.shields.io/github/actions/workflow/status/loickal/newsletter-cli/ci.yml?branch=main)](https://github.com/loickal/newsletter-cli/actions)
[![Release](https://img.shields.io/github/v/release/loickal/newsletter-cli?include_prereleases&sort=semver)](https://github.com/loickal/newsletter-cli/releases)

---

## 🧠 Overview

**Newsletter CLI** connects to your mailbox (via IMAP), analyzes incoming messages, identifies recurring newsletter senders, and lets you **unsubscribe** from them interactively — all within a modern **Bubble Tea TUI**.

It’s local-first, secure, and open-source.  
No servers, no tracking, no nonsense.

---

## ✨ Features

### Core Features (Free & Open Source)
✅ Connect via IMAP (Gmail, Outlook, etc.) with auto-discovery  
✅ Smart newsletter detection  
✅ Aggregated sender statistics  
✅ Interactive TUI built with [Charm Bracelet Bubble Tea](https://github.com/charmbracelet/bubbletea)  
✅ Mass unsubscribe with multiselect support  
✅ Automatic mailto: unsubscribe via SMTP  
✅ One-click unsubscribe via `List-Unsubscribe` header  
✅ Persistent tracking of unsubscribed newsletters  
✅ Multiple account management (add, switch, delete accounts)  
  - *Note: First account is free. Additional accounts require premium subscription*  
✅ Secure encryption using [age](https://filippo.io/age) (ChaCha20Poly1305)  
✅ Encrypted local credential storage  
✅ Config saved under `~/.config/newsletter-cli/config.json`

### 🌟 Premium Features (Subscription Required)
**Available via subscription** - See [Premium Features](#-premium-features) section below.

✅ **Cloud Sync** - Sync accounts and unsubscribed lists across devices  
✅ **Analytics Dashboard** - Web-based analytics with charts and trends  
✅ **Advanced Analytics** - Newsletter categorization and quality scoring  
✅ **Usage Statistics** - Track API usage and monitor your account  
✅ **Rate Limiting** - Tier-based API limits (Starter, Pro, Enterprise)  
  - *Note: Rate limits may change and will be communicated in updates*  
✅ **Account Limits** - Server-side enforced account limits per tier
  - Starter: 3 accounts, Pro: 10 accounts, Enterprise: 50 accounts
  - Cannot be bypassed by modifying client code  
✅ **Secure API** - Optional HMAC request signing for extra security

---

## 🧱 Installation

### 🐹 Go Install
```bash
go install github.com/loickal/newsletter-cli@latest
```

### 🍺 Homebrew
```bash
brew tap loickal/newsletter-cli
brew install newsletter-cli
```

### 🪟 Winget (Windows) - Coming Soon
Microsoft approval pending. Once approved:
```bash
winget install Loickal.NewsletterCLI
```

**Alternative for Windows**: Use Go install or download binaries from GitHub releases.

### 🐳 Docker
```bash
docker run --rm -it -v ~/.config/newsletter-cli:/config loickal/newsletter-cli analyze
```
## 🚀 Quick Start

### 1️⃣ Login once
```bash
newsletter-cli login
```
Enter your IMAP credentials — they're verified and saved locally (encrypted with age encryption).

### 2️⃣ Analyze newsletters
```bash
newsletter-cli analyze
```

➡️ Displays an interactive dashboard:
```
╭──────────────────────────────────────────────╮
│  📬 Newsletter Overview                       │
│  Total: 5 newsletters • 42 emails            │
│──────────────────────────────────────────────│
│  ✓✓ GitHub News Digest          (12 emails)  │
│     ✅ Already unsubscribed                   │
│  ▸ AWS Weekly Updates          (5 emails)   │
│     🔗 unsubscribe.link/abc123               │
│  📬 Stack Overflow Digest       (3 emails)  │
│     🔗 mailto:unsubscribe@stackoverflow.com │
│                                              │
│  [↑↓] Navigate  [Space] Select  [U] Mass    │
│  [u] Single  [/] Search  [Esc] Clear  [q]   │
╰──────────────────────────────────────────────╯
```

### 3️⃣ Manage Accounts
```bash
newsletter-cli
```
Navigate to "👤 Accounts" to:
- Add multiple email accounts
- Switch between accounts
- Delete accounts
- View active account

### Keybindings

**Dashboard:**
- `↑↓` - Navigate newsletters
- `Space` - Select/deselect for mass unsubscribe
- `U` - Unsubscribe from all selected newsletters
- `u` - Single unsubscribe (opens browser for HTTP links)
- `/` - Search/filter newsletters
- `Esc` - Clear selection
- `q` - Quit

## ⚙️ Configuration

| Path | Description |
|------|-------------|
| `~/.config/newsletter-cli/config.json` | Stores all email accounts with encrypted passwords |
| `~/.config/newsletter-cli/unsubscribed.json` | Tracks newsletters you've unsubscribed from |

### CLI Flags

You can override credentials using CLI flags:
```bash
newsletter-cli analyze --email foo@example.com --server imap.example.com:993 --days 60
```

### Multiple Accounts

Manage multiple email accounts:
- **First account is free** - No subscription required
- **Additional accounts require premium subscription**
- Each account is stored with encrypted credentials
- Switch between accounts from the Accounts screen
- Each account has its own unsubscribe history

## 🌟 Premium Features

**Newsletter CLI** offers premium features via subscription to support development and infrastructure costs.

### Available Plans

| Plan | Price | Features | Account Limit |
|------|-------|----------|---------------|
| **Starter** | $5/month | Cloud sync, Basic analytics, Web dashboard | 3 accounts |
| **Pro** | $12/month | Everything in Starter + Advanced analytics, Smart scheduling (coming soon) | 10 accounts |
| **Enterprise** | $50/month | Everything in Pro + Team features, Compliance reporting (coming soon) | 50 accounts |

### Premium Features Overview

#### ☁️ Cloud Sync
- Sync email accounts across all your devices
- Sync unsubscribed newsletters list
- Automatic conflict resolution with three-way merge
- Offline queue for failed syncs
- Automatic retry with background processing
- **Account limits enforced server-side** - Cannot be bypassed by modifying client code

#### 📊 Analytics Dashboard
- Beautiful web-based dashboard with interactive charts
- View trends over day, week, month, or year
- Newsletter and email statistics
- Unsubscribe tracking and insights
- One-click access from CLI (`[w]` key in Premium screen)

#### 🎯 Advanced Analytics (Pro+)
- **Newsletter Categorization**: Automatic classification into 7 categories
  - Technology, Finance, Marketing, Subscriptions, Promotional, News/Media, Other
- **Quality Scoring**: 0-100 score based on frequency, unsubscribe ease, and category
  - Displayed in CLI with star ratings (⭐⭐⭐⭐⭐)
- **Period-over-Period Insights**: Compare current vs previous periods
  - Percentage changes and trend analysis

#### 📈 Usage Statistics
- Track API usage over time
- View requests per endpoint
- Monitor rate limit status
- Access via CLI (`[v]` key in Premium screen)

#### 🔒 Security Features
- Tier-based rate limiting (30-500 requests/minute based on plan)
  - *Note: Rate limits are subject to change and will be communicated via updates*
- Usage tracking for abuse detection
- Optional HMAC request signing
- Server-side feature validation (cannot be bypassed)

### Getting Started with Premium

1. **Launch the app** and navigate to `☁️ Premium` from the main menu
2. **Register/Login** with your email and password
3. **Subscribe** - Press `[u]` to view plans and subscribe via Stripe
4. **Enable features** - Premium features activate automatically after subscription

### Premium Configuration

Premium settings are stored in `~/.config/newsletter-cli/premium.json`:
- API credentials (encrypted)
- Sync preferences
- Analytics settings
- Subscription status

---

## 🗺️ Roadmap

The full roadmap is available in [ROADMAP.md](ROADMAP.md).

| Phase | Focus | Status |
|-------|-------|--------|
| **v0.2.0** | Mass unsubscribe, mailto support, multiple accounts | ✅ Complete |
| **v0.4.0** | CI/CD & release pipeline | ✅ Complete |
| **v1.0.0-BETA-1** | Premium features, cloud sync, analytics, account limits | ✅ **Current** |
| **v1.0.0** | Stable release with comprehensive testing | 🔜 Future |

## 🧩 Tech Stack

| Area | Library |
|------|---------|
| CLI Framework | spf13/cobra |
| UI / TUI | charmbracelet/bubbletea, lipgloss |
| IMAP Access | emersion/go-imap |
| Encryption | filippo.io/age |
| SMTP | net/smtp (for mailto unsubscribe) |
| Packaging | GoReleaser |

## 🧪 Development

Clone and run locally:
```bash
git clone https://github.com/loickal/newsletter-cli.git
cd newsletter-cli
go run main.go analyze
```

Run all tests:
```bash
go test ./...
```
## 🧠 Semantic Commits

Follow the Conventional Commits style:

| Type | Example |
|------|---------|
| `feat(ui)` | add search filter to TUI |
| `feat(unsubscribe)` | implement mass unsubscribe |
| `fix(imap)` | handle auth errors gracefully |
| `chore(release)` | bump to v0.2.0 |

## 🤝 Contributing

Contributions are welcome!

1. Fork the repo
2. Create a feature branch (`feat/add-spinner`)
3. Write clear, semantic commits
4. Open a PR → review & merge

Please read [ROADMAP.md](ROADMAP.md) for feature planning before proposing major changes.

## 🪪 License

This project is licensed under the **GNU Affero General Public License v3.0 (AGPL-3.0)**.

**AGPL-3.0** is a strong copyleft license that requires:
- You may use, modify, and distribute the software
- If you modify the software and provide it over a network, you must share the source code
- All modifications must be released under the same AGPL-3.0 license

See the [LICENSE](LICENSE) file for the full license text, or visit [https://www.gnu.org/licenses/agpl-3.0.html](https://www.gnu.org/licenses/agpl-3.0.html) for more information.

© 2025 Loïc Kalbermatter

## 🌟 Acknowledgements

Thanks to the maintainers of:
- [emersion/go-imap](https://github.com/emersion/go-imap)
- [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea)
- [spf13/cobra](https://github.com/spf13/cobra)
- [filippo.io/age](https://filippo.io/age)

…and everyone contributing to open-source terminal tooling ❤️

> "Take control of your inbox. One unsubscribe at a time." — newsletter-cli