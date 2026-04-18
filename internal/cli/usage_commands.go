package cli

import "fmt"

func (a *App) printRootUsage() {
	printHelpHeading(a.stderr, "gig", "Remote-first release audit CLI")
	printHelpUsage(a.stderr, "gig [ticket-id | command] [flags]")
	printHelpBullets(a.stderr, "First-time users",
		"Run `gig` in a real terminal to open the guided picker.",
		"Start with GitHub unless your repository lives elsewhere.",
		"Learn `inspect`, `verify`, and `manifest` first.",
	)
	printHelpCommands(a.stderr, "Start here",
		"gig",
		"gig login github",
		"gig ABC-123 --repo github:owner/name",
		"gig resume",
		"gig ask \"what is still blocked?\"",
	)
	printHelpCommands(a.stderr, "Core workflows",
		"gig ABC-123",
		"gig verify ABC-123",
		"gig manifest ABC-123",
		"gig ABC-123 --path .",
	)
	printHelpRows(a.stderr, "Provider coverage", providerCoverageHelpRows()...)
	printHelpRows(a.stderr, "Commands",
		helpRow{Label: "workarea", Value: "Remember a project so later commands stay short"},
		helpRow{Label: "login", Value: "Authenticate to a live provider such as GitHub"},
		helpRow{Label: "inspect", Value: "Show the full ticket picture across repositories"},
		helpRow{Label: "verify", Value: "Return safe, warning, or blocked for the next move"},
		helpRow{Label: "manifest", Value: "Generate a release packet for QA and release review"},
		helpRow{Label: "plan", Value: "Build a read-only promotion plan when you need more detail"},
		helpRow{Label: "scan", Value: "Find repositories under a local path"},
		helpRow{Label: "find", Value: "List raw commits for one ticket"},
		helpRow{Label: "env status", Value: "Show where a ticket is present or behind"},
		helpRow{Label: "diff", Value: "Compare one branch to another for a ticket"},
		helpRow{Label: "snapshot", Value: "Save a repeatable ticket baseline for audit and re-check"},
		helpRow{Label: "assist", Value: "Add an optional AI briefing on top of gig evidence"},
		helpRow{Label: "ask", Value: "Continue the last AI brief with a follow-up question"},
		helpRow{Label: "resume", Value: "Show the last AI brief for the current project or repo"},
		helpRow{Label: "doctor", Value: "Check inferred topology, overrides, and repo health"},
		helpRow{Label: "resolve", Value: "Inspect or resolve active Git merge conflicts"},
		helpRow{Label: "update", Value: "Install the latest release or a specific version"},
		helpRow{Label: "version", Value: "Show the installed version"},
	)
	a.printCurrentWorkareaHint()
	printHelpCommands(a.stderr, "More help", "gig <command> --help")
}

func (a *App) printLoginUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig login <provider>")
	fmt.Fprintln(a.stderr, "Start with: github")
	fmt.Fprintln(a.stderr, "Other providers: gitlab, bitbucket, azure-devops, svn")
	fmt.Fprintln(a.stderr)
	printHelpRows(a.stderr, "Provider coverage", providerCoverageHelpRows()...)
}

func (a *App) printScanUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig scan --path .")
}

func (a *App) printFindUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig find <ticket-id> [--workarea name] [--path . | --repo <provider-target>]")
}

func (a *App) printDiffUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig diff --ticket <ticket-id> --from <branch> --to <branch> --path .")
}

