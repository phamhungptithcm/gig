package cli

import "fmt"

func (a *App) printRootUsage() {
	printHelpHeading(a.stderr, "gig", "Remote-first release audit CLI")
	printHelpUsage(a.stderr, "gig [ticket-id | command] [flags]")
	printHelpBullets(a.stderr, "First-time users",
		"Run `gig` in a real terminal to open the guided picker.",
		"The guided prompt stays open after each command; type `exit` or `quit` when done.",
		"From inside a Git checkout, gig can infer the remote repo from origin.",
		"Inside the prompt, `repo payments`, `gh owner/name`, or a pasted repo URL resolves the remote target.",
		"When a short command is missing a ticket or promotion path, gig asks for it interactively.",
		"Learn `inspect`, `verify`, and `packet` first.",
	)
	printHelpCommands(a.stderr, "Start here",
		"gig",
		"gig login",
		"gig ABC-123",
		"gig verify ABC-123",
		"gig packet ABC-123",
		"gig resume",
		"gig ask \"what is still blocked?\"",
	)
	printHelpCommands(a.stderr, "Inside prompt",
		"repo payments",
		"gh owner/name",
		"ABC-123",
		"verify ABC-123",
		"packet ABC-123",
	)
	printHelpCommands(a.stderr, "Scriptable form",
		"gig ABC-123",
		"gig verify ABC-123",
		"gig packet ABC-123",
	)
	printHelpCommands(a.stderr, "Fallback",
		"gig ABC-123 --path .",
	)
	printHelpRows(a.stderr, "Provider coverage", providerCoverageHelpRows()...)
	printHelpRows(a.stderr, "Commands",
		helpRow{Label: "project", Value: "Remember a project so later commands stay short"},
		helpRow{Label: "login", Value: "Authenticate to a live provider such as GitHub"},
		helpRow{Label: "inspect", Value: "Show the full ticket picture across repositories"},
		helpRow{Label: "verify", Value: "Return safe, warning, or blocked for the next move"},
		helpRow{Label: "packet", Value: "Generate a release packet for QA and release review"},
		helpRow{Label: "plan", Value: "Build a read-only promotion plan when you need more detail"},
		helpRow{Label: "repos", Value: "Find repositories under a local path"},
		helpRow{Label: "commits", Value: "List raw commits for one ticket"},
		helpRow{Label: "where", Value: "Show where a ticket is present or behind"},
		helpRow{Label: "diff", Value: "Compare one branch to another for a ticket"},
		helpRow{Label: "snapshot", Value: "Save a repeatable ticket baseline for audit and re-check"},
		helpRow{Label: "assist", Value: "Add an optional AI briefing on top of gig evidence"},
		helpRow{Label: "ask", Value: "Continue the last AI brief with a follow-up question"},
		helpRow{Label: "resume", Value: "Show the last AI brief for the current project or repo"},
		helpRow{Label: "doctor", Value: "Check inferred topology, overrides, and repo health"},
		helpRow{Label: "setup", Value: "Check or install missing local tools with confirmation"},
		helpRow{Label: "resolve", Value: "Inspect or resolve active Git merge conflicts"},
		helpRow{Label: "update", Value: "Install the latest release or a specific version"},
		helpRow{Label: "version", Value: "Show the installed version"},
	)
	a.printCurrentWorkareaHint()
	printHelpCommands(a.stderr, "More help", "gig <command> --help")
}

func (a *App) printLoginUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig login [provider]")
	fmt.Fprintln(a.stderr, "If provider is omitted, gig asks you to choose one.")
	fmt.Fprintln(a.stderr, "Providers: github, gitlab, bitbucket, azure-devops, svn")
	fmt.Fprintln(a.stderr)
	printHelpRows(a.stderr, "Provider coverage", providerCoverageHelpRows()...)
}

func (a *App) printScanUsage() {
	printHelpHeading(a.stderr, "gig repos", "Find local repositories under a path.")
	printHelpUsage(a.stderr, "gig repos --path .")
	printHelpRows(a.stderr, "Alias", helpRow{Label: "scan", Value: "gig scan --path ."})
}

func (a *App) printFindUsage() {
	printHelpHeading(a.stderr, "gig commits", "List raw commits for one ticket.")
	printHelpUsage(a.stderr, "gig commits <ticket-id> [--project name] [--path . | --repo <provider-target>]")
	printHelpCommands(a.stderr, "Start here",
		"gig ABC-123",
		"gig commits ABC-123",
		"gig commits ABC-123 --project payments",
	)
	printHelpRows(a.stderr, "Tip", helpRow{Label: "inspect", Value: "Use gig ABC-123 first when you want the full ticket story"})
	printHelpRows(a.stderr, "Alias", helpRow{Label: "find", Value: "gig find <ticket-id> ..."})
}

