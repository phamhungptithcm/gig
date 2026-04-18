# Error Handling And Logging

## Goal

Errors should be easy for people to understand and easy for automation to detect.

## Exit Codes

- `0`: success
- `1`: runtime failure
- `2`: command usage error
- `3`: reserved for future partial-success execution flows

## Error Categories

### Usage Errors

Examples:

- missing required flags
- missing ticket ID
- invalid command format

These should:

- print a short message
- show command usage
- return exit code `2`

### Environment Errors

Examples:

- Git is not installed
- path does not exist
- repository cannot be read

These should:

- explain the problem clearly
- return exit code `1`

### Data Errors

Examples:

- invalid ticket format
- branch not found
- config file is malformed

These should:

- say what value is bad
- suggest the expected format when possible

### Execution Errors

Examples for future promote flows:

- cherry-pick failed
- merge conflict detected
- dependency missing

These should:

- show which repo failed
- show which commit or ticket caused the problem
- stop unsafe execution unless a later recovery mode is designed

## Human-Readable Output

Default CLI output should be short and clear.

Rules:

- keep normal output readable
- keep errors direct
- do not print stack traces by default
- point to the repo, branch, ticket, or commit that failed

## Structured Logs

`gig` can now write structured diagnostics to a JSONL file when you set:

```bash
export GIG_DIAGNOSTICS_FILE=/path/to/gig-diagnostics.jsonl
```

This is opt-in.
Normal terminal output stays concise unless you enable the diagnostics file.

Current fields:

- timestamp
- level
- command
- workspace
- repo
- scm
- ticket
- fromBranch
- toBranch
- operation
- message
- error

## Logging Levels

Current levels:

- `info`
- `warning`
- `error`

Typical events today:

- command start and finish
- provider access checks
- provider login checks
- protected-branch topology evaluation
- promotion-path resolution success or failure

## Principle

Good errors help a human fix the issue fast.
Good logs help a machine or support engineer trace the issue later.
