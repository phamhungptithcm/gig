package cli

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gig/internal/buildinfo"
	"gig/internal/output"
	"gig/internal/scm"
	sessionstore "gig/internal/session"
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

	current := a.frontDoorCurrentWorkarea(ctx, store)
	assistSession, hasAssistSession, _ := a.currentAssistSessionForWorkarea(current)
	state := a.buildFrontDoorState(ctx, current, workareas, assistSession, hasAssistSession)
	if err := output.RenderFrontDoor(a.stdout, state); err != nil {
		fmt.Fprintf(a.stderr, "render failed: %v\n", err)
		return 1
	}
	if a.frontDoorPromptEnabled() {
		reader := bufio.NewReader(a.stdin)
		a.stdin = reader
		return a.runFrontDoorSession(ctx, reader, store)
	}

	return 0
}

func (a *App) frontDoorCurrentWorkarea(ctx context.Context, store *workarea.Store) *workarea.Definition {
	var current *workarea.Definition
	if definition, ok, err := store.Current(); err == nil && ok {
		current = &definition
	}
	if _, _, ok := a.inferRemoteRepositoryFromCurrentCheckout(ctx); ok {
		current = nil
	}
	return current
}

type frontDoorTurnResult struct {
	ExitCode     int
	ExitSession  bool
	RanCommand   bool
	Command      frontDoorCommand
	Stdout       string
	Stderr       string
	ResolvedLine string
}

type frontDoorSessionState struct {
	LastTicketID   string
	LastRepoTarget string
	LastPath       string
	LastFromBranch string
	LastToBranch   string
	LastAction     frontDoorAction
	LastVerdict    string
	LastCommand    *frontDoorCommand
	DefaultInput   string
	DefaultReason  string
	RepoUses       map[string]int
}