func (a *App) printDiffUsage() {
	printHelpHeading(a.stderr, "gig diff", "Compare one branch to another for a ticket.")
	printHelpUsage(a.stderr, "gig diff --ticket <ticket-id> --from <branch> --to <branch> --path .")
}

func (a *App) printInspectUsage() {
	printHelpHeading(a.stderr, "gig inspect", "Show the full ticket story across repositories.")
	printHelpUsage(a.stderr, "gig inspect <ticket-id> [--project name] [--path . | --repo <provider-target>]")
	printHelpCommands(a.stderr, "Start here",
		"gig",
		"gig inspect ABC-123",
		"gig inspect ABC-123 --project payments",
		"gig inspect ABC-123 --repo github:owner/name",
	)
	printHelpRows(a.stderr, "Common flags",
		helpRow{Label: "--repo", Value: "Inspect a live remote repository without cloning first"},
		helpRow{Label: "--project", Value: "Reuse a remembered project and its inferred defaults"},
		helpRow{Label: "--path", Value: "Use local workspace mode when remote access is not enough"},
		helpRow{Label: "--config", Value: "Optional override file when inference needs help"},
	)
	printHelpCommands(a.stderr, "Next commands",
		"gig verify ABC-123",
		"gig plan ABC-123",
		"gig packet ABC-123",
	)
}

func (a *App) printEnvUsage() {
	printHelpHeading(a.stderr, "gig where", "Show where a ticket is present or behind.")
	printHelpUsage(a.stderr, "gig where <ticket-id> [--project name] [--path . | --repo <provider-target>] [--envs dev=dev,test=test,prod=main]")
	printHelpCommands(a.stderr, "Start here",
		"gig where ABC-123",
		"gig where ABC-123 --project payments",
	)
	printHelpRows(a.stderr, "Tip", helpRow{Label: "verify", Value: "Use gig verify ABC-123 when you need a release verdict"})
	printHelpRows(a.stderr, "Alias", helpRow{Label: "env status", Value: "gig env status <ticket-id> ..."})
}

func (a *App) printEnvStatusUsage() {
	a.printEnvUsage()
}

func (a *App) printPlanUsage() {
	printHelpHeading(a.stderr, "gig plan", "Build a read-only promotion plan for one ticket or a saved release.")
	printHelpUsage(a.stderr,
		"gig plan ABC-123 [--repo github:owner/name | --project payments | --path .]",
		"gig plan --ticket-file tickets.txt [--repo github:owner/name | --project payments | --path .]",
		"gig plan --release rel-2026-04-09 --path . [--format human|json]",
	)
	printHelpCommands(a.stderr, "Start here",
		"gig plan ABC-123",
		"gig plan ABC-123 --repo github:owner/name",
		"gig plan --release rel-2026-04-09 --path .",
	)
	printHelpRows(a.stderr, "Common flags",
		helpRow{Label: "--repo", Value: "Plan directly against a live remote repository"},
		helpRow{Label: "--project", Value: "Reuse remembered repo scope and branch defaults"},
		helpRow{Label: "--from/--to", Value: "Only add these when gig cannot infer the promotion path"},
		helpRow{Label: "--envs", Value: "Override environment mapping only when needed"},
		helpRow{Label: "--json", Value: "Print JSON for automation"},
		helpRow{Label: "--format", Value: "Use json for automation or human for terminal review"},
	)
	printHelpCommands(a.stderr, "Next commands",
		"gig verify ABC-123",
		"gig packet ABC-123",
	)
}

func (a *App) printVerifyUsage() {
	printHelpHeading(a.stderr, "gig verify", "Turn ticket evidence into a safe, warning, or blocked release verdict.")
	printHelpUsage(a.stderr,
		"gig verify ABC-123 [--repo github:owner/name | --project payments | --path .]",
		"gig verify --ticket-file tickets.txt [--repo github:owner/name | --project payments | --path .]",
		"gig verify --release rel-2026-04-09 --path .",
		"gig verify ABC-123 --out verify.xlsx",
	)
	printHelpCommands(a.stderr, "Start here",
		"gig verify ABC-123",
		"gig verify ABC-123 --repo github:owner/name",
		"gig verify --release rel-2026-04-09 --path .",
	)
	printHelpRows(a.stderr, "Common flags",
		helpRow{Label: "--repo", Value: "Verify directly against a live remote repository"},
		helpRow{Label: "--project", Value: "Reuse remembered repo scope and branch defaults"},
		helpRow{Label: "--from/--to", Value: "Only add these when gig cannot infer the promotion path"},
		helpRow{Label: "--json", Value: "Print JSON for automation"},
		helpRow{Label: "--format", Value: "Use json for automation, xlsx for sharing, or csv for import"},
		helpRow{Label: "--out", Value: "Write xlsx, csv, or json export to a file"},
	)
	printHelpCommands(a.stderr, "Next commands",
		"gig plan ABC-123",
		"gig packet ABC-123",
	)
}

