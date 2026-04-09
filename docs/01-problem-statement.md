# Problem Statement

## Simple Summary

Teams often work on one business ticket in many places at the same time.

For example, one ticket may need:

- a Java service change
- a Python job change
- a SQL script
- an Angular UI update
- a Mendix workflow update

Those changes may live in different repositories. Some repositories may use Git. Some older ones may still use SVN.

This makes promotion hard.

## Why The Problem Happens

### Multi-repo work

A single ticket is often split across many codebases. People must remember every repository that was touched. This is hard, especially when many teams are working in parallel.

### Mixed Git and SVN environments

Not every enterprise team uses one SCM system. Some projects are in Git. Some are in SVN. Some companies have both at the same time. A useful tool must be ready for this reality.

### Many commits for the same ticket

A ticket rarely has just one clean commit.

Real work often looks like this:

1. developer makes the first fix
2. QA tests it
3. QA finds a problem
4. developer adds more commits
5. client reviews it
6. client asks for changes
7. developer adds more commits again

At the end, one ticket may have many commits across many repositories.

### Many verification loops

The same ticket may fail verification several times in:

- dev
- QA or test
- client test or UAT

Each failed round can create more follow-up commits. If teams do not track those commits well, later promotion becomes risky.

### Manual cherry-pick is easy to get wrong

Before moving to production, teams often cherry-pick commits by hand.

This creates common problems:

- one repo is forgotten
- one follow-up commit is missed
- the wrong commit is picked
- dependent changes are moved in the wrong order

The result can be broken releases, failed deployments, or hard-to-debug production issues.

### Dependencies between tickets

Some commits are not independent.

Example:

- `ABC-123` changes a service
- `XYZ-456` adds a database change required by that service

If only `ABC-123` is promoted, the release may break. The tool should be designed to understand this kind of dependency later.

## Why Existing Manual Process Is Not Enough

Humans can handle small cases. They do not handle repeated multi-repo release work well at scale.

People forget details.
Commit history becomes noisy.
Branch history becomes different across repos.
Audit trails become weak.

Teams need a tool that can:

- scan many repositories
- find all commits for a ticket
- compare branches
- show what is missing
- support future safe promotion automation

## Product Need

`gig` exists because release teams need a simple and reliable way to answer:

- where did this ticket change code?
- which commits belong to this ticket?
- what is still missing in the target branch?
- is it safe to promote?
