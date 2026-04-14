# gig

![gig showcase](docs/assets/gig-showcase.gif)

`gig` is a remote-first release audit CLI for one critical question:

`Did we miss any change for this ticket?`

That question gets expensive when:

- one ticket touches backend, frontend, database, scripts, or low-code assets
- QA or client review adds late follow-up fixes
- release teams have to reopen multiple repos just to decide whether the next move is safe

`gig` turns that into one deterministic workflow:

- `inspect` collects the full ticket story across repositories and branches
- `verify` returns a `safe`, `warning`, or `blocked` verdict
- `manifest generate` exports a release packet in Markdown or JSON

Why teams adopt it:

- remote-first: works directly against GitHub, GitLab, Bitbucket, Azure DevOps, and remote SVN
- zero-config-first: start with `--repo`, add `gig.yaml` only when inference needs help
- auditable by default: repository evidence first, optional AI explanation second

## Install

```bash
npm install -g @phamhungptithcm/gig
gig version
```

Direct installer fallback:

```bash
curl -fsSL https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.sh | sh
gig version
```

## Fastest Path

```bash
gig
gig login github
gig inspect ABC-123 --repo github:owner/name
gig verify --ticket ABC-123 --repo github:owner/name
gig manifest generate --ticket ABC-123 --repo github:owner/name
```

Remote target forms:

- `github:owner/name`
- `gitlab:group/project`
- `bitbucket:workspace/repo`
- `azure-devops:org/project/repo`
- `svn:https://svn.example.com/repos/app/branches/staging/ProductName`

Local fallback is still available:

```bash
gig inspect ABC-123 --path .
gig verify --ticket ABC-123 --path .
```

## Demo And Docs

- [Quick Start](docs/19-quickstart.md)
- [Demo Guide](docs/25-demo-guide.md)
- [Portfolio Guide](docs/26-portfolio-guide.md)
- [Docs site](https://phamhungptithcm.github.io/gig/)

## Scope

`gig` is strongest at ticket reconciliation, release verification, and release packet generation.
It does not try to replace code review, CI/CD, or human release approval.

The AI layer is optional.
`gig` remains the source of truth.
