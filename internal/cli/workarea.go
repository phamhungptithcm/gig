package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	inspectsvc "gig/internal/inspect"
	"gig/internal/output"
	"gig/internal/scm"
	"gig/internal/sourcecontrol"
	"gig/internal/workarea"
)

type commandScope struct {
	Workarea                 *workarea.Definition
	WorkspacePath            string
	ConfigPath               string
	RepoSpec                 string
	RepoInherited            bool
	RepoInferredFromCheckout bool
	CheckoutBranch           string
	PathInherited            bool
	ConfigInherited          bool
}

type commandDefaults struct {
	EnvironmentSpec string
	FromBranch      string
	ToBranch        string
}

type pickerItem struct {
	Value    string
	Title    string
	Subtitle string
	Keywords []string
	Current  bool
	Recent   bool
}

func (a *App) runWorkarea(ctx context.Context, args []string) int {
	_ = ctx
	if len(args) == 0 || hasHelpFlag(args) {
		a.printWorkareaUsage()
		return 0
	}

	switch args[0] {
	case "list":
		return a.runWorkareaList(args[1:])
	case "add":
		return a.runWorkareaAdd(ctx, args[1:])
	case "use":
		return a.runWorkareaUse(args[1:])
	case "show", "current":
		return a.runWorkareaShow(args[1:])
	default:
		fmt.Fprintf(a.stderr, "unknown project subcommand %q\n\n", args[0])
		a.printWorkareaUsage()
		return usageExitCode
	}
}

func (a *App) runWorkareaList(args []string) int {
	if hasHelpFlag(args) {
		a.printWorkareaListUsage()
		return 0
	}

	fs := flag.NewFlagSet("project list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", string(outputFormatHuman), "Output format: human or json")
	jsonOutput := fs.Bool("json", false, "Print JSON output")
	if err := fs.Parse(args); err != nil {
		a.printWorkareaListUsage()
		return usageExitCode
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(a.stderr, "project list does not accept positional arguments")
		a.printWorkareaListUsage()
		return usageExitCode
	}

	selectedFormat, err := parseOutputFormat(resolveFormatAlias(*format, *jsonOutput))
	if err != nil {
		fmt.Fprintf(a.stderr, "project list failed: %v\n", err)
		return usageExitCode
	}

	store, err := workarea.NewStore()
	if err != nil {
		fmt.Fprintf(a.stderr, "project list failed: %v\n", err)
		return 1
	}
	workareas, current, err := store.List()
	if err != nil {
		fmt.Fprintf(a.stderr, "project list failed: %v\n", err)
		return 1
	}
	workareas = orderedWorkareas(workareas, current)

	switch selectedFormat {
	case outputFormatHuman:
		if err := output.RenderWorkareaList(a.stdout, current, workareas); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	case outputFormatJSON:
		if err := output.RenderJSON(a.stdout, struct {
			Command   string                `json:"command"`
			Current   string                `json:"current,omitempty"`
			Workareas []workarea.Definition `json:"workareas"`
		}{
			Command:   "project list",
			Current:   current,
			Workareas: workareas,
		}); err != nil {
			fmt.Fprintf(a.stderr, "render failed: %v\n", err)
			return 1
		}
	}

	return 0
}