func (a *App) runFrontDoorSession(ctx context.Context, reader *bufio.Reader, store *workarea.Store) int {
	lastExitCode := 0
	session := frontDoorSessionState{RepoUses: make(map[string]int)}
	for {
		workareas, _, err := store.List()
		if err != nil {
			fmt.Fprintf(a.stderr, "front door failed: %v\n", err)
			return 1
		}
		current := a.frontDoorCurrentWorkarea(ctx, store)
		_, hasAssistSession, _ := a.currentAssistSessionForWorkarea(current)

		result, err := a.runFrontDoorPalette(ctx, reader, store, current, workareas, hasAssistSession, &session)
		if errors.Is(err, io.EOF) {
			return lastExitCode
		}
		if err != nil {
			fmt.Fprintf(a.stderr, "front door failed: %v\n", err)
			lastExitCode = 1
			a.renderFrontDoorSessionSuggestions(&session, current, hasAssistSession)
			continue
		}
		if result.ExitSession {
			return result.ExitCode
		}
		a.rememberFrontDoorTurn(&session, result)
		a.renderFrontDoorSessionSuggestions(&session, current, hasAssistSession)
		lastExitCode = result.ExitCode
	}
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

func (a *App) buildFrontDoorState(ctx context.Context, current *workarea.Definition, workareas []workarea.Definition, assistSession sessionstore.Session, hasAssistSession bool) output.FrontDoorState {
	var suggestionProvider scm.Type
	var suggestionStatus *sourcecontrol.ProviderStatus

	state := output.FrontDoorState{
		Current:          current,
		Workareas:        workareas,
		Version:          buildinfo.Version,
		ProviderCoverage: frontDoorProviderCoverageRows(),
	}
	if hasAssistSession {
		state.ResumeTitle = sessionstore.ResumeTitle(assistSession.Kind)
		state.ResumeSummary = assistSession.Summary
		state.ResumeScope = sessionstore.ResumeScopeLabel(assistSession)
		state.ResumeQuestion = assistSession.LastQuestion
		state.ResumeSuggestedQuestion = sessionstore.ResumeQuestion(assistSession.Kind)
		state.StatusRows = append(state.StatusRows, output.KeyValue{Label: "Assist", Value: strings.ToLower(sessionstore.ResumeTitle(assistSession.Kind)) + " ready"})
	}

	if current != nil {
		state.HeroStatus = "current project ready"
		state.Prompt = "ask gig > ABC-123"
		state.Examples = []string{
			"verify ABC-123",
			"packet ABC-123",
			"switch",
		}
		if provider, label, ok := frontDoorWorkareaProvider(*current); ok {
			providerStatus := a.frontDoorProviderStatus(ctx, provider)
			suggestionProvider = provider
			suggestionStatus = &providerStatus
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
		if detected, ok := a.detectFrontDoorRepository(ctx); ok {
			command := "gig ABC-123"
			prompt := "ask gig > local ABC-123"
			examples := []string{
				"local ABC-123",
				"verify ABC-123",
				"packet ABC-123",
			}
			if detected.Type.IsRemote() {
				command = "gig ABC-123"
				prompt = "ask gig > ABC-123"
				examples = []string{
					"ABC-123",
					"verify ABC-123",
					"packet ABC-123",
				}
			}
			state.Detected = &output.FrontDoorDetectedRepository{
				Name:    detected.Name,
				Root:    detected.Root,
				Type:    frontDoorRepositoryTypeLabel(detected.Type),
				Branch:  detected.CurrentBranch,
				Command: command,
			}
			if detected.Type.IsRemote() {
				providerStatus := a.frontDoorProviderStatus(ctx, detected.Type)
				suggestionProvider = detected.Type
				suggestionStatus = &providerStatus
			}
			state.HeroStatus = "current folder ready"
			state.Prompt = prompt
			state.Examples = examples
		} else {
			githubStatus := a.frontDoorProviderStatus(ctx, scm.TypeGitHub)
			suggestionProvider = scm.TypeGitHub
			suggestionStatus = &githubStatus
			state.HeroStatus = "no project selected yet"
			state.StatusRows = append(state.StatusRows,
				output.KeyValue{Label: "Mode", Value: "new session"},
				output.KeyValue{Label: "Provider", Value: formatFrontDoorProviderStatus(sourcecontrol.ProviderLabel(scm.TypeGitHub), githubStatus)},
			)
			state.Prompt = "ask gig > repo"
			state.Examples = []string{
				"repo",
				"repo payments",
				"gh owner/name",
				"login",
				"local ABC-123",
			}
			if githubStatus.Ready {
				state.Prompt = "ask gig > repo"
			}
		}
	}
	if hasAssistSession {
		state.Prompt = frontDoorResumePrompt(assistSession)
		state.Examples = prependFrontDoorExamples(state.Examples,
			frontDoorResumeExample(assistSession),
			"resume",
		)
	}

	if len(workareas) > 0 {
		state.StatusRows = append(state.StatusRows, output.KeyValue{Label: "Saved", Value: fmt.Sprintf("%d project(s)", len(workareas))})
	}
	repoTarget := ""
	if current != nil {
		repoTarget = current.RepoTarget
	} else if state.Detected != nil && suggestionDetectedIsRemote(*state.Detected) {
		repoTarget = state.Detected.Root
	}
	state.Suggestions = buildSmartSuggestions(suggestionContext{
		Command:        "frontdoor",
		TicketID:       "ABC-123",
		RepoTarget:     repoTarget,
		Provider:       suggestionProvider,
		AuthStatus:     suggestionStatus,
		Current:        current,
		Detected:       state.Detected,
		HasAssist:      hasAssistSession,
		ConfigPath:     "",
		ConfigDetected: false,
	})

	return state
}

func (a *App) detectFrontDoorRepository(ctx context.Context) (scm.Repository, bool) {
	remoteRepository, localRepository, ok := a.inferRemoteRepositoryFromCurrentCheckout(ctx)
	if ok {
		remoteRepository.CurrentBranch = localRepository.CurrentBranch
		return remoteRepository, true
	}
	repository, ok, err := a.scanner.Current(ctx, ".")
	if err != nil || !ok {
		return scm.Repository{}, false
	}
	return repository, true
}

func frontDoorRepositoryTypeLabel(repoType scm.Type) string {
	switch repoType {
	case scm.TypeGit:
		return "Git repository"
	case scm.TypeSVN:
		return "SVN checkout"
	default:
		return sourcecontrol.ProviderLabel(repoType)
	}
}

func frontDoorProviderCoverageRows() []output.KeyValue {
	capabilities := sourcecontrol.OrderedProviderCapabilities()
	rows := make([]output.KeyValue, 0, len(capabilities))
	for _, capability := range capabilities {
		rows = append(rows, output.KeyValue{
			Label: capability.Label,
			Value: capability.Summary(),
		})
	}
	return rows
}

func (a *App) frontDoorProviderStatus(ctx context.Context, provider scm.Type) sourcecontrol.ProviderStatus {
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
	frontDoorActionAsk                   frontDoorAction = "ask"
	frontDoorActionResume                frontDoorAction = "resume"
	frontDoorActionExit                  frontDoorAction = "exit"
	frontDoorActionPlan                  frontDoorAction = "plan"
	frontDoorActionExplain               frontDoorAction = "explain"
	frontDoorActionHelp                  frontDoorAction = "help"
	frontDoorActionLast                  frontDoorAction = "last"
	frontDoorActionNext                  frontDoorAction = "next"
	frontDoorActionProject               frontDoorAction = "project"
	frontDoorActionRepo                  frontDoorAction = "repo"
	frontDoorActionSave                  frontDoorAction = "save"
)

type frontDoorCommand struct {
	Action     frontDoorAction
	TicketID   string
	RepoTarget string
	Path       string
	FromBranch string
	ToBranch   string
	Provider   string
	Message    string
	Args       []string
	ExtraArgs  []string
	RepoQuery  string
}

func (a *App) runFrontDoorPalette(ctx context.Context, reader *bufio.Reader, store *workarea.Store, current *workarea.Definition, workareas []workarea.Definition, hasAssistSession bool, session *frontDoorSessionState) (frontDoorTurnResult, error) {
	fmt.Fprintln(a.stdout)
	fmt.Fprint(a.stdout, "ask gig > ")

	line, err := reader.ReadString('\n')
	if errors.Is(err, io.EOF) && strings.TrimSpace(line) == "" {
		return frontDoorTurnResult{}, io.EOF
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return frontDoorTurnResult{}, err
	}
	line = strings.TrimSpace(line)
	if line == "" && session != nil && strings.TrimSpace(session.DefaultInput) != "" {
		line = session.DefaultInput
		fmt.Fprintf(a.stdout, "running %s\n", line)
	}

	command, err := parseFrontDoorCommand(line, current != nil, len(workareas) > 0, hasAssistSession)
	if err != nil {
		return frontDoorTurnResult{}, err
	}
	if command.Action == frontDoorActionNext {
		if session == nil || strings.TrimSpace(session.DefaultInput) == "" {
			return frontDoorTurnResult{}, fmt.Errorf("no suggested next command is ready yet")
		}
		command, err = parseFrontDoorCommand(session.DefaultInput, current != nil, len(workareas) > 0, hasAssistSession)
		if err != nil {
			return frontDoorTurnResult{}, err
		}
	}
	if command.Action == frontDoorActionLast {
		if session == nil || session.LastCommand == nil {
			return frontDoorTurnResult{}, fmt.Errorf("no previous command is ready to rerun yet")
		}
		command = *session.LastCommand
	}

	switch command.Action {
	case frontDoorActionExit:
		fmt.Fprintln(a.stdout, "bye")
		return frontDoorTurnResult{ExitSession: true}, nil
	case frontDoorActionHelp:
		a.renderFrontDoorPromptHelp(current != nil, len(workareas) > 0, hasAssistSession, session)
		return frontDoorTurnResult{}, nil
	case frontDoorActionPicker:
		if current != nil {
			exitCode, err := a.runFrontDoorCurrentProjectActionWithReader(ctx, reader)
			return frontDoorTurnResult{ExitCode: exitCode}, err
		}
		exitCode, err := a.runFrontDoorActionWithoutCurrentProject(ctx, reader, store, workareas, frontDoorActionInspect, "")
		return frontDoorTurnResult{ExitCode: exitCode}, err
	case frontDoorActionLogin:
		args := []string{}
		if strings.TrimSpace(command.Provider) != "" {
			args = append(args, command.Provider)
		}
		execution := a.captureFrontDoorExecution(func() int {
			return a.runLoginWithReader(ctx, reader, args)
		})
		return frontDoorTurnResult{ExitCode: execution.ExitCode, RanCommand: true, Command: command, Stdout: execution.Stdout, Stderr: execution.Stderr, ResolvedLine: line}, nil
	case frontDoorActionResume:
		execution := a.captureFrontDoorExecution(func() int {
			return a.runAssistResume(nil)
		})
		return frontDoorTurnResult{ExitCode: execution.ExitCode, RanCommand: true, Command: command, Stdout: execution.Stdout, Stderr: execution.Stderr, ResolvedLine: line}, nil
	case frontDoorActionAsk:
		execution := a.captureFrontDoorExecution(func() int {
			return a.runAsk(ctx, []string{command.Message})
		})
		return frontDoorTurnResult{ExitCode: execution.ExitCode, RanCommand: true, Command: command, Stdout: execution.Stdout, Stderr: execution.Stderr, ResolvedLine: line}, nil
	case frontDoorActionProject:
		execution := a.captureFrontDoorExecution(func() int {
			return a.runWorkarea(ctx, command.Args)
		})
		return frontDoorTurnResult{ExitCode: execution.ExitCode, RanCommand: true, Command: command, Stdout: execution.Stdout, Stderr: execution.Stderr, ResolvedLine: line}, nil
	case frontDoorActionRepo:
		repository, err := a.resolveFrontDoorPromptRepository(ctx, reader, store, command.RepoQuery, command.RepoTarget)
		if err != nil {
			return frontDoorTurnResult{}, err
		}
		_ = store.RecordRepositorySelection(repository)
		fmt.Fprintf(a.stdout, "found %s\n", repository.Root)
		command.RepoTarget = repository.Root
		command.RepoQuery = ""
		return frontDoorTurnResult{RanCommand: true, Command: command, ResolvedLine: line}, nil
	case frontDoorActionSave:
		repoTarget := strings.TrimSpace(command.RepoTarget)
		if repoTarget == "" && session != nil {
			repoTarget = strings.TrimSpace(session.LastRepoTarget)
		}
		if repoTarget == "" {
			return frontDoorTurnResult{}, fmt.Errorf("no repository scope is ready to save yet; use repo <name> first")
		}
		name := strings.TrimSpace(command.Message)
		if name == "" {
			name = inferFrontDoorSaveName(repoTarget)
		}
		args := []string{"add"}
		if name != "" {
			args = append(args, name)
		}
		args = append(args, "--repo", repoTarget, "--use")
		command.RepoTarget = repoTarget
		command.Args = args
		execution := a.captureFrontDoorExecution(func() int {
			return a.runWorkarea(ctx, args)
		})
		return frontDoorTurnResult{ExitCode: execution.ExitCode, RanCommand: true, Command: command, Stdout: execution.Stdout, Stderr: execution.Stderr, ResolvedLine: line}, nil
	case frontDoorActionDiscoverGitHub, frontDoorActionEnterTarget, frontDoorActionUseCurrentFolder, frontDoorActionDiscoverOtherProvider, frontDoorActionSavedWorkarea:
		exitCode, err := a.runFrontDoorActionWithoutCurrentProject(ctx, reader, store, workareas, frontDoorActionInspect, command.TicketID, command.Action)
		return frontDoorTurnResult{ExitCode: exitCode}, err
	case frontDoorActionSwitchWorkarea:
		if current == nil {
			if len(workareas) == 0 {
				return frontDoorTurnResult{}, fmt.Errorf("no saved projects are available yet")
			}
			name, err := a.promptForWorkareaSelectionWithReader(reader, store)
			if err != nil {
				if errors.Is(err, errPickerCancelled) {
					return frontDoorTurnResult{}, nil
				}
				return frontDoorTurnResult{}, err
			}
			if _, err := store.Use(name); err != nil {
				return frontDoorTurnResult{}, err
			}
			fmt.Fprintf(a.stdout, "Current project switched to %s.\n", name)
			return frontDoorTurnResult{}, nil
		}
		exitCode, err := a.runFrontDoorCurrentProjectActionWithReader(ctx, reader)
		return frontDoorTurnResult{ExitCode: exitCode}, err
	case frontDoorActionInspect, frontDoorActionVerify, frontDoorActionManifest, frontDoorActionPlan, frontDoorActionExplain:
		command = a.resolveFrontDoorSessionCommand(ctx, command, session, current)
		if strings.TrimSpace(command.RepoTarget) == "" && strings.TrimSpace(command.RepoQuery) != "" {
			repository, err := a.resolveFrontDoorPromptRepository(ctx, reader, store, command.RepoQuery, "")
			if err != nil {
				return frontDoorTurnResult{}, err
			}
			_ = store.RecordRepositorySelection(repository)
			fmt.Fprintf(a.stdout, "found %s\n", repository.Root)
			command.RepoTarget = repository.Root
			command.RepoQuery = ""
		}
		if current == nil && strings.TrimSpace(command.RepoTarget) == "" && strings.TrimSpace(command.Path) == "" {
			if detected, ok := a.detectFrontDoorRepository(ctx); ok {
				if frontDoorCommandNeedsTicketPrompt(command) {
					ticketID, err := a.resolveFrontDoorTicketID(reader, command.TicketID)
					if err != nil {
						return frontDoorTurnResult{}, err
					}
					command.TicketID = ticketID
				}
				command.RepoTarget = strings.TrimSpace(command.RepoTarget)
				command.Path = "."
				if detected.Type.IsRemote() {
					command.Path = ""
					command.RepoTarget = detected.Root
				}
				execution := a.captureFrontDoorExecution(func() int {
					return a.executeFrontDoorCommand(ctx, command)
				})
				return frontDoorTurnResult{ExitCode: execution.ExitCode, RanCommand: true, Command: command, Stdout: execution.Stdout, Stderr: execution.Stderr, ResolvedLine: line}, nil
			}
			exitCode, err := a.runFrontDoorActionWithoutCurrentProject(ctx, reader, store, workareas, command.Action, command.TicketID)
			return frontDoorTurnResult{ExitCode: exitCode}, err
		}
		if frontDoorCommandNeedsTicketPrompt(command) {
			ticketID, err := a.resolveFrontDoorTicketID(reader, command.TicketID)
			if err != nil {
				return frontDoorTurnResult{}, err
			}
			command.TicketID = ticketID
		}
		execution := a.captureFrontDoorExecution(func() int {
			return a.executeFrontDoorCommand(ctx, command)
		})
		return frontDoorTurnResult{ExitCode: execution.ExitCode, RanCommand: true, Command: command, Stdout: execution.Stdout, Stderr: execution.Stderr, ResolvedLine: line}, nil
	default:
		return frontDoorTurnResult{}, fmt.Errorf("unsupported front-door action %q", command.Action)
	}
}

func (a *App) runFrontDoorQuickStart(ctx context.Context, store *workarea.Store, workareas []workarea.Definition) (int, error) {
	return a.runFrontDoorActionWithoutCurrentProject(ctx, bufio.NewReader(a.stdin), store, workareas, frontDoorActionInspect, "")
}

type frontDoorExecution struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

func (a *App) captureFrontDoorExecution(run func() int) frontDoorExecution {
	previousStdout := a.stdout
	previousStderr := a.stderr
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	a.stdout = io.MultiWriter(previousStdout, &stdout)
	a.stderr = io.MultiWriter(previousStderr, &stderr)
	exitCode := run()
	a.stdout = previousStdout
	a.stderr = previousStderr
	return frontDoorExecution{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}
}

func (a *App) resolveFrontDoorSessionCommand(ctx context.Context, command frontDoorCommand, session *frontDoorSessionState, current *workarea.Definition) frontDoorCommand {
	if session != nil {
		if strings.TrimSpace(command.TicketID) == "" {
			command.TicketID = session.LastTicketID
		}
		if strings.TrimSpace(command.RepoTarget) == "" && strings.TrimSpace(command.Path) == "" {
			command.RepoTarget = session.LastRepoTarget
			command.Path = session.LastPath
		}
		if strings.TrimSpace(command.FromBranch) == "" {
			command.FromBranch = session.LastFromBranch
		}
		if strings.TrimSpace(command.ToBranch) == "" {
			command.ToBranch = session.LastToBranch
		}
	}
	if strings.TrimSpace(command.RepoTarget) == "" && strings.TrimSpace(command.Path) == "" && current == nil {
		if detected, ok := a.detectFrontDoorRepository(ctx); ok {
			if detected.Type.IsRemote() {
				command.RepoTarget = detected.Root
			} else {
				command.Path = "."
			}
		}
	}
	return command
}

func (a *App) rememberFrontDoorTurn(session *frontDoorSessionState, result frontDoorTurnResult) {
	if session == nil || !result.RanCommand {
		return
	}
	command := result.Command
	if frontDoorCommandUsesTicket(command.Action) && strings.TrimSpace(command.TicketID) != "" {
		session.LastTicketID = normalizeTicketID(command.TicketID)
	}
	if strings.TrimSpace(command.RepoTarget) != "" {
		session.LastRepoTarget = canonicalFrontDoorRepoTarget(command.RepoTarget)
		session.LastPath = ""
		session.RepoUses[session.LastRepoTarget]++
	}
	if strings.TrimSpace(command.Path) != "" {
		session.LastPath = strings.TrimSpace(command.Path)
		if strings.TrimSpace(command.RepoTarget) == "" {
			session.LastRepoTarget = ""
		}
	}
	if strings.TrimSpace(command.FromBranch) != "" {
		session.LastFromBranch = strings.TrimSpace(command.FromBranch)
	}
	if strings.TrimSpace(command.ToBranch) != "" {
		session.LastToBranch = strings.TrimSpace(command.ToBranch)
	}
	session.LastAction = command.Action
	session.LastVerdict = extractFrontDoorVerdict(result.Stdout)
	stored := command
	session.LastCommand = &stored
	session.DefaultInput, session.DefaultReason = frontDoorDefaultAfterTurn(session, result)
}

func frontDoorCommandUsesTicket(action frontDoorAction) bool {
	switch action {
	case frontDoorActionInspect, frontDoorActionVerify, frontDoorActionManifest, frontDoorActionPlan, frontDoorActionExplain:
		return true
	default:
		return false
	}
}

func frontDoorCommandNeedsTicketPrompt(command frontDoorCommand) bool {
	if !frontDoorCommandUsesTicket(command.Action) {
		return false
	}
	if strings.TrimSpace(command.TicketID) != "" {
		return false
	}
	return !frontDoorArgsProvideTicketScope(command.ExtraArgs)
}

func frontDoorArgsProvideTicketScope(args []string) bool {
	for _, arg := range args {
		token := strings.TrimSpace(arg)
		switch {
		case token == "--ticket" || token == "-ticket" ||
			token == "--ticket-file" || token == "-ticket-file" ||
			token == "--release" || token == "-release":
			return true
		case strings.HasPrefix(token, "--ticket=") ||
			strings.HasPrefix(token, "-ticket=") ||
			strings.HasPrefix(token, "--ticket-file=") ||
			strings.HasPrefix(token, "-ticket-file=") ||
			strings.HasPrefix(token, "--release=") ||
			strings.HasPrefix(token, "-release="):
			return true
		}
	}
	return false
}

func frontDoorDefaultAfterTurn(session *frontDoorSessionState, result frontDoorTurnResult) (string, string) {
	if login := frontDoorLoginInputFromOutput(result.Stderr); login != "" {
		return login, "login fixes the last provider access error"
	}
	if result.ExitCode != 0 {
		return "", ""
	}
	switch result.Command.Action {
	case frontDoorActionRepo:
		return "ABC-123", "repository scope is ready; type or accept a ticket to inspect"
	case frontDoorActionInspect:
		return "verify", "inspect finished; verify checks release readiness next"
	case frontDoorActionVerify:
		switch session.LastVerdict {
		case "safe":
			return "packet", "safe verification can move straight to packet"
		case "warning", "blocked":
			return "plan", "review missing commits, risks, and manual steps next"
		default:
			return "packet", "verification finished; packet prepares handoff"
		}
	case frontDoorActionPlan:
		if session.LastVerdict == "safe" {
			return "packet", "safe plan can move to packet"
		}
		return "explain", "turn the plan into an audience-ready explanation"
	default:
		return "", ""
	}
}

func (a *App) renderFrontDoorSessionSuggestions(session *frontDoorSessionState, current *workarea.Definition, hasAssistSession bool) {
	if session == nil || (strings.TrimSpace(session.LastTicketID) == "" && strings.TrimSpace(session.LastRepoTarget) == "" && strings.TrimSpace(session.LastPath) == "") {
		return
	}
	suggestions := frontDoorSessionSuggestions(session, current, hasAssistSession)
	if len(suggestions) == 0 {
		return
	}
	ui := output.NewConsole(a.stdout)
	_ = ui.Blank()
	_ = ui.Section("Suggested next")
	rows := make([]output.KeyValue, 0, len(suggestions)+1)
	for _, suggestion := range suggestions {
		value := strings.TrimSpace(suggestion.Command)
		if value == "" {
			value = strings.TrimSpace(suggestion.Note)
		}
		if value == "" {
			continue
		}
		rows = append(rows, output.KeyValue{Label: suggestion.Label, Value: value})
	}
	if strings.TrimSpace(session.DefaultInput) != "" {
		runText := "press Enter to run " + session.DefaultInput + ", or type another command"
		if strings.TrimSpace(session.DefaultReason) != "" {
			runText += " (" + session.DefaultReason + ")"
		}
		rows = append(rows, output.KeyValue{Label: "run?", Value: runText})
	}
	_ = ui.NestedRows(rows...)
}

func frontDoorSessionSuggestions(session *frontDoorSessionState, current *workarea.Definition, hasAssistSession bool) []output.FrontDoorSuggestion {
	ticketID := strings.TrimSpace(session.LastTicketID)
	if ticketID == "" {
		ticketID = "ABC-123"
	}
	suggestions := make([]output.FrontDoorSuggestion, 0, 8)
	addCommand := func(label, command string) {
		command = strings.TrimSpace(command)
		if command == "" {
			return
		}
		suggestions = append(suggestions, output.FrontDoorSuggestion{Label: label, Command: command})
	}
	addNote := func(label, note string) {
		note = strings.TrimSpace(note)
		if note == "" {
			return
		}
		suggestions = append(suggestions, output.FrontDoorSuggestion{Label: label, Note: note})
	}

	switch session.LastAction {
	case frontDoorActionInspect:
		addCommand("verify", "verify")
		addCommand("packet", "packet")
	case frontDoorActionVerify:
		switch session.LastVerdict {
		case "safe":
			addCommand("packet", "packet")
		case "warning":
			addCommand("plan", "plan")
			addCommand("explain", "explain")
		case "blocked":
			addCommand("plan", "plan")
			addCommand("explain", "explain")
		default:
			addCommand("packet", "packet")
		}
	case frontDoorActionPlan:
		addCommand("packet", "packet")
		addCommand("explain", "explain")
	default:
		addCommand("inspect", ticketID)
		addCommand("verify", "verify")
		addCommand("packet", "packet")
	}
	if hasAssistSession {
		addCommand("ask", "ask what is still blocked?")
	}
	if session.LastRepoTarget != "" {
		addNote("scope", "using "+session.LastRepoTarget)
		if current == nil && session.RepoUses[session.LastRepoTarget] >= 2 {
			addCommand("save", "save "+inferFrontDoorSaveName(session.LastRepoTarget))
			addNote("why", "then future sessions can start with "+ticketID+", verify, and packet")
		}
	} else if session.LastPath != "" {
		addNote("scope", "using path "+session.LastPath)
	}
	if session.LastFromBranch != "" {
		addNote("branch", session.LastFromBranch+" looks like the release source")
	}
	if session.LastToBranch != "" {
		addNote("target", session.LastToBranch+" is the release target")
	}
	if strings.TrimSpace(session.LastTicketID) != "" {
		addNote("memory", "remembering "+ticketID+"; next, last, verify, packet, and explain reuse it")
	} else {
		addNote("memory", "repository is remembered; type a ticket once, then verify and packet stay short")
	}
	return suggestions
}

func (a *App) renderFrontDoorPromptHelp(hasCurrent, hasSaved, hasAssist bool, session *frontDoorSessionState) {
	ticketID := "ABC-123"
	if session != nil && strings.TrimSpace(session.LastTicketID) != "" {
		ticketID = session.LastTicketID
	}
	ui := output.NewConsole(a.stdout)
	_ = ui.Blank()
	_ = ui.Section("Prompt help")
	rows := []output.KeyValue{
		{Label: "inspect", Value: ticketID + " or i " + ticketID},
		{Label: "verify", Value: "verify or v"},
		{Label: "packet", Value: "packet or p"},
		{Label: "explain", Value: "explain"},
		{Label: "next", Value: "next or Enter when run? is shown"},
		{Label: "last", Value: "rerun the previous command"},
		{Label: "repo", Value: "repo, repo payments, or gh owner/name"},
		{Label: "save", Value: "save payments"},
		{Label: "use", Value: "use payments"},
	}
	if hasCurrent || hasSaved {
		rows = append(rows, output.KeyValue{Label: "switch", Value: "switch project"})
	}
	if hasAssist {
		rows = append(rows, output.KeyValue{Label: "resume", Value: "resume or r"})
	} else {
		rows = append(rows, output.KeyValue{Label: "resume", Value: "r tries to resume the last AI brief"})
	}
	rows = append(rows,
		output.KeyValue{Label: "url", Value: "paste a GitHub, GitLab, Bitbucket, Azure, or SVN URL"},
		output.KeyValue{Label: "local", Value: "local " + ticketID},
		output.KeyValue{Label: "exit", Value: "exit, quit, or q"},
	)
	_ = ui.NestedRows(rows...)
}

func frontDoorLoginInputFromOutput(text string) string {
	line := frontDoorLineContaining(text, "gig login ")
	if line == "" {
		return ""
	}
	fields := strings.Fields(line)
	for i := 0; i+2 < len(fields); i++ {
		if fields[i] == "gig" && fields[i+1] == "login" {
			return "login " + fields[i+2]
		}
	}
	return ""
}

func frontDoorLineContaining(text, pattern string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, pattern) {
			return line
		}
	}
	return ""
}

