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

✅ Connect via IMAP (Gmail, Outlook, etc.)  
✅ Smart newsletter detection  
✅ Aggregated sender statistics  
✅ Interactive TUI built with [Charm Bracelet Bubble Tea](https://github.com/charmbracelet/bubbletea)  
✅ One-click unsubscribe via `List-Unsubscribe` header  
✅ Encrypted local credential storage  
✅ Config saved under `~/.config/newsletter-cli/config.json`

---

## 🧱 Installation

### 🐹 Go Install
```bash
go install github.com/loickal/newsletter-cli@latest
🍺 Homebrew (coming soon)
bash
Copy code
brew install loickal/newsletter-cli/newsletter-cli
🐳 Docker
bash
Copy code
docker run --rm -it -v ~/.config/newsletter-cli:/config loickal/newsletter-cli analyze
🚀 Quick Start
1️⃣ Login once
bash
Copy code
newsletter-cli login
Enter your IMAP credentials — they’re verified and saved locally (encrypted).

2️⃣ Analyze newsletters
bash
Copy code
newsletter-cli analyze
➡️ Displays an interactive dashboard like:

scss
Copy code
╭────────────────────────────────────────────╮
│  📬 Newsletter Overview (Last 30 Days)     │
│────────────────────────────────────────────│
│  ▸ GitHub News Digest          (12 emails) │
│    AWS Weekly Updates          (5 emails)  │
│    Stack Overflow Digest       (3 emails)  │
│                                            │
│  [↑↓] Navigate   [u] Unsubscribe  [q] Quit │
╰────────────────────────────────────────────╯
Press u to unsubscribe, q to quit.

⚙️ Configuration
Path	Description
~/.config/newsletter-cli/config.json	Stores email, IMAP server, and encrypted password
~/.cache/newsletter-cli/ (planned)	Local cache of senders and unsubscribe links

You can override credentials using CLI flags (coming soon):

bash
Copy code
newsletter-cli analyze --email foo@example.com --server imap.example.com:993
🗺️ Roadmap
The full roadmap is available in ROADMAP.md.
Highlights:

Phase	Focus
v0.2.0	Auto-unsubscribe & CLI flags
v0.3.0	Caching, logging, better config
v0.4.0	CI/CD & release pipeline
v1.0.0	Stable release with Homebrew support

🧩 Tech Stack
Area	Library
CLI Framework	spf13/cobra
UI / TUI	charmbracelet/bubbletea, lipgloss
IMAP Access	emersion/go-imap
Config	spf13/viper
Packaging	GoReleaser (planned)

🧪 Development
Clone and run locally:

bash
Copy code
git clone https://github.com/loickal/newsletter-cli.git
cd newsletter-cli
go run main.go analyze
Run all tests:

bash
Copy code
go test ./...
🧠 Semantic Commits
Follow the Conventional Commits style:

Type	Example
feat(ui): add search filter to TUI	
feat(unsubscribe): implement --all flag	
fix(imap): handle auth errors gracefully	
chore(release): bump to v0.2.0	

🤝 Contributing
Contributions are welcome!

Fork the repo

Create a feature branch (feat/add-spinner)

Write clear, semantic commits

Open a PR → review & merge

Please read ROADMAP.md for feature planning before proposing major changes.

🪪 License
MIT License © 2025 Loïc Kalbermatter

🌟 Acknowledgements
Thanks to the maintainers of:

emersion/go-imap

charmbracelet/bubbletea

spf13/cobra

…and everyone contributing to open-source terminal tooling ❤️

“Take control of your inbox. One unsubscribe at a time.” — newsletter-cli