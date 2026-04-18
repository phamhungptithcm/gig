# Branching And Release

## Branch Model

- `staging` is the active integration branch
- feature and bug-fix branches start from `staging`
- pull requests should go back into `staging`
- `main` is the stable branch for releases
- the scheduled promotion is `staging -> main`

## CI Expectations

- pull requests into `staging` and `main` run formatting, vet, test, and build checks
- pushes to `staging` and `main` run the same CI checks

## Release Automation

When code is pushed to `main`:

- the next release tag is calculated in `vYYYY.MM.DD` format
- release archives are built for macOS, Linux, and Windows
- release notes are generated from the commits since the previous tag
- release notes are grouped by area instead of one flat commit list
- breaking commits are surfaced as upgrade notes
- a compare link is added when the repository remote can be resolved to GitHub
- GitHub Release assets are published
- the npm package `@hunpeolabs/gig` is published from this same repository
- the release tag is mapped to npm package version `YYYY.M.D` for registry compatibility

The release workflow is split into:

- metadata resolution
- release verification across Go, docs, and npm package staging
- npm publication
- npm registry verification with retry for first-visibility lag
- GitHub Release publication after npm succeeds

npm publication is now a release requirement.
If neither trusted publishing nor a token fallback is configured, the release workflow fails during verification before it creates a GitHub Release.

npm publication supports two modes:

1. `trusted`
   steady-state publish through GitHub Actions OIDC after the package already exists on npm
2. `token`
   bootstrap or fallback publish through repository secret `NPM_PUBLISH_TOKEN`

The safer steady-state setup is:

1. open npm package settings for `@hunpeolabs/gig`
2. add a trusted publisher for GitHub Actions
3. set owner `phamhungptithcm`
4. set repository `gig`
5. set workflow filename `release.yml`
6. set environment name `npm-release`
7. optionally require approvals on the GitHub environment `npm-release`
8. set repository variable `NPM_TRUSTED_PUBLISHING=true` after npm trusted publishing is configured and ready

The npm package scope and the GitHub repository owner are different here:

- npm package scope: `@hunpeolabs/gig`
- GitHub repository for the trusted publisher: `phamhungptithcm/gig`

Bootstrap note:

- npm trusted publisher settings live on the package, so `@hunpeolabs/gig` usually needs to exist first
- for the first publish, set repository secret `NPM_PUBLISH_TOKEN` so the release workflow can bootstrap the package automatically from GitHub Actions
- `NPM_PUBLISH_TOKEN` must come from an npm identity that can publish to `@hunpeolabs/gig`
- if the token path is used, prefer an npm automation token or a granular token with bypass 2FA enabled
- after the package exists, configure trusted publishing on npm, verify one GitHub Actions publish, then remove the token fallback if you no longer want it
- if you leave `NPM_PUBLISH_TOKEN` configured, the workflow can still use it as an emergency fallback when trusted publishing is not enabled

Recovery note:

- if a GitHub Release already exists but npm publication failed, do not push a no-op commit just to retry
- open the `Release` workflow in GitHub Actions and run it manually with `release_tag=vYYYY.MM.DD`
- the workflow will re-run verification from the current `main` workflow, skip GitHub Release creation if it already exists, and only retry the missing npm publish work for that release tag

## Easiest Bootstrap Commands

If you want the shortest path with the least memory burden, use these commands from the repo root:

```bash
./scripts/npm-release.sh prepare
./scripts/npm-release.sh bootstrap
gh variable set NPM_TRUSTED_PUBLISHING --repo phamhungptithcm/gig --body true
```

If you prefer `make`:

```bash
make npm-release-prepare
make npm-release-bootstrap
```

What each command does:

- `prepare`: open a searchable release-tag selector, stage the npm package, and run `npm pack --dry-run`
- `bootstrap`: open the same selector, publish the first package version locally, configure npm trusted publishing, verify npm state, and print the exact next commands
- `verify`: show the published version and trusted publisher configuration

If `fzf` is installed, the selector behaves like a searchable dropdown with type-to-filter.
If `fzf` is not installed, the script falls back to a search prompt and numbered list.

Important setup details:

- the workflow filename is case-sensitive and must match `release.yml` exactly
- trusted publishing works on GitHub-hosted runners, which this workflow uses
- automatic provenance on npm requires a public package published from a public repository
- the first publish can now be done through GitHub Actions by setting repository secret `NPM_PUBLISH_TOKEN`
- `package.json` repository metadata must continue to match `https://github.com/phamhungptithcm/gig.git`
- after the first successful trusted publish, switch the npm package to `Require two-factor authentication and disallow tokens`

After trusted publishing works, revoke old npm automation tokens and disallow token-based publishing for the package.

## Release Note Standard

The release workflow uses `./scripts/release-notes.sh [previous-tag] <current-tag>`.

The generated release notes should always follow the same shape:

- `Summary` with the release focus and change counts
- `Highlights` grouped into CLI and workflows, source-control-native access, packaging and release automation, docs, and maintenance
- `Upgrade Notes` when a breaking commit is detected
- `Full Changelog` with a compare link when possible

To keep the output readable, commit subjects should follow Conventional Commits whenever possible:

- use scopes such as `cli`, `sourcecontrol`, `release`, or `docs`
- keep the subject line short and user-visible
- reserve `docs`, `chore`, `build`, `ci`, and `test` for non-product work so the generator can place them correctly
- use `!` only for real breaking changes because it is promoted into `Upgrade Notes`

## Branch Protection Follow-Up

After the workflow changes are merged and CI has produced at least one successful run on `main` and `staging`, sync the required checks:

```bash
./scripts/sync-required-checks.sh phamhungptithcm/gig
```

That script only updates branch protection when these checks already exist and are passing on the target branch:

- `go`
- `packaging`
- `docs`

This avoids locking merges by requiring a check name before GitHub has seen it on the branch.

## Docs Deployment

When docs inputs change on `main`:

- MkDocs builds the site
- GitHub Pages publishes the updated docs

## Easiest First-Time User Flow

1. install `gig`
2. run `gig --help`
3. run `gig version`
4. run `gig inspect ABC-123 --path .`
5. run `gig verify ABC-123 --from test --to main --path .`

## Suggested Weekly Team Flow

1. developers branch from `staging`
2. developers open pull requests back into `staging`
3. QA and review happen on `staging`
4. at release time, open a promotion pull request from `staging` to `main`
5. merging to `main` publishes the next release and updates docs if needed

## Where `gig` Fits In That Flow

Use `gig` before the promotion step when you need to answer:

- what changed for this ticket?
- is `test` behind `dev`?
- is `main` still missing a follow-up fix?
- does this ticket need manual review because of DB or config changes?
