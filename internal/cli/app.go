package cli

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	assistsvc "gig/internal/assistant"
	"gig/internal/buildinfo"
	"gig/internal/config"
	conflictsvc "gig/internal/conflict"
	diffsvc "gig/internal/diff"
	doctorsvc "gig/internal/doctor"
	inspectsvc "gig/internal/inspect"
	manifestsvc "gig/internal/manifest"
	"gig/internal/output"
	plansvc "gig/internal/plan"
	releaseplansvc "gig/internal/releaseplan"
	"gig/internal/repo"
	"gig/internal/scm"
	azuredevopsscm "gig/internal/scm/azuredevops"
	bitbucketscm "gig/internal/scm/bitbucket"
	gitscm "gig/internal/scm/git"
	githubscm "gig/internal/scm/github"
	gitlabscm "gig/internal/scm/gitlab"
	svnscm "gig/internal/scm/svn"
	snapshotsvc "gig/internal/snapshot"
	"gig/internal/sourcecontrol"
	ticketsvc "gig/internal/ticket"
	updatesvc "gig/internal/update"
)

const usageExitCode = 2
const optionalOverrideFileHelp = "Optional gig override file"

type App struct {
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
	scanner *repo.Scanner
}

func NewApp(stdout, stderr io.Writer) (*App, error) {
	return NewAppWithIO(os.Stdin, stdout, stderr)
}

func NewAppWithIO(stdin io.Reader, stdout, stderr io.Writer) (*App, error) {
	parser, err := ticketsvc.NewParser(config.Default().TicketPattern)
	if err != nil {
		return nil, err
	}

	return &App{
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		scanner: repo.NewScanner(newRegistry(parser)),
	}, nil
}

func (a *App) Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return a.runFrontDoor(ctx)
	}

	switch args[0] {
	case "scan":
		return a.runScan(ctx, args[1:])
	case "find":
		return a.runFind(ctx, args[1:])
	case "diff":
		return a.runDiff(ctx, args[1:])
	case "inspect":
		return a.runInspect(ctx, args[1:])
	case "env":
		return a.runEnv(ctx, args[1:])
	case "plan":
		return a.runPlan(ctx, args[1:])
	case "verify":
		return a.runVerify(ctx, args[1:])
	case "manifest":
		return a.runManifest(ctx, args[1:])
	case "snapshot":
		return a.runSnapshot(ctx, args[1:])
	case "assist":
		return a.runAssist(ctx, args[1:])
	case "workarea", "project":
		return a.runWorkarea(ctx, args[1:])
	case "login":
		return a.runLogin(ctx, args[1:])
	case "doctor":
		return a.runDoctor(ctx, args[1:])
	case "resolve":
		return a.runResolve(ctx, args[1:])
	case "update":
		return a.runUpdate(ctx, args[1:])
	case "version", "-v", "--version":
		return a.runVersion()
	case "help", "-h", "--help":
		a.printRootUsage()
		return 0
	default:
		if looksLikeTicketID(args[0]) {
			return a.runInspect(ctx, args)
		}
		fmt.Fprintf(a.stderr, "unknown command %q\n\n", args[0])
		a.printRootUsage()
		return usageExitCode
	}
}

func (a *App) runScan(ctx context.Context, args []string) int {
	if hasHelpFlag(args) {
		a.printScanUsage()
		return 0
	}

	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a repository or workspace")
	if err := fs.Parse(args); err != nil {
		a.printScanUsage()
		return usageExitCode
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(a.stderr, "scan does not accept positional arguments")
		a.printScanUsage()
		return usageExitCode
	}

	resolvedPath, err := normalizeCLIPath(*path)
	if err != nil {
		fmt.Fprintf(a.stderr, "scan failed: %v\n", err)
		return 1
	}

	repositories, err := a.scanner.Discover(ctx, resolvedPath)
	if err != nil {
		fmt.Fprintf(a.stderr, "scan failed: %v\n", err)
		return 1
	}

	if err := output.RenderScan(a.stdout, resolvedPath, repositories); err != nil {
		fmt.Fprintf(a.stderr, "render failed: %v\n", err)
		return 1
	}

	return 0
}

