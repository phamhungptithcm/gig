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

func TestStoreSaveInferredDefaultsFillsBlanksOnly(t *testing.T) {
	t.Parallel()

	store := NewStoreAt(filepath.Join(t.TempDir(), "workareas.json"))
	store.now = func() time.Time {
		return time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	}

	if _, err := store.Save(Definition{
		Name:       "Payments",
		RepoTarget: "github:acme/payments",
	}, true); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	updated, changed, err := store.SaveInferredDefaults("Payments", "staging", "main", "staging=staging,prod=main")
	if err != nil {
		t.Fatalf("SaveInferredDefaults() error = %v", err)
	}
	if !changed {
		t.Fatal("SaveInferredDefaults() changed = false, want true")
	}
	if updated.FromBranch != "staging" || updated.ToBranch != "main" {
		t.Fatalf("branches = %s -> %s, want staging -> main", updated.FromBranch, updated.ToBranch)
	}
	if updated.EnvironmentSpec != "staging=staging,prod=main" {
		t.Fatalf("EnvironmentSpec = %q, want inferred mapping", updated.EnvironmentSpec)
	}

	updated, changed, err = store.SaveInferredDefaults("Payments", "release/test", "production", "qa=qa,prod=production")
	if err != nil {
		t.Fatalf("SaveInferredDefaults() second call error = %v", err)
	}
	if changed {
		t.Fatal("SaveInferredDefaults() second call changed = true, want false")
	}
	if updated.FromBranch != "staging" || updated.ToBranch != "main" {
		t.Fatalf("branches changed unexpectedly to %s -> %s", updated.FromBranch, updated.ToBranch)
	}
	if updated.EnvironmentSpec != "staging=staging,prod=main" {
		t.Fatalf("EnvironmentSpec changed unexpectedly to %q", updated.EnvironmentSpec)
	}
}

func TestStoreEnsureRemoteRepositoryCreatesAndReusesCurrentWorkarea(t *testing.T) {
	t.Parallel()

	store := NewStoreAt(filepath.Join(t.TempDir(), "workareas.json"))
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time {
		now = now.Add(time.Minute)
		return now
	}

	saved, created, err := store.EnsureRemoteRepository(scm.Repository{
		Name: "payments",
		Root: "github:acme/payments",
		Type: scm.TypeGitHub,
	})
	if err != nil {
		t.Fatalf("EnsureRemoteRepository() error = %v", err)
	}
	if !created {
		t.Fatal("EnsureRemoteRepository() created = false, want true")
	}
	if saved.Name != "payments" {
		t.Fatalf("saved.Name = %q, want payments", saved.Name)
	}

	reused, created, err := store.EnsureRemoteRepository(scm.Repository{
		Name: "payments",
		Root: "github:acme/payments",
		Type: scm.TypeGitHub,
	})
	if err != nil {
		t.Fatalf("EnsureRemoteRepository() second call error = %v", err)
	}
	if created {
		t.Fatal("EnsureRemoteRepository() second call created = true, want false")
	}
	if reused.Name != "payments" {
		t.Fatalf("reused.Name = %q, want payments", reused.Name)
	}

	if _, _, err := store.EnsureRemoteRepository(scm.Repository{
		Name: "payments",
		Root: "github:other/payments",
		Type: scm.TypeGitHub,
	}); err != nil {
		t.Fatalf("EnsureRemoteRepository() collision case error = %v", err)
	}
	workareas, current, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(workareas) != 2 {
		t.Fatalf("len(workareas) = %d, want 2", len(workareas))
	}
	if current != "payments-2" {
		t.Fatalf("current = %q, want payments-2", current)
	}
}
