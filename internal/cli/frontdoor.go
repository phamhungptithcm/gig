package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gig/internal/buildinfo"
	"gig/internal/output"
	"gig/internal/scm"
	"gig/internal/sourcecontrol"
	"gig/internal/workarea"

	"golang.org/x/term"
)

func (a *App) runFrontDoor(ctx context.Context) int {
	store, err := workarea.NewStore()
	if err != nil {
		fmt.Fprintf(a.stderr, "front door failed: %v\n", err)
		a.printRootUsage()
		return 1
	}

	workareas, _, err := store.List()
	if err != nil {
		fmt.Fprintf(a.stderr, "front door failed: %v\n", err)
		a.printRootUsage()
		return 1
	}

	var current *workarea.Definition
	if definition, ok, err := store.Current(); err == nil && ok {
		current = &definition
	}

	state := a.buildFrontDoorState(ctx, current, workareas)
	if err := output.RenderFrontDoor(a.stdout, state); err != nil {
		fmt.Fprintf(a.stderr, "render failed: %v\n", err)
		return 1
	}
	if a.frontDoorPromptEnabled() {
		reader := bufio.NewReader(a.stdin)
		exitCode, err := a.runFrontDoorPalette(ctx, reader, store, current, workareas)
		if err != nil {
			fmt.Fprintf(a.stderr, "front door failed: %v\n", err)
			return 1
		}
		if exitCode >= 0 {
			return exitCode
		}
	}

	return 0
}

func (a *App) frontDoorPromptEnabled() bool {
	if file, ok := a.stdin.(*os.File); ok {
		return term.IsTerminal(int(file.Fd()))
	}

	type lenReader interface {
		Len() int
	}
	if reader, ok := a.stdin.(lenReader); ok {
		return reader.Len() > 0
	}

	return false
}

func (a *App) buildFrontDoorState(ctx context.Context, current *workarea.Definition, workareas []workarea.Definition) output.FrontDoorState {
	state := output.FrontDoorState{
		Current:   current,
		Workareas: workareas,
		Version:   buildinfo.Version,
	}

	if current != nil {
		state.HeroStatus = "current project ready"
		state.Prompt = "ask gig > ABC-123"
		state.Examples = []string{
			"verify ABC-123",
			"manifest ABC-123",
			"switch",
		}
		if provider, label, ok := frontDoorWorkareaProvider(*current); ok {
			providerStatus := a.frontDoorProviderStatus(ctx, provider)
			state.HeroStatus = formatFrontDoorProviderStatus(label, providerStatus)
			state.StatusRows = append(state.StatusRows,
				output.KeyValue{Label: "Mode", Value: "current project"},
				output.KeyValue{Label: "Provider", Value: formatFrontDoorProviderStatus(label, providerStatus)},
			)
			if !providerStatus.Ready && providerStatus.Detail != "" {
				state.Examples = []string{
					"login " + strings.ToLower(string(provider)),
					"ABC-123",
					"verify ABC-123",
				}
			}
		} else {
			state.StatusRows = append(state.StatusRows,
				output.KeyValue{Label: "Mode", Value: "current project"},
				output.KeyValue{Label: "Provider", Value: "local workspace"},
			)
		}
	} else {
		githubStatus := a.frontDoorProviderStatus(ctx, scm.TypeGitHub)
		state.HeroStatus = "no project selected yet"
		state.StatusRows = append(state.StatusRows,
			output.KeyValue{Label: "Mode", Value: "new session"},
			output.KeyValue{Label: "Provider", Value: formatFrontDoorProviderStatus(sourcecontrol.ProviderLabel(scm.TypeGitHub), githubStatus)},
		)
		state.Prompt = "ask gig > repo github:owner/name ABC-123"
		state.Examples = []string{
			"ABC-123",
			"repo github:owner/name ABC-123",
			"login github",
		}
		if githubStatus.Ready {
			state.Prompt = "ask gig > ABC-123"
			state.Examples = []string{
				"ABC-123",
				"repo github:owner/name ABC-123",
				"verify ABC-123 github:owner/name",
			}
		}
	}

	if len(workareas) > 0 {
		state.StatusRows = append(state.StatusRows, output.KeyValue{Label: "Saved", Value: fmt.Sprintf("%d project(s)", len(workareas))})
	}

	return state
}

func (a *App) frontDoorProviderStatus(ctx context.Context, provider scm.Type) sourcecontrol.ProviderStatus {
	if !a.terminalPickerEnabled() {
		return sourcecontrol.ProviderStatus{Provider: provider, Detail: "not checked"}
	}
	return sourcecontrol.CheckProviderStatus(ctx, provider, a.stdin)
}