func (a *App) runFind(ctx context.Context, args []string) int {
	reorderedArgs, err := reorderArgsWithSinglePositional(args, "-path", "--path", "-config", "--config", "-repo", "--repo", "-workarea", "--workarea")
	if err != nil {
		fmt.Fprintf(a.stderr, "find failed: %v\n", err)
		a.printFindUsage()
		return usageExitCode
	}
	args = reorderedArgs

	if hasHelpFlag(args) {
		a.printFindUsage()
		return 0
	}

	fs := flag.NewFlagSet("find", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a repository or workspace")
	configPath := fs.String("config", "", optionalOverrideFileHelp)
	repoTarget := fs.String("repo", "", "Remote repository target, for example github:owner/name or gitlab:group/project")
	workareaName := fs.String("workarea", "", "Saved workarea to use")
	if err := fs.Parse(args); err != nil {
		a.printFindUsage()
		return usageExitCode
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(a.stderr, "find requires exactly one <ticket-id> argument")
		a.printFindUsage()
		return usageExitCode
	}

	ticketID := normalizeTicketID(fs.Arg(0))
	scope, err := a.resolveCommandScope(*path, *configPath, *repoTarget, *workareaName, flagProvided(fs, "path"), flagProvided(fs, "config"), flagProvided(fs, "repo"))
	if err != nil {
		fmt.Fprintf(a.stderr, "find failed: %v\n", err)
		return 1
	}
	a.announceWorkareaSelection(scope, commandDefaults{})

	remoteMode := strings.TrimSpace(scope.RepoSpec) != ""
	runtime, err := newCommandRuntimeWithOptions(scope.WorkspacePath, scope.ConfigPath, runtimeOptions{DisableAutoConfig: remoteMode})
	if err != nil {
		fmt.Fprintf(a.stderr, "find failed: %v\n", err)
		return 1
	}

	repositories, scopeLabel, err := a.resolveCommandRepositories(ctx, scope.WorkspacePath, scope.RepoSpec, runtime)
	if err != nil {
		fmt.Fprintf(a.stderr, "find failed: %v\n", err)
		return 1
	}

	results, err := runtime.finder.FindInRepositories(ctx, repositories, ticketID)
	if err != nil {
		fmt.Fprintf(a.stderr, "find failed: %v\n", err)
		return 1
	}

	if err := output.RenderFind(a.stdout, ticketID, scopeLabel, len(repositories), results); err != nil {
		fmt.Fprintf(a.stderr, "render failed: %v\n", err)
		return 1
	}
	a.rememberProjectMemory(scope, commandDefaults{}, runtime, repositories, nil, "", "")

	return 0
}

func (a *App) runDiff(ctx context.Context, args []string) int {
	if hasHelpFlag(args) {
		a.printDiffUsage()
		return 0
	}

	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a repository or workspace")
	configPath := fs.String("config", "", optionalOverrideFileHelp)
	ticketID := fs.String("ticket", "", "Ticket ID to compare")
	fromBranch := fs.String("from", "", "Source branch")
	toBranch := fs.String("to", "", "Target branch")

	if err := fs.Parse(args); err != nil {
		a.printDiffUsage()
		return usageExitCode
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(a.stderr, "diff does not accept positional arguments")
		a.printDiffUsage()
		return usageExitCode
	}

	resolvedPath, err := normalizeCLIPath(*path)
	if err != nil {
		fmt.Fprintf(a.stderr, "diff failed: %v\n", err)
		return 1
	}

	normalizedTicketID := normalizeTicketID(*ticketID)
	runtime, err := newCommandRuntime(resolvedPath, *configPath)
	if err != nil {
		fmt.Fprintf(a.stderr, "diff failed: %v\n", err)
		return 1
	}

	repositories, err := runtime.scanner.Discover(ctx, resolvedPath)
	if err != nil {
		fmt.Fprintf(a.stderr, "diff failed: %v\n", err)
		return 1
	}

	results, err := runtime.diff.CompareTicketInRepositories(ctx, repositories, normalizedTicketID, *fromBranch, *toBranch)
	if err != nil {
		fmt.Fprintf(a.stderr, "diff failed: %v\n", err)
		return 1
	}

	if err := output.RenderDiff(a.stdout, normalizedTicketID, *fromBranch, *toBranch, resolvedPath, len(repositories), results); err != nil {
		fmt.Fprintf(a.stderr, "render failed: %v\n", err)
		return 1
	}

	return 0
}

func (a *App) runInspect(ctx context.Context, args []string) int {
	reorderedArgs, err := reorderArgsWithSinglePositional(args, "-path", "--path", "-config", "--config", "-repo", "--repo", "-workarea", "--workarea")
	if err != nil {
		fmt.Fprintf(a.stderr, "inspect failed: %v\n", err)
		a.printInspectUsage()
		return usageExitCode
	}
	args = reorderedArgs

	if hasHelpFlag(args) {
		a.printInspectUsage()
		return 0
	}

	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a repository or workspace")
	configPath := fs.String("config", "", optionalOverrideFileHelp)
	repoTarget := fs.String("repo", "", "Remote repository target, for example github:owner/name or gitlab:group/project")
	workareaName := fs.String("workarea", "", "Saved workarea to use")
	if err := fs.Parse(args); err != nil {
		a.printInspectUsage()
		return usageExitCode
	}
	if fs.NArg() != 1 {
		printUsageFailure(a.stderr, "inspect", "provide exactly one ticket ID.", "gig inspect ABC-123", "gig ABC-123 --repo github:owner/name")
		a.printInspectUsage()
		return usageExitCode
	}

	ticketID := normalizeTicketID(fs.Arg(0))
	scope, err := a.resolveCommandScope(*path, *configPath, *repoTarget, *workareaName, flagProvided(fs, "path"), flagProvided(fs, "config"), flagProvided(fs, "repo"))
	if err != nil {
		fmt.Fprintf(a.stderr, "inspect failed: %v\n", err)
		return 1
	}
	a.announceWorkareaSelection(scope, commandDefaults{})

	remoteMode := strings.TrimSpace(scope.RepoSpec) != ""
	runtime, err := newCommandRuntimeWithOptions(scope.WorkspacePath, scope.ConfigPath, runtimeOptions{DisableAutoConfig: remoteMode})
	if err != nil {
		fmt.Fprintf(a.stderr, "inspect failed: %v\n", err)
		return 1
	}

	repositories, scopeLabel, err := a.resolveCommandRepositories(ctx, scope.WorkspacePath, scope.RepoSpec, runtime)
	if err != nil {
		fmt.Fprintf(a.stderr, "inspect failed: %v\n", err)
		return 1
	}

	results, err := runtime.inspect.InspectInRepositories(ctx, repositories, ticketID)
	if err != nil {
		fmt.Fprintf(a.stderr, "inspect failed: %v\n", err)
		return 1
	}

	if err := output.RenderInspect(a.stdout, ticketID, scopeLabel, len(repositories), results); err != nil {
		fmt.Fprintf(a.stderr, "render failed: %v\n", err)
		return 1
	}
	a.rememberProjectMemory(scope, commandDefaults{}, runtime, repositories, nil, "", "")

	return 0
}

func (a *App) runEnv(ctx context.Context, args []string) int {
	if len(args) == 0 || hasHelpFlag(args) {
		a.printEnvUsage()
		return 0
	}

	switch args[0] {
	case "status":
		return a.runEnvStatus(ctx, args[1:])
	default:
		fmt.Fprintf(a.stderr, "unknown env subcommand %q\n\n", args[0])
		a.printEnvUsage()
		return usageExitCode
	}
}

func (a *App) runEnvStatus(ctx context.Context, args []string) int {
	reorderedArgs, err := reorderArgsWithSinglePositional(args, "-path", "--path", "-config", "--config", "-envs", "--envs", "-repo", "--repo", "-workarea", "--workarea")
	if err != nil {
		fmt.Fprintf(a.stderr, "env status failed: %v\n", err)
		a.printEnvStatusUsage()
		return usageExitCode
	}
	args = reorderedArgs

	if hasHelpFlag(args) {
		a.printEnvStatusUsage()
		return 0
	}

	fs := flag.NewFlagSet("env status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a repository or workspace")
	configPath := fs.String("config", "", optionalOverrideFileHelp)
	envsSpec := fs.String("envs", "", "Comma-separated environment mapping, for example dev=dev,test=test,prod=main")
	repoTarget := fs.String("repo", "", "Remote repository target, for example github:owner/name or gitlab:group/project")
	workareaName := fs.String("workarea", "", "Saved workarea to use")
	if err := fs.Parse(args); err != nil {
		a.printEnvStatusUsage()
		return usageExitCode
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(a.stderr, "env status requires exactly one <ticket-id> argument")
		a.printEnvStatusUsage()
		return usageExitCode
	}

	scope, err := a.resolveCommandScope(*path, *configPath, *repoTarget, *workareaName, flagProvided(fs, "path"), flagProvided(fs, "config"), flagProvided(fs, "repo"))
	if err != nil {
		fmt.Fprintf(a.stderr, "env status failed: %v\n", err)
		return 1
	}
	defaults := resolveCommandDefaults(scope.Workarea, *envsSpec, "", "", flagProvided(fs, "envs"), false, false)
	a.announceWorkareaSelection(scope, defaults)

	remoteMode := strings.TrimSpace(scope.RepoSpec) != ""
	runtime, err := newCommandRuntimeWithOptions(scope.WorkspacePath, scope.ConfigPath, runtimeOptions{DisableAutoConfig: remoteMode})
	if err != nil {
		fmt.Fprintf(a.stderr, "env status failed: %v\n", err)
		return 1
	}

	repositories, scopeLabel, err := a.resolveCommandRepositories(ctx, scope.WorkspacePath, scope.RepoSpec, runtime)
	if err != nil {
		fmt.Fprintf(a.stderr, "env status failed: %v\n", err)
		return 1
	}
	environments, err := resolveOperationEnvironments(ctx, runtime, repositories, defaults.EnvironmentSpec)
	if err != nil {
		fmt.Fprintf(a.stderr, "env status failed: %v\n", err)
		return usageExitCode
	}
	a.rememberProjectMemory(scope, defaults, runtime, repositories, environments, "", "")

	ticketID := normalizeTicketID(fs.Arg(0))
	results, err := runtime.inspect.EnvironmentStatusInRepositories(ctx, repositories, ticketID, environments)
	if err != nil {
		fmt.Fprintf(a.stderr, "env status failed: %v\n", err)
		return 1
	}

	if err := output.RenderEnvironmentStatus(a.stdout, ticketID, scopeLabel, environments, len(repositories), results); err != nil {
		fmt.Fprintf(a.stderr, "render failed: %v\n", err)
		return 1
	}

	return 0
}

func (a *App) runPlan(ctx context.Context, args []string) int {
	reorderedArgs, err := reorderArgsWithSinglePositional(args,
		"-path", "--path",
		"-config", "--config",
		"-repo", "--repo",
		"-workarea", "--workarea",
		"-release", "--release",
		"-ticket", "--ticket",
		"-ticket-file", "--ticket-file",
		"-from", "--from",
		"-to", "--to",
		"-envs", "--envs",
		"-format", "--format",
	)
	if err != nil {
		fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
		a.printPlanUsage()
		return usageExitCode
	}
	args = reorderedArgs

	if hasHelpFlag(args) {
		a.printPlanUsage()
		return 0
	}

	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a repository or workspace")
	configPath := fs.String("config", "", optionalOverrideFileHelp)
	repoTarget := fs.String("repo", "", "Remote repository target, for example github:owner/name or gitlab:group/project")
	workareaName := fs.String("workarea", "", "Saved workarea to use")
	releaseID := fs.String("release", "", "Release ID to plan from saved snapshots")
	ticketID := fs.String("ticket", "", "Ticket ID to plan")
	ticketFile := fs.String("ticket-file", "", "Path to a file with one ticket ID per line")
	fromBranch := fs.String("from", "", "Source branch")
	toBranch := fs.String("to", "", "Target branch")
	envsSpec := fs.String("envs", "", "Comma-separated environment mapping, for example dev=dev,test=test,prod=main")
	format := fs.String("format", string(outputFormatHuman), "Output format: human or json")

	if err := fs.Parse(args); err != nil {
		a.printPlanUsage()
		return usageExitCode
	}
	if fs.NArg() > 1 {
		printUsageFailure(a.stderr, "plan", "provide at most one positional ticket ID.", "gig plan ABC-123", "gig plan --release rel-2026-04-09 --path .")
		a.printPlanUsage()
		return usageExitCode
	}
	if fs.NArg() == 1 {
		if strings.TrimSpace(*releaseID) != "" || strings.TrimSpace(*ticketID) != "" || strings.TrimSpace(*ticketFile) != "" {
			printUsageFailure(a.stderr, "plan", "choose either one ticket ID, --ticket-file, or --release.", "gig plan ABC-123", "gig plan --ticket-file tickets.txt --repo github:owner/name")
			a.printPlanUsage()
			return usageExitCode
		}
		*ticketID = normalizeTicketID(fs.Arg(0))
	}

	outputFormat, err := parseOutputFormat(*format)
	if err != nil {
		fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
		return usageExitCode
	}

	scope, err := a.resolveCommandScope(*path, *configPath, *repoTarget, *workareaName, flagProvided(fs, "path"), flagProvided(fs, "config"), flagProvided(fs, "repo"))
	if err != nil {
		fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
		return 1
	}
	defaults := resolveCommandDefaults(scope.Workarea, *envsSpec, *fromBranch, *toBranch, flagProvided(fs, "envs"), flagProvided(fs, "from"), flagProvided(fs, "to"))
	a.announceWorkareaSelection(scope, defaults)

	remoteMode := strings.TrimSpace(scope.RepoSpec) != ""
	runtime, err := newCommandRuntimeWithOptions(scope.WorkspacePath, scope.ConfigPath, runtimeOptions{DisableAutoConfig: remoteMode})
	if err != nil {
		fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
		return 1
	}

	normalizedReleaseID := strings.TrimSpace(*releaseID)
	if normalizedReleaseID != "" {
		if strings.TrimSpace(scope.RepoSpec) != "" && !scope.RepoInherited {
			printUsageFailure(a.stderr, "plan", "do not combine --release with an explicit --repo target.", "gig plan --release rel-2026-04-09 --path .", "gig plan ABC-123 --repo github:owner/name")
			a.printPlanUsage()
			return usageExitCode
		}
		if *ticketID != "" || *ticketFile != "" {
			printUsageFailure(a.stderr, "plan", "choose either --release or ticket-based flags, not both.", "gig plan --release rel-2026-04-09 --path .", "gig plan ABC-123 --repo github:owner/name")
			a.printPlanUsage()
			return usageExitCode
		}
		if strings.TrimSpace(*fromBranch) != "" || strings.TrimSpace(*toBranch) != "" || strings.TrimSpace(*envsSpec) != "" {
			printUsageFailure(a.stderr, "plan", "--from, --to, and --envs are only for ticket-based planning.", "gig plan --release rel-2026-04-09 --path .", "gig plan ABC-123 --from test --to main --path .")
			a.printPlanUsage()
			return usageExitCode
		}

		normalizedReleaseID, err = snapshotsvc.NormalizeReleaseID(normalizedReleaseID)
		if err != nil {
			fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
			return usageExitCode
		}

		snapshots, snapshotDir, err := snapshotsvc.LoadReleaseSnapshots(scope.WorkspacePath, normalizedReleaseID)
		if err != nil {
			fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
			return 1
		}
		releasePlan, err := releaseplansvc.Build(normalizedReleaseID, scope.WorkspacePath, snapshotDir, snapshots)
		if err != nil {
			fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
			return 1
		}

		switch outputFormat {
		case outputFormatHuman:
			if err := output.RenderReleasePlan(a.stdout, releasePlan); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		case outputFormatJSON:
			if err := output.RenderJSON(a.stdout, struct {
				Command     string                     `json:"command"`
				Workspace   string                     `json:"workspace"`
				ReleasePlan releaseplansvc.ReleasePlan `json:"releasePlan"`
			}{
				Command:     "plan",
				Workspace:   scope.WorkspacePath,
				ReleasePlan: releasePlan,
			}); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		}

		return 0
	}

	ticketIDs, resolvedTicketFile, err := resolveTicketIDs(*ticketID, *ticketFile, runtime.parser)
	if err != nil {
		fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
		return usageExitCode
	}

	repositories, scopeLabel, err := a.resolveCommandRepositories(ctx, scope.WorkspacePath, scope.RepoSpec, runtime)
	if err != nil {
		fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
		return 1
	}

	environments, resolvedFromBranch, resolvedToBranch, err := resolveOperationContext(ctx, runtime, repositories, defaults.EnvironmentSpec, defaults.FromBranch, defaults.ToBranch)
	if err != nil {
		fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
		return usageExitCode
	}
	a.rememberProjectMemory(scope, defaults, runtime, repositories, environments, resolvedFromBranch, resolvedToBranch)

	promotionPlans := make([]plansvc.PromotionPlan, 0, len(ticketIDs))
	for _, ticketID := range ticketIDs {
		promotionPlan, err := runtime.planner.BuildPromotionPlanInRepositories(ctx, repositories, ticketID, resolvedFromBranch, resolvedToBranch, environments)
		if err != nil {
			fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
			return 1
		}
		promotionPlans = append(promotionPlans, promotionPlan)
	}

	switch outputFormat {
	case outputFormatHuman:
		if len(promotionPlans) == 1 {
			if err := output.RenderPromotionPlan(a.stdout, promotionPlans[0]); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		} else {
			if err := output.RenderPromotionPlanBatch(a.stdout, promotionPlans); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		}
	case outputFormatJSON:
		if len(promotionPlans) == 1 {
			if err := output.RenderJSON(a.stdout, struct {
				Command   string                `json:"command"`
				Workspace string                `json:"workspace"`
				Plan      plansvc.PromotionPlan `json:"plan"`
			}{
				Command:   "plan",
				Workspace: scopeLabel,
				Plan:      promotionPlans[0],
			}); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		} else {
			if err := output.RenderJSON(a.stdout, struct {
				Command    string                  `json:"command"`
				Workspace  string                  `json:"workspace"`
				TicketFile string                  `json:"ticketFile,omitempty"`
				Plans      []plansvc.PromotionPlan `json:"plans"`
			}{
				Command:    "plan",
				Workspace:  scopeLabel,
				TicketFile: resolvedTicketFile,
				Plans:      promotionPlans,
			}); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		}
	}

	return 0
}

func (a *App) runVerify(ctx context.Context, args []string) int {
	reorderedArgs, err := reorderArgsWithSinglePositional(args,
		"-path", "--path",
		"-config", "--config",
		"-repo", "--repo",
		"-workarea", "--workarea",
		"-release", "--release",
		"-ticket", "--ticket",
		"-ticket-file", "--ticket-file",
		"-from", "--from",
		"-to", "--to",
		"-envs", "--envs",
		"-format", "--format",
	)
	if err != nil {
		fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
		a.printVerifyUsage()
		return usageExitCode
	}
	args = reorderedArgs

	if hasHelpFlag(args) {
		a.printVerifyUsage()
		return 0
	}

	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a repository or workspace")
	configPath := fs.String("config", "", optionalOverrideFileHelp)
	repoTarget := fs.String("repo", "", "Remote repository target, for example github:owner/name or gitlab:group/project")
	workareaName := fs.String("workarea", "", "Saved workarea to use")
	releaseID := fs.String("release", "", "Release ID to verify from saved snapshots")
	ticketID := fs.String("ticket", "", "Ticket ID to verify")
	ticketFile := fs.String("ticket-file", "", "Path to a file with one ticket ID per line")
	fromBranch := fs.String("from", "", "Source branch")
	toBranch := fs.String("to", "", "Target branch")
	envsSpec := fs.String("envs", "", "Comma-separated environment mapping, for example dev=dev,test=test,prod=main")
	format := fs.String("format", string(outputFormatHuman), "Output format: human or json")

	if err := fs.Parse(args); err != nil {
		a.printVerifyUsage()
		return usageExitCode
	}
	if fs.NArg() > 1 {
		printUsageFailure(a.stderr, "verify", "provide at most one positional ticket ID.", "gig verify ABC-123", "gig verify --release rel-2026-04-09 --path .")
		a.printVerifyUsage()
		return usageExitCode
	}
	if fs.NArg() == 1 {
		if strings.TrimSpace(*releaseID) != "" || strings.TrimSpace(*ticketID) != "" || strings.TrimSpace(*ticketFile) != "" {
			printUsageFailure(a.stderr, "verify", "choose either one ticket ID, --ticket-file, or --release.", "gig verify ABC-123", "gig verify --ticket-file tickets.txt --repo github:owner/name")
			a.printVerifyUsage()
			return usageExitCode
		}
		*ticketID = normalizeTicketID(fs.Arg(0))
	}

	outputFormat, err := parseOutputFormat(*format)
	if err != nil {
		fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
		return usageExitCode
	}

	scope, err := a.resolveCommandScope(*path, *configPath, *repoTarget, *workareaName, flagProvided(fs, "path"), flagProvided(fs, "config"), flagProvided(fs, "repo"))
	if err != nil {
		fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
		return 1
	}
	defaults := resolveCommandDefaults(scope.Workarea, *envsSpec, *fromBranch, *toBranch, flagProvided(fs, "envs"), flagProvided(fs, "from"), flagProvided(fs, "to"))
	a.announceWorkareaSelection(scope, defaults)

	remoteMode := strings.TrimSpace(scope.RepoSpec) != ""
	runtime, err := newCommandRuntimeWithOptions(scope.WorkspacePath, scope.ConfigPath, runtimeOptions{DisableAutoConfig: remoteMode})
	if err != nil {
		fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
		return 1
	}

	normalizedReleaseID := strings.TrimSpace(*releaseID)
	if normalizedReleaseID != "" {
		if strings.TrimSpace(scope.RepoSpec) != "" && !scope.RepoInherited {
			printUsageFailure(a.stderr, "verify", "do not combine --release with an explicit --repo target.", "gig verify --release rel-2026-04-09 --path .", "gig verify ABC-123 --repo github:owner/name")
			a.printVerifyUsage()
			return usageExitCode
		}
		if *ticketID != "" || *ticketFile != "" {
			printUsageFailure(a.stderr, "verify", "choose either --release or ticket-based flags, not both.", "gig verify --release rel-2026-04-09 --path .", "gig verify ABC-123 --repo github:owner/name")
			a.printVerifyUsage()
			return usageExitCode
		}
		if strings.TrimSpace(*fromBranch) != "" || strings.TrimSpace(*toBranch) != "" || strings.TrimSpace(*envsSpec) != "" {
			printUsageFailure(a.stderr, "verify", "--from, --to, and --envs are only for ticket-based verification.", "gig verify --release rel-2026-04-09 --path .", "gig verify ABC-123 --from test --to main --path .")
			a.printVerifyUsage()
			return usageExitCode
		}

		normalizedReleaseID, err = snapshotsvc.NormalizeReleaseID(normalizedReleaseID)
		if err != nil {
			fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
			return usageExitCode
		}
		snapshots, snapshotDir, err := snapshotsvc.LoadReleaseSnapshots(scope.WorkspacePath, normalizedReleaseID)
		if err != nil {
			fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
			return 1
		}

		verifications := make([]plansvc.Verification, 0, len(snapshots))
		for _, snapshot := range snapshots {
			verifications = append(verifications, snapshot.Verification)
		}

		switch outputFormat {
		case outputFormatHuman:
			if err := output.RenderReleaseVerificationBatch(a.stdout, normalizedReleaseID, snapshotDir, verifications); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		case outputFormatJSON:
			if err := output.RenderJSON(a.stdout, struct {
				Command       string                 `json:"command"`
				Workspace     string                 `json:"workspace"`
				ReleaseID     string                 `json:"releaseId"`
				SnapshotDir   string                 `json:"snapshotDir"`
				Verifications []plansvc.Verification `json:"verifications"`
			}{
				Command:       "verify",
				Workspace:     scope.WorkspacePath,
				ReleaseID:     normalizedReleaseID,
				SnapshotDir:   snapshotDir,
				Verifications: verifications,
			}); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		}

		return 0
	}

	ticketIDs, resolvedTicketFile, err := resolveTicketIDs(*ticketID, *ticketFile, runtime.parser)
	if err != nil {
		fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
		return usageExitCode
	}

	repositories, scopeLabel, err := a.resolveCommandRepositories(ctx, scope.WorkspacePath, scope.RepoSpec, runtime)
	if err != nil {
		fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
		return 1
	}

	environments, resolvedFromBranch, resolvedToBranch, err := resolveOperationContext(ctx, runtime, repositories, defaults.EnvironmentSpec, defaults.FromBranch, defaults.ToBranch)
	if err != nil {
		fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
		return usageExitCode
	}
	a.rememberProjectMemory(scope, defaults, runtime, repositories, environments, resolvedFromBranch, resolvedToBranch)

	verifications := make([]plansvc.Verification, 0, len(ticketIDs))
	for _, ticketID := range ticketIDs {
		promotionPlan, err := runtime.planner.BuildPromotionPlanInRepositories(ctx, repositories, ticketID, resolvedFromBranch, resolvedToBranch, environments)
		if err != nil {
			fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
			return 1
		}
		verifications = append(verifications, plansvc.BuildVerification(promotionPlan))
	}

	switch outputFormat {
	case outputFormatHuman:
		if len(verifications) == 1 {
			if err := output.RenderVerification(a.stdout, verifications[0]); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		} else {
			if err := output.RenderVerificationBatch(a.stdout, verifications); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		}
	case outputFormatJSON:
		if len(verifications) == 1 {
			if err := output.RenderJSON(a.stdout, struct {
				Command      string               `json:"command"`
				Workspace    string               `json:"workspace"`
				Verification plansvc.Verification `json:"verification"`
			}{
				Command:      "verify",
				Workspace:    scopeLabel,
				Verification: verifications[0],
			}); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		} else {
			if err := output.RenderJSON(a.stdout, struct {
				Command       string                 `json:"command"`
				Workspace     string                 `json:"workspace"`
				TicketFile    string                 `json:"ticketFile,omitempty"`
				Verifications []plansvc.Verification `json:"verifications"`
			}{
				Command:       "verify",
				Workspace:     scopeLabel,
				TicketFile:    resolvedTicketFile,
				Verifications: verifications,
			}); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		}
	}

	return 0
}

func (a *App) runManifest(ctx context.Context, args []string) int {
	if len(args) == 0 || hasHelpFlag(args) {
		a.printManifestUsage()
		return 0
	}

	switch args[0] {
	case "generate":
		return a.runManifestGenerate(ctx, args[1:])
	default:
		return a.runManifestGenerate(ctx, args)
	}
}

func (a *App) runManifestGenerate(ctx context.Context, args []string) int {
	reorderedArgs, err := reorderArgsWithSinglePositional(args,
		"-path", "--path",
		"-config", "--config",
		"-repo", "--repo",
		"-workarea", "--workarea",
		"-release", "--release",
		"-ticket", "--ticket",
		"-ticket-file", "--ticket-file",
		"-from", "--from",
		"-to", "--to",
		"-envs", "--envs",
		"-format", "--format",
	)
	if err != nil {
		fmt.Fprintf(a.stderr, "manifest failed: %v\n", err)
		a.printManifestGenerateUsage()
		return usageExitCode
	}
	args = reorderedArgs

	if hasHelpFlag(args) {
		a.printManifestGenerateUsage()
		return 0
	}

	fs := flag.NewFlagSet("manifest", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a repository or workspace")
	configPath := fs.String("config", "", optionalOverrideFileHelp)
	repoTarget := fs.String("repo", "", "Remote repository target, for example github:owner/name or gitlab:group/project")
	workareaName := fs.String("workarea", "", "Saved workarea to use")
	releaseID := fs.String("release", "", "Release ID to package from saved snapshots")
	ticketID := fs.String("ticket", "", "Ticket ID to package")
	ticketFile := fs.String("ticket-file", "", "Path to a file with one ticket ID per line")
	fromBranch := fs.String("from", "", "Source branch")
	toBranch := fs.String("to", "", "Target branch")
	envsSpec := fs.String("envs", "", "Comma-separated environment mapping, for example dev=dev,test=test,prod=main")
	format := fs.String("format", string(manifestFormatMarkdown), "Output format: markdown or json")

	if err := fs.Parse(args); err != nil {
		a.printManifestGenerateUsage()
		return usageExitCode
	}
	if fs.NArg() > 1 {
		printUsageFailure(a.stderr, "manifest", "provide at most one positional ticket ID.", "gig manifest ABC-123", "gig manifest --release rel-2026-04-09 --path .")
		a.printManifestGenerateUsage()
		return usageExitCode
	}
	if fs.NArg() == 1 {
		if strings.TrimSpace(*releaseID) != "" || strings.TrimSpace(*ticketID) != "" || strings.TrimSpace(*ticketFile) != "" {
			printUsageFailure(a.stderr, "manifest", "choose either one ticket ID, --ticket-file, or --release.", "gig manifest ABC-123", "gig manifest --ticket-file tickets.txt --repo github:owner/name")
			a.printManifestGenerateUsage()
			return usageExitCode
		}
		*ticketID = normalizeTicketID(fs.Arg(0))
	}

	selectedFormat, err := parseManifestFormat(*format)
	if err != nil {
		fmt.Fprintf(a.stderr, "manifest failed: %v\n", err)
		return usageExitCode
	}

	scope, err := a.resolveCommandScope(*path, *configPath, *repoTarget, *workareaName, flagProvided(fs, "path"), flagProvided(fs, "config"), flagProvided(fs, "repo"))
	if err != nil {
		fmt.Fprintf(a.stderr, "manifest failed: %v\n", err)
		return 1
	}
	defaults := resolveCommandDefaults(scope.Workarea, *envsSpec, *fromBranch, *toBranch, flagProvided(fs, "envs"), flagProvided(fs, "from"), flagProvided(fs, "to"))
	a.announceWorkareaSelection(scope, defaults)

	remoteMode := strings.TrimSpace(scope.RepoSpec) != ""
	runtime, err := newCommandRuntimeWithOptions(scope.WorkspacePath, scope.ConfigPath, runtimeOptions{DisableAutoConfig: remoteMode})
	if err != nil {
		fmt.Fprintf(a.stderr, "manifest failed: %v\n", err)
		return 1
	}

	normalizedReleaseID := strings.TrimSpace(*releaseID)
	if normalizedReleaseID != "" {
		if strings.TrimSpace(scope.RepoSpec) != "" && !scope.RepoInherited {
			printUsageFailure(a.stderr, "manifest", "do not combine --release with an explicit --repo target.", "gig manifest --release rel-2026-04-09 --path .", "gig manifest ABC-123 --repo github:owner/name")
			a.printManifestGenerateUsage()
			return usageExitCode
		}
		if *ticketID != "" || *ticketFile != "" {
			printUsageFailure(a.stderr, "manifest", "choose either --release or ticket-based flags, not both.", "gig manifest --release rel-2026-04-09 --path .", "gig manifest ABC-123 --repo github:owner/name")
			a.printManifestGenerateUsage()
			return usageExitCode
		}
		if strings.TrimSpace(*fromBranch) != "" || strings.TrimSpace(*toBranch) != "" || strings.TrimSpace(*envsSpec) != "" {
			printUsageFailure(a.stderr, "manifest", "--from, --to, and --envs are only for ticket-based packets.", "gig manifest --release rel-2026-04-09 --path .", "gig manifest ABC-123 --from test --to main --path .")
			a.printManifestGenerateUsage()
			return usageExitCode
		}

		normalizedReleaseID, err = snapshotsvc.NormalizeReleaseID(normalizedReleaseID)
		if err != nil {
			fmt.Fprintf(a.stderr, "manifest failed: %v\n", err)
			return usageExitCode
		}
		snapshots, snapshotDir, err := snapshotsvc.LoadReleaseSnapshots(scope.WorkspacePath, normalizedReleaseID)
		if err != nil {
			fmt.Fprintf(a.stderr, "manifest failed: %v\n", err)
			return 1
		}

		packets := make([]manifestsvc.ReleasePacket, 0, len(snapshots))
		for _, snapshot := range snapshots {
			packets = append(packets, manifestsvc.BuildReleasePacket(scope.WorkspacePath, runtime.loaded, snapshot.Plan))
		}

		switch selectedFormat {
		case manifestFormatMarkdown:
			if err := output.RenderReleasePacketBundleMarkdownForRelease(a.stdout, normalizedReleaseID, snapshotDir, packets); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		case manifestFormatJSON:
			if err := output.RenderJSON(a.stdout, struct {
				Command     string                      `json:"command"`
				Workspace   string                      `json:"workspace"`
				ReleaseID   string                      `json:"releaseId"`
				SnapshotDir string                      `json:"snapshotDir"`
				Packets     []manifestsvc.ReleasePacket `json:"packets"`
			}{
				Command:     "manifest generate",
				Workspace:   scope.WorkspacePath,
				ReleaseID:   normalizedReleaseID,
				SnapshotDir: snapshotDir,
				Packets:     packets,
			}); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		}

		return 0
	}

	ticketIDs, resolvedTicketFile, err := resolveTicketIDs(*ticketID, *ticketFile, runtime.parser)
	if err != nil {
		fmt.Fprintf(a.stderr, "manifest failed: %v\n", err)
		return usageExitCode
	}

	repositories, scopeLabel, err := a.resolveCommandRepositories(ctx, scope.WorkspacePath, scope.RepoSpec, runtime)
	if err != nil {
		fmt.Fprintf(a.stderr, "manifest failed: %v\n", err)
		return 1
	}

	environments, resolvedFromBranch, resolvedToBranch, err := resolveOperationContext(ctx, runtime, repositories, defaults.EnvironmentSpec, defaults.FromBranch, defaults.ToBranch)
	if err != nil {
		fmt.Fprintf(a.stderr, "manifest failed: %v\n", err)
		return usageExitCode
	}
	a.rememberProjectMemory(scope, defaults, runtime, repositories, environments, resolvedFromBranch, resolvedToBranch)

	packets := make([]manifestsvc.ReleasePacket, 0, len(ticketIDs))
	for _, ticketID := range ticketIDs {
		promotionPlan, err := runtime.planner.BuildPromotionPlanInRepositories(ctx, repositories, ticketID, resolvedFromBranch, resolvedToBranch, environments)
		if err != nil {
			fmt.Fprintf(a.stderr, "manifest failed: %v\n", err)
			return 1
		}
		packets = append(packets, manifestsvc.BuildReleasePacket(scopeLabel, runtime.loaded, promotionPlan))
	}

	switch selectedFormat {
	case manifestFormatMarkdown:
		if len(packets) == 1 {
			if err := output.RenderReleasePacketMarkdown(a.stdout, packets[0]); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		} else {
			if err := output.RenderReleasePacketBundleMarkdown(a.stdout, packets); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		}
	case manifestFormatJSON:
		if len(packets) == 1 {
			if err := output.RenderJSON(a.stdout, struct {
				Command string                    `json:"command"`
				Packet  manifestsvc.ReleasePacket `json:"packet"`
			}{
				Command: "manifest generate",
				Packet:  packets[0],
			}); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		} else {
			if err := output.RenderJSON(a.stdout, struct {
				Command    string                      `json:"command"`
				Workspace  string                      `json:"workspace"`
				TicketFile string                      `json:"ticketFile,omitempty"`
				Packets    []manifestsvc.ReleasePacket `json:"packets"`
			}{
				Command:    "manifest generate",
				Workspace:  scopeLabel,
				TicketFile: resolvedTicketFile,
				Packets:    packets,
			}); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		}
	}

	return 0
}

func (a *App) runSnapshot(ctx context.Context, args []string) int {
	if len(args) == 0 || hasHelpFlag(args) {
		a.printSnapshotUsage()
		return 0
	}

	switch args[0] {
	case "create":
		return a.runSnapshotCreate(ctx, args[1:])
	default:
		fmt.Fprintf(a.stderr, "unknown snapshot subcommand %q\n\n", args[0])
		a.printSnapshotUsage()
		return usageExitCode
	}
}

func (a *App) runAssist(ctx context.Context, args []string) int {
	if len(args) == 0 || hasHelpFlag(args) {
		a.printAssistUsage()
		return 0
	}

	switch args[0] {
	case "doctor":
		return a.runAssistDoctor(ctx, args[1:])
	case "audit":
		return a.runAssistAudit(ctx, args[1:])
	case "release":
		return a.runAssistRelease(ctx, args[1:])
	case "resolve":
		return a.runAssistResolve(ctx, args[1:])
	case "setup":
		return a.runAssistSetup(ctx, args[1:])
	default:
		fmt.Fprintf(a.stderr, "unknown assist subcommand %q\n\n", args[0])
		a.printAssistUsage()
		return usageExitCode
	}
}

func (a *App) runAssistAudit(ctx context.Context, args []string) int {
	if hasHelpFlag(args) {
		a.printAssistAuditUsage()
		return 0
	}

	fs := flag.NewFlagSet("assist audit", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a repository or workspace")
	configPath := fs.String("config", "", optionalOverrideFileHelp)
	repoTarget := fs.String("repo", "", "Remote repository target, for example github:owner/name or gitlab:group/project")
	workareaName := fs.String("workarea", "", "Saved workarea to use")
	ticketID := fs.String("ticket", "", "Ticket ID to brief")
	fromBranch := fs.String("from", "", "Source branch")
	toBranch := fs.String("to", "", "Target branch")
	envsSpec := fs.String("envs", "", "Comma-separated environment mapping, for example dev=dev,test=test,prod=main")
	deerflowURL := fs.String("url", "", "Base URL for DeerFlow, for example http://localhost:2026")
	mode := fs.String("mode", string(assistsvc.ModePro), "Execution mode: flash, standard, pro, or ultra")
	audience := fs.String("audience", string(assistsvc.AudienceReleaseManager), "Audience: qa, client, or release-manager")
	format := fs.String("format", string(outputFormatHuman), "Output format: human or json")

	if err := fs.Parse(args); err != nil {
		a.printAssistAuditUsage()
		return usageExitCode
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(a.stderr, "assist audit does not accept positional arguments")
		a.printAssistAuditUsage()
		return usageExitCode
	}

	outputFormat, err := parseOutputFormat(*format)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist audit failed: %v\n", err)
		return usageExitCode
	}

	runMode, err := assistsvc.ParseRunMode(*mode)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist audit failed: %v\n", err)
		return usageExitCode
	}
	selectedAudience, err := assistsvc.ParseAudience(*audience)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist audit failed: %v\n", err)
		return usageExitCode
	}

	scope, err := a.resolveCommandScope(*path, *configPath, *repoTarget, *workareaName, flagProvided(fs, "path"), flagProvided(fs, "config"), flagProvided(fs, "repo"))
	if err != nil {
		fmt.Fprintf(a.stderr, "assist audit failed: %v\n", err)
		return 1
	}
	defaults := resolveCommandDefaults(scope.Workarea, *envsSpec, *fromBranch, *toBranch, flagProvided(fs, "envs"), flagProvided(fs, "from"), flagProvided(fs, "to"))
	a.announceWorkareaSelection(scope, defaults)

	remoteMode := strings.TrimSpace(scope.RepoSpec) != ""
	runtime, err := newCommandRuntimeWithOptions(scope.WorkspacePath, scope.ConfigPath, runtimeOptions{DisableAutoConfig: remoteMode})
	if err != nil {
		fmt.Fprintf(a.stderr, "assist audit failed: %v\n", err)
		return 1
	}

	normalizedTicketID := normalizeTicketID(*ticketID)
	if normalizedTicketID == "" {
		fmt.Fprintln(a.stderr, "assist audit failed: --ticket is required")
		a.printAssistAuditUsage()
		return usageExitCode
	}
	if err := runtime.parser.Validate(normalizedTicketID); err != nil {
		fmt.Fprintf(a.stderr, "assist audit failed: %v\n", err)
		return usageExitCode
	}

	repositories, scopeLabel, err := a.resolveCommandRepositories(ctx, scope.WorkspacePath, scope.RepoSpec, runtime)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist audit failed: %v\n", err)
		return 1
	}

	environments, resolvedFromBranch, resolvedToBranch, err := resolveOperationContext(ctx, runtime, repositories, defaults.EnvironmentSpec, defaults.FromBranch, defaults.ToBranch)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist audit failed: %v\n", err)
		return usageExitCode
	}
	a.rememberProjectMemory(scope, defaults, runtime, repositories, environments, resolvedFromBranch, resolvedToBranch)

	commandTarget := fmt.Sprintf("--path %s", shellSingleQuote(scope.WorkspacePath))
	if strings.TrimSpace(scope.RepoSpec) != "" {
		commandTarget = fmt.Sprintf("--repo %s", strings.TrimSpace(scope.RepoSpec))
	}

	result, err := runtime.assistant.Audit(ctx, assistsvc.AuditRequest{
		WorkspacePath: scope.WorkspacePath,
		ScopeLabel:    scopeLabel,
		CommandTarget: commandTarget,
		ConfigPath:    strings.TrimSpace(scope.ConfigPath),
		TicketID:      normalizedTicketID,
		FromBranch:    resolvedFromBranch,
		ToBranch:      resolvedToBranch,
		Environments:  environments,
		Repositories:  repositories,
		LoadedConfig:  runtime.loaded,
	}, assistsvc.ExecuteOptions{
		BaseURL:  *deerflowURL,
		Mode:     runMode,
		Audience: selectedAudience,
	})
	if err != nil {
		fmt.Fprintf(a.stderr, "assist audit failed: %v\n", err)
		return 1
	}

	switch outputFormat {
	case outputFormatHuman:
		if err := output.RenderAssistantAudit(a.stdout, result); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	case outputFormatJSON:
		if err := output.RenderJSON(a.stdout, struct {
			Command string                `json:"command"`
			Scope   string                `json:"scope"`
			Result  assistsvc.AuditResult `json:"result"`
		}{
			Command: "assist audit",
			Scope:   scopeLabel,
			Result:  result,
		}); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	}

	return 0
}

