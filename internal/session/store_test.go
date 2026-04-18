package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	inspectsvc "gig/internal/inspect"
	"gig/internal/scm"
)

func TestStoreSaveCurrentAndLoadCurrent(t *testing.T) {
	t.Parallel()

	store := NewStoreAt(filepath.Join(t.TempDir(), "assist-sessions.json"))
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	saved, err := store.SaveCurrent(Session{
		Kind:          KindAudit,
		ScopeLabel:    "github:acme/payments",
		WorkspacePath: "/tmp/workspace",
		TicketID:      "ABC-123",
		FromBranch:    "staging",
		ToBranch:      "main",
		Environments:  []inspectsvc.Environment{{Name: "prod", Branch: "main"}},
		Repositories:  []scm.Repository{{Root: "github:acme/payments", Type: scm.TypeGitHub}},
		ThreadID:      "thread-123",
		Summary:       "Audit ABC-123 on github:acme/payments",
	})
	if err != nil {
		t.Fatalf("SaveCurrent() error = %v", err)
	}
	if saved.ID == "" {
		t.Fatalf("saved session id = empty, want generated id")
	}

	current, ok, err := store.Current()
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if !ok {
		t.Fatalf("Current() ok = false, want true")
	}
	if current.ThreadID != "thread-123" {
		t.Fatalf("thread id = %q, want thread-123", current.ThreadID)
	}
	if current.UpdatedAt.IsZero() {
		t.Fatalf("updatedAt = zero, want timestamp")
	}
}

func TestStoreTouchUpdatesCurrentSession(t *testing.T) {
	t.Parallel()

	store := NewStoreAt(filepath.Join(t.TempDir(), "assist-sessions.json"))
	store.now = func() time.Time {
		return time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	}

	saved, err := store.SaveCurrent(Session{
		Kind:       KindRelease,
		ScopeLabel: "github:acme/payments",
		ReleaseID:  "rel-2026-04-14",
		ThreadID:   "thread-1",
		Summary:    "Release rel-2026-04-14 on github:acme/payments",
	})
	if err != nil {
		t.Fatalf("SaveCurrent() error = %v", err)
	}

	store.now = func() time.Time {
		return time.Date(2026, 4, 14, 12, 5, 0, 0, time.UTC)
	}

	touched, err := store.Touch(saved.ID, "what is still blocked?", "One ticket is still blocked.", "thread-1")
	if err != nil {
		t.Fatalf("Touch() error = %v", err)
	}
	if touched.LastQuestion != "what is still blocked?" {
		t.Fatalf("LastQuestion = %q, want updated value", touched.LastQuestion)
	}
	if touched.LastResponse != "One ticket is still blocked." {
		t.Fatalf("LastResponse = %q, want updated value", touched.LastResponse)
	}
	if touched.UpdatedAt.Equal(saved.UpdatedAt) {
		t.Fatalf("UpdatedAt = %v, want a newer timestamp than %v", touched.UpdatedAt, saved.UpdatedAt)
	}
}

func TestStoreCurrentForScopeWithEmptyFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "assist-sessions.json")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := NewStoreAt(path)
	current, ok, err := store.CurrentForScope("payments", "github:acme/payments", "")
	if err != nil {
		t.Fatalf("CurrentForScope() error = %v", err)
	}
	if ok {
		t.Fatalf("CurrentForScope() ok = true, want false with empty file: %#v", current)
	}
}

func TestStoreCurrentForScopePrefersWorkareaThenRepoTarget(t *testing.T) {
	t.Parallel()

	store := NewStoreAt(filepath.Join(t.TempDir(), "assist-sessions.json"))
	store.now = func() time.Time {
		return time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	}

	if _, err := store.SaveCurrent(Session{
		Kind:       KindAudit,
		ScopeLabel: "github:acme/payments",
		RepoTarget: "github:acme/payments",
		TicketID:   "ABC-123",
		ThreadID:   "thread-repo",
		Summary:    "Audit ABC-123 on github:acme/payments",
	}); err != nil {
		t.Fatalf("SaveCurrent(repo) error = %v", err)
	}

	store.now = func() time.Time {
		return time.Date(2026, 4, 14, 12, 5, 0, 0, time.UTC)
	}

	if _, err := store.SaveCurrent(Session{
		Kind:         KindAudit,
		ScopeLabel:   "github:acme/payments",
		WorkareaName: "payments",
		RepoTarget:   "github:acme/payments",
		TicketID:     "ABC-123",
		ThreadID:     "thread-workarea",
		Summary:      "Audit ABC-123 on github:acme/payments",
	}); err != nil {
		t.Fatalf("SaveCurrent(workarea) error = %v", err)
	}

	current, ok, err := store.CurrentForScope("payments", "github:acme/payments", "")
	if err != nil {
		t.Fatalf("CurrentForScope(workarea) error = %v", err)
	}
	if !ok {
		t.Fatal("CurrentForScope(workarea) ok = false, want true")
	}
	if current.ThreadID != "thread-workarea" {
		t.Fatalf("CurrentForScope(workarea) thread = %q, want thread-workarea", current.ThreadID)
	}

	current, ok, err = store.CurrentForScope("billing", "github:acme/payments", "")
	if err != nil {
		t.Fatalf("CurrentForScope(repo) error = %v", err)
	}
	if !ok {
		t.Fatal("CurrentForScope(repo) ok = false, want true")
	}
	if current.ThreadID != "thread-repo" {
		t.Fatalf("CurrentForScope(repo) thread = %q, want thread-repo", current.ThreadID)
	}
}

func TestStoreCurrentForScopeReturnsParseErrorForCorruptedFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "assist-sessions.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := NewStoreAt(path)
	if _, _, err := store.CurrentForScope("payments", "github:acme/payments", ""); err == nil {
		t.Fatal("CurrentForScope() error = nil, want parse error")
	}
}
