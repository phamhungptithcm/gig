package workarea

import (
	"path/filepath"
	"testing"
	"time"

	"gig/internal/scm"
)

func TestStoreSaveUseAndCurrent(t *testing.T) {
	t.Parallel()

	store := NewStoreAt(filepath.Join(t.TempDir(), "workareas.json"))
	store.now = func() time.Time {
		return time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	}

	saved, err := store.Save(Definition{
		Name:            "Payments",
		RepoTarget:      "github:acme/payments",
		FromBranch:      "develop",
		ToBranch:        "staging",
		EnvironmentSpec: "dev=develop,test=staging,prod=main",
	}, false)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if saved.Name != "Payments" {
		t.Fatalf("saved.Name = %q, want %q", saved.Name, "Payments")
	}

	current, ok, err := store.Current()
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if !ok {
		t.Fatalf("Current() ok = false, want true")
	}
	if current.Name != "Payments" {
		t.Fatalf("current.Name = %q, want %q", current.Name, "Payments")
	}

	used, err := store.Use("payments")
	if err != nil {
		t.Fatalf("Use() error = %v", err)
	}
	if used.LastUsedAt.IsZero() {
		t.Fatalf("used.LastUsedAt = zero, want timestamp")
	}
	if got := store.ScopePath(used); got != filepath.Join(filepath.Dir(store.FilePath()), "workareas", "payments") {
		t.Fatalf("ScopePath() = %q, want cache path", got)
	}
}

func TestStoreSaveRequiresRepoOrPath(t *testing.T) {
	t.Parallel()

	store := NewStoreAt(filepath.Join(t.TempDir(), "workareas.json"))
	if _, err := store.Save(Definition{Name: "Empty"}, false); err == nil {
		t.Fatal("Save() error = nil, want validation error")
	}
}

func TestStoreCurrentFallsBackToOnlyWorkarea(t *testing.T) {
	t.Parallel()

	store := NewStoreAt(filepath.Join(t.TempDir(), "workareas.json"))
	if _, err := store.Save(Definition{
		Name: "Ops",
		Path: filepath.Join(t.TempDir(), "ops"),
	}, false); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	current, ok, err := store.Current()
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if !ok {
		t.Fatalf("Current() ok = false, want true")
	}
	if current.Name != "Ops" {
		t.Fatalf("current.Name = %q, want %q", current.Name, "Ops")
	}
}

func TestStoreRecentRepositoriesOrdersBySelectionTime(t *testing.T) {
	t.Parallel()

	store := NewStoreAt(filepath.Join(t.TempDir(), "workareas.json"))
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time {
		now = now.Add(time.Minute)
		return now
	}

	if err := store.RecordRepositorySelection(scm.Repository{Name: "payments", Root: "github:acme/payments", Type: scm.TypeGitHub}); err != nil {
		t.Fatalf("RecordRepositorySelection() error = %v", err)
	}
	if err := store.RecordRepositorySelection(scm.Repository{Name: "billing", Root: "github:acme/billing", Type: scm.TypeGitHub}); err != nil {
		t.Fatalf("RecordRepositorySelection() error = %v", err)
	}

	recent, err := store.RecentRepositories(scm.TypeGitHub, 10)
	if err != nil {
		t.Fatalf("RecentRepositories() error = %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("len(recent) = %d, want 2", len(recent))
	}
	if recent[0].Root != "github:acme/billing" {
		t.Fatalf("recent[0].Root = %q, want github:acme/billing", recent[0].Root)
	}
}