func (a *App) runAssistRelease(ctx context.Context, args []string) int {
	if hasHelpFlag(args) {
		a.printAssistReleaseUsage()
		return 0
	}

	fs := flag.NewFlagSet("assist release", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a repository or workspace")
	configPath := fs.String("config", "", optionalOverrideFileHelp)
	repoTarget := fs.String("repo", "", "Remote repository target, for example github:owner/name or gitlab:group/project")
	workareaName := fs.String("workarea", "", "Saved workarea to use")
	releaseID := fs.String("release", "", "Release ID to brief from saved snapshots")
	ticketID := fs.String("ticket", "", "Ticket ID to include in a live release bundle")
	ticketFile := fs.String("ticket-file", "", "Path to a file with one ticket ID per line for a live release bundle")
	fromBranch := fs.String("from", "", "Source branch")
	toBranch := fs.String("to", "", "Target branch")
	envsSpec := fs.String("envs", "", "Comma-separated environment mapping, for example dev=dev,test=test,prod=main")
	deerflowURL := fs.String("url", "", "Base URL for DeerFlow, for example http://localhost:2026")
	mode := fs.String("mode", string(assistsvc.ModePro), "Execution mode: flash, standard, pro, or ultra")
	audience := fs.String("audience", string(assistsvc.AudienceReleaseManager), "Audience: qa, client, or release-manager")
	format := fs.String("format", string(outputFormatHuman), "Output format: human or json")

	if err := fs.Parse(args); err != nil {
		a.printAssistReleaseUsage()
		return usageExitCode
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(a.stderr, "assist release does not accept positional arguments")
		a.printAssistReleaseUsage()
		return usageExitCode
	}

	outputFormat, err := parseOutputFormat(*format)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist release failed: %v\n", err)
		return usageExitCode
	}

	runMode, err := assistsvc.ParseRunMode(*mode)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist release failed: %v\n", err)
		return usageExitCode
	}
	selectedAudience, err := assistsvc.ParseAudience(*audience)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist release failed: %v\n", err)
		return usageExitCode
	}

	scope, err := a.resolveCommandScope(*path, *configPath, *repoTarget, *workareaName, flagProvided(fs, "path"), flagProvided(fs, "config"), flagProvided(fs, "repo"))
	if err != nil {
		fmt.Fprintf(a.stderr, "assist release failed: %v\n", err)
		return 1
	}
	defaults := resolveCommandDefaults(scope.Workarea, *envsSpec, *fromBranch, *toBranch, flagProvided(fs, "envs"), flagProvided(fs, "from"), flagProvided(fs, "to"))
	a.announceWorkareaSelection(scope, defaults)

	remoteMode := strings.TrimSpace(scope.RepoSpec) != ""
	runtime, err := newCommandRuntimeWithOptions(scope.WorkspacePath, scope.ConfigPath, runtimeOptions{DisableAutoConfig: remoteMode})
	if err != nil {
		fmt.Fprintf(a.stderr, "assist release failed: %v\n", err)
		return 1
	}

	normalizedReleaseID := strings.TrimSpace(*releaseID)
	if normalizedReleaseID == "" {
		fmt.Fprintln(a.stderr, "assist release failed: --release is required")
		a.printAssistReleaseUsage()
		return usageExitCode
	}
	normalizedReleaseID, err = snapshotsvc.NormalizeReleaseID(normalizedReleaseID)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist release failed: %v\n", err)
		return usageExitCode
	}

	var (
		ticketIDs          []string
		scopeLabel         = scope.WorkspacePath
		commandTarget      = fmt.Sprintf("--path %s", shellSingleQuote(scope.WorkspacePath))
		environments       []inspectsvc.Environment
		resolvedFromBranch string
		resolvedToBranch   string
		repositories       []scm.Repository
	)

	if strings.TrimSpace(*ticketID) != "" || strings.TrimSpace(*ticketFile) != "" {
		ticketIDs, _, err = resolveTicketIDs(*ticketID, *ticketFile, runtime.parser)
		if err != nil {
			fmt.Fprintf(a.stderr, "assist release failed: %v\n", err)
			return usageExitCode
		}

		repositories, scopeLabel, err = a.resolveCommandRepositories(ctx, scope.WorkspacePath, scope.RepoSpec, runtime)
		if err != nil {
			fmt.Fprintf(a.stderr, "assist release failed: %v\n", err)
			return 1
		}

		environments, resolvedFromBranch, resolvedToBranch, err = resolveOperationContext(ctx, runtime, repositories, defaults.EnvironmentSpec, defaults.FromBranch, defaults.ToBranch)
		if err != nil {
			fmt.Fprintf(a.stderr, "assist release failed: %v\n", err)
			return usageExitCode
		}
		a.rememberProjectMemory(scope, defaults, runtime, repositories, environments, resolvedFromBranch, resolvedToBranch)

		if strings.TrimSpace(scope.RepoSpec) != "" {
			commandTarget = fmt.Sprintf("--repo %s", strings.TrimSpace(scope.RepoSpec))
		}
	} else {
		if strings.TrimSpace(scope.RepoSpec) != "" && !scope.RepoInherited {
			fmt.Fprintln(a.stderr, "assist release failed: --repo requires --ticket or --ticket-file")
			a.printAssistReleaseUsage()
			return usageExitCode
		}
		if strings.TrimSpace(*fromBranch) != "" || strings.TrimSpace(*toBranch) != "" || strings.TrimSpace(*envsSpec) != "" {
			fmt.Fprintln(a.stderr, "assist release failed: --from, --to, and --envs are only used with --ticket or --ticket-file")
			a.printAssistReleaseUsage()
			return usageExitCode
		}
	}

	result, err := runtime.assistant.Release(ctx, assistsvc.ReleaseRequest{
		WorkspacePath: scope.WorkspacePath,
		ScopeLabel:    scopeLabel,
		CommandTarget: commandTarget,
		ConfigPath:    strings.TrimSpace(scope.ConfigPath),
		ReleaseID:     normalizedReleaseID,
		TicketIDs:     ticketIDs,
		FromBranch:    resolvedFromBranch,
		ToBranch:      resolvedToBranch,
		Environments:  environments,
		Repositories:  repositories,
		LoadedConfig:  runtime.loaded,
	}, assistsvc.ExecuteOptions{
		BaseURL:  *deerflowURL,
		Mode:     runMode,
		Audience: selectedAudience,
	})
	if err != nil {
		fmt.Fprintf(a.stderr, "assist release failed: %v\n", err)
		return 1
	}

	switch outputFormat {
	case outputFormatHuman:
		if err := output.RenderAssistantRelease(a.stdout, result); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	case outputFormatJSON:
		if err := output.RenderJSON(a.stdout, struct {
			Command string                  `json:"command"`
			Scope   string                  `json:"scope"`
			Result  assistsvc.ReleaseResult `json:"result"`
		}{
			Command: "assist release",
			Scope:   result.Bundle.ScopeLabel,
			Result:  result,
		}); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	}

	return 0
}

