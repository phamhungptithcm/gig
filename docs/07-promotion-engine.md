# Promotion Engine

## Status

This engine is planned, not fully implemented in the current MVP.

## Goal

Move from "show me what is missing" to "help me move the right changes safely."

## Promotion Flow

### 1. Collect Commits

Find all commits for the requested ticket across all detected repositories.

### 2. Compare Branches

Compare the source branch and target branch for each repository.

Goal:
find which ticket commits are already present and which are still missing.

### 3. Build Plan

Create a clear promotion plan.

The plan should show:

- repository name
- commits to apply
- commit order
- dependencies
- warnings

### 4. Detect Missing Items

Before execution, the engine should detect:

- missing commits
- missing dependent tickets
- branch mismatches
- repositories that could not be read

### 5. Dry-Run

In dry-run mode, the tool should simulate the promotion flow without changing anything.

Dry-run should answer:

- what would be picked
- what might fail
- where conflicts may happen

### 6. Execute

Only after explicit confirmation, the tool may execute the plan.

Safety rules:

- no silent write actions
- clear confirmation before execution
- stop and report on critical failure

### 7. Report

After execution, the tool should print a simple report.

The report should show:

- what was applied
- what was skipped
- what failed
- what needs manual follow-up

## Design Principles

- safe by default
- dry-run first
- clear plan before execution
- easy to audit
- no hidden destructive behavior
