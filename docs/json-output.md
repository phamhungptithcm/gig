# JSON Output

Use JSON when another tool needs to consume `gig` evidence.

```bash
gig verify ABC-123 --repo github:owner/name --format json
gig packet ABC-123 --repo github:owner/name --format json
gig plan ABC-123 --repo github:owner/name --format json
```

## When To Use JSON

- CI release checks
- dashboard ingestion
- release packet automation
- regression snapshots
- scripts that need verdicts and missing evidence

## Practical Contract

Treat JSON as additive:

- new fields may appear
- existing fields should keep their meaning
- consumers should ignore unknown fields
- prefer semantic fields such as verdict, repository root, missing commits, risk signals, and provider evidence

Some nested legacy structures currently reflect Go field names.
If you need strict contracts, pin the `gig` version and test against representative JSON output.

## Common Commands

```bash
gig verify ABC-123 --repo github:owner/name --format json
gig verify --ticket-file tickets.txt --repo github:owner/name --format json
gig packet ABC-123 --repo github:owner/name --format json
gig snapshot create --ticket ABC-123 --path . --from staging --to main --format json --output snapshot.json
```

## Consumer Advice

- Check the command name and verdict first.
- Do not parse human output.
- Ignore unknown fields.
- Do not assume every provider has the same evidence depth.
- Use provider capability docs to decide which evidence fields can be expected.