func (a *App) runLogin(ctx context.Context, args []string) int {
	if len(args) == 0 || hasHelpFlag(args) {
		a.printLoginUsage()
		return 0
	}
	if len(args) != 1 {
		fmt.Fprintln(a.stderr, "login requires exactly one <provider> argument")
		a.printLoginUsage()
		return usageExitCode
	}

	provider, err := sourcecontrol.ParseProvider(args[0])
	if err != nil {
		fmt.Fprintf(a.stderr, "login failed: %v\n", err)
		a.printLoginUsage()
		return usageExitCode
	}

	if err := sourcecontrol.Login(ctx, provider, a.stdin, a.stdout, a.stderr); err != nil {
		fmt.Fprintf(a.stderr, "login failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(a.stdout, "%s authentication is ready.\n", sourcecontrol.ProviderLabel(provider))
	return 0
}

func (a *App) runSnapshotCreate(ctx context.Context, args []string) int {
	if hasHelpFlag(args) {
		a.printSnapshotCreateUsage()
		return 0
	}

	fs := flag.NewFlagSet("snapshot create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a repository or workspace")
	configPath := fs.String("config", "", optionalOverrideFileHelp)
	repoTarget := fs.String("repo", "", "Remote repository target, for example github:owner/name or gitlab:group/project")
	workareaName := fs.String("workarea", "", "Saved workarea to use")
	ticketID := fs.String("ticket", "", "Ticket ID to capture")
	fromBranch := fs.String("from", "", "Source branch")
	toBranch := fs.String("to", "", "Target branch")
	releaseID := fs.String("release", "", "Release ID to attach this snapshot to")
	envsSpec := fs.String("envs", "", "Comma-separated environment mapping, for example dev=dev,test=test,prod=main")
	format := fs.String("format", string(outputFormatHuman), "Output format: human or json")
	outputPath := fs.String("output", "", "Write the snapshot JSON to this path")

	if err := fs.Parse(args); err != nil {
		a.printSnapshotCreateUsage()
		return usageExitCode
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(a.stderr, "snapshot create does not accept positional arguments")
		a.printSnapshotCreateUsage()
		return usageExitCode
	}

	selectedFormat, err := parseOutputFormat(*format)
	if err != nil {
		fmt.Fprintf(a.stderr, "snapshot create failed: %v\n", err)
		return usageExitCode
	}

	scope, err := a.resolveCommandScope(*path, *configPath, *repoTarget, *workareaName, flagProvided(fs, "path"), flagProvided(fs, "config"), flagProvided(fs, "repo"))
	if err != nil {
		fmt.Fprintf(a.stderr, "snapshot create failed: %v\n", err)
		return 1
	}
	defaults := resolveCommandDefaults(scope.Workarea, *envsSpec, *fromBranch, *toBranch, flagProvided(fs, "envs"), flagProvided(fs, "from"), flagProvided(fs, "to"))
	a.announceWorkareaSelection(scope, defaults)

	remoteMode := strings.TrimSpace(scope.RepoSpec) != ""
	runtime, err := newCommandRuntimeWithOptions(scope.WorkspacePath, scope.ConfigPath, runtimeOptions{DisableAutoConfig: remoteMode})
	if err != nil {
		fmt.Fprintf(a.stderr, "snapshot create failed: %v\n", err)
		return 1
	}

	normalizedTicketID := normalizeTicketID(*ticketID)
	if normalizedTicketID == "" {
		fmt.Fprintln(a.stderr, "snapshot create failed: --ticket is required")
		a.printSnapshotCreateUsage()
		return usageExitCode
	}
	if err := runtime.parser.Validate(normalizedTicketID); err != nil {
		fmt.Fprintf(a.stderr, "snapshot create failed: %v\n", err)
		return usageExitCode
	}
	if strings.TrimSpace(defaults.FromBranch) == "" || strings.TrimSpace(defaults.ToBranch) == "" {
		if !remoteMode {
			fmt.Fprintln(a.stderr, "snapshot create failed: both --from and --to branches are required")
			a.printSnapshotCreateUsage()
			return usageExitCode
		}
	}

	repositories, scopeLabel, err := a.resolveCommandRepositories(ctx, scope.WorkspacePath, scope.RepoSpec, runtime)
	if err != nil {
		fmt.Fprintf(a.stderr, "snapshot create failed: %v\n", err)
		return 1
	}

	environments, resolvedFromBranch, resolvedToBranch, err := resolveOperationContext(ctx, runtime, repositories, defaults.EnvironmentSpec, defaults.FromBranch, defaults.ToBranch)
	if err != nil {
		fmt.Fprintf(a.stderr, "snapshot create failed: %v\n", err)
		return usageExitCode
	}
	a.rememberProjectMemory(scope, defaults, runtime, repositories, environments, resolvedFromBranch, resolvedToBranch)

	normalizedReleaseID := ""
	if strings.TrimSpace(*releaseID) != "" {
		normalizedReleaseID, err = snapshotsvc.NormalizeReleaseID(*releaseID)
		if err != nil {
			fmt.Fprintf(a.stderr, "snapshot create failed: %v\n", err)
			return usageExitCode
		}
	}

	var snapshot snapshotsvc.TicketSnapshot
	if remoteMode {
		snapshot, err = runtime.snapshot.CaptureInRepositoriesWithOptions(ctx, scopeLabel, runtime.loaded, repositories, normalizedTicketID, resolvedFromBranch, resolvedToBranch, environments, snapshotsvc.CaptureOptions{
			ReleaseID: normalizedReleaseID,
		})
	} else {
		snapshot, err = runtime.snapshot.CaptureWithOptions(ctx, scope.WorkspacePath, runtime.loaded, normalizedTicketID, resolvedFromBranch, resolvedToBranch, environments, snapshotsvc.CaptureOptions{
			ReleaseID: normalizedReleaseID,
		})
	}
	if err != nil {
		fmt.Fprintf(a.stderr, "snapshot create failed: %v\n", err)
		return 1
	}

	resolvedOutputPath := ""
	switch {
	case strings.TrimSpace(*outputPath) != "":
		resolvedOutputPath, err = normalizeCLIPath(*outputPath)
		if err != nil {
			fmt.Fprintf(a.stderr, "snapshot create failed: %v\n", err)
			return 1
		}
	case normalizedReleaseID != "":
		resolvedOutputPath = snapshotsvc.DefaultReleaseSnapshotPath(scope.WorkspacePath, normalizedReleaseID, normalizedTicketID)
	}
	if resolvedOutputPath != "" {
		if err := writeJSONFile(resolvedOutputPath, snapshot); err != nil {
			fmt.Fprintf(a.stderr, "snapshot create failed: %v\n", err)
			return 1
		}
	}

	switch selectedFormat {
	case outputFormatHuman:
		if err := output.RenderSnapshot(a.stdout, snapshot, resolvedOutputPath); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	case outputFormatJSON:
		if err := output.RenderJSON(a.stdout, struct {
			Command  string                     `json:"command"`
			Output   string                     `json:"output,omitempty"`
			Snapshot snapshotsvc.TicketSnapshot `json:"snapshot"`
		}{
			Command:  "snapshot create",
			Output:   resolvedOutputPath,
			Snapshot: snapshot,
		}); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	}

	return 0
}

func (a *App) runDoctor(ctx context.Context, args []string) int {
	if hasHelpFlag(args) {
		a.printDoctorUsage()
		return 0
	}

	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a repository or workspace")
	configPath := fs.String("config", "", optionalOverrideFileHelp)
	format := fs.String("format", string(outputFormatHuman), "Output format: human or json")

	if err := fs.Parse(args); err != nil {
		a.printDoctorUsage()
		return usageExitCode
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(a.stderr, "doctor does not accept positional arguments")
		a.printDoctorUsage()
		return usageExitCode
	}

	outputFormat, err := parseOutputFormat(*format)
	if err != nil {
		fmt.Fprintf(a.stderr, "doctor failed: %v\n", err)
		return usageExitCode
	}

	resolvedPath, err := normalizeCLIPath(*path)
	if err != nil {
		fmt.Fprintf(a.stderr, "doctor failed: %v\n", err)
		return 1
	}

	runtime, err := newCommandRuntime(resolvedPath, *configPath)
	if err != nil {
		fmt.Fprintf(a.stderr, "doctor failed: %v\n", err)
		return 1
	}

	report, err := runtime.doctor.Run(ctx, resolvedPath, runtime.loaded)
	if err != nil {
		fmt.Fprintf(a.stderr, "doctor failed: %v\n", err)
		return 1
	}

	switch outputFormat {
	case outputFormatHuman:
		if err := output.RenderDoctor(a.stdout, report); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	case outputFormatJSON:
		if err := output.RenderJSON(a.stdout, struct {
			Command string           `json:"command"`
			Report  doctorsvc.Report `json:"report"`
		}{
			Command: "doctor",
			Report:  report,
		}); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	}

	return 0
}

func (a *App) runAssistResolve(ctx context.Context, args []string) int {
	if hasHelpFlag(args) {
		a.printAssistResolveUsage()
		return 0
	}

	fs := flag.NewFlagSet("assist resolve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a Git repository or a child path inside it")
	configPath := fs.String("config", "", optionalOverrideFileHelp)
	ticketID := fs.String("ticket", "", "Optional ticket ID used for scope warnings")
	deerflowURL := fs.String("url", "", "Base URL for DeerFlow, for example http://localhost:2026")
	mode := fs.String("mode", string(assistsvc.ModePro), "Execution mode: flash, standard, pro, or ultra")
	audience := fs.String("audience", string(assistsvc.AudienceReleaseManager), "Audience: qa, client, or release-manager")
	format := fs.String("format", string(outputFormatHuman), "Output format: human or json")

	if err := fs.Parse(args); err != nil {
		a.printAssistResolveUsage()
		return usageExitCode
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(a.stderr, "assist resolve does not accept positional arguments")
		a.printAssistResolveUsage()
		return usageExitCode
	}

	outputFormat, err := parseOutputFormat(*format)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist resolve failed: %v\n", err)
		return usageExitCode
	}

	runMode, err := assistsvc.ParseRunMode(*mode)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist resolve failed: %v\n", err)
		return usageExitCode
	}
	selectedAudience, err := assistsvc.ParseAudience(*audience)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist resolve failed: %v\n", err)
		return usageExitCode
	}

	resolvedPath, err := normalizeCLIPath(*path)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist resolve failed: %v\n", err)
		return 1
	}

	runtime, err := newCommandRuntime(resolvedPath, *configPath)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist resolve failed: %v\n", err)
		return 1
	}

	if strings.TrimSpace(*ticketID) != "" {
		if err := runtime.parser.Validate(*ticketID); err != nil {
			fmt.Fprintf(a.stderr, "assist resolve failed: %v\n", err)
			return usageExitCode
		}
	}

	result, err := runtime.assistant.Resolve(ctx, assistsvc.ResolveRequest{
		WorkspacePath: resolvedPath,
		ScopeLabel:    resolvedPath,
		CommandTarget: fmt.Sprintf("--path %s", shellSingleQuote(resolvedPath)),
		ConfigPath:    strings.TrimSpace(*configPath),
		TicketID:      strings.TrimSpace(*ticketID),
	}, assistsvc.ExecuteOptions{
		BaseURL:  *deerflowURL,
		Mode:     runMode,
		Audience: selectedAudience,
	})
	if err != nil {
		if errors.Is(err, conflictsvc.ErrNoConflict) {
			fmt.Fprintln(a.stderr, "assist resolve failed: no active Git conflict state was found")
			return 1
		}
		fmt.Fprintf(a.stderr, "assist resolve failed: %v\n", err)
		return 1
	}

	switch outputFormat {
	case outputFormatHuman:
		if err := output.RenderAssistantResolve(a.stdout, result); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	case outputFormatJSON:
		if err := output.RenderJSON(a.stdout, struct {
			Command string                  `json:"command"`
			Scope   string                  `json:"scope"`
			Result  assistsvc.ResolveResult `json:"result"`
		}{
			Command: "assist resolve",
			Scope:   result.Bundle.ScopeLabel,
			Result:  result,
		}); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	}

	return 0
}

