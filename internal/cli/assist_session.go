package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	assistsvc "gig/internal/assistant"
	conflictsvc "gig/internal/conflict"
	inspectsvc "gig/internal/inspect"
	"gig/internal/output"
	"gig/internal/scm"
	sessionstore "gig/internal/session"
	"gig/internal/workarea"
)

func (a *App) runAsk(ctx context.Context, args []string) int {
	return a.runAssistChat(ctx, args)
}

func (a *App) runAssistChat(ctx context.Context, args []string) int {
	reorderedArgs, err := reorderArgsWithSinglePositional(args, "-message", "--message", "-url", "--url", "-mode", "--mode", "-audience", "--audience", "-format", "--format")
	if err != nil {
		fmt.Fprintf(a.stderr, "assist chat failed: %v\n", err)
		a.printAssistChatUsage()
		return usageExitCode
	}
	args = reorderedArgs

	if hasHelpFlag(args) {
		a.printAssistChatUsage()
		return 0
	}

	fs := flag.NewFlagSet("assist chat", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	questionFlag := fs.String("message", "", "Follow-up question for the current assist session")
	deerflowURL := fs.String("url", "", "Base URL for DeerFlow, for example http://localhost:2026")
	mode := fs.String("mode", "", "Execution mode: flash, standard, pro, or ultra")
	audience := fs.String("audience", "", "Audience override: qa, client, or release-manager")
	format := fs.String("format", string(outputFormatHuman), "Output format: human or json")

	if err := fs.Parse(args); err != nil {
		a.printAssistChatUsage()
		return usageExitCode
	}

	outputFormat, err := parseOutputFormat(*format)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist chat failed: %v\n", err)
		return usageExitCode
	}

	question := strings.TrimSpace(*questionFlag)
	if question == "" {
		question = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if question == "" {
		fmt.Fprintln(a.stderr, "assist chat failed: a follow-up question is required")
		a.printAssistChatUsage()
		return usageExitCode
	}

	store, err := sessionstore.NewStore()
	if err != nil {
		fmt.Fprintf(a.stderr, "assist chat failed: %v\n", err)
		return 1
	}
	saved, ok, err := a.currentAssistSession()
	if err != nil {
		fmt.Fprintf(a.stderr, "assist chat failed: %v\n", err)
		return 1
	}
	if !ok {
		fmt.Fprintln(a.stderr, "assist chat failed: no saved assist session was found; run gig assist audit, release, or resolve first")
		return 1
	}

	selectedMode := strings.TrimSpace(saved.Mode)
	if strings.TrimSpace(*mode) != "" {
		selectedMode = strings.TrimSpace(*mode)
	}
	runMode, err := assistsvc.ParseRunMode(selectedMode)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist chat failed: %v\n", err)
		return usageExitCode
	}

	selectedAudience := strings.TrimSpace(saved.Audience)
	if strings.TrimSpace(*audience) != "" {
		selectedAudience = strings.TrimSpace(*audience)
	}
	audienceValue, err := assistsvc.ParseAudience(selectedAudience)
	if err != nil {
		fmt.Fprintf(a.stderr, "assist chat failed: %v\n", err)
		return usageExitCode
	}

	remoteMode := strings.TrimSpace(saved.RepoTarget) != ""
	runtime, err := newCommandRuntimeWithOptions(saved.WorkspacePath, saved.ConfigPath, runtimeOptions{DisableAutoConfig: remoteMode})
	if err != nil {
		fmt.Fprintf(a.stderr, "assist chat failed: %v\n", err)
		return 1
	}

	options := assistsvc.ExecuteOptions{
		BaseURL:  *deerflowURL,
		Mode:     runMode,
		Audience: audienceValue,
	}

	switch saved.Kind {
	case sessionstore.KindAudit:
		result, err := runtime.assistant.FollowUpAudit(ctx, assistsvc.AuditRequest{
			WorkspacePath: saved.WorkspacePath,
			ScopeLabel:    saved.ScopeLabel,
			CommandTarget: saved.CommandTarget,
			ConfigPath:    saved.ConfigPath,
			TicketID:      saved.TicketID,
			FromBranch:    saved.FromBranch,
			ToBranch:      saved.ToBranch,
			Environments:  append([]inspectsvc.Environment(nil), saved.Environments...),
			Repositories:  append([]scm.Repository(nil), saved.Repositories...),
			LoadedConfig:  runtime.loaded,
		}, question, saved.ThreadID, options)
		if err != nil {
			fmt.Fprintf(a.stderr, "assist chat failed: %v\n", err)
			return 1
		}
		saved.ThreadID = result.ThreadID
		saved.LastQuestion = question
		saved.LastResponse = result.Response
		saved.Mode = string(result.Mode)
		saved.Audience = string(result.Audience)
		if _, err := store.SaveCurrent(saved); err != nil {
			fmt.Fprintf(a.stderr, "assist chat failed: %v\n", err)
			return 1
		}
		return renderAssistChatResult(a, outputFormat, question, saved, result)
	case sessionstore.KindRelease:
		result, err := runtime.assistant.FollowUpRelease(ctx, assistsvc.ReleaseRequest{
			WorkspacePath: saved.WorkspacePath,
			ScopeLabel:    saved.ScopeLabel,
			CommandTarget: saved.CommandTarget,
			ConfigPath:    saved.ConfigPath,
			ReleaseID:     saved.ReleaseID,
			TicketIDs:     append([]string(nil), saved.TicketIDs...),
			FromBranch:    saved.FromBranch,
			ToBranch:      saved.ToBranch,
			Environments:  saved.Environments,
			Repositories:  append([]scm.Repository(nil), saved.Repositories...),
			LoadedConfig:  runtime.loaded,
		}, question, saved.ThreadID, options)
		if err != nil {
			fmt.Fprintf(a.stderr, "assist chat failed: %v\n", err)
			return 1
		}
		saved.ThreadID = result.ThreadID
		saved.LastQuestion = question
		saved.LastResponse = result.Response
		saved.Mode = string(result.Mode)
		saved.Audience = string(result.Audience)
		if _, err := store.SaveCurrent(saved); err != nil {
			fmt.Fprintf(a.stderr, "assist chat failed: %v\n", err)
			return 1
		}
		return renderAssistChatResult(a, outputFormat, question, saved, result)
	case sessionstore.KindResolve:
		result, err := runtime.assistant.FollowUpResolve(ctx, assistsvc.ResolveRequest{
			WorkspacePath: saved.WorkspacePath,
			ScopeLabel:    saved.ScopeLabel,
			CommandTarget: saved.CommandTarget,
			ConfigPath:    saved.ConfigPath,
			TicketID:      saved.TicketID,
		}, question, saved.ThreadID, options)
		if err != nil {
			if errors.Is(err, conflictsvc.ErrNoConflict) {
				fmt.Fprintln(a.stderr, "assist chat failed: no active Git conflict state was found")
				return 1
			}
			fmt.Fprintf(a.stderr, "assist chat failed: %v\n", err)
			return 1
		}
		saved.ThreadID = result.ThreadID
		saved.LastQuestion = question
		saved.LastResponse = result.Response
		saved.Mode = string(result.Mode)
		saved.Audience = string(result.Audience)
		if _, err := store.SaveCurrent(saved); err != nil {
			fmt.Fprintf(a.stderr, "assist chat failed: %v\n", err)
			return 1
		}
		return renderAssistChatResult(a, outputFormat, question, saved, result)
	default:
		fmt.Fprintf(a.stderr, "assist chat failed: unsupported saved session kind %q\n", saved.Kind)
		return 1
	}
}

