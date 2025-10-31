# 📬 Newsletter CLI

> A beautiful, privacy-friendly terminal tool to analyze and unsubscribe from newsletters — right from your inbox.

[![Go Version](https://img.shields.io/github/go-mod/go-version/loickal/newsletter-cli)](https://go.dev/)
[![License](https://img.shields.io/github/license/loickal/newsletter-cli)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/loickal/newsletter-cli/ci.yml?branch=main)](https://github.com/loickal/newsletter-cli/actions)
[![Release](https://img.shields.io/github/v/release/loickal/newsletter-cli?include_prereleases&sort=semver)](https://github.com/loickal/newsletter-cli/releases)

---

## 🧠 Overview

**Newsletter CLI** connects to your mailbox (via IMAP), analyzes incoming messages, identifies recurring newsletter senders, and lets you **unsubscribe** from them interactively — all within a modern **Bubble Tea TUI**.

It’s local-first, secure, and open-source.  
No servers, no tracking, no nonsense.

---

## ✨ Features

✅ Connect via IMAP (Gmail, Outlook, etc.) with auto-discovery  
✅ Smart newsletter detection  
✅ Aggregated sender statistics  
✅ Interactive TUI built with [Charm Bracelet Bubble Tea](https://github.com/charmbracelet/bubbletea)  
✅ Mass unsubscribe with multiselect support  
✅ Automatic mailto: unsubscribe via SMTP  
✅ One-click unsubscribe via `List-Unsubscribe` header  
✅ Persistent tracking of unsubscribed newsletters  
✅ Multiple account management (add, switch, delete accounts)  
✅ Secure encryption using [age](https://filippo.io/age) (ChaCha20Poly1305)  
✅ Encrypted local credential storage  
✅ Config saved under `~/.config/newsletter-cli/config.json`

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

### 🪟 Winget (Windows)
```bash
winget install Loickal.NewsletterCLI
```

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
- Each account is stored with encrypted credentials
- Switch between accounts from the Accounts screen
- Each account has its own unsubscribe history

## 🗺️ Roadmap

The full roadmap is available in [ROADMAP.md](ROADMAP.md).

| Phase | Focus | Status |
|-------|-------|--------|
| **v0.2.0** | Mass unsubscribe, mailto support, multiple accounts | ✅ Complete |
| **v0.3.0** | Caching, logging, better config | 🟡 Planned |
| **v0.4.0** | CI/CD & release pipeline | ✅ Complete |
| **v1.0.0** | Stable release with all features | 🔜 Future |

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

MIT License © 2025 Loïc Kalbermatter

## 🌟 Acknowledgements

Thanks to the maintainers of:
- [emersion/go-imap](https://github.com/emersion/go-imap)
- [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea)
- [spf13/cobra](https://github.com/spf13/cobra)
- [filippo.io/age](https://filippo.io/age)

…and everyone contributing to open-source terminal tooling ❤️

> "Take control of your inbox. One unsubscribe at a time." — newsletter-cli