func (a *App) printInspectUsage() {
	printHelpHeading(a.stderr, "gig inspect", "Show the full ticket story across repositories.")
	printHelpUsage(a.stderr, "gig inspect <ticket-id> [--workarea name] [--path . | --repo <provider-target>]")
	printHelpCommands(a.stderr, "Start here",
		"gig inspect ABC-123",
		"gig ABC-123 --repo github:owner/name",
		"gig inspect ABC-123 --workarea payments",
	)
	printHelpRows(a.stderr, "Common flags",
		helpRow{Label: "--repo", Value: "Inspect a live remote repository without cloning first"},
		helpRow{Label: "--workarea", Value: "Reuse a remembered project and its inferred defaults"},
		helpRow{Label: "--path", Value: "Use local workspace mode when remote access is not enough"},
		helpRow{Label: "--config", Value: "Optional override file when inference needs help"},
	)
	printHelpCommands(a.stderr, "Next commands",
		"gig verify ABC-123",
		"gig plan ABC-123",
		"gig manifest ABC-123",
	)
}

func (a *App) printEnvUsage() {
	fmt.Fprintln(a.stderr, "Usage:")
	fmt.Fprintln(a.stderr, "  gig env status <ticket-id> [--workarea name] [--path . | --repo <provider-target>] [--envs dev=dev,test=test,prod=main]")
}

func (a *App) printEnvStatusUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig env status <ticket-id> [--workarea name] [--path . | --repo <provider-target>] [--envs dev=dev,test=test,prod=main]")
}

func (a *App) printPlanUsage() {
	printHelpHeading(a.stderr, "gig plan", "Build a read-only promotion plan for one ticket or a saved release.")
	printHelpUsage(a.stderr,
		"gig plan ABC-123 [--repo github:owner/name | --workarea payments | --path .]",
		"gig plan --ticket-file tickets.txt [--repo github:owner/name | --workarea payments | --path .]",
		"gig plan --release rel-2026-04-09 --path . [--format human|json]",
	)
	printHelpCommands(a.stderr, "Start here",
		"gig plan ABC-123",
		"gig plan ABC-123 --repo github:owner/name",
		"gig plan --release rel-2026-04-09 --path .",
	)
	printHelpRows(a.stderr, "Common flags",
		helpRow{Label: "--repo", Value: "Plan directly against a live remote repository"},
		helpRow{Label: "--workarea", Value: "Reuse remembered repo scope and branch defaults"},
		helpRow{Label: "--from/--to", Value: "Only add these when gig cannot infer the promotion path"},
		helpRow{Label: "--envs", Value: "Override environment mapping only when needed"},
		helpRow{Label: "--format", Value: "Use json for automation or human for terminal review"},
	)
	printHelpCommands(a.stderr, "Next commands",
		"gig verify ABC-123",
		"gig manifest ABC-123",
	)
}

func (a *App) printVerifyUsage() {
	printHelpHeading(a.stderr, "gig verify", "Turn ticket evidence into a safe, warning, or blocked release verdict.")
	printHelpUsage(a.stderr,
		"gig verify ABC-123 [--repo github:owner/name | --workarea payments | --path .]",
		"gig verify --ticket-file tickets.txt [--repo github:owner/name | --workarea payments | --path .]",
		"gig verify --release rel-2026-04-09 --path .",
	)
	printHelpCommands(a.stderr, "Start here",
		"gig verify ABC-123",
		"gig verify ABC-123 --repo github:owner/name",
		"gig verify --release rel-2026-04-09 --path .",
	)
	printHelpRows(a.stderr, "Common flags",
		helpRow{Label: "--repo", Value: "Verify directly against a live remote repository"},
		helpRow{Label: "--workarea", Value: "Reuse remembered repo scope and branch defaults"},
		helpRow{Label: "--from/--to", Value: "Only add these when gig cannot infer the promotion path"},
		helpRow{Label: "--format", Value: "Add json output for automation"},
	)
	printHelpCommands(a.stderr, "Next commands",
		"gig plan ABC-123",
		"gig manifest ABC-123",
	)
}