func (a *App) runWorkareaAdd(ctx context.Context, args []string) int {
	reorderedArgs, err := reorderArgsWithSinglePositional(args, "-path", "--path", "-config", "--config", "-repo", "--repo", "-from", "--from", "-to", "--to", "-envs", "--envs", "-provider", "--provider", "-org", "--org", "-project", "--project")
	if err != nil {
		fmt.Fprintf(a.stderr, "project add failed: %v\n", err)
		a.printWorkareaAddUsage()
		return usageExitCode
	}
	args = reorderedArgs

	if hasHelpFlag(args) {
		a.printWorkareaAddUsage()
		return 0
	}

	fs := flag.NewFlagSet("project add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", "", "Local workspace path for this project")
	configPath := fs.String("config", "", optionalOverrideFileHelp)
	repoTarget := fs.String("repo", "", "Remote repository target, for example github:owner/name")
	providerValue := fs.String("provider", "", "Provider to discover from when --repo is omitted")
	organizationValue := fs.String("org", "", "Owner, group, workspace, or organization filter for discovery")
	projectValue := fs.String("project", "", "Project filter for Azure DevOps discovery")
	fromBranch := fs.String("from", "", "Default source branch")
	toBranch := fs.String("to", "", "Default target branch")
	envsSpec := fs.String("envs", "", "Default environments, for example dev=dev,test=test,prod=main")
	useNow := fs.Bool("use", false, "Make this the current project after saving")

	if err := fs.Parse(args); err != nil {
		a.printWorkareaAddUsage()
		return usageExitCode
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(a.stderr, "project add accepts at most one <name> argument")
		a.printWorkareaAddUsage()
		return usageExitCode
	}

	name := strings.TrimSpace("")
	if fs.NArg() == 1 {
		name = strings.TrimSpace(fs.Arg(0))
	}

	repoSpec := strings.TrimSpace(*repoTarget)
	resolvedPath := strings.TrimSpace(*path)
	if repoSpec == "" && resolvedPath == "" {
		discoveredRepository, err := a.discoverWorkareaRepository(ctx, strings.TrimSpace(*providerValue), strings.TrimSpace(*organizationValue), strings.TrimSpace(*projectValue))
		if err != nil {
			if errors.Is(err, errPickerCancelled) {
				fmt.Fprintln(a.stdout, "Project setup cancelled.")
				return 0
			}
			fmt.Fprintf(a.stderr, "project add failed: %v\n", err)
			return 1
		}
		repoSpec = discoveredRepository.Root
		if name == "" {
			name = discoveredRepository.Name
		}
	}
	if resolvedPath != "" {
		resolvedPath, err = normalizeCLIPath(resolvedPath)
		if err != nil {
			fmt.Fprintf(a.stderr, "project add failed: %v\n", err)
			return 1
		}
	}
	if strings.TrimSpace(*configPath) != "" {
		*configPath, err = normalizeCLIPath(*configPath)
		if err != nil {
			fmt.Fprintf(a.stderr, "project add failed: %v\n", err)
			return 1
		}
	}
	if name == "" {
		name = inferWorkareaName(repoSpec, resolvedPath)
	}
	if name == "" {
		fmt.Fprintln(a.stderr, "project add failed: a project name is required")
		a.printWorkareaAddUsage()
		return usageExitCode
	}
	if repoSpec != "" {
		if _, err := sourcecontrol.ParseRepositoryTargets(repoSpec); err != nil {
			fmt.Fprintf(a.stderr, "project add failed: %v\n", err)
			return usageExitCode
		}
	}

	store, err := workarea.NewStore()
	if err != nil {
		fmt.Fprintf(a.stderr, "project add failed: %v\n", err)
		return 1
	}

	saved, err := store.Save(workarea.Definition{
		Name:            name,
		RepoTarget:      repoSpec,
		Path:            resolvedPath,
		ConfigPath:      strings.TrimSpace(*configPath),
		FromBranch:      strings.TrimSpace(*fromBranch),
		ToBranch:        strings.TrimSpace(*toBranch),
		EnvironmentSpec: strings.TrimSpace(*envsSpec),
	}, *useNow)
	if err != nil {
		fmt.Fprintf(a.stderr, "project add failed: %v\n", err)
		return 1
	}

	if err := output.RenderWorkareaDetail(a.stdout, saved, store.ScopePath(saved), *useNow); err != nil {
		fmt.Fprintf(a.stderr, "render failed: %v\n", err)
		return 1
	}
	if repository, err := parseSingleRepository(saved.RepoTarget); err == nil {
		_ = store.RecordRepositorySelection(repository)
	}
	if *useNow {
		fmt.Fprintln(a.stdout, "Status: current")
	} else {
		fmt.Fprintln(a.stdout, "Status: saved")
	}
	return 0
}

