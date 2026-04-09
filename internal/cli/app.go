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
	gitscm "gig/internal/scm/git"
	svnscm "gig/internal/scm/svn"
	snapshotsvc "gig/internal/snapshot"
	ticketsvc "gig/internal/ticket"
	updatesvc "gig/internal/update"
)

const usageExitCode = 2

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
		a.printRootUsage()
		return usageExitCode
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
	reorderedArgs, err := reorderArgsWithSinglePositional(args, "-path", "--path", "-config", "--config")
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
	configPath := fs.String("config", "", "Path to a gig config file")
	if err := fs.Parse(args); err != nil {
		a.printFindUsage()
		return usageExitCode
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(a.stderr, "find requires exactly one <ticket-id> argument")
		a.printFindUsage()
		return usageExitCode
	}

	resolvedPath, err := normalizeCLIPath(*path)
	if err != nil {
		fmt.Fprintf(a.stderr, "find failed: %v\n", err)
		return 1
	}

	ticketID := normalizeTicketID(fs.Arg(0))
	runtime, err := newCommandRuntime(resolvedPath, *configPath)
	if err != nil {
		fmt.Fprintf(a.stderr, "find failed: %v\n", err)
		return 1
	}

	repositories, err := runtime.scanner.Discover(ctx, resolvedPath)
	if err != nil {
		fmt.Fprintf(a.stderr, "find failed: %v\n", err)
		return 1
	}

	results, err := runtime.finder.FindInRepositories(ctx, repositories, ticketID)
	if err != nil {
		fmt.Fprintf(a.stderr, "find failed: %v\n", err)
		return 1
	}

	if err := output.RenderFind(a.stdout, ticketID, resolvedPath, len(repositories), results); err != nil {
		fmt.Fprintf(a.stderr, "render failed: %v\n", err)
		return 1
	}

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
	configPath := fs.String("config", "", "Path to a gig config file")
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
	reorderedArgs, err := reorderArgsWithSinglePositional(args, "-path", "--path", "-config", "--config")
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
	configPath := fs.String("config", "", "Path to a gig config file")
	if err := fs.Parse(args); err != nil {
		a.printInspectUsage()
		return usageExitCode
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(a.stderr, "inspect requires exactly one <ticket-id> argument")
		a.printInspectUsage()
		return usageExitCode
	}

	resolvedPath, err := normalizeCLIPath(*path)
	if err != nil {
		fmt.Fprintf(a.stderr, "inspect failed: %v\n", err)
		return 1
	}

	ticketID := normalizeTicketID(fs.Arg(0))
	runtime, err := newCommandRuntime(resolvedPath, *configPath)
	if err != nil {
		fmt.Fprintf(a.stderr, "inspect failed: %v\n", err)
		return 1
	}

	results, scannedRepoCount, err := runtime.inspect.Inspect(ctx, resolvedPath, ticketID)
	if err != nil {
		fmt.Fprintf(a.stderr, "inspect failed: %v\n", err)
		return 1
	}

	if err := output.RenderInspect(a.stdout, ticketID, resolvedPath, scannedRepoCount, results); err != nil {
		fmt.Fprintf(a.stderr, "render failed: %v\n", err)
		return 1
	}

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
	reorderedArgs, err := reorderArgsWithSinglePositional(args, "-path", "--path", "-config", "--config", "-envs", "--envs")
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
	configPath := fs.String("config", "", "Path to a gig config file")
	envsSpec := fs.String("envs", "", "Comma-separated environment mapping, for example dev=dev,test=test,prod=main")
	if err := fs.Parse(args); err != nil {
		a.printEnvStatusUsage()
		return usageExitCode
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(a.stderr, "env status requires exactly one <ticket-id> argument")
		a.printEnvStatusUsage()
		return usageExitCode
	}

	resolvedPath, err := normalizeCLIPath(*path)
	if err != nil {
		fmt.Fprintf(a.stderr, "env status failed: %v\n", err)
		return 1
	}

	runtime, err := newCommandRuntime(resolvedPath, *configPath)
	if err != nil {
		fmt.Fprintf(a.stderr, "env status failed: %v\n", err)
		return 1
	}

	environments, err := resolveEnvironments(*envsSpec, runtime.loaded)
	if err != nil {
		fmt.Fprintf(a.stderr, "env status failed: %v\n", err)
		return usageExitCode
	}

	ticketID := normalizeTicketID(fs.Arg(0))
	results, scannedRepoCount, err := runtime.inspect.EnvironmentStatus(ctx, resolvedPath, ticketID, environments)
	if err != nil {
		fmt.Fprintf(a.stderr, "env status failed: %v\n", err)
		return 1
	}

	if err := output.RenderEnvironmentStatus(a.stdout, ticketID, resolvedPath, environments, scannedRepoCount, results); err != nil {
		fmt.Fprintf(a.stderr, "render failed: %v\n", err)
		return 1
	}

	return 0
}