func (a *App) printManifestUsage() {
	printHelpHeading(a.stderr, "gig manifest", "Generate a release packet for QA, client, and release review.")
	printHelpUsage(a.stderr,
		"gig manifest ABC-123 [--repo github:owner/name | --workarea payments | --path .]",
		"gig manifest --ticket-file tickets.txt [--repo github:owner/name | --workarea payments | --path .]",
		"gig manifest --release rel-2026-04-09 --path .",
	)
	printHelpCommands(a.stderr, "Start here",
		"gig manifest ABC-123",
		"gig manifest ABC-123 --repo github:owner/name",
		"gig manifest --release rel-2026-04-09 --path .",
	)
	printHelpRows(a.stderr, "Common flags",
		helpRow{Label: "--repo", Value: "Generate a packet directly from a live remote repository"},
		helpRow{Label: "--workarea", Value: "Reuse remembered repo scope and branch defaults"},
		helpRow{Label: "--format", Value: "Keep markdown for handoff or use json for tooling"},
		helpRow{Label: "--from/--to", Value: "Only add these when gig cannot infer the promotion path"},
		helpRow{Label: "Alias", Value: "`gig manifest generate ...` still works for existing scripts"},
	)
	printHelpCommands(a.stderr, "Next commands",
		"gig assist audit --ticket ABC-123 --audience release-manager",
		"gig assist release --release rel-2026-04-09 --path .",
	)
}

func (a *App) printManifestGenerateUsage() {
	a.printManifestUsage()
}

func (a *App) printSnapshotUsage() {
	fmt.Fprintln(a.stderr, "Usage:")
	fmt.Fprintln(a.stderr, "  gig snapshot create --ticket <ticket-id> [--workarea name] [--from <branch>] [--to <branch>] [--path . | --repo <provider-target>] [--release <release-id>] [--envs dev=dev,test=test,prod=main] [--format human|json] [--output snapshot.json]")
}

func (a *App) printSnapshotCreateUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig snapshot create --ticket <ticket-id> [--workarea name] [--from <branch>] [--to <branch>] [--path . | --repo <provider-target>] [--release <release-id>] [--envs dev=dev,test=test,prod=main] [--format human|json] [--output snapshot.json]")
}

func (a *App) printAssistUsage() {
	fmt.Fprintln(a.stderr, "Usage:")
	fmt.Fprintln(a.stderr, "  gig assist doctor [--path .] [--url http://localhost:2026] [--format human|json]")
	fmt.Fprintln(a.stderr, "  gig assist setup [--path .] [--format human|json]")
	fmt.Fprintln(a.stderr, "  gig assist audit --ticket ABC-123 [--repo github:owner/name | --workarea payments | --path .]")
	fmt.Fprintln(a.stderr, "  gig assist chat \"what is still blocked?\"")
	fmt.Fprintln(a.stderr, "  gig assist release --release rel-2026-04-09 [--path . | --ticket-file tickets.txt --repo github:owner/name]")
	fmt.Fprintln(a.stderr, "  gig assist resume")
	fmt.Fprintln(a.stderr, "  gig assist resolve --path . [--ticket ABC-123]")
	fmt.Fprintln(a.stderr)
	fmt.Fprintln(a.stderr, "Common flags:")
	fmt.Fprintln(a.stderr, "  --audience qa|client|release-manager   --mode flash|standard|pro|ultra")
	fmt.Fprintln(a.stderr, "  --url http://localhost:2026            --format human|json")
}

func (a *App) printAssistDoctorUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig assist doctor [--path .] [--url http://localhost:2026] [--format human|json]")
	fmt.Fprintln(a.stderr)
	fmt.Fprintln(a.stderr, "Checks whether the bundled DeerFlow sidecar is configured, startable, and reachable.")
}

func (a *App) printAssistSetupUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig assist setup [--path .] [--format human|json]")
	fmt.Fprintln(a.stderr)
	fmt.Fprintln(a.stderr, "Bootstraps the bundled deer-flow sidecar config and prints the next start command.")
	fmt.Fprintln(a.stderr, "Run gig assist doctor first if you want a readiness check without writing files.")
}

