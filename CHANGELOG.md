# 📦 Changelog

All notable changes to this project will be documented here.

The format is based on [Conventional Commits](https://www.conventionalcommits.org).

---

## [v1.0.0-BETA-1] - 2025-11-01

### 🎉 Beta Release

This is the first beta release of Newsletter CLI with premium features! We're excited to share our subscription-based features while gathering feedback for the stable v1.0.0 release.

### ⚠️ Beta Notice

This is a **beta release**. Some features may have bugs, and we're actively working on improvements based on user feedback. For production use, we recommend waiting for the stable v1.0.0 release.

### ✨ Added

#### Premium Features
- **Cloud Sync** - Sync email accounts and unsubscribed lists across all your devices
  - Automatic conflict resolution with three-way merge
  - Offline queue for failed syncs
  - Automatic retry with background processing
  - Server-side validation and security
  
- **Analytics Dashboard** - Beautiful web-based dashboard with interactive charts
  - View trends over day, week, month, or year periods
  - Newsletter and email statistics
  - Unsubscribe tracking and insights
  - One-click access from CLI (`[w]` key in Premium screen)

- **Advanced Analytics** (Pro+ tiers)
  - **Newsletter Categorization**: Automatic classification into 7 categories
    - Technology, Finance, Marketing, Subscriptions, Promotional, News/Media, Other
  - **Quality Scoring**: 0-100 score with star ratings (⭐⭐⭐⭐⭐)
    - Based on frequency, unsubscribe ease, and category
  - **Period-over-Period Insights**: Compare current vs previous periods

- **Usage Statistics** - Track API usage and monitor your account
  - View requests per endpoint
  - Monitor rate limit status
  - Access via CLI (`[v]` key in Premium screen)

- **Stripe Subscription Integration**
  - Plan selection UI with feature display
  - One-click checkout via Stripe Checkout
  - Subscription management via Stripe Customer Portal
  - Automatic subscription status sync via webhooks
  - Three tiers: Starter ($5/mo), Pro ($12/mo), Enterprise ($50/mo)

- **Security Features**
  - **Tier-based Rate Limiting**: Starter (30/min), Pro (100/min), Enterprise (500/min)
  - **Usage Tracking**: Automatic logging for abuse detection
  - **Optional HMAC Request Signing**: Extra security for API requests
  - **Server-Side Feature Validation**: All premium features secured server-side

- **Account Management**
  - **Multiple Accounts Support**: Manage multiple email accounts
  - **First Account Free**: No subscription required for first account
  - **Account Limits**: Client-side and server-side validation
    - Starter: 3 accounts, Pro: 10 accounts, Enterprise: 50 accounts
    - Client-side checks prevent adding accounts exceeding tier limits
    - Server-side validation enforces limits on sync (cannot be bypassed)
    - Clear error messages guide users to upgrade when limit reached

#### Core Features
- Multiple account management (free users get 1 account, premium required for additional)
- Account switching and deletion
- Per-account unsubscribe history tracking

### 🔒 Security

- **Server-Side Validation**: All premium features validated server-side
  - Cannot be bypassed by modifying client code
  - Subscription status checked on every API request
  - Account limits enforced on sync attempts

- **Privacy-First**: Analytics data anonymized (SHA-256 hashing)
  - Sender domains hashed
  - Account IDs hashed
  - No personally identifiable information stored

### 🐛 Known Issues

- Beta release - some edge cases may not be handled perfectly
- Rate limits may be adjusted based on usage patterns
- Webhook processing may have minor delays in subscription status updates

### 📝 Documentation

- Comprehensive README with premium features section
- Updated ROADMAP with current feature status
- Security documentation for premium features
- Account limit documentation

### 🔄 Changed

- **License**: Changed from MIT to AGPL-3.0
- **Multiple Accounts**: Now requires premium subscription (first account remains free)
- **Premium Features**: Moved categorization and quality scoring to server-side API

### 📦 Distribution

- Homebrew: `brew install loickal/newsletter-cli/newsletter-cli`
- Go Install: `go install github.com/loickal/newsletter-cli@v1.0.0-BETA-1`
- Docker: Available via GitHub releases
- Winget: Pending Microsoft approval

---

## [v0.4.0] - 2024

### Added
- CI/CD with GoReleaser
- Automated releases
- Homebrew tap
- Winget package (pending approval)

## [v0.2.0] - 2024

### Added
- Mass unsubscribe with multiselect
- Mailto unsubscribe support
- Multiple account management
- Age encryption for credentials

## [v0.1.0] - 2024

### Added
- Initial MVP release
- IMAP connection and newsletter detection
- Basic TUI dashboard
- Unsubscribe functionality

---

[Unreleased]: https://github.com/loickal/newsletter-cli/compare/v1.0.0-BETA-1...HEAD
[v1.0.0-BETA-1]: https://github.com/loickal/newsletter-cli/compare/v0.4.0...v1.0.0-BETA-1
