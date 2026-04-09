# Product Overview

## What `gig` Is

`gig` is a command-line tool for developers and release engineers.

It helps teams answer a simple but important question:

"Did we move every commit for this ticket to the next branch?"

## What Problem It Solves

Many teams work in more than one repository. One ticket may touch backend code, frontend code, database scripts, and low-code apps like Mendix at the same time.

When the same ticket is tested many times, more commits are added over time. Later, someone must move the right changes from one branch to another. This is often done by hand. Hand work is slow and easy to get wrong.

`gig` helps reduce that risk by finding ticket-related commits, comparing branches, and showing what is missing before promotion.

## Main Users

- developers who make and fix changes for the same ticket many times
- QA and release engineers who need a clear view before promotion
- teams working in multi-repo and mixed-SCM environments

## Value

- fewer missed commits during release work
- one place to check many repositories
- a safer promotion process
- a clear path to future automation

## What The First Versions Do

The early versions focus on safe read-only work:

- detect repositories in a workspace
- find commits by ticket ID
- compare source and target branches for missing changes

Promotion automation comes later, after discovery and comparison are stable and trusted.