func (a *App) runWorkareaUse(args []string) int {
	if hasHelpFlag(args) {
		a.printWorkareaUseUsage()
		return 0
	}
	if len(args) > 1 {
		fmt.Fprintln(a.stderr, "project use accepts at most one <name> argument")
		a.printWorkareaUseUsage()
		return usageExitCode
	}

	store, err := workarea.NewStore()
	if err != nil {
		fmt.Fprintf(a.stderr, "project use failed: %v\n", err)
		return 1
	}

	name := ""
	if len(args) == 1 {
		name = strings.TrimSpace(args[0])
	} else {
		name, err = a.promptForWorkareaSelection(store)
		if err != nil {
			if errors.Is(err, errPickerCancelled) {
				fmt.Fprintln(a.stdout, "Project switch cancelled.")
				return 0
			}
			fmt.Fprintf(a.stderr, "project use failed: %v\n", err)
			return 1
		}
	}

	selected, err := store.Use(name)
	if err != nil {
		fmt.Fprintf(a.stderr, "project use failed: %v\n", err)
		return 1
	}
	if err := output.RenderWorkareaDetail(a.stdout, selected, store.ScopePath(selected), true); err != nil {
		fmt.Fprintf(a.stderr, "render failed: %v\n", err)
		return 1
	}
	if repository, err := parseSingleRepository(selected.RepoTarget); err == nil {
		_ = store.RecordRepositorySelection(repository)
	}
	return 0
}

func (a *App) runWorkareaShow(args []string) int {
	if hasHelpFlag(args) {
		a.printWorkareaShowUsage()
		return 0
	}
	if len(args) > 1 {
		fmt.Fprintln(a.stderr, "project show accepts at most one <name> argument")
		a.printWorkareaShowUsage()
		return usageExitCode
	}

	store, err := workarea.NewStore()
	if err != nil {
		fmt.Fprintf(a.stderr, "project show failed: %v\n", err)
		return 1
	}

	var (
		selected  workarea.Definition
		ok        bool
		isCurrent bool
	)
	if len(args) == 1 {
		selected, ok, err = store.Get(args[0])
		if err != nil {
			fmt.Fprintf(a.stderr, "project show failed: %v\n", err)
			return 1
		}
		if !ok {
			fmt.Fprintf(a.stderr, "project show failed: project %q was not found\n", strings.TrimSpace(args[0]))
			return 1
		}
		current, currentOK, err := store.Current()
		if err != nil {
			fmt.Fprintf(a.stderr, "project show failed: %v\n", err)
			return 1
		}
		isCurrent = currentOK && strings.EqualFold(current.Name, selected.Name)
	} else {
		selected, ok, err = store.Current()
		if err != nil {
			fmt.Fprintf(a.stderr, "project show failed: %v\n", err)
			return 1
		}
		if !ok {
			fmt.Fprintln(a.stderr, "project show failed: no current project is selected")
			return 1
		}
		isCurrent = true
	}

	if err := output.RenderWorkareaDetail(a.stdout, selected, store.ScopePath(selected), isCurrent); err != nil {
		fmt.Fprintf(a.stderr, "render failed: %v\n", err)
		return 1
	}
	return 0
}

func (a *App) promptForWorkareaSelection(store *workarea.Store) (string, error) {
	return a.promptForWorkareaSelectionWithReader(bufio.NewReader(a.stdin), store)
}

