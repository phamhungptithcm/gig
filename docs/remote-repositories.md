# Remote Repositories

Remote mode is the preferred first path. From inside a supported Git checkout, `gig` infers the remote target from `origin`, so the common commands do not need `--repo`.

```bash
gig ABC-123
gig verify ABC-123
gig packet ABC-123
```

Use explicit targets when you are outside the checkout, running CI, or auditing a different repo.

## Target Syntax

```text
github:owner/name
gitlab:group/project
bitbucket:workspace/repo
azure-devops:org/project/repo
svn:https://svn.example.com/repos/app/branches/staging/ProductName
```

Use targets with `--repo`:

```bash
gig ABC-123 --repo github:owner/name
gig verify ABC-123 --repo gitlab:group/project
gig packet ABC-123 --repo azure-devops:org/project/repo
```

## Login

```bash
gig login
```

`gig login` asks which provider to use when it cannot infer safely.

Provider-specific login:

```bash
gig login github
gig login gitlab
gig login bitbucket
gig login azure-devops
gig login svn
```

## Provider Coverage

| Provider | Coverage |
| --- | --- |
| GitHub | Deep release evidence: PRs, deployments, checks, linked issues, releases |
| GitLab | Deep release evidence: merge requests, deployments, checks, linked issues, releases |
| Bitbucket | Basic release evidence: pull requests, deployments, branching model |
| Azure DevOps | Deep release evidence: pull requests, deployments, checks, linked work items |
| Remote SVN | Audit topology only: branch and trunk discovery |

## Branch Inference

For remote providers, `gig` tries protected/default branch metadata first.
If the inferred path is high-confidence, commands stay short:

```bash
gig verify ABC-123
```

If the path is ambiguous, `gig` asks for explicit promotion intent in an interactive terminal. In scripts, keep the command explicit:

```bash
gig verify ABC-123 --repo github:owner/name --from staging --to main
```

`--envs` is optional and only needed when the environment order needs a manual override.

Save repeated topology in a project:

```bash
gig project add payments --repo github:owner/name --from staging --to main --use
```

## SVN Notes

Remote SVN targets must not include username or password in the URL.

Use:

```bash
gig login svn
```

or environment credentials:

```bash
export GIG_SVN_USERNAME=demo
export GIG_SVN_PASSWORD=secret
```

Then run:

```bash
gig ABC-123 --repo svn:https://svn.example.com/repos/app/branches/staging/ProductName
```

## When To Use Local Fallback Instead

Use [Local Fallback](local-fallback.md) when:

- provider login is unavailable
- the provider path cannot expose the needed SVN/Mendix layout
- you need local-only files or conflict state
- you are debugging repository discovery
