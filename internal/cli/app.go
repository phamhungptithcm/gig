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
	"path/filepath"
	"strings"

	assistsvc "gig/internal/assistant"
	"gig/internal/buildinfo"
	"gig/internal/config"
	conflictsvc "gig/internal/conflict"
	"gig/internal/diagnostics"
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
	logger := diagnostics.NewFromEnv(os.LookupEnv)
	if logger != nil {
		ctx = diagnostics.WithLogger(ctx, logger)
	}
	commandName := "frontdoor"
	if len(args) == 0 {
		diagnostics.Emit(ctx, "info", "command.start", "command started", diagnostics.Meta{Command: commandName}, nil)
		exitCode := a.runFrontDoor(ctx)
		diagnostics.Emit(ctx, "info", "command.finish", "command finished", diagnostics.Meta{
			Command: commandName,
			Details: map[string]any{"exitCode": exitCode},
		}, nil)
		return exitCode
	}
	commandName = strings.TrimSpace(args[0])
	diagnostics.Emit(ctx, "info", "command.start", "command started", diagnostics.Meta{
		Command: commandName,
		Details: map[string]any{"args": append([]string(nil), args[1:]...)},
	}, nil)

	var exitCode int
	switch args[0] {
	case "scan":
		exitCode = a.runScan(ctx, args[1:])
	case "find":
		exitCode = a.runFind(ctx, args[1:])
	case "diff":
		exitCode = a.runDiff(ctx, args[1:])
	case "inspect":
		exitCode = a.runInspect(ctx, args[1:])
	case "env":
		exitCode = a.runEnv(ctx, args[1:])
	case "plan":
		exitCode = a.runPlan(ctx, args[1:])
	case "verify":
		exitCode = a.runVerify(ctx, args[1:])
	case "manifest":
		exitCode = a.runManifest(ctx, args[1:])
	case "snapshot":
		exitCode = a.runSnapshot(ctx, args[1:])
	case "assist":
		exitCode = a.runAssist(ctx, args[1:])
	case "ask":
		exitCode = a.runAsk(ctx, args[1:])
	case "resume":
		exitCode = a.runAssistResume(args[1:])
	case "workarea", "project":
		exitCode = a.runWorkarea(ctx, args[1:])
	case "login":
		exitCode = a.runLogin(ctx, args[1:])
	case "doctor":
		exitCode = a.runDoctor(ctx, args[1:])
	case "resolve":
		exitCode = a.runResolve(ctx, args[1:])
	case "update":
		exitCode = a.runUpdate(ctx, args[1:])
	case "version", "-v", "--version":
		exitCode = a.runVersion()
	case "help", "-h", "--help":
		a.printRootUsage()
		exitCode = 0
	default:
		if looksLikeTicketID(args[0]) {
			exitCode = a.runInspect(ctx, args)
			break
		}
		fmt.Fprintf(a.stderr, "unknown command %q\n\n", args[0])
		a.printRootUsage()
		exitCode = usageExitCode
	}
	diagnostics.Emit(ctx, "info", "command.finish", "command finished", diagnostics.Meta{
		Command: commandName,
		Details: map[string]any{"exitCode": exitCode},
	}, nil)
	return exitCode
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
	case "chat", "ask":
		return a.runAssistChat(ctx, args[1:])
	case "release":
		return a.runAssistRelease(ctx, args[1:])
	case "resolve":
		return a.runAssistResolve(ctx, args[1:])
	case "resume":
		return a.runAssistResume(args[1:])
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

	request := assistsvc.AuditRequest{
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
	}
	result, err := runtime.assistant.Audit(ctx, request, assistsvc.ExecuteOptions{
		BaseURL:  *deerflowURL,
		Mode:     runMode,
		Audience: selectedAudience,
	})
	if err != nil {
		fmt.Fprintf(a.stderr, "assist audit failed: %v\n", err)
		return 1
	}
	a.persistAuditSession(scope, request, result, runMode, selectedAudience)

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

	request := assistsvc.ReleaseRequest{
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
	}
	result, err := runtime.assistant.Release(ctx, request, assistsvc.ExecuteOptions{
		BaseURL:  *deerflowURL,
		Mode:     runMode,
		Audience: selectedAudience,
	})
	if err != nil {
		fmt.Fprintf(a.stderr, "assist release failed: %v\n", err)
		return 1
	}
	a.persistReleaseSession(scope, request, result, runMode, selectedAudience)

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

	request := assistsvc.ResolveRequest{
		WorkspacePath: resolvedPath,
		ScopeLabel:    resolvedPath,
		CommandTarget: fmt.Sprintf("--path %s", shellSingleQuote(resolvedPath)),
		ConfigPath:    strings.TrimSpace(*configPath),
		TicketID:      strings.TrimSpace(*ticketID),
	}
	result, err := runtime.assistant.Resolve(ctx, request, assistsvc.ExecuteOptions{
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
	a.persistResolveSession(request.ScopeLabel, request.WorkspacePath, request.ConfigPath, request.TicketID, result, runMode, selectedAudience)

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
