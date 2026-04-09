# Config Spec

## Goal

The config file should let teams set workspace and SCM defaults without changing code.

Config loading is planned. The current MVP uses built-in defaults only.

## Example Config

```yaml
workspaceRoots:
  - /Users/me/workspace
scanRecursive: true
defaultBranches:
  dev: develop
  test: test
  prod: main
commitPattern: '[A-Z]+-\\d+'
supportedScm:
  - git
  - svn
excludedPaths:
  - node_modules
  - dist
  - target
```

## Field Meaning

### `workspaceRoots`

List of common root folders to scan.

### `scanRecursive`

If `true`, the scanner walks child folders recursively.

### `defaultBranches`

Maps logical environment names to real branch names.

Example:

- `dev` -> `develop`
- `test` -> `test`
- `prod` -> `main`

### `commitPattern`

Regex used to find ticket IDs in commit messages.

### `supportedScm`

List of SCM types the tool should try to detect.

### `excludedPaths`

Folders that should never be scanned.

Useful for:

- build output folders
- dependency caches
- large generated directories

## Planned Precedence

When config loading is added, values should be resolved in this order:

1. command flags
2. local project config
3. user config
4. built-in defaults

## Design Notes

- config should stay small and easy to read
- config errors should be clear
- unknown fields should warn or fail based on chosen strictness mode
