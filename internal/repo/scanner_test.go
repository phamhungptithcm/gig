package repo_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"gig/internal/config"
	"gig/internal/repo"
	"gig/internal/scm"
	gitscm "gig/internal/scm/git"
	svnscm "gig/internal/scm/svn"
	"gig/internal/ticket"
)

func TestScannerDiscoversNestedRepositories(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	repoA := filepath.Join(workspace, "service-a")
	repoB := filepath.Join(workspace, "apps", "service-b")

	mustMkdir(t, filepath.Join(repoA, ".git"))
	mustMkdir(t, filepath.Join(repoB, ".git"))

	scanner := newScanner(t)
	repositories, err := scanner.Discover(context.Background(), workspace)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(repositories) != 2 {
		t.Fatalf("Discover() returned %d repositories, want 2", len(repositories))
	}

	gotRoots := []string{repositories[0].Root, repositories[1].Root}
	wantRoots := []string{repoA, repoB}
	sort.Strings(gotRoots)
	sort.Strings(wantRoots)

	if !reflect.DeepEqual(gotRoots, wantRoots) {
		t.Fatalf("repository roots = %#v, want %#v", gotRoots, wantRoots)
	}
}

func TestScannerDetectsEnclosingRepository(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	repoRoot := filepath.Join(workspace, "service-a")
	nestedPath := filepath.Join(repoRoot, "pkg", "module")

	mustMkdir(t, filepath.Join(repoRoot, ".git"))
	mustMkdir(t, nestedPath)

	scanner := newScanner(t)
	repositories, err := scanner.Discover(context.Background(), nestedPath)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(repositories) != 1 {
		t.Fatalf("Discover() returned %d repositories, want 1", len(repositories))
	}
	if repositories[0].Root != repoRoot {
		t.Fatalf("repositories[0].Root = %q, want %q", repositories[0].Root, repoRoot)
	}
	if repositories[0].Type != scm.TypeGit {
		t.Fatalf("repositories[0].Type = %q, want %q", repositories[0].Type, scm.TypeGit)
	}
}

func newScanner(t *testing.T) *repo.Scanner {
	t.Helper()

	parser, err := ticket.NewParser(config.Default().TicketPattern)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	registry := scm.NewRegistry(
		gitscm.NewAdapter(parser),
		svnscm.NewAdapter(),
	)

	return repo.NewScanner(registry)
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}
