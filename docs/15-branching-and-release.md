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

- the next patch tag is calculated
- release archives are built for macOS, Linux, and Windows
- release notes are generated from the commits since the previous tag
- GitHub Release assets are published
- package-manager metadata for `gig-cli` is regenerated from this same repository

## Docs Deployment

When docs inputs change on `main`:

- MkDocs builds the site
- GitHub Pages publishes the updated docs

## Easiest First-Time User Flow

1. install `gig`
2. run `gig --help`
3. run `gig version`
4. run `gig inspect ABC-123 --path .`
5. run `gig verify --ticket ABC-123 --from test --to main --path . --envs dev=dev,test=test,prod=main`

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