func (a *App) promptForWorkareaSelectionWithReader(reader *bufio.Reader, store *workarea.Store) (string, error) {
	workareas, current, err := store.List()
	if err != nil {
		return "", err
	}
	if len(workareas) == 0 {
		return "", fmt.Errorf("no projects saved yet")
	}
	ordered := orderedWorkareas(workareas, current)
	items := make([]pickerItem, 0, len(ordered))
	for _, definition := range ordered {
		target := definition.RepoTarget
		if target == "" {
			target = definition.Path
		}
		items = append(items, pickerItem{
			Value:    definition.Name,
			Title:    definition.Name,
			Subtitle: target,
			Keywords: []string{definition.Name, target},
			Current:  strings.EqualFold(definition.Name, current),
			Recent:   !definition.LastUsedAt.IsZero() && !strings.EqualFold(definition.Name, current),
		})
	}

	selected, err := a.runPicker(reader, "Select a project:", items)
	if err != nil {
		return "", err
	}
	return selected.Value, nil
}

func (a *App) discoverWorkareaRepository(ctx context.Context, providerValue, organization, project string) (scm.Repository, error) {
	return a.discoverWorkareaRepositoryWithReader(ctx, bufio.NewReader(a.stdin), providerValue, organization, project)
}

func (a *App) discoverWorkareaRepositoryWithReader(ctx context.Context, reader *bufio.Reader, providerValue, organization, project string) (scm.Repository, error) {
	provider, err := a.resolveDiscoveryProvider(reader, providerValue)
	if err != nil {
		return scm.Repository{}, err
	}
	if provider == scm.TypeAzureDevOps && strings.TrimSpace(organization) == "" {
		organization, err = a.promptForLine(reader, "Azure DevOps organization")
		if err != nil {
			return scm.Repository{}, err
		}
	}

	repositories, err := sourcecontrol.DiscoverRepositories(ctx, provider, sourcecontrol.RepositoryDiscoveryOptions{
		Organization: organization,
		Project:      project,
	}, reader, a.stdout, a.stderr)
	if err != nil {
		return scm.Repository{}, err
	}
	if len(repositories) == 0 {
		return scm.Repository{}, fmt.Errorf("no repositories were discovered for %s", sourcecontrol.ProviderLabel(provider))
	}

	store, err := workarea.NewStore()
	if err != nil {
		return scm.Repository{}, err
	}
	recentRepositories, err := store.RecentRepositories(provider, 10)
	if err != nil {
		return scm.Repository{}, err
	}
	repositories = orderRepositoriesForPicker(repositories, recentRepositories)

	items := make([]pickerItem, 0, len(repositories))
	for _, repository := range repositories {
		items = append(items, pickerItem{
			Value:    repository.Root,
			Title:    repository.Name,
			Subtitle: repository.Root,
			Keywords: []string{repository.Name, repository.Root},
			Recent:   containsRecentRepository(recentRepositories, repository.Root),
		})
	}

	selected, err := a.runPicker(reader, fmt.Sprintf("Select a %s repository:", sourcecontrol.ProviderLabel(provider)), items)
	if err != nil {
		return scm.Repository{}, err
	}
	for _, repository := range repositories {
		if repository.Root == selected.Value {
			return repository, nil
		}
	}
	return scm.Repository{}, fmt.Errorf("selected repository %q was not found", selected.Value)
}

func (a *App) resolveDiscoveryProvider(reader *bufio.Reader, providerValue string) (scm.Type, error) {
	if strings.TrimSpace(providerValue) != "" {
		provider, err := sourcecontrol.ParseProvider(providerValue)
		if err != nil {
			return "", err
		}
		if !sourcecontrol.SupportsRepositoryDiscovery(provider) {
			return "", fmt.Errorf("%s repository discovery is not supported", sourcecontrol.ProviderLabel(provider))
		}
		return provider, nil
	}

	providers := sourcecontrol.DiscoverableProviders()
	items := make([]pickerItem, 0, len(providers))
	for _, provider := range providers {
		subtitle := "Browse repositories from this provider."
		if provider == scm.TypeGitHub {
			subtitle = "Recommended first-run path. gig can list your GitHub repositories after login."
		}
		items = append(items, pickerItem{
			Value:    string(provider),
			Title:    sourcecontrol.ProviderLabel(provider),
			Subtitle: subtitle,
			Keywords: []string{string(provider), sourcecontrol.ProviderLabel(provider)},
		})
	}
	selected, err := a.runPicker(reader, "Pick a provider:", items)
	if err != nil {
		return "", err
	}
	return scm.Type(selected.Value), nil
}