func formatFrontDoorProviderStatus(label string, status sourcecontrol.ProviderStatus) string {
	detail := strings.TrimSpace(status.Detail)
	switch {
	case detail == "":
		return label
	case detail == "not checked":
		return label + " first-run path"
	default:
		return fmt.Sprintf("%s %s", label, detail)
	}
}

func frontDoorWorkareaProvider(definition workarea.Definition) (scm.Type, string, bool) {
	repoTarget := strings.TrimSpace(definition.RepoTarget)
	if repoTarget == "" {
		return "", "", false
	}
	repositories, err := sourcecontrol.ParseRepositoryTargets(repoTarget)
	if err != nil || len(repositories) != 1 {
		return "", "", false
	}
	return repositories[0].Type, sourcecontrol.ProviderLabel(repositories[0].Type), true
}

type frontDoorAction string

const (
	frontDoorActionPicker                frontDoorAction = "picker"
	frontDoorActionInspect               frontDoorAction = "inspect"
	frontDoorActionVerify                frontDoorAction = "verify"
	frontDoorActionManifest              frontDoorAction = "manifest"
	frontDoorActionLogin                 frontDoorAction = "login"
	frontDoorActionDiscoverGitHub        frontDoorAction = "discover-github"
	frontDoorActionEnterTarget           frontDoorAction = "enter-target"
	frontDoorActionUseCurrentFolder      frontDoorAction = "use-current-folder"
	frontDoorActionDiscoverOtherProvider frontDoorAction = "discover-other-provider"
	frontDoorActionSavedWorkarea         frontDoorAction = "saved-workarea"
	frontDoorActionSwitchWorkarea        frontDoorAction = "switch-workarea"
)

type frontDoorCommand struct {
	Action     frontDoorAction
	TicketID   string
	RepoTarget string
	Provider   string
}

func (a *App) runFrontDoorPalette(ctx context.Context, reader *bufio.Reader, store *workarea.Store, current *workarea.Definition, workareas []workarea.Definition) (int, error) {
	fmt.Fprintln(a.stdout)
	fmt.Fprint(a.stdout, "ask gig > ")

	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return -1, err
	}

	command, err := parseFrontDoorCommand(line, current != nil, len(workareas) > 0)
	if err != nil {
		return -1, err
	}

	switch command.Action {
	case frontDoorActionPicker:
		if current != nil {
			return a.runFrontDoorCurrentProjectActionWithReader(ctx, reader)
		}
		return a.runFrontDoorActionWithoutCurrentProject(ctx, reader, store, workareas, frontDoorActionInspect, "")
	case frontDoorActionLogin:
		return a.runLogin(ctx, []string{command.Provider}), nil
	case frontDoorActionDiscoverGitHub, frontDoorActionEnterTarget, frontDoorActionUseCurrentFolder, frontDoorActionDiscoverOtherProvider, frontDoorActionSavedWorkarea:
		return a.runFrontDoorActionWithoutCurrentProject(ctx, reader, store, workareas, frontDoorActionInspect, command.TicketID, command.Action)
	case frontDoorActionSwitchWorkarea:
		if current == nil {
			if len(workareas) == 0 {
				return -1, fmt.Errorf("no saved projects are available yet")
			}
			name, err := a.promptForWorkareaSelectionWithReader(reader, store)
			if err != nil {
				if errors.Is(err, errPickerCancelled) {
					return 0, nil
				}
				return -1, err
			}
			if _, err := store.Use(name); err != nil {
				return -1, err
			}
			fmt.Fprintf(a.stdout, "Current project switched to %s.\n", name)
			return 0, nil
		}
		return a.runFrontDoorCurrentProjectActionWithReader(ctx, reader)
	case frontDoorActionInspect, frontDoorActionVerify, frontDoorActionManifest:
		if current == nil && strings.TrimSpace(command.RepoTarget) == "" {
			return a.runFrontDoorActionWithoutCurrentProject(ctx, reader, store, workareas, command.Action, command.TicketID)
		}
		ticketID, err := a.resolveFrontDoorTicketID(reader, command.TicketID)
		if err != nil {
			return -1, err
		}
		if strings.TrimSpace(command.RepoTarget) != "" {
			return a.executeFrontDoorAction(ctx, command.Action, ticketID, command.RepoTarget, ""), nil
		}
		return a.executeFrontDoorAction(ctx, command.Action, ticketID, "", ""), nil
	default:
		return -1, fmt.Errorf("unsupported front-door action %q", command.Action)
	}
}

