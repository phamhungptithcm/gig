# gig

`gig` is a cross-platform Go CLI for release workflows that need to track ticket-related commits across multiple repositories before promotion.

## What The MVP Covers

- recursive workspace scanning
- Git repository detection
- ticket-based commit search across repositories
- branch comparison for a ticket using Git-first logic
- human-readable grouped CLI output

## Delivery Flow

- `staging` is the integration branch.
- Feature and bug-fix branches start from `staging` and open pull requests back into `staging`.
- `main` receives the scheduled promotion from `staging`.
- Every push to `main` produces the next release tag, release notes, and build artifacts.

## Documentation Map

- [Product overview](00-product-overview.md)
- [CLI specification](03-cli-spec.md)
- [Architecture](04-architecture.md)
- [Test strategy](11-test-strategy.md)
- [Roadmap](13-roadmap.md)
- [Branching and release](15-branching-and-release.md)
