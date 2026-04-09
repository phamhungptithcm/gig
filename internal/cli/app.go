package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"gig/internal/buildinfo"
	"gig/internal/config"
	diffsvc "gig/internal/diff"
	inspectsvc "gig/internal/inspect"
	"gig/internal/output"
	plansvc "gig/internal/plan"
	"gig/internal/repo"
	"gig/internal/scm"
	gitscm "gig/internal/scm/git"
	svnscm "gig/internal/scm/svn"
	ticketsvc "gig/internal/ticket"
)

const usageExitCode = 2

type App struct {
	stdout  io.Writer
	stderr  io.Writer
	scanner *repo.Scanner
	finder  *ticketsvc.Service
	diff    *diffsvc.Service
	inspect *inspectsvc.Service
	planner *plansvc.Service
}

func NewApp(stdout, stderr io.Writer) (*App, error) {
	cfg := config.Default()
	parser, err := ticketsvc.NewParser(cfg.TicketPattern)
	if err != nil {
		return nil, err
	}

	registry := scm.NewRegistry(
		gitscm.NewAdapter(parser),
		svnscm.NewAdapter(),
	)

	scanner := repo.NewScanner(registry)

	return &App{
		stdout:  stdout,
		stderr:  stderr,
		scanner: scanner,
		finder:  ticketsvc.NewService(scanner, registry, parser),
		diff:    diffsvc.NewService(scanner, registry, parser),
		inspect: inspectsvc.NewService(scanner, registry, parser),
		planner: plansvc.NewService(scanner, registry, parser),
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
	reorderedArgs, err := reorderArgsWithSinglePositional(args, "-path", "--path")
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
	repositories, err := a.scanner.Discover(ctx, resolvedPath)
	if err != nil {
		fmt.Fprintf(a.stderr, "find failed: %v\n", err)
		return 1
	}

	results, err := a.finder.FindInRepositories(ctx, repositories, ticketID)
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
	repositories, err := a.scanner.Discover(ctx, resolvedPath)
	if err != nil {
		fmt.Fprintf(a.stderr, "diff failed: %v\n", err)
		return 1
	}

	results, err := a.diff.CompareTicketInRepositories(ctx, repositories, normalizedTicketID, *fromBranch, *toBranch)
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
	reorderedArgs, err := reorderArgsWithSinglePositional(args, "-path", "--path")
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
	results, scannedRepoCount, err := a.inspect.Inspect(ctx, resolvedPath, ticketID)
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
	reorderedArgs, err := reorderArgsWithSinglePositional(args, "-path", "--path", "-envs", "--envs")
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
	envsSpec := fs.String("envs", defaultEnvironmentSpec, "Comma-separated environment mapping, for example dev=dev,test=test,prod=main")
	if err := fs.Parse(args); err != nil {
		a.printEnvStatusUsage()
		return usageExitCode
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(a.stderr, "env status requires exactly one <ticket-id> argument")
		a.printEnvStatusUsage()
		return usageExitCode
	}

	environments, err := parseEnvironmentSpec(*envsSpec)
	if err != nil {
		fmt.Fprintf(a.stderr, "env status failed: %v\n", err)
		return usageExitCode
	}

	resolvedPath, err := normalizeCLIPath(*path)
	if err != nil {
		fmt.Fprintf(a.stderr, "env status failed: %v\n", err)
		return 1
	}

	ticketID := normalizeTicketID(fs.Arg(0))
	results, scannedRepoCount, err := a.inspect.EnvironmentStatus(ctx, resolvedPath, ticketID, environments)
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
	ticketID := fs.String("ticket", "", "Ticket ID to plan")
	fromBranch := fs.String("from", "", "Source branch")
	toBranch := fs.String("to", "", "Target branch")
	envsSpec := fs.String("envs", defaultEnvironmentSpec, "Comma-separated environment mapping, for example dev=dev,test=test,prod=main")
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

	environments, err := parseEnvironmentSpec(*envsSpec)
	if err != nil {
		fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
		return usageExitCode
	}

	resolvedPath, err := normalizeCLIPath(*path)
	if err != nil {
		fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
		return 1
	}

	normalizedTicketID := normalizeTicketID(*ticketID)
	promotionPlan, err := a.planner.BuildPromotionPlan(ctx, resolvedPath, normalizedTicketID, *fromBranch, *toBranch, environments)
	if err != nil {
		fmt.Fprintf(a.stderr, "plan failed: %v\n", err)
		return 1
	}

	switch outputFormat {
	case outputFormatHuman:
		if err := output.RenderPromotionPlan(a.stdout, promotionPlan); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	case outputFormatJSON:
		if err := output.RenderJSON(a.stdout, struct {
			Command   string                `json:"command"`
			Workspace string                `json:"workspace"`
			Plan      plansvc.PromotionPlan `json:"plan"`
		}{
			Command:   "plan",
			Workspace: resolvedPath,
			Plan:      promotionPlan,
		}); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
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
	ticketID := fs.String("ticket", "", "Ticket ID to verify")
	fromBranch := fs.String("from", "", "Source branch")
	toBranch := fs.String("to", "", "Target branch")
	envsSpec := fs.String("envs", defaultEnvironmentSpec, "Comma-separated environment mapping, for example dev=dev,test=test,prod=main")
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

	environments, err := parseEnvironmentSpec(*envsSpec)
	if err != nil {
		fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
		return usageExitCode
	}

	resolvedPath, err := normalizeCLIPath(*path)
	if err != nil {
		fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
		return 1
	}

	normalizedTicketID := normalizeTicketID(*ticketID)
	verification, err := a.planner.VerifyPromotion(ctx, resolvedPath, normalizedTicketID, *fromBranch, *toBranch, environments)
	if err != nil {
		fmt.Fprintf(a.stderr, "verify failed: %v\n", err)
		return 1
	}

	switch outputFormat {
	case outputFormatHuman:
		if err := output.RenderVerification(a.stdout, verification); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	case outputFormatJSON:
		if err := output.RenderJSON(a.stdout, struct {
			Command      string               `json:"command"`
			Workspace    string               `json:"workspace"`
			Verification plansvc.Verification `json:"verification"`
		}{
			Command:      "verify",
			Workspace:    resolvedPath,
			Verification: verification,
		}); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	}

	return 0
}

func (a *App) printRootUsage() {
	fmt.Fprintln(a.stderr, "gig helps release workflows find ticket-related commits across multiple repositories.")
	fmt.Fprintln(a.stderr)
	fmt.Fprintln(a.stderr, "Usage:")
	fmt.Fprintln(a.stderr, "  gig scan --path .")
	fmt.Fprintln(a.stderr, "  gig find <ticket-id> --path .")
	fmt.Fprintln(a.stderr, "  gig diff --ticket <ticket-id> --from <branch> --to <branch> --path .")
	fmt.Fprintln(a.stderr, "  gig inspect <ticket-id> --path .")
	fmt.Fprintln(a.stderr, "  gig env status <ticket-id> --path . [--envs dev=dev,test=test,prod=main]")
	fmt.Fprintln(a.stderr, "  gig verify --ticket <ticket-id> --from <branch> --to <branch> --path . [--envs ...] [--format human|json]")
	fmt.Fprintln(a.stderr, "  gig plan --ticket <ticket-id> --from <branch> --to <branch> --path . [--envs ...] [--format human|json]")
	fmt.Fprintln(a.stderr, "  gig version")
}

func (a *App) printScanUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig scan --path .")
}

func (a *App) printFindUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig find <ticket-id> --path .")
}

func (a *App) printDiffUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig diff --ticket <ticket-id> --from <branch> --to <branch> --path .")
}

func (a *App) printInspectUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig inspect <ticket-id> --path .")
}

func (a *App) printEnvUsage() {
	fmt.Fprintln(a.stderr, "Usage:")
	fmt.Fprintln(a.stderr, "  gig env status <ticket-id> --path . [--envs dev=dev,test=test,prod=main]")
}

func (a *App) printEnvStatusUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig env status <ticket-id> --path . [--envs dev=dev,test=test,prod=main]")
}

func (a *App) printPlanUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig plan --ticket <ticket-id> --from <branch> --to <branch> --path . [--envs dev=dev,test=test,prod=main] [--format human|json]")
}

func (a *App) printVerifyUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig verify --ticket <ticket-id> --from <branch> --to <branch> --path . [--envs dev=dev,test=test,prod=main] [--format human|json]")
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