func (a *App) runAssistResume(args []string) int {
	if hasHelpFlag(args) {
		a.printAssistResumeUsage()
		return 0
	}
	if len(args) != 0 {
		fmt.Fprintln(a.stderr, "assist resume does not accept positional arguments")
		a.printAssistResumeUsage()
		return usageExitCode
	}

	saved, ok, err := a.currentAssistSession()
	if err != nil {
		fmt.Fprintf(a.stderr, "assist resume failed: %v\n", err)
		return 1
	}
	if !ok {
		fmt.Fprintln(a.stderr, "assist resume failed: no saved assist session was found")
		return 1
	}

	fmt.Fprintf(a.stdout, "%s: %s\n", sessionstore.ResumeTitle(saved.Kind), saved.Summary)
	fmt.Fprintf(a.stdout, "Scope: %s\n", blankIfEmpty(sessionstore.ResumeScopeLabel(saved), blankIfEmpty(saved.ScopeLabel, saved.WorkspacePath)))
	if saved.ThreadID != "" {
		fmt.Fprintf(a.stdout, "Thread: %s\n", saved.ThreadID)
	}
	if saved.LastQuestion != "" {
		fmt.Fprintf(a.stdout, "Last question: %s\n", saved.LastQuestion)
	}
	fmt.Fprintf(a.stdout, "Updated: %s\n", saved.UpdatedAt.Local().Format(time.RFC3339))
	fmt.Fprintln(a.stdout)
	fmt.Fprintln(a.stdout, "Continue with:")
	fmt.Fprintf(a.stdout, "  gig ask %q\n", sessionstore.ResumeQuestion(saved.Kind))
	return 0
}