func (a *App) runFrontDoorQuickStart(ctx context.Context, store *workarea.Store, workareas []workarea.Definition) (int, error) {
	return a.runFrontDoorActionWithoutCurrentProject(ctx, bufio.NewReader(a.stdin), store, workareas, frontDoorActionInspect, "")
}

func (a *App) runFrontDoorActionWithoutCurrentProject(ctx context.Context, reader *bufio.Reader, store *workarea.Store, workareas []workarea.Definition, action frontDoorAction, presetTicket string, preferredModes ...frontDoorAction) (int, error) {
	selectedAction := frontDoorActionPicker
	if len(preferredModes) > 0 {
		selectedAction = preferredModes[0]
	}

	items := []pickerItem{
		{
			Value:    "discover-github",
			Title:    "Pick a GitHub repository",
			Subtitle: "Recommended first run. gig can browse your GitHub repos after login.",
			Keywords: []string{"github", "recommended", "remote"},
		},
		{
			Value:    "enter-target",
			Title:    "Paste a repository target",
			Subtitle: "Use a target like github:owner/name when you already know the repo.",
			Keywords: []string{"repo", "target", "github:owner/name"},
		},
		{
			Value:    "use-current-folder",
			Title:    "Use the current folder",
			Subtitle: "Local Git or SVN fallback if you already have the code checked out.",
			Keywords: []string{"local", "path", "folder", "workspace"},
		},
		{
			Value:    "discover-other-provider",
			Title:    "Pick from another provider",
			Subtitle: "GitLab, Bitbucket, Azure DevOps, or another supported remote source.",
			Keywords: []string{"gitlab", "bitbucket", "azure", "provider"},
		},
	}
	if len(workareas) > 0 {
		items = append(items, pickerItem{
			Value:    "saved-workarea",
			Title:    "Use a saved project",
			Subtitle: "Reuse a remembered repo so you can skip provider and branch setup.",
			Keywords: []string{"workarea", "saved", "project"},
			Recent:   true,
		})
	}

	if selectedAction == frontDoorActionPicker {
		fmt.Fprintln(a.stdout)
		selected, err := a.runPicker(reader, "How should gig look this up?", items)
		if err != nil {
			if errors.Is(err, errPickerCancelled) {
				return 0, nil
			}
			return -1, err
		}
		selectedAction = frontDoorAction(selected.Value)
	}

	switch selectedAction {
	case frontDoorActionDiscoverGitHub:
		repository, err := a.discoverWorkareaRepositoryWithReader(ctx, reader, "github", "", "")
		if err != nil {
			if errors.Is(err, errPickerCancelled) {
				return 0, nil
			}
			return -1, err
		}
		ticketID, err := a.resolveFrontDoorTicketID(reader, presetTicket)
		if err != nil {
			return -1, err
		}
		return a.executeFrontDoorAction(ctx, action, ticketID, repository.Root, ""), nil
	case frontDoorActionEnterTarget:
		fmt.Fprintln(a.stdout, "Repository target example: github:owner/name")
		repoTarget, err := a.promptForLine(reader, "Repository target")
		if err != nil {
			return -1, err
		}
		if _, err := sourcecontrol.ParseRepositoryTargets(repoTarget); err != nil {
			return -1, err
		}
		ticketID, err := a.resolveFrontDoorTicketID(reader, presetTicket)
		if err != nil {
			return -1, err
		}
		return a.executeFrontDoorAction(ctx, action, ticketID, repoTarget, ""), nil
	case frontDoorActionUseCurrentFolder:
		ticketID, err := a.resolveFrontDoorTicketID(reader, presetTicket)
		if err != nil {
			return -1, err
		}
		return a.executeFrontDoorAction(ctx, action, ticketID, "", "."), nil
	case frontDoorActionDiscoverOtherProvider:
		repository, err := a.discoverWorkareaRepositoryWithReader(ctx, reader, "", "", "")
		if err != nil {
			if errors.Is(err, errPickerCancelled) {
				return 0, nil
			}
			return -1, err
		}
		ticketID, err := a.resolveFrontDoorTicketID(reader, presetTicket)
		if err != nil {
			return -1, err
		}
		return a.executeFrontDoorAction(ctx, action, ticketID, repository.Root, ""), nil
	case frontDoorActionSavedWorkarea:
		name, err := a.promptForWorkareaSelectionWithReader(reader, store)
		if err != nil {
			if errors.Is(err, errPickerCancelled) {
				return 0, nil
			}
			return -1, err
		}
		if _, err := store.Use(name); err != nil {
			return -1, err
		}
		ticketID, err := a.resolveFrontDoorTicketID(reader, presetTicket)
		if err != nil {
			return -1, err
		}
		return a.executeFrontDoorAction(ctx, action, ticketID, "", ""), nil
	default:
		return -1, fmt.Errorf("invalid quick-start choice %q", selectedAction)
	}
}

