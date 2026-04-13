package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gig/internal/output"
	"gig/internal/scm"
	"gig/internal/sourcecontrol"
	"gig/internal/workarea"
)

type commandScope struct {
	Workarea        *workarea.Definition
	WorkspacePath   string
	ConfigPath      string
	RepoSpec        string
	RepoInherited   bool
	PathInherited   bool
	ConfigInherited bool
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
		fmt.Fprintf(a.stderr, "unknown workarea subcommand %q\n\n", args[0])
		a.printWorkareaUsage()
		return usageExitCode
	}
}

func (a *App) runWorkareaList(args []string) int {
	if hasHelpFlag(args) {
		a.printWorkareaListUsage()
		return 0
	}

	fs := flag.NewFlagSet("workarea list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", string(outputFormatHuman), "Output format: human or json")
	if err := fs.Parse(args); err != nil {
		a.printWorkareaListUsage()
		return usageExitCode
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(a.stderr, "workarea list does not accept positional arguments")
		a.printWorkareaListUsage()
		return usageExitCode
	}

	selectedFormat, err := parseOutputFormat(*format)
	if err != nil {
		fmt.Fprintf(a.stderr, "workarea list failed: %v\n", err)
		return usageExitCode
	}

	store, err := workarea.NewStore()
	if err != nil {
		fmt.Fprintf(a.stderr, "workarea list failed: %v\n", err)
		return 1
	}
	workareas, current, err := store.List()
	if err != nil {
		fmt.Fprintf(a.stderr, "workarea list failed: %v\n", err)
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
			Command:   "workarea list",
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
		fmt.Fprintf(a.stderr, "workarea add failed: %v\n", err)
		a.printWorkareaAddUsage()
		return usageExitCode
	}
	args = reorderedArgs

	if hasHelpFlag(args) {
		a.printWorkareaAddUsage()
		return 0
	}

	fs := flag.NewFlagSet("workarea add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	path := fs.String("path", "", "Local workspace path for this workarea")
	configPath := fs.String("config", "", "Optional gig config file to keep with this workarea")
	repoTarget := fs.String("repo", "", "Remote repository target, for example github:owner/name")
	providerValue := fs.String("provider", "", "Provider to discover from when --repo is omitted")
	organizationValue := fs.String("org", "", "Owner, group, workspace, or organization filter for discovery")
	projectValue := fs.String("project", "", "Project filter for Azure DevOps discovery")
	fromBranch := fs.String("from", "", "Default source branch")
	toBranch := fs.String("to", "", "Default target branch")
	envsSpec := fs.String("envs", "", "Default environments, for example dev=dev,test=test,prod=main")
	useNow := fs.Bool("use", false, "Make this the current workarea after saving")

	if err := fs.Parse(args); err != nil {
		a.printWorkareaAddUsage()
		return usageExitCode
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(a.stderr, "workarea add accepts at most one <name> argument")
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
			fmt.Fprintf(a.stderr, "workarea add failed: %v\n", err)
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
			fmt.Fprintf(a.stderr, "workarea add failed: %v\n", err)
			return 1
		}
	}
	if strings.TrimSpace(*configPath) != "" {
		*configPath, err = normalizeCLIPath(*configPath)
		if err != nil {
			fmt.Fprintf(a.stderr, "workarea add failed: %v\n", err)
			return 1
		}
	}
	if name == "" {
		name = inferWorkareaName(repoSpec, resolvedPath)
	}
	if name == "" {
		fmt.Fprintln(a.stderr, "workarea add failed: a workarea name is required")
		a.printWorkareaAddUsage()
		return usageExitCode
	}
	if repoSpec != "" {
		if _, err := sourcecontrol.ParseRepositoryTargets(repoSpec); err != nil {
			fmt.Fprintf(a.stderr, "workarea add failed: %v\n", err)
			return usageExitCode
		}
	}

	store, err := workarea.NewStore()
	if err != nil {
		fmt.Fprintf(a.stderr, "workarea add failed: %v\n", err)
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
		fmt.Fprintf(a.stderr, "workarea add failed: %v\n", err)
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
		fmt.Fprintln(a.stderr, "workarea use accepts at most one <name> argument")
		a.printWorkareaUseUsage()
		return usageExitCode
	}

	store, err := workarea.NewStore()
	if err != nil {
		fmt.Fprintf(a.stderr, "workarea use failed: %v\n", err)
		return 1
	}

	name := ""
	if len(args) == 1 {
		name = strings.TrimSpace(args[0])
	} else {
		name, err = a.promptForWorkareaSelection(store)
		if err != nil {
			fmt.Fprintf(a.stderr, "workarea use failed: %v\n", err)
			return 1
		}
	}

	selected, err := store.Use(name)
	if err != nil {
		fmt.Fprintf(a.stderr, "workarea use failed: %v\n", err)
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
		fmt.Fprintln(a.stderr, "workarea show accepts at most one <name> argument")
		a.printWorkareaShowUsage()
		return usageExitCode
	}

	store, err := workarea.NewStore()
	if err != nil {
		fmt.Fprintf(a.stderr, "workarea show failed: %v\n", err)
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
			fmt.Fprintf(a.stderr, "workarea show failed: %v\n", err)
			return 1
		}
		if !ok {
			fmt.Fprintf(a.stderr, "workarea show failed: workarea %q was not found\n", strings.TrimSpace(args[0]))
			return 1
		}
		current, currentOK, err := store.Current()
		if err != nil {
			fmt.Fprintf(a.stderr, "workarea show failed: %v\n", err)
			return 1
		}
		isCurrent = currentOK && strings.EqualFold(current.Name, selected.Name)
	} else {
		selected, ok, err = store.Current()
		if err != nil {
			fmt.Fprintf(a.stderr, "workarea show failed: %v\n", err)
			return 1
		}
		if !ok {
			fmt.Fprintln(a.stderr, "workarea show failed: no current workarea is selected")
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
	workareas, current, err := store.List()
	if err != nil {
		return "", err
	}
	if len(workareas) == 0 {
		return "", fmt.Errorf("no workareas saved yet")
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

	selected, err := a.runPicker(bufio.NewReader(a.stdin), "Select a workarea:", items)
	if err != nil {
		return "", err
	}
	return selected.Value, nil
}

func (a *App) discoverWorkareaRepository(ctx context.Context, providerValue, organization, project string) (scm.Repository, error) {
	reader := bufio.NewReader(a.stdin)

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
	}, a.stdin, a.stdout, a.stderr)
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
	fmt.Fprintln(a.stdout, "Select a provider:")
	for index, provider := range providers {
		fmt.Fprintf(a.stdout, "  %d. %s\n", index+1, sourcecontrol.ProviderLabel(provider))
	}
	fmt.Fprint(a.stdout, "Choice: ")

	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return "", fmt.Errorf("a selection is required")
	}
	index, err := strconv.Atoi(line)
	if err != nil {
		return "", fmt.Errorf("invalid selection %q", line)
	}
	if index < 1 || index > len(providers) {
		return "", fmt.Errorf("selection %d is out of range", index)
	}
	return providers[index-1], nil
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