func renderAssistChatResult(a *App, format outputFormat, question string, saved sessionstore.Session, result any) int {
	switch typed := result.(type) {
	case assistsvc.AuditResult:
		switch format {
		case outputFormatHuman:
			if _, err := fmt.Fprintf(a.stdout, "Follow-up: %s\n\n", question); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
			if err := output.RenderAssistantAudit(a.stdout, typed); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		case outputFormatJSON:
			if err := output.RenderJSON(a.stdout, struct {
				Command  string                `json:"command"`
				Question string                `json:"question"`
				Session  sessionstore.Session  `json:"session"`
				Result   assistsvc.AuditResult `json:"result"`
			}{
				Command:  "ask",
				Question: question,
				Session:  saved,
				Result:   typed,
			}); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		}
	case assistsvc.ReleaseResult:
		switch format {
		case outputFormatHuman:
			if _, err := fmt.Fprintf(a.stdout, "Follow-up: %s\n\n", question); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
			if err := output.RenderAssistantRelease(a.stdout, typed); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		case outputFormatJSON:
			if err := output.RenderJSON(a.stdout, struct {
				Command  string                  `json:"command"`
				Question string                  `json:"question"`
				Session  sessionstore.Session    `json:"session"`
				Result   assistsvc.ReleaseResult `json:"result"`
			}{
				Command:  "ask",
				Question: question,
				Session:  saved,
				Result:   typed,
			}); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		}
	case assistsvc.ResolveResult:
		switch format {
		case outputFormatHuman:
			if _, err := fmt.Fprintf(a.stdout, "Follow-up: %s\n\n", question); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
			if err := output.RenderAssistantResolve(a.stdout, typed); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		case outputFormatJSON:
			if err := output.RenderJSON(a.stdout, struct {
				Command  string                  `json:"command"`
				Question string                  `json:"question"`
				Session  sessionstore.Session    `json:"session"`
				Result   assistsvc.ResolveResult `json:"result"`
			}{
				Command:  "ask",
				Question: question,
				Session:  saved,
				Result:   typed,
			}); err != nil {
				fmt.Fprintf(a.stderr, "render failed: %v\n", err)
				return 1
			}
		}
	default:
		fmt.Fprintf(a.stderr, "render failed: unsupported assist result type %T\n", result)
		return 1
	}
	return 0
}

func (a *App) currentAssistSession() (sessionstore.Session, bool, error) {
	return a.currentAssistSessionForWorkarea(nil)
}