func (a *App) runFrontDoorCurrentProjectAction(ctx context.Context) (int, error) {
	return a.runFrontDoorCurrentProjectActionWithReader(ctx, bufio.NewReader(a.stdin))
}

func (a *App) runFrontDoorCurrentProjectActionWithReader(ctx context.Context, reader *bufio.Reader) (int, error) {

	store, err := workarea.NewStore()
	if err != nil {
		return -1, err
	}
	workareas, _, err := store.List()
	if err != nil {
		return -1, err
	}

	items := []pickerItem{
		{
			Value:    "inspect",
			Title:    "Inspect one ticket",
			Subtitle: "See every commit, branch, PR, and risk hint for a ticket.",
			Keywords: []string{"inspect", "ticket", "audit"},
		},
		{
			Value:    "verify",
			Title:    "Verify release readiness",
			Subtitle: "Get a safe, warning, or blocked verdict for the next move.",
			Keywords: []string{"verify", "safe", "blocked", "warning"},
		},
		{
			Value:    "manifest",
			Title:    "Generate a release packet",
			Subtitle: "Export Markdown or JSON for QA, client review, and handoff.",
			Keywords: []string{"manifest", "packet", "release"},
		},
	}
	if len(workareas) > 1 {
		items = append(items, pickerItem{
			Value:    "switch-workarea",
			Title:    "Switch saved project",
			Subtitle: "Change the current project before you run the next ticket command.",
			Keywords: []string{"switch", "workarea", "project"},
			Recent:   true,
		})
	}

	fmt.Fprintln(a.stdout)
	selected, err := a.runPicker(reader, "What do you want to do next?", items)
	if err != nil {
		if errors.Is(err, errPickerCancelled) {
			return 0, nil
		}
		return -1, err
	}

	if selected.Value == "switch-workarea" {
		name, err := a.promptForWorkareaSelectionWithReader(reader, store)
		if err != nil {
			if errors.Is(err, errPickerCancelled) {
				return 0, nil
			}
			return -1, err
		}
		if _, err := store.Use(name); err != nil {
			return -1, err
		}
		fmt.Fprintf(a.stdout, "Current project switched to %s.\n", name)
		return 0, nil
	}

	ticketID, err := a.resolveFrontDoorTicketID(reader, "")
	if err != nil {
		return -1, err
	}

	switch selected.Value {
	case "inspect":
		return a.executeFrontDoorAction(ctx, frontDoorActionInspect, ticketID, "", ""), nil
	case "verify":
		return a.executeFrontDoorAction(ctx, frontDoorActionVerify, ticketID, "", ""), nil
	case "manifest":
		return a.executeFrontDoorAction(ctx, frontDoorActionManifest, ticketID, "", ""), nil
	default:
		return -1, fmt.Errorf("invalid front-door choice %q", selected.Value)
	}
}

func (a *App) resolveFrontDoorTicketID(reader *bufio.Reader, preset string) (string, error) {
	preset = normalizeTicketID(preset)
	if preset != "" {
		return preset, nil
	}
	return a.promptForLine(reader, "Ticket ID")
}

func (a *App) executeFrontDoorAction(ctx context.Context, action frontDoorAction, ticketID, repoTarget, path string) int {
	ticketID = normalizeTicketID(ticketID)
	switch action {
	case frontDoorActionInspect:
		args := []string{ticketID}
		if strings.TrimSpace(repoTarget) != "" {
			args = append(args, "--repo", repoTarget)
		}
		if strings.TrimSpace(path) != "" {
			args = append(args, "--path", path)
		}
		return a.runInspect(ctx, args)
	case frontDoorActionVerify:
		args := []string{"--ticket", ticketID}
		if strings.TrimSpace(repoTarget) != "" {
			args = append(args, "--repo", repoTarget)
		}
		if strings.TrimSpace(path) != "" {
			args = append(args, "--path", path)
		}
		return a.runVerify(ctx, args)
	case frontDoorActionManifest:
		args := []string{"--ticket", ticketID}
		if strings.TrimSpace(repoTarget) != "" {
			args = append(args, "--repo", repoTarget)
		}
		if strings.TrimSpace(path) != "" {
			args = append(args, "--path", path)
		}
		return a.runManifestGenerate(ctx, args)
	default:
		fmt.Fprintf(a.stderr, "front door failed: unsupported action %q\n", action)
		return 1
	}
}

