# Demo Guide

This page shows the fastest way to produce a clean, repeatable demo for `gig`.

## Why The Demo Matters

`gig` lands best when people can see the product promise quickly:

- guided CLI front door
- command-palette front door that accepts ticket-like queries
- remote-first workflow
- clear ticket audit
- release verdict instead of raw Git output

A strong demo helps docs, portfolio pages, hiring conversations, and GitHub stars for the same reason: it makes the value obvious fast.

## Run The Deterministic Walkthrough

```bash
./scripts/demo/frontdoor.sh
```

What it shows:

- `gig` opening the guided front door
- the branded `googling in git` terminal layout
- saving a remote-style workarea
- re-opening `gig` with current-project shortcuts
- `gig assist doctor` readiness output
- `gig assist setup` bootstrap output

The script is deterministic and does not require live provider access.

## Record The Terminal Cast

```bash
./scripts/demo/record-frontdoor.sh
```

Default output:

```text
docs/assets/gig-demo.cast
```

If `asciinema` is not installed, the script falls back to the deterministic local cast builder and still writes the same `.cast` file.

You can also choose a custom output path:

```bash
./scripts/demo/record-frontdoor.sh /tmp/gig-demo.cast
```

## Optional: Render Shareable Assets

If you want a README- or social-friendly showcase asset:

```bash
./scripts/demo/render-share-assets.sh
```

Default outputs:

```text
docs/assets/gig-showcase.gif
docs/assets/gig-showcase.mp4
```

This script renders the repo's SVG demo states into PNG frames with `qlmanage`, then builds a short MP4 and GIF with `ffmpeg`.

## Recommended Demo Sequence

For a 45 to 60 second product demo, keep the flow tight:

1. show `gig` with no arguments
2. show the command palette idea: `ABC-123` or `repo github:owner/name ABC-123`
3. show one remote-style workarea being saved
4. show `gig inspect` or the guided shortcut path
5. close with `gig verify`, `gig manifest`, or `gig assist audit`

That sequence sells the strongest product story without drowning the viewer in command surface area.

## Publishing Checklist

Use the demo in these places:

- README hero link or thumbnail
- GitHub release notes
- LinkedIn or X post with one clear sentence about the release-audit problem
- portfolio page or job application follow-up

When writing the caption, lead with:

- `gig = googling in git`
- multi-repo ticket reconciliation
- remote-first release audit
- deterministic evidence with optional AI explanation

## Supporting Assets

The repo already includes stable visual assets for documentation and project showcases:

- `docs/assets/gig-showcase.gif`
- `docs/assets/gig-showcase.mp4`
- `docs/assets/gig-demo-thumbnail.svg`
- `docs/assets/ticket-inspect-demo.svg`
- `docs/assets/ticket-verify-demo.svg`
- `docs/assets/ticket-assist-demo.svg`

Use those when you need a consistent visual story without rerecording screenshots.