func (a *App) promptForLine(reader *bufio.Reader, label string) (string, error) {
	fmt.Fprintf(a.stdout, "%s: ", label)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return "", fmt.Errorf("%s is required", strings.ToLower(label))
	}
	return line, nil
}

func (a *App) resolveCommandScope(ctx context.Context, pathValue, configValue, repoValue, workareaName string, pathExplicit, configExplicit, repoExplicit bool) (commandScope, error) {
	store, err := workarea.NewStore()
	if err != nil {
		return commandScope{}, err
	}

	var selected *workarea.Definition
	var checkoutRemote scm.Repository
	var checkoutLocal scm.Repository
	checkoutRemoteOK := false

	if name := strings.TrimSpace(workareaName); name != "" {
		definition, ok, err := store.Get(name)
		if err != nil {
			return commandScope{}, err
		}
		if !ok {
			return commandScope{}, fmt.Errorf("project %q was not found", name)
		}
		if touched, err := store.Use(definition.Name); err == nil {
			definition = touched
		}
		selected = &definition
	} else if !pathExplicit && !repoExplicit {
		checkoutRemote, checkoutLocal, checkoutRemoteOK = a.inferRemoteRepositoryFromCurrentCheckout(ctx)
	}
	if selected == nil && !checkoutRemoteOK && strings.TrimSpace(workareaName) == "" && !pathExplicit && !repoExplicit {
		definition, ok, err := store.Current()
		if err != nil {
			return commandScope{}, err
		}
		if ok {
			if touched, err := store.Touch(definition.Name); err == nil {
				definition = touched
			}
			selected = &definition
		}
	}

	repoSpec := strings.TrimSpace(repoValue)
	repoInherited := false
	repoInferredFromCheckout := false
	checkoutBranch := ""
	if repoSpec == "" && checkoutRemoteOK {
		repoSpec = checkoutRemote.Root
		repoInferredFromCheckout = true
		checkoutBranch = strings.TrimSpace(checkoutRemote.CurrentBranch)
	}
	if selected != nil && !repoExplicit && !pathExplicit && repoSpec == "" {
		repoSpec = selected.RepoTarget
		repoInherited = repoSpec != ""
	}

	configPath := strings.TrimSpace(configValue)
	configInherited := false
	if selected != nil && !configExplicit && configPath == "" {
		configPath = selected.ConfigPath
		configInherited = configPath != ""
	}
	if configPath != "" {
		configPath, err = normalizeCLIPath(configPath)
		if err != nil {
			return commandScope{}, err
		}
	}

	workspacePath := ""
	pathInherited := false
	switch {
	case pathExplicit:
		workspacePath, err = normalizeCLIPath(pathValue)
	case checkoutRemoteOK && strings.TrimSpace(checkoutLocal.Root) != "":
		workspacePath = checkoutLocal.Root
	case selected != nil && strings.TrimSpace(selected.Path) != "":
		workspacePath, err = normalizeCLIPath(selected.Path)
		pathInherited = true
	case selected != nil:
		workspacePath = store.ScopePath(*selected)
		err = os.MkdirAll(workspacePath, 0o755)
		pathInherited = true
	default:
		workspacePath, err = normalizeCLIPath(pathValue)
	}
	if err != nil {
		return commandScope{}, err
	}

	return commandScope{
		Workarea:                 selected,
		WorkspacePath:            workspacePath,
		ConfigPath:               configPath,
		RepoSpec:                 repoSpec,
		RepoInherited:            repoInherited,
		RepoInferredFromCheckout: repoInferredFromCheckout,
		CheckoutBranch:           checkoutBranch,
		PathInherited:            pathInherited,
		ConfigInherited:          configInherited,
	}, nil
}

