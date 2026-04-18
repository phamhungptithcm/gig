# Config Spec

This page explains the config file that `gig` can load today.

The goal is simple:

- map your real environment branches
- describe repos in human terms
- stop repeating the same flags on every command

## Supported File Names

`gig` will auto-detect these names:

- `gig.yaml`
- `gig.yml`
- `.gig.yaml`
- `.gig.yml`

It searches upward from the `--path` you pass to the command.

If you want to point at a specific file, use `--config`.

## What The Config Can Do Today

The current config supports:

- custom ticket pattern regex
- environment-to-branch mapping
- repository catalog entries with service name, owner, kind, and notes

This config is already used by:

- `gig env status`
- `gig verify`
- `gig plan`
- `gig manifest`
- `gig doctor`

## Example Config

```yaml
ticketPattern: '\b[A-Z][A-Z0-9]+-\d+\b'

environments:
  - name: dev
    branch: develop
  - name: test
    branch: release/test
  - name: prod
    branch: main

repositories:
  - path: services/accounts-api
    service: Accounts API
    owner: Backend Team
    kind: app
    notes:
      - Verify login and billing summary in QA.

  - path: db/shared-schema
    service: Shared Schema
    owner: Platform Team
    kind: db
    notes:
      - Review migration order and rollback notes before release.
```

There is also a sample file in the repo:

- [gig.example.yaml](https://github.com/phamhungptithcm/gig/blob/main/gig.example.yaml)

## Field Meaning

### `ticketPattern`

Regex used to validate and find ticket IDs in commit messages.

Use this only if your team does not follow the default pattern.

Default:

```text
\b[A-Z][A-Z0-9]+-\d+\b
```

### `environments`

List of logical environments and the real branch name each one maps to.

Example:

- `dev` -> `develop`
- `test` -> `release/test`
- `prod` -> `main`

Commands such as `env status`, `verify`, `plan`, and `manifest` use this mapping when `--envs` is not passed.

### `repositories`

List of repo catalog entries.

Each entry can include:

- `path`
  repo path relative to the workspace root, or an absolute path
- `name`
  optional fallback if you prefer matching by repo name
- `service`
  human-friendly service or app name
- `owner`
  team or person responsible for that repo
- `kind`
  repo type such as `app`, `db`, `mendix`, or `infra`
- `notes`
  simple release notes, reminders, or QA hints for that repo

## Matching Rules

`gig` tries to match repository entries in this order:

1. `path`
2. `name`

Using `path` is the safest option for real teams because repo names can repeat across larger workspaces.

Today, `path` should point to a detected repository root.

If one Git repo contains many services or apps in subfolders, `gig` still treats that as one repository for scan and release analysis.

## Why The Repo Catalog Matters

The repo catalog is what makes the output feel useful to humans, not just correct to a machine.

It powers:

- service names in release packets
- owner callouts for release coordination
- repo kind checks in `gig doctor`
- release notes and reminders that are specific to each repo

## Precedence

Values are resolved in this order:

1. command flags
2. explicit file passed with `--config`
3. auto-detected config file from the workspace path
4. built-in defaults

Example:

- if you pass `--envs`, that wins over the config file
- if you do not pass `--envs`, `gig` uses the config file
- if no config file exists, `gig` falls back to built-in defaults

## Built-In Defaults

If there is no config file, `gig` still works with these defaults:

- ticket pattern: `\b[A-Z][A-Z0-9]+-\d+\b`
- environments:
  - `dev` -> `dev`
  - `test` -> `test`
  - `prod` -> `main`

## Best Practices

- keep the file short and practical
- use real team names in `owner`
- use repo `path` whenever possible
- add notes only when they help release, QA, or client review
- use `gig doctor` after editing the file
