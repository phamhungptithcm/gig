# Mendix Considerations

## Why Mendix Needs Special Care

Mendix projects can be higher risk during promotion than plain text code.

One important reason is that some project files can be harder to merge and review with normal source control workflows.

## High-Risk Areas

### `.mpr` File Risk

The `.mpr` file can be large, complex, and hard to merge safely.

This means:

- merge conflicts may be harder to understand
- normal text diff tools may not be enough
- manual review may still be required

### Conflict Resolution Difficulty

Mendix conflicts are often harder to auto-resolve than normal code conflicts.

A tool should be careful here.

## Recommended MVP Behavior

For early versions, `gig` should warn, not auto-resolve.

Good warning examples:

- Mendix repo detected
- high-risk file types present
- manual review recommended before promote

## Current SVN Support

`gig` can now read Mendix projects that live in SVN working copies for these commands:

- `gig scan`
- `gig find`
- `gig inspect`

That means ticket search can find SVN revisions and `inspect` can still flag risky files such as `.mpr`.

It still does not compare SVN branches for promotion commands such as:

- `gig diff`
- `gig env status`
- `gig verify`
- `gig plan`
- `gig manifest generate`

## Safe Direction

In early promote phases, the tool should:

- detect Mendix repositories
- flag high-risk files
- warn before execution
- allow human approval

It should not:

- promise safe automatic conflict resolution
- hide Mendix-specific risks

## Future Possibilities

- stronger Mendix file detection
- risk scoring in promotion plans
- custom warnings for known conflict-prone modules