func (a *App) printManifestUsage() {
	printHelpHeading(a.stderr, "gig packet", "Generate a release packet for QA, client, and release review.")
	printHelpUsage(a.stderr,
		"gig packet ABC-123 [--repo github:owner/name | --project payments | --path .]",
		"gig packet --ticket-file tickets.txt [--repo github:owner/name | --project payments | --path .]",
		"gig packet --release rel-2026-04-09 --path .",
		"gig packet ABC-123 --out release-packet.xlsx",
	)
	printHelpCommands(a.stderr, "Start here",
		"gig packet ABC-123",
		"gig packet ABC-123 --repo github:owner/name",
		"gig packet --release rel-2026-04-09 --path .",
	)
	printHelpRows(a.stderr, "Common flags",
		helpRow{Label: "--repo", Value: "Generate a packet directly from a live remote repository"},
		helpRow{Label: "--project", Value: "Reuse remembered repo scope and branch defaults"},
		helpRow{Label: "--json", Value: "Print JSON for tooling"},
		helpRow{Label: "--format", Value: "Use markdown, json, xlsx, or csv"},
		helpRow{Label: "--out", Value: "Write xlsx/json to a file or csv to a directory"},
		helpRow{Label: "--from/--to", Value: "Only add these when gig cannot infer the promotion path"},
		helpRow{Label: "Alias", Value: "`gig manifest ...` and `gig manifest generate ...` still work for existing scripts"},
	)
	printHelpCommands(a.stderr, "Next commands",
		"gig explain ABC-123 --audience release-manager",
		"gig assist release --release rel-2026-04-09 --path .",
	)
}

func (a *App) printManifestGenerateUsage() {
	a.printManifestUsage()
}

func (a *App) printSnapshotUsage() {
	printHelpHeading(a.stderr, "gig snapshot", "Save a repeatable ticket baseline for audit and re-check.")
	printHelpUsage(a.stderr, "gig snapshot create <ticket-id> [--project name] [--path . | --repo <target>] [--release <release-id>] [--output snapshot.json]")
	printHelpCommands(a.stderr, "Start here",
		"gig snapshot create ABC-123",
		"gig snapshot create ABC-123 --project payments",
		"gig snapshot create ABC-123 --release rel-2026-04-09",
	)
	printHelpRows(a.stderr, "Common flags",
		helpRow{Label: "--release", Value: "Group snapshots under a saved release ID"},
		helpRow{Label: "--from/--to", Value: "Optional promotion path when inference needs help"},
		helpRow{Label: "--output", Value: "Write the snapshot JSON file"},
		helpRow{Label: "--json", Value: "Print JSON for automation"},
	)
}

func (a *App) printSnapshotCreateUsage() {
	a.printSnapshotUsage()
}

func (a *App) printAssistUsage() {
	printHelpUsage(a.stderr,
		"gig assist doctor [--path .] [--url http://localhost:2026] [--format human|json]",
		"gig assist setup [--path .] [--format human|json]",
		"gig explain ABC-123 [--repo github:owner/name | --project payments | --path .]",
		"gig assist audit ABC-123 [--repo github:owner/name | --project payments | --path .]",
		"gig assist chat \"what is still blocked?\"",
		"gig assist release --release rel-2026-04-09 [--path . | --ticket-file tickets.txt --repo github:owner/name]",
		"gig assist resume",
		"gig assist resolve --path . [--ticket ABC-123]",
	)
	printHelpRows(a.stderr, "Common flags",
		helpRow{Label: "--audience", Value: "qa, client, or release-manager"},
		helpRow{Label: "--mode", Value: "flash, standard, pro, or ultra"},
		helpRow{Label: "--url", Value: "DeerFlow sidecar URL, for example http://localhost:2026"},
		helpRow{Label: "--format", Value: "human or json"},
	)
}

