# Branching And Release

## Branch Model

- `main` is the stable branch used for weekly promotion and release publishing.
- `staging` is the active integration branch.
- Feature and bug-fix branches should be created from `staging`.
- Feature and bug-fix pull requests should target `staging`.
- The scheduled promotion pull request should merge `staging` into `main`.

## CI Expectations

- Pull requests into `staging` and `main` run Go formatting, vet, test, and build checks.
- Pushes to `staging` and `main` run the same CI workflow.

## Release Automation

- Every push to `main` computes the next patch tag in the `v0.1.x` line.
- A GitHub Release is created with generated archives for Linux, macOS, and Windows.
- Release notes are generated from the commits included since the previous release tag.

## Documentation Deployment

- Markdown content from `docs/` is published to GitHub Pages through MkDocs.
- The Pages workflow runs on changes to the documentation site inputs on `main`.

## Suggested Weekly Cadence

1. Developers branch from `staging`.
2. Developers open pull requests back into `staging`.
3. `staging` stays open for QA and review during the week.
4. At the scheduled release window, open a promotion pull request from `staging` to `main`.
5. Merging that promotion triggers the GitHub Release and updates GitHub Pages if docs changed.