func resolveCommandDefaults(selected *workarea.Definition, envSpec, fromBranch, toBranch string, envExplicit, fromExplicit, toExplicit bool) commandDefaults {
	defaults := commandDefaults{
		EnvironmentSpec: strings.TrimSpace(envSpec),
		FromBranch:      strings.TrimSpace(fromBranch),
		ToBranch:        strings.TrimSpace(toBranch),
	}
	if selected == nil {
		return defaults
	}
	if !envExplicit && defaults.EnvironmentSpec == "" {
		defaults.EnvironmentSpec = selected.EnvironmentSpec
	}
	if !fromExplicit && defaults.FromBranch == "" {
		defaults.FromBranch = selected.FromBranch
	}
	if !toExplicit && defaults.ToBranch == "" {
		defaults.ToBranch = selected.ToBranch
	}
	return defaults
}

func applyScopePromotionDefaults(scope commandScope, defaults commandDefaults) commandDefaults {
	if defaults.FromBranch == "" && scope.RepoInferredFromCheckout && isLikelyPromotionSourceBranch(scope.CheckoutBranch) {
		defaults.FromBranch = strings.TrimSpace(scope.CheckoutBranch)
	}
	return defaults
}

func isLikelyPromotionSourceBranch(branch string) bool {
	branch = strings.ToLower(strings.TrimSpace(branch))
	switch branch {
	case "dev", "develop", "development", "integration", "test", "qa", "uat", "staging", "stage", "preprod", "pre-prod":
		return true
	case "", "head", "main", "master", "prod", "production", "trunk":
		return false
	default:
		return strings.HasPrefix(branch, "release/") || strings.HasPrefix(branch, "rc/")
	}
}

func (a *App) rememberProjectMemory(scope commandScope, defaults commandDefaults, runtime commandRuntime, repositories []scm.Repository, environments []inspectsvc.Environment, fromBranch, toBranch string) {
	if !containsRemoteRepositories(repositories) {
		return
	}

	store, err := workarea.NewStore()
	if err != nil {
		return
	}

	workareaName := ""
	switch {
	case scope.Workarea != nil:
		workareaName = scope.Workarea.Name
	case strings.TrimSpace(scope.RepoSpec) != "":
		repository, err := parseSingleRepository(scope.RepoSpec)
		if err != nil {
			return
		}
		saved, _, err := store.EnsureRemoteRepository(repository)
		if err != nil {
			return
		}
		workareaName = saved.Name
		_ = store.RecordRepositorySelection(repository)
	default:
		return
	}

	if workareaName == "" || runtime.loaded.Found || strings.TrimSpace(scope.ConfigPath) != "" {
		return
	}

	environmentSpec := ""
	if scope.Workarea == nil || (scope.Workarea.EnvironmentSpec == "" && defaults.EnvironmentSpec == "") {
		environmentSpec = formatEnvironmentSpec(environments)
	}

	resolvedFromBranch := ""
	if scope.Workarea == nil || (scope.Workarea.FromBranch == "" && defaults.FromBranch == "") {
		resolvedFromBranch = strings.TrimSpace(fromBranch)
	}

	resolvedToBranch := ""
	if scope.Workarea == nil || (scope.Workarea.ToBranch == "" && defaults.ToBranch == "") {
		resolvedToBranch = strings.TrimSpace(toBranch)
	}

	if environmentSpec == "" && resolvedFromBranch == "" && resolvedToBranch == "" {
		return
	}
	_, _, _ = store.SaveInferredDefaults(workareaName, resolvedFromBranch, resolvedToBranch, environmentSpec)
}