func (a *App) currentAssistSessionForWorkarea(current *workarea.Definition) (sessionstore.Session, bool, error) {
	store, err := sessionstore.NewStore()
	if err != nil {
		return sessionstore.Session{}, false, err
	}
	if current != nil {
		return store.CurrentForScope(current.Name, current.RepoTarget, current.Path)
	}
	if workareaStore, err := workarea.NewStore(); err == nil {
		if definition, ok, err := workareaStore.Current(); err == nil && ok {
			return store.CurrentForScope(definition.Name, definition.RepoTarget, definition.Path)
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		if session, ok, err := store.CurrentForScope("", "", cwd); err != nil || ok {
			return session, ok, err
		}
	}
	return store.Current()
}

func (a *App) persistAuditSession(scope commandScope, request assistsvc.AuditRequest, result assistsvc.AuditResult, mode assistsvc.RunMode, audience assistsvc.Audience) {
	store, err := sessionstore.NewStore()
	if err != nil {
		return
	}
	_, _ = store.SaveCurrent(sessionstore.Session{
		Kind:          sessionstore.KindAudit,
		ScopeLabel:    request.ScopeLabel,
		WorkspacePath: request.WorkspacePath,
		WorkareaName:  workareaName(scope),
		RepoTarget:    strings.TrimSpace(scope.RepoSpec),
		CommandTarget: request.CommandTarget,
		ConfigPath:    request.ConfigPath,
		TicketID:      request.TicketID,
		FromBranch:    request.FromBranch,
		ToBranch:      request.ToBranch,
		Environments:  append([]inspectsvc.Environment(nil), request.Environments...),
		Repositories:  append([]scm.Repository(nil), request.Repositories...),
		Audience:      string(audience),
		Mode:          string(mode),
		ThreadID:      result.ThreadID,
		Summary:       sessionstore.BuildSummary(sessionstore.KindAudit, request.TicketID, "", request.ScopeLabel),
		LastResponse:  result.Response,
	})
}

func (a *App) persistReleaseSession(scope commandScope, request assistsvc.ReleaseRequest, result assistsvc.ReleaseResult, mode assistsvc.RunMode, audience assistsvc.Audience) {
	store, err := sessionstore.NewStore()
	if err != nil {
		return
	}
	_, _ = store.SaveCurrent(sessionstore.Session{
		Kind:          sessionstore.KindRelease,
		ScopeLabel:    request.ScopeLabel,
		WorkspacePath: request.WorkspacePath,
		WorkareaName:  workareaName(scope),
		RepoTarget:    strings.TrimSpace(scope.RepoSpec),
		CommandTarget: request.CommandTarget,
		ConfigPath:    request.ConfigPath,
		TicketIDs:     append([]string(nil), request.TicketIDs...),
		ReleaseID:     request.ReleaseID,
		FromBranch:    request.FromBranch,
		ToBranch:      request.ToBranch,
		Environments:  append([]inspectsvc.Environment(nil), request.Environments...),
		Repositories:  append([]scm.Repository(nil), request.Repositories...),
		Audience:      string(audience),
		Mode:          string(mode),
		ThreadID:      result.ThreadID,
		Summary:       sessionstore.BuildSummary(sessionstore.KindRelease, "", request.ReleaseID, result.Bundle.ScopeLabel),
		LastResponse:  result.Response,
	})
}

func (a *App) persistResolveSession(scopeLabel, workspacePath, configPath, ticketID string, result assistsvc.ResolveResult, mode assistsvc.RunMode, audience assistsvc.Audience) {
	store, err := sessionstore.NewStore()
	if err != nil {
		return
	}
	_, _ = store.SaveCurrent(sessionstore.Session{
		Kind:          sessionstore.KindResolve,
		ScopeLabel:    scopeLabel,
		WorkspacePath: workspacePath,
		CommandTarget: fmt.Sprintf("--path %s", shellSingleQuote(workspacePath)),
		ConfigPath:    configPath,
		TicketID:      ticketID,
		Audience:      string(audience),
		Mode:          string(mode),
		ThreadID:      result.ThreadID,
		Summary:       sessionstore.BuildSummary(sessionstore.KindResolve, ticketID, "", scopeLabel),
		LastResponse:  result.Response,
	})
}

func workareaName(scope commandScope) string {
	if scope.Workarea == nil {
		return ""
	}
	return scope.Workarea.Name
}