func (a *App) runResolve(ctx context.Context, args []string) int {
	if len(args) == 0 || hasHelpFlag(args) {
		a.printResolveUsage()
		return 0
	}

	switch args[0] {
	case "status":
		return a.runResolveStatus(ctx, args[1:])
	case "start":
		return a.runResolveStart(ctx, args[1:])
	default:
		fmt.Fprintf(a.stderr, "unknown resolve subcommand %q\n\n", args[0])
		a.printResolveUsage()
		return usageExitCode
	}
}

func (a *App) runResolveStatus(ctx context.Context, args []string) int {
	if hasHelpFlag(args) {
		a.printResolveStatusUsage()
		return 0
	}

	fs := flag.NewFlagSet("resolve status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a Git repository or a child path inside it")
	configPath := fs.String("config", "", optionalOverrideFileHelp)
	ticketID := fs.String("ticket", "", "Optional ticket ID used for scope warnings")
	format := fs.String("format", string(outputFormatHuman), "Output format: human or json")
	if err := fs.Parse(args); err != nil {
		a.printResolveStatusUsage()
		return usageExitCode
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(a.stderr, "resolve status does not accept positional arguments")
		a.printResolveStatusUsage()
		return usageExitCode
	}

	outputFormat, err := parseOutputFormat(*format)
	if err != nil {
		fmt.Fprintf(a.stderr, "resolve status failed: %v\n", err)
		return usageExitCode
	}

	resolvedPath, err := normalizeCLIPath(*path)
	if err != nil {
		fmt.Fprintf(a.stderr, "resolve status failed: %v\n", err)
		return 1
	}

	runtime, err := newCommandRuntime(resolvedPath, *configPath)
	if err != nil {
		fmt.Fprintf(a.stderr, "resolve status failed: %v\n", err)
		return 1
	}

	status, err := runtime.conflicts.Status(ctx, resolvedPath, *ticketID)
	if err != nil {
		if errors.Is(err, conflictsvc.ErrNoConflict) {
			fmt.Fprintln(a.stdout, "No active Git conflict state was found.")
			return 0
		}
		fmt.Fprintf(a.stderr, "resolve status failed: %v\n", err)
		return 1
	}

	switch outputFormat {
	case outputFormatHuman:
		if err := output.RenderConflictStatus(a.stdout, status); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	case outputFormatJSON:
		if err := output.RenderJSON(a.stdout, struct {
			Command string             `json:"command"`
			Status  conflictsvc.Status `json:"status"`
		}{
			Command: "resolve status",
			Status:  status,
		}); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	}

	return 0
}