func extractFrontDoorVerdict(text string) string {
	for _, line := range strings.Split(text, "\n") {
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "verdict") {
			continue
		}
		switch {
		case strings.Contains(lower, "blocked"):
			return "blocked"
		case strings.Contains(lower, "warning"):
			return "warning"
		case strings.Contains(lower, "safe"):
			return "safe"
		}
	}
	return ""
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
			Title:    "Paste a repository URL or target",
			Subtitle: "Paste a provider URL, Git remote, SVN URL, or canonical target.",
			Keywords: []string{"repo", "target", "github:owner/name", "url", "remote"},
		},
		{
			Value:    "use-current-folder",
			Title:    "Use the current folder",
			Subtitle: "Local Git or SVN fallback when you do not want provider-backed lookup.",
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
		fmt.Fprintln(a.stdout, "Repository example: https://github.com/owner/name or github:owner/name")
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
	return a.executeFrontDoorCommand(ctx, frontDoorCommand{
		Action:     action,
		TicketID:   ticketID,
		RepoTarget: repoTarget,
		Path:       path,
	})
}

func (a *App) executeFrontDoorCommand(ctx context.Context, command frontDoorCommand) int {
	ticketID := normalizeTicketID(command.TicketID)
	args := []string{}
	if ticketID != "" {
		args = append(args, ticketID)
	}
	if strings.TrimSpace(command.RepoTarget) != "" {
		args = append(args, "--repo", strings.TrimSpace(command.RepoTarget))
	}
	if strings.TrimSpace(command.Path) != "" {
		args = append(args, "--path", strings.TrimSpace(command.Path))
	}
	if strings.TrimSpace(command.FromBranch) != "" {
		args = append(args, "--from", strings.TrimSpace(command.FromBranch))
	}
	if strings.TrimSpace(command.ToBranch) != "" {
		args = append(args, "--to", strings.TrimSpace(command.ToBranch))
	}
	if command.Action != frontDoorActionInspect && strings.TrimSpace(command.Path) != "" && strings.TrimSpace(command.FromBranch) == "" && strings.TrimSpace(command.ToBranch) == "" {
		args = append(args, a.frontDoorLocalPromotionArgs(ctx, strings.TrimSpace(command.Path))...)
	}
	args = append(args, command.ExtraArgs...)

	switch command.Action {
	case frontDoorActionInspect:
		return a.runInspect(ctx, args)
	case frontDoorActionVerify:
		return a.runVerify(ctx, args)
	case frontDoorActionManifest:
		return a.runManifest(ctx, args)
	case frontDoorActionPlan:
		return a.runPlan(ctx, args)
	case frontDoorActionExplain:
		return a.runAssistAudit(ctx, args)
	default:
		fmt.Fprintf(a.stderr, "front door failed: unsupported action %q\n", command.Action)
		return 1
	}
}