func (a *App) announceWorkareaSelection(scope commandScope, defaults commandDefaults) {
	if scope.Workarea == nil && scope.RepoInferredFromCheckout {
		message := fmt.Sprintf("Using current checkout remote %s", scope.RepoSpec)
		if branch := strings.TrimSpace(scope.CheckoutBranch); branch != "" {
			message += fmt.Sprintf(" on %s", branch)
		}
		if defaults.FromBranch != "" || defaults.ToBranch != "" {
			message += fmt.Sprintf(", %s -> %s", blankIfEmpty(defaults.FromBranch, "auto"), blankIfEmpty(defaults.ToBranch, "auto"))
		}
		fmt.Fprintln(a.stderr, message)
		return
	}

	if scope.Workarea == nil {
		return
	}

	target := scope.RepoSpec
	if target == "" {
		target = scope.WorkspacePath
	}

	message := fmt.Sprintf("Using project %s (%s)", scope.Workarea.Name, target)
	if defaults.FromBranch != "" || defaults.ToBranch != "" {
		message += fmt.Sprintf(", %s -> %s", blankIfEmpty(defaults.FromBranch, "auto"), blankIfEmpty(defaults.ToBranch, "auto"))
	}
	fmt.Fprintln(a.stderr, message)
}

func blankIfEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func formatEnvironmentSpec(environments []inspectsvc.Environment) string {
	parts := make([]string, 0, len(environments))
	for _, environment := range environments {
		name := strings.TrimSpace(environment.Name)
		branch := strings.TrimSpace(environment.Branch)
		if name == "" || branch == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", name, branch))
	}
	return strings.Join(parts, ",")
}

func inferWorkareaName(repoSpec, pathValue string) string {
	if strings.TrimSpace(repoSpec) != "" {
		repositories, err := sourcecontrol.ParseRepositoryTargets(repoSpec)
		if err == nil && len(repositories) == 1 {
			return repositories[0].Name
		}
	}

	pathValue = strings.TrimSpace(pathValue)
	if pathValue != "" {
		base := filepath.Base(pathValue)
		if base != "." && base != string(filepath.Separator) {
			return base
		}
	}

	return ""
}

func orderedWorkareas(workareas []workarea.Definition, current string) []workarea.Definition {
	ordered := append([]workarea.Definition(nil), workareas...)
	sort.SliceStable(ordered, func(i, j int) bool {
		leftCurrent := strings.EqualFold(ordered[i].Name, current)
		rightCurrent := strings.EqualFold(ordered[j].Name, current)
		switch {
		case leftCurrent != rightCurrent:
			return leftCurrent
		case ordered[i].LastUsedAt.Equal(ordered[j].LastUsedAt):
			return strings.ToLower(ordered[i].Name) < strings.ToLower(ordered[j].Name)
		case ordered[i].LastUsedAt.IsZero():
			return false
		case ordered[j].LastUsedAt.IsZero():
			return true
		default:
			return ordered[i].LastUsedAt.After(ordered[j].LastUsedAt)
		}
	})
	return ordered
}

func orderRepositoriesForPicker(repositories []scm.Repository, recent []workarea.RecentRepository) []scm.Repository {
	ordered := append([]scm.Repository(nil), repositories...)
	recencyRank := make(map[string]int, len(recent))
	for index, repository := range recent {
		recencyRank[strings.TrimSpace(repository.Root)] = index
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		leftRank, leftOK := recencyRank[strings.TrimSpace(ordered[i].Root)]
		rightRank, rightOK := recencyRank[strings.TrimSpace(ordered[j].Root)]
		switch {
		case leftOK && rightOK:
			return leftRank < rightRank
		case leftOK != rightOK:
			return leftOK
		case strings.EqualFold(ordered[i].Name, ordered[j].Name):
			return strings.ToLower(strings.TrimSpace(ordered[i].Root)) < strings.ToLower(strings.TrimSpace(ordered[j].Root))
		default:
			return strings.ToLower(ordered[i].Name) < strings.ToLower(ordered[j].Name)
		}
	})
	return ordered
}

func containsRecentRepository(recent []workarea.RecentRepository, root string) bool {
	for _, repository := range recent {
		if strings.EqualFold(strings.TrimSpace(repository.Root), strings.TrimSpace(root)) {
			return true
		}
	}
	return false
}