func (a *App) printAssistAuditUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig assist audit --ticket ABC-123 [--repo github:owner/name | --workarea payments | --path .]")
	fmt.Fprintln(a.stderr)
	fmt.Fprintln(a.stderr, "Tips:")
	fmt.Fprintln(a.stderr, "  use the current workarea when no --repo or --path is set")
	fmt.Fprintln(a.stderr, "  add --audience qa|client|release-manager")
	fmt.Fprintln(a.stderr, "  add --url http://localhost:2026 when DeerFlow is not on the default local port")
}

func (a *App) printAssistReleaseUsage() {
	fmt.Fprintln(a.stderr, "Usage:")
	fmt.Fprintln(a.stderr, "  gig assist release --release rel-2026-04-09 --path .")
	fmt.Fprintln(a.stderr, "  gig assist release --release rel-2026-04-09 --ticket-file tickets.txt --repo github:owner/name")
	fmt.Fprintln(a.stderr)
	fmt.Fprintln(a.stderr, "Tips:")
	fmt.Fprintln(a.stderr, "  use --path when the release already has saved snapshots")
	fmt.Fprintln(a.stderr, "  use --ticket or --ticket-file with --repo for a live remote bundle")
}

func (a *App) printAssistChatUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig assist chat [--message \"what changed?\"] [--mode flash|standard|pro|ultra] [--audience qa|client|release-manager] [--url http://localhost:2026] [--format human|json]")
	fmt.Fprintln(a.stderr)
	fmt.Fprintln(a.stderr, "Tips:")
	fmt.Fprintln(a.stderr, "  run gig assist audit, release, or resolve first so gig has a saved deterministic session to continue")
	fmt.Fprintln(a.stderr, "  use gig ask \"...\" as the short form from the root command")
}

func (a *App) printAssistResumeUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig assist resume")
	fmt.Fprintln(a.stderr, "   or: gig resume")
	fmt.Fprintln(a.stderr)
	fmt.Fprintln(a.stderr, "Shows the saved assist session for the current workarea, repo target, or workspace so gig can continue with gig ask.")
}

func (a *App) printAssistResolveUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig assist resolve --path . [--ticket ABC-123]")
	fmt.Fprintln(a.stderr)
	fmt.Fprintln(a.stderr, "Tips:")
	fmt.Fprintln(a.stderr, "  use this only after Git has already stopped on a conflict")
	fmt.Fprintln(a.stderr, "  use gig resolve start when you are ready to apply a choice")
}

func (a *App) printDoctorUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig doctor --path . [--format human|json]")
	fmt.Fprintln(a.stderr)
	fmt.Fprintln(a.stderr, "Tips:")
	fmt.Fprintln(a.stderr, "  gig uses provider metadata, inferred branch topology, and built-in defaults when no override file is present")
	fmt.Fprintln(a.stderr, "  add --config only when your team needs explicit branch or repository metadata overrides")
	fmt.Fprintln(a.stderr, "  set GIG_DIAGNOSTICS_FILE=/path/to/gig-diagnostics.jsonl when you want structured auth and topology traces")
}

func (a *App) printResolveUsage() {
	fmt.Fprintln(a.stderr, "Usage:")
	fmt.Fprintln(a.stderr, "  gig resolve status --path . [--ticket ABC-123] [--format human|json]")
	fmt.Fprintln(a.stderr, "  gig resolve start --path . [--ticket ABC-123]")
}

func (a *App) printResolveStatusUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig resolve status --path . [--ticket ABC-123] [--format human|json]")
}

func (a *App) printResolveStartUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig resolve start --path . [--ticket ABC-123]")
}

func (a *App) printUpdateUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig update [<version>] [--version vYYYY.MM.DD] [--install-dir /path/to/bin] [--repo owner/name]")
	fmt.Fprintln(a.stderr, "Tip: the direct installer path is the canonical install and update flow")
}