func (a *App) runResolveStart(ctx context.Context, args []string) int {
	if hasHelpFlag(args) {
		a.printResolveStartUsage()
		return 0
	}

	fs := flag.NewFlagSet("resolve start", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a Git repository or a child path inside it")
	configPath := fs.String("config", "", optionalOverrideFileHelp)
	ticketID := fs.String("ticket", "", "Optional ticket ID used for scope warnings")
	if err := fs.Parse(args); err != nil {
		a.printResolveStartUsage()
		return usageExitCode
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(a.stderr, "resolve start does not accept positional arguments")
		a.printResolveStartUsage()
		return usageExitCode
	}

	resolvedPath, err := normalizeCLIPath(*path)
	if err != nil {
		fmt.Fprintf(a.stderr, "resolve start failed: %v\n", err)
		return 1
	}

	runtime, err := newCommandRuntime(resolvedPath, *configPath)
	if err != nil {
		fmt.Fprintf(a.stderr, "resolve start failed: %v\n", err)
		return 1
	}

	if err := runtime.conflicts.RunInteractive(ctx, resolvedPath, *ticketID, conflictsvc.InteractiveOptions{
		Stdin:  a.stdin,
		Stdout: a.stdout,
		Stderr: a.stderr,
	}); err != nil {
		if errors.Is(err, conflictsvc.ErrNoConflict) {
			fmt.Fprintln(a.stderr, "resolve start failed: no active Git conflict state was found")
			return 1
		}
		fmt.Fprintf(a.stderr, "resolve start failed: %v\n", err)
		return 1
	}

	return 0
}

func (a *App) runUpdate(ctx context.Context, args []string) int {
	if hasHelpFlag(args) {
		a.printUpdateUsage()
		return 0
	}

	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	versionFlag := fs.String("version", "latest", "Install the latest release or a specific tag")
	repoFlag := fs.String("repo", "", "GitHub repo that hosts gig releases")
	installDirFlag := fs.String("install-dir", "", "Override the install directory for a direct install")

	if err := fs.Parse(args); err != nil {
		a.printUpdateUsage()
		return usageExitCode
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(a.stderr, "update accepts at most one positional <version> argument")
		a.printUpdateUsage()
		return usageExitCode
	}

	version := updatesvc.NormalizeVersion(*versionFlag)
	if fs.NArg() == 1 {
		if version != "latest" {
			fmt.Fprintln(a.stderr, "update accepts either --version or a positional <version>, not both")
			a.printUpdateUsage()
			return usageExitCode
		}
		version = updatesvc.NormalizeVersion(fs.Arg(0))
	}

	repoName := strings.TrimSpace(*repoFlag)
	if repoName == "" {
		repoName = strings.TrimSpace(os.Getenv("GIG_REPO"))
	}
	if repoName == "" {
		repoName = "phamhungptithcm/gig"
	}

	executablePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	resolvedExecutablePath := executablePath
	if linkedPath, linkErr := filepath.EvalSymlinks(executablePath); linkErr == nil {
		resolvedExecutablePath = linkedPath
	}

	installDir := strings.TrimSpace(*installDirFlag)
	installMode := updatesvc.DetectInstallMode(resolvedExecutablePath, os.LookupEnv)
	if installDir != "" {
		installMode = updatesvc.ModeDirect
	} else {
		installDir = filepath.Dir(resolvedExecutablePath)
	}

	switch installMode {
	case updatesvc.ModeNPM:
		fmt.Fprintf(a.stdout, "Detected an npm-managed install at %s\n", resolvedExecutablePath)
		return a.runNPMUpdate(ctx, updatesvc.ResolveNPMPackageName(os.LookupEnv), version)
	case updatesvc.ModeHomebrew:
		fmt.Fprintf(a.stderr, "update failed: Homebrew installs are no longer published for gig. Reinstall with `npm install -g %s` or use the direct installer.\n", updatesvc.DefaultNPMPackageName)
		return 1
	case updatesvc.ModeScoop:
		fmt.Fprintf(a.stderr, "update failed: Scoop installs are no longer published for gig. Reinstall with `npm install -g %s` or use the direct installer.\n", updatesvc.DefaultNPMPackageName)
		return 1
	default:
		if runtime.GOOS == "windows" {
			return a.runWindowsInstallerUpdate(ctx, repoName, version, installDir)
		}
		return a.runPOSIXInstallerUpdate(ctx, repoName, version, installDir)
	}
}

func (a *App) runNPMUpdate(ctx context.Context, packageName, releaseVersion string) int {
	npmVersion, err := updatesvc.NormalizeNPMVersion(releaseVersion)
	if err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	spec := packageName + "@latest"
	if npmVersion != "latest" {
		spec = packageName + "@" + npmVersion
	}

	if runtime.GOOS == "windows" {
		return a.runWindowsNPMUpdate(ctx, spec)
	}

	return a.runExternalCommand(ctx, "npm", []string{"install", "-g", spec})
}

func (a *App) runExternalCommand(ctx context.Context, name string, args []string) int {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = a.stdout
	cmd.Stderr = a.stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	return 0
}

func (a *App) runPOSIXInstallerUpdate(ctx context.Context, repoName, version, installDir string) int {
	installerURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/scripts/install.sh", repoName)
	command := fmt.Sprintf(
		"if command -v curl >/dev/null 2>&1; then curl -fsSL %s | sh; elif command -v wget >/dev/null 2>&1; then wget -qO- %s | sh; else echo 'curl or wget is required to update gig.' >&2; exit 1; fi",
		shellSingleQuote(installerURL),
		shellSingleQuote(installerURL),
	)

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = a.stdout
	cmd.Stderr = a.stderr
	cmd.Env = append(os.Environ(),
		"GIG_REPO="+repoName,
		"GIG_VERSION="+version,
		"GIG_INSTALL_DIR="+installDir,
	)

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	return 0
}

func (a *App) runWindowsInstallerUpdate(ctx context.Context, repoName, version, installDir string) int {
	installerURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/scripts/install.ps1", repoName)

	scriptFile, err := os.CreateTemp("", "gig-update-*.ps1")
	if err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	scriptBody := fmt.Sprintf(`$ErrorActionPreference = "Stop"
$installer = [ScriptBlock]::Create((Invoke-RestMethod -Uri '%s'))
& $installer -Repo '%s' -Version '%s' -InstallDir '%s' -WaitForPid %d
Remove-Item -LiteralPath $PSCommandPath -Force -ErrorAction SilentlyContinue
`,
		powerShellSingleQuote(installerURL),
		powerShellSingleQuote(repoName),
		powerShellSingleQuote(version),
		powerShellSingleQuote(installDir),
		os.Getpid(),
	)

	if _, err := scriptFile.WriteString(scriptBody); err != nil {
		scriptFile.Close()
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}
	if err := scriptFile.Close(); err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	command := fmt.Sprintf(
		"Start-Process powershell -WindowStyle Hidden -ArgumentList @('-NoProfile','-ExecutionPolicy','Bypass','-File','%s')",
		powerShellSingleQuote(scriptFile.Name()),
	)

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = a.stdout
	cmd.Stderr = a.stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	fmt.Fprintln(a.stdout, "gig update started in the background.")
	fmt.Fprintln(a.stdout, "Open a new terminal in a few seconds, then run: gig version")
	return 0
}

func (a *App) runWindowsNPMUpdate(ctx context.Context, packageSpec string) int {
	scriptFile, err := os.CreateTemp("", "gig-update-npm-*.ps1")
	if err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	scriptBody := fmt.Sprintf(`$ErrorActionPreference = "Stop"
for ($attempt = 0; $attempt -lt 240; $attempt++) {
	if (-not (Get-Process -Id %d -ErrorAction SilentlyContinue)) {
		break
	}
	Start-Sleep -Milliseconds 500
}
npm.cmd install -g '%s'
Remove-Item -LiteralPath $PSCommandPath -Force -ErrorAction SilentlyContinue
`,
		os.Getpid(),
		powerShellSingleQuote(packageSpec),
	)

	if _, err := scriptFile.WriteString(scriptBody); err != nil {
		scriptFile.Close()
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}
	if err := scriptFile.Close(); err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	command := fmt.Sprintf(
		"Start-Process powershell -WindowStyle Hidden -ArgumentList @('-NoProfile','-ExecutionPolicy','Bypass','-File','%s')",
		powerShellSingleQuote(scriptFile.Name()),
	)

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = a.stdout
	cmd.Stderr = a.stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(a.stderr, "update failed: %v\n", err)
		return 1
	}

	fmt.Fprintln(a.stdout, "gig update started in the background.")
	fmt.Fprintln(a.stdout, "Open a new terminal in a few seconds, then run: gig version")
	return 0
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func powerShellSingleQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
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
	)
	printHelpCommands(a.stderr, "Core workflows",
		"gig ABC-123",
		"gig verify ABC-123",
		"gig manifest ABC-123",
		"gig ABC-123 --path .",
	)
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
	fmt.Fprintln(a.stderr, "  gig assist release --release rel-2026-04-09 [--path . | --ticket-file tickets.txt --repo github:owner/name]")
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
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}

	return false
}