func (a *App) frontDoorLocalPromotionArgs(ctx context.Context, path string) []string {
	repository, ok, err := a.scanner.Current(ctx, path)
	if err != nil || !ok {
		return nil
	}

	fromBranch := strings.TrimSpace(repository.CurrentBranch)
	if fromBranch == "" || isFrontDoorProductionBranch(fromBranch) {
		return nil
	}

	toBranch := "main"
	if repository.Type == scm.TypeSVN {
		toBranch = "trunk"
	}
	if strings.EqualFold(fromBranch, toBranch) {
		return nil
	}
	return []string{"--from", fromBranch, "--to", toBranch}
}

func isFrontDoorProductionBranch(branch string) bool {
	switch strings.ToLower(strings.TrimSpace(branch)) {
	case "main", "master", "prod", "production", "trunk":
		return true
	default:
		return false
	}
}

func parseFrontDoorCommand(line string, hasCurrent, hasSaved, hasAssist bool) (frontDoorCommand, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return frontDoorCommand{Action: frontDoorActionPicker}, nil
	}

	tokens := strings.Fields(trimmed)
	if len(tokens) == 0 {
		return frontDoorCommand{Action: frontDoorActionPicker}, nil
	}
	if strings.EqualFold(tokens[0], "gig") {
		tokens = tokens[1:]
		if len(tokens) == 0 {
			return frontDoorCommand{Action: frontDoorActionPicker}, nil
		}
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
		case "?", "help":
			return frontDoorCommand{Action: frontDoorActionHelp}, nil
		case "menu", "pick", "browse":
			return frontDoorCommand{Action: frontDoorActionPicker}, nil
		case "i":
			return frontDoorCommand{Action: frontDoorActionInspect}, nil
		case "v":
			return frontDoorCommand{Action: frontDoorActionVerify}, nil
		case "p":
			return frontDoorCommand{Action: frontDoorActionManifest}, nil
		case "r":
			return frontDoorCommand{Action: frontDoorActionResume}, nil
		case "last":
			return frontDoorCommand{Action: frontDoorActionLast}, nil
		case "next":
			return frontDoorCommand{Action: frontDoorActionNext}, nil
		case "repo", "target":
			return frontDoorCommand{Action: frontDoorActionRepo}, nil
		case "save":
			return frontDoorCommand{Action: frontDoorActionSave}, nil
		case "use":
			return frontDoorCommand{Action: frontDoorActionProject, Args: []string{"use"}}, nil
		case "github":
			return frontDoorCommand{Action: frontDoorActionDiscoverGitHub}, nil
		case "gh", "gl", "bb", "ado", "azdo", "svn":
			return frontDoorCommand{Action: frontDoorActionRepo, RepoQuery: tokens[0]}, nil
		case "local", "folder":
			return frontDoorCommand{Action: frontDoorActionUseCurrentFolder, Path: "."}, nil
		case "switch", "project", "workarea":
			return frontDoorCommand{Action: frontDoorActionSwitchWorkarea}, nil
		case "resume":
			if hasAssist {
				return frontDoorCommand{Action: frontDoorActionResume}, nil
			}
		case "exit", "quit", "q":
			return frontDoorCommand{Action: frontDoorActionExit}, nil
		}
	}

	filtered, flagRepoTarget, flagPath, flagFromBranch, flagToBranch, err := parseFrontDoorInlineFlags(tokens)
	if err != nil {
		return frontDoorCommand{}, err
	}
	tokens = filtered
	if len(tokens) == 0 {
		return frontDoorCommand{Action: frontDoorActionPicker}, nil
	}

	repoTarget := flagRepoTarget
	filtered = make([]string, 0, len(tokens))
	for _, token := range tokens {
		if repoTarget == "" && isFrontDoorRepoTarget(token) {
			repoTarget = token
			continue
		}
		filtered = append(filtered, token)
	}
	tokens = filtered
	if len(tokens) == 0 && repoTarget != "" {
		return frontDoorCommand{Action: frontDoorActionInspect, RepoTarget: repoTarget, Path: flagPath, FromBranch: flagFromBranch, ToBranch: flagToBranch}, nil
	}

	first := strings.ToLower(tokens[0])
	switch first {
	case "inspect", "find", "i":
		ticketID, extraArgs := frontDoorTicketAndExtraArgs(tokens[1:])
		return frontDoorCommand{Action: frontDoorActionInspect, TicketID: ticketID, RepoTarget: repoTarget, Path: flagPath, FromBranch: flagFromBranch, ToBranch: flagToBranch, ExtraArgs: extraArgs}, nil
	case "verify", "v":
		ticketID, extraArgs := frontDoorTicketAndExtraArgs(tokens[1:])
		return frontDoorCommand{Action: frontDoorActionVerify, TicketID: ticketID, RepoTarget: repoTarget, Path: flagPath, FromBranch: flagFromBranch, ToBranch: flagToBranch, ExtraArgs: extraArgs}, nil
	case "manifest", "packet", "p":
		args := tokens[1:]
		if len(args) > 0 && strings.EqualFold(args[0], "generate") {
			args = args[1:]
		}
		ticketID, extraArgs := frontDoorTicketAndExtraArgs(args)
		return frontDoorCommand{Action: frontDoorActionManifest, TicketID: ticketID, RepoTarget: repoTarget, Path: flagPath, FromBranch: flagFromBranch, ToBranch: flagToBranch, ExtraArgs: extraArgs}, nil
	case "plan":
		ticketID, extraArgs := frontDoorTicketAndExtraArgs(tokens[1:])
		return frontDoorCommand{Action: frontDoorActionPlan, TicketID: ticketID, RepoTarget: repoTarget, Path: flagPath, FromBranch: flagFromBranch, ToBranch: flagToBranch, ExtraArgs: extraArgs}, nil
	case "explain", "e":
		ticketID, extraArgs := frontDoorTicketAndExtraArgs(tokens[1:])
		return frontDoorCommand{Action: frontDoorActionExplain, TicketID: ticketID, RepoTarget: repoTarget, Path: flagPath, FromBranch: flagFromBranch, ToBranch: flagToBranch, ExtraArgs: extraArgs}, nil
	case "login":
		provider := ""
		if len(tokens) > 1 {
			provider = strings.TrimSpace(tokens[1])
		}
		return frontDoorCommand{Action: frontDoorActionLogin, Provider: provider}, nil
	case "ask":
		message := strings.TrimSpace(strings.Join(tokens[1:], " "))
		if message == "" {
			message = "what changed since the last brief?"
		}
		return frontDoorCommand{Action: frontDoorActionAsk, Message: message}, nil
	case "resume", "r":
		return frontDoorCommand{Action: frontDoorActionResume}, nil
	case "exit", "quit", "q":
		return frontDoorCommand{Action: frontDoorActionExit}, nil
	case "?", "help":
		return frontDoorCommand{Action: frontDoorActionHelp}, nil
	case "last":
		return frontDoorCommand{Action: frontDoorActionLast}, nil
	case "next":
		return frontDoorCommand{Action: frontDoorActionNext}, nil
	case "project", "workarea":
		return frontDoorCommand{Action: frontDoorActionProject, Args: append([]string(nil), tokens[1:]...)}, nil
	case "use":
		return frontDoorCommand{Action: frontDoorActionProject, Args: append([]string{"use"}, tokens[1:]...)}, nil
	case "save":
		return frontDoorCommand{Action: frontDoorActionSave, Message: strings.TrimSpace(strings.Join(tokens[1:], " ")), RepoTarget: repoTarget}, nil
	case "gh", "gl", "bb", "ado", "azdo", "svn":
		return frontDoorProviderAliasCommand(first, tokens[1:], flagPath, flagFromBranch, flagToBranch)
	case "repo", "target":
		if repoTarget != "" {
			ticketID, extraArgs := frontDoorTicketAndExtraArgs(tokens[1:])
			if ticketID != "" || frontDoorArgsProvideTicketScope(extraArgs) {
				return frontDoorCommand{Action: frontDoorActionInspect, TicketID: ticketID, RepoTarget: repoTarget, Path: flagPath, FromBranch: flagFromBranch, ToBranch: flagToBranch, ExtraArgs: extraArgs}, nil
			}
			return frontDoorCommand{Action: frontDoorActionRepo, RepoTarget: repoTarget}, nil
		}
		query, ticketID, extraArgs := frontDoorRepoQueryTicketExtra(tokens[1:])
		if query == "" {
			return frontDoorCommand{Action: frontDoorActionRepo}, nil
		}
		if ticketID != "" {
			return frontDoorCommand{Action: frontDoorActionInspect, TicketID: ticketID, RepoQuery: query, Path: flagPath, FromBranch: flagFromBranch, ToBranch: flagToBranch, ExtraArgs: extraArgs}, nil
		}
		return frontDoorCommand{Action: frontDoorActionRepo, RepoQuery: query}, nil
	case "local", "folder":
		ticketID, extraArgs := frontDoorTicketAndExtraArgs(tokens[1:])
		return frontDoorCommand{Action: frontDoorActionInspect, TicketID: ticketID, Path: ".", FromBranch: flagFromBranch, ToBranch: flagToBranch, ExtraArgs: extraArgs}, nil
	case "switch":
		return frontDoorCommand{Action: frontDoorActionSwitchWorkarea}, nil
	}

	if hasAssist && !looksLikeTicketID(tokens[0]) {
		return frontDoorCommand{Action: frontDoorActionAsk, Message: trimmed}, nil
	}

	ticketID, extraArgs := frontDoorTicketAndExtraArgs(tokens)
	return frontDoorCommand{Action: frontDoorActionInspect, TicketID: ticketID, RepoTarget: repoTarget, Path: flagPath, FromBranch: flagFromBranch, ToBranch: flagToBranch, ExtraArgs: extraArgs}, nil
}

