# Demo Guide

This page shows the fastest way to produce a repeatable terminal demo for `gig`.

## Why These Scripts Exist

The repo now includes a deterministic terminal walkthrough so you can:

- sanity-check the guided front door
- demo workarea setup without hitting live provider APIs
- show the DeerFlow onboarding flow with `assist doctor` and `assist setup`
- record a shareable terminal cast for README updates, release posts, or portfolio use
- reuse static terminal mockups for `inspect`, `verify`, and `assist audit` in landing pages or docs

## Quick Demo

Run the scripted terminal walkthrough:

```bash
./scripts/demo/frontdoor.sh
```

What it shows:

- `gig` with no args opening the guided front door
- saving a remote-style workarea
- `gig` showing the current project shortcuts
- `gig assist doctor` reporting DeerFlow readiness
- `gig assist setup` bootstrapping the bundled sidecar config
- `gig assist doctor` showing the next readiness state after bootstrap

The script is deterministic and does not need live provider login.
It uses a temporary workarea file and a temporary DeerFlow fixture so it does not dirty the repo.

## Record An Asciinema Cast

If you have [`asciinema`](https://asciinema.org/) installed:

```bash
./scripts/demo/record-frontdoor.sh
```

That records the same walkthrough to:

```text
docs/assets/gig-demo.cast
```

If `asciinema` is not installed, the script falls back to the deterministic local cast builder and still writes the same `.cast` file.

You can also pass a custom output path:

```bash
./scripts/demo/record-frontdoor.sh /tmp/gig-demo.cast
```

## Recommended Use

Use the scripted demo when you want:

- a reliable terminal capture for README or docs updates
- a short social clip showing the product direction
- a portfolio artifact that highlights the front door and DeerFlow integration

For deeper product demos, start from this script and layer on real provider login plus live `inspect`, `verify`, or `assist audit` flows after the scripted onboarding section.

## Static Demo Assets

The README also uses static SVG mockups for the main product story:

- `docs/assets/ticket-inspect-demo.svg`
- `docs/assets/ticket-verify-demo.svg`
- `docs/assets/ticket-assist-demo.svg`

Use those when you want screenshot-like assets that stay stable across README renders and docs builds.