func (a *App) runPlan(ctx context.Context, args []string) int {
	if hasHelpFlag(args) {
		a.printPlanUsage()
		return 0
	}

	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a repository or workspace")
	configPath := fs.String("config", "", "Path to a gig config file")
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
	if fs.NArg() != 0 {
		fmt.Fprintln(a.stderr, "plan does not accept positional arguments")
		a.printPlanUsage()
		return usageExitCode
	}

	outputFormat, err := parseOutputFormat(*format)
	if err != nil {
		fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
		return usageExitCode
	}

	resolvedPath, err := normalizeCLIPath(*path)
	if err != nil {
		fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
		return 1
	}

	runtime, err := newCommandRuntime(resolvedPath, *configPath)
	if err != nil {
		fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
		return 1
	}

	normalizedReleaseID := strings.TrimSpace(*releaseID)
	if normalizedReleaseID != "" {
		if *ticketID != "" || *ticketFile != "" {
			fmt.Fprintln(a.stderr, "plan failed: use either --release or ticket-based flags, not both")
			a.printPlanUsage()
			return usageExitCode
		}
		if strings.TrimSpace(*fromBranch) != "" || strings.TrimSpace(*toBranch) != "" || strings.TrimSpace(*envsSpec) != "" {
			fmt.Fprintln(a.stderr, "plan failed: --from, --to, and --envs are not used with --release")
			a.printPlanUsage()
			return usageExitCode
		}

		normalizedReleaseID, err = snapshotsvc.NormalizeReleaseID(normalizedReleaseID)
		if err != nil {
			fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
			return usageExitCode
		}

		snapshots, snapshotDir, err := snapshotsvc.LoadReleaseSnapshots(resolvedPath, normalizedReleaseID)
		if err != nil {
			fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
			return 1
		}
		releasePlan, err := releaseplansvc.Build(normalizedReleaseID, resolvedPath, snapshotDir, snapshots)
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
				Workspace:   resolvedPath,
				ReleasePlan: releasePlan,
			}); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		}

		return 0
	}

	environments, err := resolveEnvironments(*envsSpec, runtime.loaded)
	if err != nil {
		fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
		return usageExitCode
	}

	ticketIDs, resolvedTicketFile, err := resolveTicketIDs(*ticketID, *ticketFile, runtime.parser)
	if err != nil {
		fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
		return usageExitCode
	}

	promotionPlans := make([]plansvc.PromotionPlan, 0, len(ticketIDs))
	for _, ticketID := range ticketIDs {
		promotionPlan, err := runtime.planner.BuildPromotionPlan(ctx, resolvedPath, ticketID, *fromBranch, *toBranch, environments)
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
				Workspace: resolvedPath,
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
				Workspace:  resolvedPath,
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
	if hasHelpFlag(args) {
		a.printVerifyUsage()
		return 0
	}

	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a repository or workspace")
	configPath := fs.String("config", "", "Path to a gig config file")
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
	if fs.NArg() != 0 {
		fmt.Fprintln(a.stderr, "verify does not accept positional arguments")
		a.printVerifyUsage()
		return usageExitCode
	}

	outputFormat, err := parseOutputFormat(*format)
	if err != nil {
		fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
		return usageExitCode
	}

	resolvedPath, err := normalizeCLIPath(*path)
	if err != nil {
		fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
		return 1
	}

	runtime, err := newCommandRuntime(resolvedPath, *configPath)
	if err != nil {
		fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
		return 1
	}

	normalizedReleaseID := strings.TrimSpace(*releaseID)
	if normalizedReleaseID != "" {
		if *ticketID != "" || *ticketFile != "" {
			fmt.Fprintln(a.stderr, "verify failed: use either --release or ticket-based flags, not both")
			a.printVerifyUsage()
			return usageExitCode
		}
		if strings.TrimSpace(*fromBranch) != "" || strings.TrimSpace(*toBranch) != "" || strings.TrimSpace(*envsSpec) != "" {
			fmt.Fprintln(a.stderr, "verify failed: --from, --to, and --envs are not used with --release")
			a.printVerifyUsage()
			return usageExitCode
		}

		normalizedReleaseID, err = snapshotsvc.NormalizeReleaseID(normalizedReleaseID)
		if err != nil {
			fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
			return usageExitCode
		}
		snapshots, snapshotDir, err := snapshotsvc.LoadReleaseSnapshots(resolvedPath, normalizedReleaseID)
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
				Workspace:     resolvedPath,
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

	environments, err := resolveEnvironments(*envsSpec, runtime.loaded)
	if err != nil {
		fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
		return usageExitCode
	}

	ticketIDs, resolvedTicketFile, err := resolveTicketIDs(*ticketID, *ticketFile, runtime.parser)
	if err != nil {
		fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
		return usageExitCode
	}

	verifications := make([]plansvc.Verification, 0, len(ticketIDs))
	for _, ticketID := range ticketIDs {
		verification, err := runtime.planner.VerifyPromotion(ctx, resolvedPath, ticketID, *fromBranch, *toBranch, environments)
		if err != nil {
			fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
			return 1
		}
		verifications = append(verifications, verification)
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
				Workspace:    resolvedPath,
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
				Workspace:     resolvedPath,
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
		fmt.Fprintf(a.stderr, "unknown manifest subcommand %q\n\n", args[0])
		a.printManifestUsage()
		return usageExitCode
	}
}

func (a *App) runManifestGenerate(ctx context.Context, args []string) int {
	if hasHelpFlag(args) {
		a.printManifestGenerateUsage()
		return 0
	}

	fs := flag.NewFlagSet("manifest generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a repository or workspace")
	configPath := fs.String("config", "", "Path to a gig config file")
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
	if fs.NArg() != 0 {
		fmt.Fprintln(a.stderr, "manifest generate does not accept positional arguments")
		a.printManifestGenerateUsage()
		return usageExitCode
	}

	selectedFormat, err := parseManifestFormat(*format)
	if err != nil {
		fmt.Fprintf(a.stderr, "manifest generate failed: %v\n", err)
		return usageExitCode
	}

	resolvedPath, err := normalizeCLIPath(*path)
	if err != nil {
		fmt.Fprintf(a.stderr, "manifest generate failed: %v\n", err)
		return 1
	}

	runtime, err := newCommandRuntime(resolvedPath, *configPath)
	if err != nil {
		fmt.Fprintf(a.stderr, "manifest generate failed: %v\n", err)
		return 1
	}

	normalizedReleaseID := strings.TrimSpace(*releaseID)
	if normalizedReleaseID != "" {
		if *ticketID != "" || *ticketFile != "" {
			fmt.Fprintln(a.stderr, "manifest generate failed: use either --release or ticket-based flags, not both")
			a.printManifestGenerateUsage()
			return usageExitCode
		}
		if strings.TrimSpace(*fromBranch) != "" || strings.TrimSpace(*toBranch) != "" || strings.TrimSpace(*envsSpec) != "" {
			fmt.Fprintln(a.stderr, "manifest generate failed: --from, --to, and --envs are not used with --release")
			a.printManifestGenerateUsage()
			return usageExitCode
		}

		normalizedReleaseID, err = snapshotsvc.NormalizeReleaseID(normalizedReleaseID)
		if err != nil {
			fmt.Fprintf(a.stderr, "manifest generate failed: %v\n", err)
			return usageExitCode
		}
		snapshots, snapshotDir, err := snapshotsvc.LoadReleaseSnapshots(resolvedPath, normalizedReleaseID)
		if err != nil {
			fmt.Fprintf(a.stderr, "manifest generate failed: %v\n", err)
			return 1
		}

		packets := make([]manifestsvc.ReleasePacket, 0, len(snapshots))
		for _, snapshot := range snapshots {
			packets = append(packets, manifestsvc.BuildReleasePacket(resolvedPath, runtime.loaded, snapshot.Plan))
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
				Workspace:   resolvedPath,
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

	environments, err := resolveEnvironments(*envsSpec, runtime.loaded)
	if err != nil {
		fmt.Fprintf(a.stderr, "manifest generate failed: %v\n", err)
		return usageExitCode
	}

	ticketIDs, resolvedTicketFile, err := resolveTicketIDs(*ticketID, *ticketFile, runtime.parser)
	if err != nil {
		fmt.Fprintf(a.stderr, "manifest generate failed: %v\n", err)
		return usageExitCode
	}

	packets := make([]manifestsvc.ReleasePacket, 0, len(ticketIDs))
	for _, ticketID := range ticketIDs {
		packet, err := runtime.manifest.Generate(ctx, resolvedPath, runtime.loaded, ticketID, *fromBranch, *toBranch, environments)
		if err != nil {
			fmt.Fprintf(a.stderr, "manifest generate failed: %v\n", err)
			return 1
		}
		packets = append(packets, packet)
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
				Workspace:  resolvedPath,
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

func (a *App) runSnapshotCreate(ctx context.Context, args []string) int {
	if hasHelpFlag(args) {
		a.printSnapshotCreateUsage()
		return 0
	}

	fs := flag.NewFlagSet("snapshot create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", ".", "Path to a repository or workspace")
	configPath := fs.String("config", "", "Path to a gig config file")
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

	resolvedPath, err := normalizeCLIPath(*path)
	if err != nil {
		fmt.Fprintf(a.stderr, "snapshot create failed: %v\n", err)
		return 1
	}

	runtime, err := newCommandRuntime(resolvedPath, *configPath)
	if err != nil {
		fmt.Fprintf(a.stderr, "snapshot create failed: %v\n", err)
		return 1
	}

	environments, err := resolveEnvironments(*envsSpec, runtime.loaded)
	if err != nil {
		fmt.Fprintf(a.stderr, "snapshot create failed: %v\n", err)
		return usageExitCode
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
	if strings.TrimSpace(*fromBranch) == "" || strings.TrimSpace(*toBranch) == "" {
		fmt.Fprintln(a.stderr, "snapshot create failed: both --from and --to branches are required")
		a.printSnapshotCreateUsage()
		return usageExitCode
	}

	normalizedReleaseID := ""
	if strings.TrimSpace(*releaseID) != "" {
		normalizedReleaseID, err = snapshotsvc.NormalizeReleaseID(*releaseID)
		if err != nil {
			fmt.Fprintf(a.stderr, "snapshot create failed: %v\n", err)
			return usageExitCode
		}
	}

	snapshot, err := runtime.snapshot.CaptureWithOptions(ctx, resolvedPath, runtime.loaded, normalizedTicketID, *fromBranch, *toBranch, environments, snapshotsvc.CaptureOptions{
		ReleaseID: normalizedReleaseID,
	})
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
		resolvedOutputPath = snapshotsvc.DefaultReleaseSnapshotPath(resolvedPath, normalizedReleaseID, normalizedTicketID)
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
	configPath := fs.String("config", "", "Path to a gig config file")
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
	configPath := fs.String("config", "", "Path to a gig config file")
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
	configPath := fs.String("config", "", "Path to a gig config file")
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
	installMode := updatesvc.DetectInstallMode(resolvedExecutablePath)
	if installDir != "" {
		installMode = updatesvc.ModeDirect
	} else {
		installDir = filepath.Dir(resolvedExecutablePath)
	}

	switch installMode {
	case updatesvc.ModeHomebrew:
		if version != "latest" {
			fmt.Fprintf(a.stderr, "update failed: Homebrew installs track the latest stable release only. Use the install script or GitHub release asset for %s.\n", version)
			return 1
		}

		fmt.Fprintf(a.stdout, "Detected a Homebrew-managed install at %s\n", resolvedExecutablePath)
		if code := a.runExternalCommand(ctx, "brew", []string{"update"}); code != 0 {
			return code
		}
		return a.runExternalCommand(ctx, "brew", []string{"upgrade", "gig-cli"})
	case updatesvc.ModeScoop:
		if version != "latest" {
			fmt.Fprintf(a.stderr, "update failed: Scoop installs track the latest stable release only. Use the PowerShell installer or GitHub release asset for %s.\n", version)
			return 1
		}

		fmt.Fprintf(a.stdout, "Detected a Scoop-managed install at %s\n", resolvedExecutablePath)
		return a.runExternalCommand(ctx, "scoop", []string{"update", "gig-cli"})
	default:
		if runtime.GOOS == "windows" {
			return a.runWindowsInstallerUpdate(ctx, repoName, version, installDir)
		}
		return a.runPOSIXInstallerUpdate(ctx, repoName, version, installDir)
	}
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

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func powerShellSingleQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
func (a *App) printRootUsage() {
	fmt.Fprintln(a.stderr, "gig helps teams check whether a ticket is really ready for the next release step.")
	fmt.Fprintln(a.stderr)
	fmt.Fprintln(a.stderr, "Usage:")
	fmt.Fprintln(a.stderr, "  gig <command> [flags]")
	fmt.Fprintln(a.stderr)
	fmt.Fprintln(a.stderr, "Start here:")
	fmt.Fprintln(a.stderr, "  gig --help")
	fmt.Fprintln(a.stderr, "  gig inspect ABC-123 --path .")
	fmt.Fprintln(a.stderr, "  gig verify --ticket ABC-123 --from test --to main --path .")
	fmt.Fprintln(a.stderr, "  gig manifest generate --ticket ABC-123 --from test --to main --path .")
	fmt.Fprintln(a.stderr, "  gig doctor --path .")
	fmt.Fprintln(a.stderr, "  gig update")
	fmt.Fprintln(a.stderr)
	fmt.Fprintln(a.stderr, "Commands:")
	fmt.Fprintln(a.stderr, "  scan        Find repositories under a path")
	fmt.Fprintln(a.stderr, "  find        Find commits for one ticket")
	fmt.Fprintln(a.stderr, "  inspect     Show the full ticket picture across repositories")
	fmt.Fprintln(a.stderr, "  env status  Show where a ticket is present or behind across environments")
	fmt.Fprintln(a.stderr, "  diff        Compare one branch to another for a ticket")
	fmt.Fprintln(a.stderr, "  verify      Return safe, warning, or blocked for the next move")
	fmt.Fprintln(a.stderr, "  plan        Build a read-only promotion plan")
	fmt.Fprintln(a.stderr, "  manifest    Generate a release packet for QA, client, and release review")
	fmt.Fprintln(a.stderr, "  snapshot    Save a repeatable ticket baseline for audit and re-check")
	fmt.Fprintln(a.stderr, "  doctor      Check config coverage, env mappings, and repo catalog health")
	fmt.Fprintln(a.stderr, "  resolve     Inspect or resolve active Git merge conflicts")
	fmt.Fprintln(a.stderr, "  update      Install the latest release or a specific version")
	fmt.Fprintln(a.stderr, "  version     Show the installed version")
	fmt.Fprintln(a.stderr)
	fmt.Fprintln(a.stderr, "More help:")
	fmt.Fprintln(a.stderr, "  gig <command> --help")
}

func (a *App) printScanUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig scan --path .")
}

func (a *App) printFindUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig find <ticket-id> --path . [--config gig.yaml]")
}

func (a *App) printDiffUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig diff --ticket <ticket-id> --from <branch> --to <branch> --path . [--config gig.yaml]")
}

func (a *App) printInspectUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig inspect <ticket-id> --path . [--config gig.yaml]")
}

func (a *App) printEnvUsage() {
	fmt.Fprintln(a.stderr, "Usage:")
	fmt.Fprintln(a.stderr, "  gig env status <ticket-id> --path . [--config gig.yaml] [--envs dev=dev,test=test,prod=main]")
}

func (a *App) printEnvStatusUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig env status <ticket-id> --path . [--config gig.yaml] [--envs dev=dev,test=test,prod=main]")
}

func (a *App) printPlanUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig plan --release <release-id> --path . [--config gig.yaml] [--format human|json]")
	fmt.Fprintln(a.stderr, "   or: gig plan (--ticket <ticket-id> | --ticket-file tickets.txt) --from <branch> --to <branch> --path . [--config gig.yaml] [--envs dev=dev,test=test,prod=main] [--format human|json]")
}

func (a *App) printVerifyUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig verify --release <release-id> --path . [--config gig.yaml] [--format human|json]")
	fmt.Fprintln(a.stderr, "   or: gig verify (--ticket <ticket-id> | --ticket-file tickets.txt) --from <branch> --to <branch> --path . [--config gig.yaml] [--envs dev=dev,test=test,prod=main] [--format human|json]")
}

