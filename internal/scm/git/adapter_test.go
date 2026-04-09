package git_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gig/internal/config"
	"gig/internal/scm"
	gitscm "gig/internal/scm/git"
	"gig/internal/ticket"
)

func TestAdapterSearchCommits(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for adapter tests")
	}

	repoRoot := initRepository(t)
	runGit(t, repoRoot, "checkout", "-b", "dev")
	writeFile(t, filepath.Join(repoRoot, "app.txt"), "hello")
	runGit(t, repoRoot, "add", "app.txt")
	runGit(t, repoRoot, "commit", "-m", "ABC-123 fix login bug")

	adapter := newAdapter(t)
	commits, err := adapter.SearchCommits(context.Background(), repoRoot, scm.SearchQuery{TicketID: "ABC-123"})
	if err != nil {
		t.Fatalf("SearchCommits() error = %v", err)
	}

	if len(commits) != 1 {
		t.Fatalf("SearchCommits() returned %d commits, want 1", len(commits))
	}
	if !strings.Contains(commits[0].Subject, "ABC-123") {
		t.Fatalf("SearchCommits() subject = %q, want ticket ID included", commits[0].Subject)
	}
}

func TestAdapterCompareBranchesDetectsMissingAndCherryPickedCommits(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for adapter tests")
	}

	repoRoot := initRepository(t)
	runGit(t, repoRoot, "checkout", "-b", "dev")
	writeFile(t, filepath.Join(repoRoot, "feature.txt"), "line one\n")
	runGit(t, repoRoot, "add", "feature.txt")
	runGit(t, repoRoot, "commit", "-m", "ABC-123 add feature flag")
	sourceHash := strings.TrimSpace(runGit(t, repoRoot, "rev-parse", "HEAD"))

	runGit(t, repoRoot, "checkout", "main")
	runGit(t, repoRoot, "checkout", "-b", "test")

	adapter := newAdapter(t)
	result, err := adapter.CompareBranches(context.Background(), repoRoot, scm.CompareQuery{
		TicketID:   "ABC-123",
		FromBranch: "dev",
		ToBranch:   "test",
	})
	if err != nil {
		t.Fatalf("CompareBranches() error = %v", err)
	}

	if len(result.MissingCommits) != 1 {
		t.Fatalf("CompareBranches() missing = %d commits, want 1", len(result.MissingCommits))
	}
	if result.MissingCommits[0].Hash != sourceHash {
		t.Fatalf("CompareBranches() missing hash = %q, want %q", result.MissingCommits[0].Hash, sourceHash)
	}

	runGit(t, repoRoot, "cherry-pick", sourceHash)

	result, err = adapter.CompareBranches(context.Background(), repoRoot, scm.CompareQuery{
		TicketID:   "ABC-123",
		FromBranch: "dev",
		ToBranch:   "test",
	})
	if err != nil {
		t.Fatalf("CompareBranches() after cherry-pick error = %v", err)
	}

	if len(result.MissingCommits) != 0 {
		t.Fatalf("CompareBranches() after cherry-pick missing = %d commits, want 0", len(result.MissingCommits))
	}
}

func newAdapter(t *testing.T) *gitscm.Adapter {
	t.Helper()

	parser, err := ticket.NewParser(config.Default().TicketPattern)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	return gitscm.NewAdapter(parser)
}

func initRepository(t *testing.T) string {
	t.Helper()

	repoRoot := t.TempDir()
	runGit(t, repoRoot, "init")
	runGit(t, repoRoot, "config", "user.name", "Gig Test")
	runGit(t, repoRoot, "config", "user.email", "gig@example.com")
	writeFile(t, filepath.Join(repoRoot, "README.md"), "seed")
	runGit(t, repoRoot, "add", "README.md")
	runGit(t, repoRoot, "commit", "-m", "initial commit")
	runGit(t, repoRoot, "branch", "-m", "main")

	return repoRoot
}

func runGit(t *testing.T, repoRoot string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(output)))
	}

	return string(output)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