func (a *App) printAssistDoctorUsage() {
	printHelpUsage(a.stderr, "gig assist doctor [--path .] [--url http://localhost:2026] [--format human|json]")
	printHelpBullets(a.stderr, "Checks", "Whether the bundled DeerFlow sidecar is configured, startable, and reachable.")
}

func (a *App) printAssistSetupUsage() {
	printHelpUsage(a.stderr, "gig assist setup [--path .] [--format human|json]")
	printHelpBullets(a.stderr, "Setup",
		"Bootstraps the bundled deer-flow sidecar config and prints the next start command.",
		"Run gig assist doctor first when you want a readiness check without writing files.",
	)
}

func (a *App) printAssistAuditUsage() {
	printHelpUsage(a.stderr,
		"gig explain ABC-123 [--repo github:owner/name | --project payments | --path .]",
		"gig assist audit ABC-123 [--repo github:owner/name | --project payments | --path .]",
	)
	printHelpBullets(a.stderr, "Tips",
		"Use the current checkout or current project when no --repo or --path is set.",
		"Add --audience qa|client|release-manager.",
		"Add --json for automation.",
		"Add --url http://localhost:2026 when DeerFlow is not on the default local port.",
	)
}

func (a *App) printAssistReleaseUsage() {
	printHelpUsage(a.stderr,
		"gig assist release --release rel-2026-04-09 --path .",
		"gig assist release --release rel-2026-04-09 --ticket-file tickets.txt --repo github:owner/name",
	)
	printHelpBullets(a.stderr, "Tips",
		"Use --path when the release already has saved snapshots.",
		"Use --ticket or --ticket-file with --repo for a live remote bundle.",
	)
}

func (a *App) printAssistChatUsage() {
	printHelpUsage(a.stderr, "gig assist chat [--message \"what changed?\"] [--mode flash|standard|pro|ultra] [--audience qa|client|release-manager] [--url http://localhost:2026] [--format human|json]")
	printHelpBullets(a.stderr, "Tips",
		"Run gig assist audit, release, or resolve first so gig has a saved deterministic session to continue.",
		"Use gig ask \"...\" as the short form from the root command.",
	)
}

func (a *App) printAssistResumeUsage() {
	printHelpUsage(a.stderr, "gig assist resume", "gig resume")
	printHelpBullets(a.stderr, "Resume", "Shows the saved assist session for the current project, repo target, or workspace so gig can continue with gig ask.")
}

func (a *App) printAssistResolveUsage() {
	printHelpUsage(a.stderr, "gig assist resolve --path . [--ticket ABC-123]")
	printHelpBullets(a.stderr, "Tips",
		"Use this only after Git has already stopped on a conflict.",
		"Use gig resolve start when you are ready to apply a choice.",
	)
}

func (a *App) printDoctorUsage() {
	printHelpUsage(a.stderr, "gig doctor [--path .] [--fix] [--format human|json] [--json]")
	printHelpBullets(a.stderr, "Tips",
		"Gig uses provider metadata, inferred branch topology, and built-in defaults when no override file is present.",
		"Add --config only when your team needs explicit branch or repository metadata overrides.",
		"Add --fix to print setup commands for missing required tools.",
		"Set GIG_DIAGNOSTICS_FILE=/path/to/gig-diagnostics.jsonl when you want structured auth and topology traces.",
	)
}

func (a *App) printSetupUsage() {
	printHelpUsage(a.stderr, "gig setup [--provider github|gitlab|azure-devops|svn|all] [--install-missing] [--yes]")
	printHelpBullets(a.stderr, "Safety",
		"Setup never installs tools unless --install-missing is present.",
		"Without --yes, setup asks for confirmation before running install commands.",
		"Run gig login <provider> after installing provider tools.",
	)
}

func (a *App) printResolveUsage() {
	printHelpUsage(a.stderr,
		"gig resolve status --path . [--ticket ABC-123] [--format human|json]",
		"gig resolve start --path . [--ticket ABC-123]",
	)
}

func (a *App) printResolveStatusUsage() {
	printHelpUsage(a.stderr, "gig resolve status --path . [--ticket ABC-123] [--format human|json]")
}

func (a *App) printResolveStartUsage() {
	printHelpUsage(a.stderr, "gig resolve start --path . [--ticket ABC-123]")
}

func (a *App) printUpdateUsage() {
	printHelpUsage(a.stderr, "gig update [<version>] [--version vYYYY.MM.DD] [--install-dir /path/to/bin] [--repo owner/name]")
	printHelpBullets(a.stderr, "Tip", "The direct installer path is the canonical install and update flow.")
}