func parseSingleRepository(repoSpec string) (scm.Repository, error) {
	repositories, err := sourcecontrol.ParseRepositoryTargets(repoSpec)
	if err != nil {
		return scm.Repository{}, err
	}
	if len(repositories) != 1 {
		return scm.Repository{}, fmt.Errorf("expected exactly one repository target")
	}
	return repositories[0], nil
}

func flagProvided(fs *flag.FlagSet, name string) bool {
	provided := false
	fs.Visit(func(flag *flag.Flag) {
		if flag.Name == name {
			provided = true
		}
	})
	return provided
}

func (a *App) printWorkareaUsage() {
	printHelpHeading(a.stderr, "gig project", "Save repo context so future commands stay short.")
	printHelpUsage(a.stderr,
		"gig project add [<name>] [--repo <target> | --provider github|gitlab|bitbucket|azure-devops] [--use]",
		"gig project use [<name>]",
		"gig project list [--format human|json] [--json]",
		"gig project show [<name>]",
	)
	printHelpCommands(a.stderr, "Inside prompt",
		"gig",
		"repo payments",
		"save payments",
	)
	printHelpCommands(a.stderr, "CLI form",
		"gig project add payments --repo github:owner/name --use",
		"gig project use payments",
	)
	printHelpRows(a.stderr, "Common flags",
		helpRow{Label: "--repo", Value: "Remember a live remote target"},
		helpRow{Label: "--provider", Value: "Browse provider repos when --repo is omitted"},
		helpRow{Label: "--from/--to", Value: "Optional branch defaults when inference needs help"},
		helpRow{Label: "--use", Value: "Make this the current project now"},
	)
	printHelpRows(a.stderr, "Alias", helpRow{Label: "workarea", Value: "gig workarea ..."})
}

func (a *App) printWorkareaListUsage() {
	printHelpUsage(a.stderr, "gig project list [--format human|json] [--json]")
	printHelpRows(a.stderr, "Alias", helpRow{Label: "workarea list", Value: "gig workarea list ..."})
}

func (a *App) printWorkareaAddUsage() {
	printHelpHeading(a.stderr, "gig project add", "Remember a project for shorter future commands.")
	printHelpUsage(a.stderr, "gig project add [<name>] [--repo <target> | --provider github|gitlab|bitbucket|azure-devops] [--use]")
	printHelpCommands(a.stderr, "Start here",
		"gig",
		"repo payments",
		"save payments",
		"gig project add payments --repo github:owner/name --use",
	)
	printHelpRows(a.stderr, "Common flags",
		helpRow{Label: "--repo", Value: "Remote target or pasted provider URL"},
		helpRow{Label: "--path", Value: "Local fallback workspace"},
		helpRow{Label: "--from/--to", Value: "Optional branch defaults when inference needs help"},
		helpRow{Label: "--envs", Value: "Optional environment mapping"},
		helpRow{Label: "--use", Value: "Make this the current project now"},
	)
	printHelpRows(a.stderr, "Alias", helpRow{Label: "workarea add", Value: "gig workarea add ..."})
}

func (a *App) printWorkareaUseUsage() {
	printHelpUsage(a.stderr, "gig project use [<name>]")
	printHelpCommands(a.stderr, "Examples", "gig project use payments", "use payments")
	printHelpRows(a.stderr, "Alias", helpRow{Label: "workarea use", Value: "gig workarea use ..."})
}

func (a *App) printWorkareaShowUsage() {
	printHelpUsage(a.stderr, "gig project show [<name>]")
	printHelpRows(a.stderr, "Alias", helpRow{Label: "workarea show", Value: "gig workarea show ..."})
}

func (a *App) printCurrentWorkareaHint() {
	store, err := workarea.NewStore()
	if err != nil {
		return
	}
	current, ok, err := store.Current()
	if err != nil || !ok {
		return
	}

	target := current.RepoTarget
	if target == "" {
		target = store.ScopePath(current)
	}

	fmt.Fprintln(a.stderr)
	fmt.Fprintf(a.stderr, "Current project: %s (%s)\n", current.Name, target)
}