func normalizeCLIPath(path string) (string, error) {
	if path == "" {
		path = "."
	}

	return filepath.Abs(path)
}

func normalizeTicketID(ticketID string) string {
	return strings.ToUpper(strings.TrimSpace(ticketID))
}

func looksLikeTicketID(ticketID string) bool {
	parser, err := ticketsvc.NewParser(config.Default().TicketPattern)
	if err != nil {
		return false
	}
	return parser.Validate(normalizeTicketID(ticketID)) == nil
}

func resolveTicketIDs(ticketID, ticketFile string, parser ticketsvc.Parser) ([]string, string, error) {
	normalizedTicketID := normalizeTicketID(ticketID)
	ticketFile = strings.TrimSpace(ticketFile)

	switch {
	case normalizedTicketID != "" && ticketFile != "":
		return nil, "", fmt.Errorf("use either --ticket or --ticket-file, not both")
	case normalizedTicketID == "" && ticketFile == "":
		return nil, "", fmt.Errorf("either --ticket or --ticket-file is required")
	case normalizedTicketID != "":
		if err := parser.Validate(normalizedTicketID); err != nil {
			return nil, "", err
		}
		return []string{normalizedTicketID}, "", nil
	default:
		resolvedTicketFile, err := normalizeCLIPath(ticketFile)
		if err != nil {
			return nil, "", err
		}
		ticketIDs, err := readTicketIDsFromFile(resolvedTicketFile, parser)
		if err != nil {
			return nil, "", err
		}
		return ticketIDs, resolvedTicketFile, nil
	}
}

func readTicketIDsFromFile(path string, parser ticketsvc.Parser) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	seen := make(map[string]struct{})
	ticketIDs := make([]string, 0, 8)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		ticketID := normalizeTicketID(line)
		if err := parser.Validate(ticketID); err != nil {
			return nil, fmt.Errorf("ticket file %s line %d: %w", path, lineNumber, err)
		}
		if _, ok := seen[ticketID]; ok {
			continue
		}
		seen[ticketID] = struct{}{}
		ticketIDs = append(ticketIDs, ticketID)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(ticketIDs) == 0 {
		return nil, fmt.Errorf("ticket file %s did not contain any ticket IDs", path)
	}

	return ticketIDs, nil
}