func parseFrontDoorCommand(line string, hasCurrent, hasSaved bool) (frontDoorCommand, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return frontDoorCommand{Action: frontDoorActionPicker}, nil
	}

	tokens := strings.Fields(trimmed)
	if len(tokens) == 0 {
		return frontDoorCommand{Action: frontDoorActionPicker}, nil
	}

	if len(tokens) == 1 {
		switch strings.ToLower(tokens[0]) {
		case "1":
			if hasCurrent {
				return frontDoorCommand{Action: frontDoorActionInspect}, nil
			}
			return frontDoorCommand{Action: frontDoorActionDiscoverGitHub}, nil
		case "2":
			if hasCurrent {
				return frontDoorCommand{Action: frontDoorActionVerify}, nil
			}
			return frontDoorCommand{Action: frontDoorActionEnterTarget}, nil
		case "3":
			if hasCurrent {
				return frontDoorCommand{Action: frontDoorActionManifest}, nil
			}
			return frontDoorCommand{Action: frontDoorActionUseCurrentFolder}, nil
		case "4":
			if hasCurrent {
				return frontDoorCommand{Action: frontDoorActionSwitchWorkarea}, nil
			}
			return frontDoorCommand{Action: frontDoorActionDiscoverOtherProvider}, nil
		case "5":
			if hasSaved && !hasCurrent {
				return frontDoorCommand{Action: frontDoorActionSavedWorkarea}, nil
			}
		case "?", "help", "menu", "pick", "browse":
			return frontDoorCommand{Action: frontDoorActionPicker}, nil
		case "github":
			return frontDoorCommand{Action: frontDoorActionDiscoverGitHub}, nil
		case "local", "folder":
			return frontDoorCommand{Action: frontDoorActionUseCurrentFolder}, nil
		case "switch", "project", "workarea":
			return frontDoorCommand{Action: frontDoorActionSwitchWorkarea}, nil
		}
	}

	repoTarget := ""
	filtered := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if repoTarget == "" && isFrontDoorRepoTarget(token) {
			repoTarget = token
			continue
		}
		filtered = append(filtered, token)
	}
	tokens = filtered
	if len(tokens) == 0 && repoTarget != "" {
		return frontDoorCommand{Action: frontDoorActionInspect, RepoTarget: repoTarget}, nil
	}

	first := strings.ToLower(tokens[0])
	switch first {
	case "inspect", "find":
		return frontDoorCommand{Action: frontDoorActionInspect, TicketID: frontDoorTicketArg(tokens[1:]), RepoTarget: repoTarget}, nil
	case "verify":
		return frontDoorCommand{Action: frontDoorActionVerify, TicketID: frontDoorTicketArg(tokens[1:]), RepoTarget: repoTarget}, nil
	case "manifest":
		args := tokens[1:]
		if len(args) > 0 && strings.EqualFold(args[0], "generate") {
			args = args[1:]
		}
		return frontDoorCommand{Action: frontDoorActionManifest, TicketID: frontDoorTicketArg(args), RepoTarget: repoTarget}, nil
	case "login":
		provider := "github"
		if len(tokens) > 1 {
			provider = strings.TrimSpace(tokens[1])
		}
		return frontDoorCommand{Action: frontDoorActionLogin, Provider: provider}, nil
	case "repo", "target":
		if repoTarget == "" {
			return frontDoorCommand{}, fmt.Errorf("type a repository target such as github:owner/name")
		}
		return frontDoorCommand{Action: frontDoorActionInspect, TicketID: frontDoorTicketArg(tokens[1:]), RepoTarget: repoTarget}, nil
	case "local", "folder":
		return frontDoorCommand{Action: frontDoorActionUseCurrentFolder, TicketID: frontDoorTicketArg(tokens[1:])}, nil
	case "switch", "project", "workarea":
		return frontDoorCommand{Action: frontDoorActionSwitchWorkarea}, nil
	}

	return frontDoorCommand{Action: frontDoorActionInspect, TicketID: tokens[0], RepoTarget: repoTarget}, nil
}

func isFrontDoorRepoTarget(value string) bool {
	_, err := sourcecontrol.ParseRepositoryTargets(value)
	return err == nil
}

func frontDoorTicketArg(tokens []string) string {
	if len(tokens) == 0 {
		return ""
	}
	return tokens[0]
}