func (a *App) printManifestUsage() {
	fmt.Fprintln(a.stderr, "Usage:")
	fmt.Fprintln(a.stderr, "  gig manifest generate --release <release-id> --path . [--config gig.yaml] [--format markdown|json]")
	fmt.Fprintln(a.stderr, "  gig manifest generate (--ticket <ticket-id> | --ticket-file tickets.txt) --from <branch> --to <branch> --path . [--config gig.yaml] [--envs dev=dev,test=test,prod=main] [--format markdown|json]")
}

func (a *App) printManifestGenerateUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig manifest generate --release <release-id> --path . [--config gig.yaml] [--format markdown|json]")
	fmt.Fprintln(a.stderr, "   or: gig manifest generate (--ticket <ticket-id> | --ticket-file tickets.txt) --from <branch> --to <branch> --path . [--config gig.yaml] [--envs dev=dev,test=test,prod=main] [--format markdown|json]")
}

func (a *App) printSnapshotUsage() {
	fmt.Fprintln(a.stderr, "Usage:")
	fmt.Fprintln(a.stderr, "  gig snapshot create --ticket <ticket-id> --from <branch> --to <branch> --path . [--config gig.yaml] [--release <release-id>] [--envs dev=dev,test=test,prod=main] [--format human|json] [--output snapshot.json]")
}