func parseFrontDoorInlineFlags(tokens []string) ([]string, string, string, string, string, error) {
	var repoTarget, path, fromBranch, toBranch string
	filtered := make([]string, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		readValue := func(flag string) (string, error) {
			if i+1 >= len(tokens) {
				return "", fmt.Errorf("%s requires a value", flag)
			}
			i++
			return strings.TrimSpace(tokens[i]), nil
		}
		switch {
		case token == "--repo" || token == "-repo":
			value, err := readValue(token)
			if err != nil {
				return nil, "", "", "", "", err
			}
			repoTarget = value
		case strings.HasPrefix(token, "--repo="):
			repoTarget = strings.TrimSpace(strings.TrimPrefix(token, "--repo="))
		case token == "--path" || token == "-path":
			value, err := readValue(token)
			if err != nil {
				return nil, "", "", "", "", err
			}
			path = value
		case strings.HasPrefix(token, "--path="):
			path = strings.TrimSpace(strings.TrimPrefix(token, "--path="))
		case token == "--from" || token == "-from":
			value, err := readValue(token)
			if err != nil {
				return nil, "", "", "", "", err
			}
			fromBranch = value
		case strings.HasPrefix(token, "--from="):
			fromBranch = strings.TrimSpace(strings.TrimPrefix(token, "--from="))
		case token == "--to" || token == "-to":
			value, err := readValue(token)
			if err != nil {
				return nil, "", "", "", "", err
			}
			toBranch = value
		case strings.HasPrefix(token, "--to="):
			toBranch = strings.TrimSpace(strings.TrimPrefix(token, "--to="))
		default:
			filtered = append(filtered, token)
		}
	}
	return filtered, repoTarget, path, fromBranch, toBranch, nil
}

func isFrontDoorRepoTarget(value string) bool {
	_, err := sourcecontrol.ParseRepositoryTargets(value)
	return err == nil
}

func frontDoorTicketAndExtraArgs(tokens []string) (string, []string) {
	if len(tokens) == 0 {
		return "", nil
	}
	if strings.HasPrefix(strings.TrimSpace(tokens[0]), "-") {
		return "", append([]string(nil), tokens...)
	}
	return tokens[0], append([]string(nil), tokens[1:]...)
}

func frontDoorResumePrompt(session sessionstore.Session) string {
	return "ask gig > " + frontDoorResumeExample(session)
}

func frontDoorResumeExample(session sessionstore.Session) string {
	return sessionstore.ResumeQuestion(session.Kind)
}

func prependFrontDoorExamples(examples []string, values ...string) []string {
	seen := make(map[string]struct{}, len(examples)+len(values))
	combined := make([]string, 0, len(examples)+len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		combined = append(combined, value)
	}
	for _, value := range examples {
		key := strings.ToLower(strings.TrimSpace(value))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		combined = append(combined, value)
	}
	return combined
}