func (a *App) resolveCommandScope(pathValue, configValue, repoValue, workareaName string, pathExplicit, configExplicit, repoExplicit bool) (commandScope, error) {
	store, err := workarea.NewStore()
	if err != nil {
		return commandScope{}, err
	}

	var selected *workarea.Definition
	if name := strings.TrimSpace(workareaName); name != "" {
		definition, ok, err := store.Get(name)
		if err != nil {
			return commandScope{}, err
		}
		if !ok {
			return commandScope{}, fmt.Errorf("workarea %q was not found", name)
		}
		if touched, err := store.Use(definition.Name); err == nil {
			definition = touched
		}
		selected = &definition
	} else if !pathExplicit && !repoExplicit {
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
		Workarea:        selected,
		WorkspacePath:   workspacePath,
		ConfigPath:      configPath,
		RepoSpec:        repoSpec,
		RepoInherited:   repoInherited,
		PathInherited:   pathInherited,
		ConfigInherited: configInherited,
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

func (a *App) announceWorkareaSelection(scope commandScope, defaults commandDefaults) {
	if scope.Workarea == nil {
		return
	}

	target := scope.RepoSpec
	if target == "" {
		target = scope.WorkspacePath
	}

	message := fmt.Sprintf("Using workarea %s (%s)", scope.Workarea.Name, target)
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

func (a *App) runPicker(reader *bufio.Reader, heading string, items []pickerItem) (pickerItem, error) {
	if len(items) == 0 {
		return pickerItem{}, fmt.Errorf("no choices are available")
	}

	filtered := append([]pickerItem(nil), items...)
	for {
		fmt.Fprintln(a.stdout, heading)
		for index, item := range filtered {
			badges := make([]string, 0, 2)
			if item.Current {
				badges = append(badges, "current")
			}
			if item.Recent {
				badges = append(badges, "recent")
			}
			fmt.Fprintf(a.stdout, "  %d. %s", index+1, item.Title)
			if len(badges) > 0 {
				fmt.Fprintf(a.stdout, " [%s]", strings.Join(badges, ", "))
			}
			if strings.TrimSpace(item.Subtitle) != "" {
				fmt.Fprintf(a.stdout, "  %s", item.Subtitle)
			}
			fmt.Fprintln(a.stdout)
		}
		fmt.Fprint(a.stdout, "Choice or filter text ('/' clears): ")

		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return pickerItem{}, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return pickerItem{}, fmt.Errorf("a selection is required")
		}
		if line == "/" {
			filtered = append([]pickerItem(nil), items...)
			continue
		}
		index, err := strconv.Atoi(line)
		if err == nil {
			if index < 1 || index > len(filtered) {
				return pickerItem{}, fmt.Errorf("selection %d is out of range", index)
			}
			return filtered[index-1], nil
		}

		next := filterPickerItems(items, line)
		if len(next) == 0 {
			fmt.Fprintf(a.stdout, "No matches for %q.\n", line)
			continue
		}
		if len(next) == 1 {
			return next[0], nil
		}
		filtered = next
	}
}

func filterPickerItems(items []pickerItem, query string) []pickerItem {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return append([]pickerItem(nil), items...)
	}

	filtered := make([]pickerItem, 0, len(items))
	for _, item := range items {
		fields := []string{item.Title, item.Subtitle, item.Value}
		fields = append(fields, item.Keywords...)
		for _, field := range fields {
			if strings.Contains(strings.ToLower(strings.TrimSpace(field)), query) {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
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
	fmt.Fprintln(a.stderr, "Usage:")
	fmt.Fprintln(a.stderr, "  gig workarea list [--format human|json]")
	fmt.Fprintln(a.stderr, "  gig workarea add [<name>] [--repo <provider-target>] [--provider github|gitlab|bitbucket|azure-devops] [--org owner] [--project name] [--path /path/to/workspace] [--config gig.yaml] [--from <branch>] [--to <branch>] [--envs dev=dev,test=test,prod=main] [--use]")
	fmt.Fprintln(a.stderr, "  gig workarea use [<name>]")
	fmt.Fprintln(a.stderr, "  gig workarea show [<name>]")
}

func (a *App) printWorkareaListUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig workarea list [--format human|json]")
}

func (a *App) printWorkareaAddUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig workarea add [<name>] [--repo <provider-target>] [--provider github|gitlab|bitbucket|azure-devops] [--org owner] [--project name] [--path /path/to/workspace] [--config gig.yaml] [--from <branch>] [--to <branch>] [--envs dev=dev,test=test,prod=main] [--use]")
}

func (a *App) printWorkareaUseUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig workarea use [<name>]")
}

func (a *App) printWorkareaShowUsage() {
	fmt.Fprintln(a.stderr, "Usage: gig workarea show [<name>]")
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
	fmt.Fprintf(a.stderr, "Current workarea: %s (%s)\n", current.Name, target)
}