func (a *App) printSnapshotCreateUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig snapshot create --ticket <ticket-id> --from <branch> --to <branch> --path . [--config gig.yaml] [--release <release-id>] [--envs dev=dev,test=test,prod=main] [--format human|json] [--output snapshot.json]")
}

func (a *App) printDoctorUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig doctor --path . [--config gig.yaml] [--format human|json]")
}

func (a *App) printResolveUsage() {
	fmt.Fprintln(a.stderr, "Usage:")
	fmt.Fprintln(a.stderr, "  gig resolve status --path . [--config gig.yaml] [--ticket ABC-123] [--format human|json]")
	fmt.Fprintln(a.stderr, "  gig resolve start --path . [--config gig.yaml] [--ticket ABC-123]")
}

func (a *App) printResolveStatusUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig resolve status --path . [--config gig.yaml] [--ticket ABC-123] [--format human|json]")
}

func (a *App) printResolveStartUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig resolve start --path . [--config gig.yaml] [--ticket ABC-123]")
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

type commandRuntime struct {
	loaded    config.Loaded
	parser    ticketsvc.Parser
	scanner   *repo.Scanner
	conflicts *conflictsvc.Service
	finder    *ticketsvc.Service
	diff      *diffsvc.Service
	inspect   *inspectsvc.Service
	planner   *plansvc.Service
	snapshot  *snapshotsvc.Service
	doctor    *doctorsvc.Service
	manifest  *manifestsvc.Service
}

func newCommandRuntime(path, configPath string) (commandRuntime, error) {
	loaded, err := config.LoadForPath(path, configPath)
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

	return commandRuntime{
		loaded:    loaded,
		parser:    parser,
		scanner:   scanner,
		conflicts: conflictsvc.NewService(scanner, registry, parser),
		finder:    ticketsvc.NewService(scanner, registry, parser),
		diff:      diffsvc.NewService(scanner, registry, parser),
		inspect:   inspector,
		planner:   planner,
		snapshot:  snapshotsvc.NewService(inspector, planner),
		doctor:    doctorsvc.NewService(scanner, registry),
		manifest:  manifestsvc.NewService(planner),
	}, nil
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
		svnscm.NewAdapter(parser),
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