func (a *App) runVersion() int {
	fmt.Fprintln(a.stdout, buildinfo.Summary())
	fmt.Fprintf(a.stdout, "commit: %s\n", buildinfo.Commit)
	fmt.Fprintf(a.stdout, "built: %s\n", buildinfo.Date)
	return 0
}

const defaultEnvironmentSpec = "dev=dev,test=test,prod=main"

type outputFormat string

const (
	outputFormatHuman outputFormat = "human"
	outputFormatJSON  outputFormat = "json"
)

type manifestFormat string

const (
	manifestFormatMarkdown manifestFormat = "markdown"
	manifestFormatJSON     manifestFormat = "json"
)

func parseOutputFormat(raw string) (outputFormat, error) {
	switch outputFormat(strings.ToLower(strings.TrimSpace(raw))) {
	case outputFormatHuman:
		return outputFormatHuman, nil
	case outputFormatJSON:
		return outputFormatJSON, nil
	default:
		return "", fmt.Errorf("unsupported format %q", raw)
	}
}

func parseManifestFormat(raw string) (manifestFormat, error) {
	switch manifestFormat(strings.ToLower(strings.TrimSpace(raw))) {
	case manifestFormatMarkdown:
		return manifestFormatMarkdown, nil
	case manifestFormatJSON:
		return manifestFormatJSON, nil
	default:
		return "", fmt.Errorf("unsupported format %q", raw)
	}
}

func parseEnvironmentSpec(spec string) ([]inspectsvc.Environment, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, fmt.Errorf("at least one environment mapping is required")
	}

	parts := strings.Split(spec, ",")
	environments := make([]inspectsvc.Environment, 0, len(parts))
	seen := map[string]struct{}{}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		name := part
		branch := part
		if strings.Contains(part, "=") {
			fields := strings.SplitN(part, "=", 2)
			name = strings.TrimSpace(fields[0])
			branch = strings.TrimSpace(fields[1])
		}
		if name == "" || branch == "" {
			return nil, fmt.Errorf("invalid environment mapping %q", part)
		}
		if _, ok := seen[name]; ok {
			return nil, fmt.Errorf("duplicate environment name %q", name)
		}
		seen[name] = struct{}{}
		environments = append(environments, inspectsvc.Environment{
			Name:   name,
			Branch: branch,
		})
	}

	if len(environments) == 0 {
		return nil, fmt.Errorf("at least one environment mapping is required")
	}

	return environments, nil
}

func resolveEnvironments(spec string, loaded config.Loaded) ([]inspectsvc.Environment, error) {
	if strings.TrimSpace(spec) != "" {
		return parseEnvironmentSpec(spec)
	}

	environments := make([]inspectsvc.Environment, 0, len(loaded.Config.Environments))
	for _, environment := range loaded.Config.Environments {
		environments = append(environments, inspectsvc.Environment{
			Name:   environment.Name,
			Branch: environment.Branch,
		})
	}
	if len(environments) == 0 {
		return nil, fmt.Errorf("at least one environment mapping is required")
	}
	return environments, nil
}

func (a *App) resolveCommandRepositories(ctx context.Context, workspacePath, repoSpec string, runtime commandRuntime) ([]scm.Repository, string, error) {
	if strings.TrimSpace(repoSpec) == "" {
		repositories, err := runtime.scanner.Discover(ctx, workspacePath)
		return repositories, workspacePath, err
	}

	repositories, err := sourcecontrol.ParseRepositoryTargets(repoSpec)
	if err != nil {
		return nil, "", err
	}
	if err := sourcecontrol.ValidateRemoteAuditSupport(repositories, runtime.adapters); err != nil {
		return nil, "", err
	}
	if err := a.ensureSourceControlAccess(ctx, repositories); err != nil {
		return nil, "", err
	}

	return repositories, sourcecontrol.FormatScopeLabel(repositories, workspacePath), nil
}

func (a *App) ensureSourceControlAccess(ctx context.Context, repositories []scm.Repository) error {
	return sourcecontrol.EnsureAccess(ctx, repositories, a.stdin, a.stdout, a.stderr)
}

func resolveOperationContext(ctx context.Context, runtime commandRuntime, repositories []scm.Repository, envSpec, fromBranch, toBranch string) ([]inspectsvc.Environment, string, string, error) {
	environments, err := resolveOperationEnvironments(ctx, runtime, repositories, envSpec)
	if err != nil {
		return nil, "", "", err
	}

	fromBranch = strings.TrimSpace(fromBranch)
	toBranch = strings.TrimSpace(toBranch)
	if fromBranch != "" && toBranch != "" {
		return environments, fromBranch, toBranch, nil
	}

	if !containsRemoteRepositories(repositories) {
		return nil, "", "", fmt.Errorf("both --from and --to branches are required")
	}

	inferredFrom, inferredTo, err := sourcecontrol.InferPromotionBranches(environments, fromBranch, toBranch)
	if err != nil {
		return nil, "", "", err
	}

	return environments, inferredFrom, inferredTo, nil
}

func resolveOperationEnvironments(ctx context.Context, runtime commandRuntime, repositories []scm.Repository, spec string) ([]inspectsvc.Environment, error) {
	if strings.TrimSpace(spec) != "" {
		return parseEnvironmentSpec(spec)
	}

	if runtime.loaded.ExplicitEnvironments || !containsRemoteRepositories(repositories) {
		return resolveEnvironments("", runtime.loaded)
	}

	protectedBranches, err := protectedBranchesForRepositories(ctx, runtime, repositories)
	if err != nil {
		return nil, err
	}
	if len(protectedBranches) == 0 {
		return resolveEnvironments("", runtime.loaded)
	}

	environments := sourcecontrol.InferEnvironments(protectedBranches)
	if len(environments) == 0 {
		return nil, fmt.Errorf("unable to infer protected branch topology for the selected remote repository")
	}

	return environments, nil
}

func protectedBranchesForRepositories(ctx context.Context, runtime commandRuntime, repositories []scm.Repository) ([]string, error) {
	branches := make([]string, 0)
	seen := map[string]struct{}{}

	for _, repository := range repositories {
		adapter, ok := runtime.adapters.For(repository.Type)
		if !ok {
			continue
		}
		provider, ok := adapter.(scm.ProtectedBranchProvider)
		if !ok {
			continue
		}
		protectedBranches, err := provider.ProtectedBranches(ctx, repository.Root)
		if err != nil {
			return nil, err
		}
		for _, branch := range protectedBranches {
			if _, ok := seen[branch]; ok {
				continue
			}
			seen[branch] = struct{}{}
			branches = append(branches, branch)
		}
	}

	return branches, nil
}

func containsRemoteRepositories(repositories []scm.Repository) bool {
	for _, repository := range repositories {
		if repository.Type.IsRemote() {
			return true
		}
	}
	return false
}

type commandRuntime struct {
	loaded    config.Loaded
	parser    ticketsvc.Parser
	adapters  *scm.Registry
	scanner   *repo.Scanner
	conflicts *conflictsvc.Service
	finder    *ticketsvc.Service
	diff      *diffsvc.Service
	inspect   *inspectsvc.Service
	planner   *plansvc.Service
	snapshot  *snapshotsvc.Service
	doctor    *doctorsvc.Service
	manifest  *manifestsvc.Service
	assistant *assistsvc.Service
}

func newCommandRuntime(path, configPath string) (commandRuntime, error) {
	return newCommandRuntimeWithOptions(path, configPath, runtimeOptions{})
}

type runtimeOptions struct {
	DisableAutoConfig bool
}

func newCommandRuntimeWithOptions(path, configPath string, options runtimeOptions) (commandRuntime, error) {
	loaded, err := loadRuntimeConfig(path, configPath, options)
	if err != nil {
		return commandRuntime{}, err
	}

	parser, err := ticketsvc.NewParser(loaded.Config.TicketPattern)
	if err != nil {
		return commandRuntime{}, err
	}

	registry := newRegistry(parser)
	scanner := repo.NewScanner(registry)
	inspector := inspectsvc.NewService(scanner, registry, parser)
	planner := plansvc.NewService(scanner, registry, parser)
	manifest := manifestsvc.NewService(planner)
	conflicts := conflictsvc.NewService(scanner, registry, parser)

	return commandRuntime{
		loaded:    loaded,
		parser:    parser,
		adapters:  registry,
		scanner:   scanner,
		conflicts: conflicts,
		finder:    ticketsvc.NewService(scanner, registry, parser),
		diff:      diffsvc.NewService(scanner, registry, parser),
		inspect:   inspector,
		planner:   planner,
		snapshot:  snapshotsvc.NewService(inspector, planner),
		doctor:    doctorsvc.NewService(scanner, registry),
		manifest:  manifest,
		assistant: assistsvc.NewService(inspector, planner, manifest, conflicts),
	}, nil
}

func loadRuntimeConfig(path, configPath string, options runtimeOptions) (config.Loaded, error) {
	if options.DisableAutoConfig && strings.TrimSpace(configPath) == "" {
		return config.Loaded{
			Config: config.Default(),
		}, nil
	}

	return config.LoadForPath(path, configPath)
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	var buffer bytes.Buffer
	if err := output.RenderJSON(&buffer, value); err != nil {
		return err
	}

	return os.WriteFile(path, buffer.Bytes(), 0o644)
}

func newRegistry(parser ticketsvc.Parser) *scm.Registry {
	return scm.NewRegistry(
		gitscm.NewAdapter(parser),
		azuredevopsscm.NewAdapter(parser),
		bitbucketscm.NewAdapter(parser),
		githubscm.NewAdapter(parser),
		gitlabscm.NewAdapter(parser),
		svnscm.NewAdapter(parser),
		svnscm.NewRemoteAdapter(parser),
	)
}

func reorderArgsWithSinglePositional(args []string, flagsWithValues ...string) ([]string, error) {
	valueFlagSet := make(map[string]struct{}, len(flagsWithValues))
	for _, flagName := range flagsWithValues {
		valueFlagSet[flagName] = struct{}{}
	}

	reordered := make([]string, 0, len(args))
	positionals := make([]string, 0, 1)

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "-h" || arg == "--help":
			reordered = append(reordered, arg)
		case isValueFlag(arg, valueFlagSet):
			reordered = append(reordered, arg)
			if i+1 >= len(args) {
				return nil, fmt.Errorf("flag %q requires a value", arg)
			}
			reordered = append(reordered, args[i+1])
			i++
		case isInlineValueFlag(arg, valueFlagSet):
			reordered = append(reordered, arg)
		case strings.HasPrefix(arg, "-"):
			reordered = append(reordered, arg)
		default:
			positionals = append(positionals, arg)
		}
	}

	if len(positionals) > 1 {
		return nil, fmt.Errorf("requires exactly one positional argument")
	}

	return append(reordered, positionals...), nil
}

func isValueFlag(arg string, valueFlags map[string]struct{}) bool {
	_, ok := valueFlags[arg]
	return ok
}

func isInlineValueFlag(arg string, valueFlags map[string]struct{}) bool {
	for flagName := range valueFlags {
		if strings.HasPrefix(arg, flagName+"=") {
			return true
		}
	}

	return false
}
