package session

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	inspectsvc "gig/internal/inspect"
	"gig/internal/scm"
)

type Kind string

const (
	KindAudit   Kind = "audit"
	KindRelease Kind = "release"
	KindResolve Kind = "resolve"
)

type Session struct {
	ID            string                   `json:"id"`
	Kind          Kind                     `json:"kind"`
	ScopeLabel    string                   `json:"scopeLabel,omitempty"`
	WorkspacePath string                   `json:"workspacePath,omitempty"`
	WorkareaName  string                   `json:"workareaName,omitempty"`
	RepoTarget    string                   `json:"repoTarget,omitempty"`
	CommandTarget string                   `json:"commandTarget,omitempty"`
	ConfigPath    string                   `json:"configPath,omitempty"`
	TicketID      string                   `json:"ticketId,omitempty"`
	TicketIDs     []string                 `json:"ticketIds,omitempty"`
	ReleaseID     string                   `json:"releaseId,omitempty"`
	FromBranch    string                   `json:"fromBranch,omitempty"`
	ToBranch      string                   `json:"toBranch,omitempty"`
	Environments  []inspectsvc.Environment `json:"environments,omitempty"`
	Repositories  []scm.Repository         `json:"repositories,omitempty"`
	Audience      string                   `json:"audience,omitempty"`
	Mode          string                   `json:"mode,omitempty"`
	ThreadID      string                   `json:"threadId,omitempty"`
	Summary       string                   `json:"summary,omitempty"`
	LastQuestion  string                   `json:"lastQuestion,omitempty"`
	LastResponse  string                   `json:"lastResponse,omitempty"`
	CreatedAt     time.Time                `json:"createdAt,omitempty"`
	UpdatedAt     time.Time                `json:"updatedAt,omitempty"`
}

type State struct {
	Current  string    `json:"current,omitempty"`
	Sessions []Session `json:"sessions,omitempty"`
}

func BuildID(session Session) string {
	parts := []string{
		string(session.Kind),
		normalizeIDPart(sessionScopeIdentity(session)),
		normalizeIDPart(session.ScopeLabel),
		normalizeIDPart(session.TicketID),
		normalizeIDPart(session.ReleaseID),
	}
	return strings.Join(parts, "|")
}

func BuildSummary(kind Kind, ticketID, releaseID, scopeLabel string) string {
	switch kind {
	case KindAudit:
		return fmt.Sprintf("Audit %s on %s", blankAsDefault(ticketID, "ticket"), blankAsDefault(scopeLabel, "current scope"))
	case KindRelease:
		return fmt.Sprintf("Release %s on %s", blankAsDefault(releaseID, "release"), blankAsDefault(scopeLabel, "current scope"))
	case KindResolve:
		if strings.TrimSpace(ticketID) != "" {
			return fmt.Sprintf("Resolve %s in %s", ticketID, blankAsDefault(scopeLabel, "current scope"))
		}
		return fmt.Sprintf("Resolve conflict in %s", blankAsDefault(scopeLabel, "current scope"))
	default:
		return blankAsDefault(scopeLabel, "assist session")
	}
}

func normalizeIDPart(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "_"
	}
	return value
}

func blankAsDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func ResumeTitle(kind Kind) string {
	switch kind {
	case KindAudit:
		return "Resume last audit"
	case KindRelease:
		return "Resume last release"
	case KindResolve:
		return "Resume last resolve"
	default:
		return "Resume last AI brief"
	}
}

func ResumeQuestion(kind Kind) string {
	switch kind {
	case KindRelease:
		return "what changed since the last release brief?"
	case KindResolve:
		return "what is the safest next conflict step?"
	default:
		return "what is still blocked?"
	}
}

func ResumeScopeLabel(session Session) string {
	switch {
	case strings.TrimSpace(session.WorkareaName) != "":
		return "workarea " + strings.TrimSpace(session.WorkareaName)
	case strings.TrimSpace(session.RepoTarget) != "":
		return strings.TrimSpace(session.RepoTarget)
	case strings.TrimSpace(session.ScopeLabel) != "":
		return strings.TrimSpace(session.ScopeLabel)
	case strings.TrimSpace(session.WorkspacePath) != "":
		return strings.TrimSpace(session.WorkspacePath)
	default:
		return ""
	}
}

func sessionScopeIdentity(session Session) string {
	switch {
	case strings.TrimSpace(session.WorkareaName) != "":
		return "workarea:" + strings.TrimSpace(session.WorkareaName)
	case strings.TrimSpace(session.RepoTarget) != "":
		return "repo:" + strings.TrimSpace(session.RepoTarget)
	case strings.TrimSpace(session.WorkspacePath) != "":
		return "workspace:" + normalizeWorkspacePath(session.WorkspacePath)
	case strings.TrimSpace(session.ScopeLabel) != "":
		return "scope:" + strings.TrimSpace(session.ScopeLabel)
	default:
		return "scope:_"
	}
}

func normalizeWorkspacePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return filepath.Clean(value)
}
