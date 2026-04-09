package conflict_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gig/internal/config"
	conflictsvc "gig/internal/conflict"
	"gig/internal/repo"
	"gig/internal/scm"
	gitscm "gig/internal/scm/git"
	"gig/internal/ticket"
)

func TestServiceStatusDetectsMergeConflict(t *testing.T) {
	t.Parallel()

	repoRoot := createMergeConflictRepo(t)
	service := newService(t)

	status, err := service.Status(context.Background(), repoRoot, "ABC-123")
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if got := status.Operation.Type; got != scm.ConflictOperationMerge {
		t.Fatalf("operation type = %q, want %q", got, scm.ConflictOperationMerge)
	}
	if status.ResolvableFiles != 1 {
		t.Fatalf("ResolvableFiles = %d, want 1", status.ResolvableFiles)
	}
	if len(status.Files) != 1 || status.Files[0].BlockCount != 1 {
		t.Fatalf("files = %#v, want one supported file with one conflict block", status.Files)
	}
	if !contains(status.Operation.IncomingSide.TicketIDs, "ABC-123") {
		t.Fatalf("incoming tickets = %v, want ABC-123", status.Operation.IncomingSide.TicketIDs)
	}
}

func TestServiceStatusDetectsRebaseConflict(t *testing.T) {
	t.Parallel()

	repoRoot := createRebaseConflictRepo(t)
	service := newService(t)

	status, err := service.Status(context.Background(), repoRoot, "")
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if got := status.Operation.Type; got != scm.ConflictOperationRebase {
		t.Fatalf("operation type = %q, want %q", got, scm.ConflictOperationRebase)
	}
	if got := status.Operation.IncomingSide.Label; got != "Replayed pick" {
		t.Fatalf("incoming label = %q, want Replayed pick", got)
	}
	if got := status.Operation.SequenceBranch; got == "" {
		t.Fatalf("sequence branch should be populated during rebase")
	}
}

func TestServiceStatusDetectsCherryPickConflict(t *testing.T) {
	t.Parallel()

	repoRoot := createCherryPickConflictRepo(t)
	service := newService(t)

	status, err := service.Status(context.Background(), repoRoot, "")
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if got := status.Operation.Type; got != scm.ConflictOperationCherryPick {
		t.Fatalf("operation type = %q, want %q", got, scm.ConflictOperationCherryPick)
	}
	if got := status.Operation.ContinuationCommand; got != "git cherry-pick --continue" {
		t.Fatalf("continuation command = %q, want git cherry-pick --continue", got)
	}
}

func TestServiceApplyResolutionAndStageFile(t *testing.T) {
	t.Parallel()

	repoRoot := createMergeConflictRepo(t)
	service := newService(t)

	session, active, err := service.LoadActiveConflict(context.Background(), repoRoot, "", 0, "ABC-123")
	if err != nil {
		t.Fatalf("LoadActiveConflict() error = %v", err)
	}
	if active == nil {
		t.Fatalf("LoadActiveConflict() returned nil active conflict")
	}

	if err := service.ApplyResolution(context.Background(), repoRoot, active.File.Path, active.Block.Index, active.Operation, conflictsvc.ResolutionIncoming); err != nil {
		t.Fatalf("ApplyResolution() error = %v", err)
	}
	if err := service.StageFile(context.Background(), repoRoot, active.File.Path); err != nil {
		t.Fatalf("StageFile() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(repoRoot, active.File.Path))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.Contains(string(content), "<<<<<<<") {
		t.Fatalf("resolved file still contains conflict markers: %q", string(content))
	}

	status, err := service.Status(context.Background(), repoRoot, "")
	if err != nil {
		t.Fatalf("Status() after stage error = %v", err)
	}
	if len(status.Files) != 0 {
		t.Fatalf("status files after stage = %d, want 0", len(status.Files))
	}
	if session.CurrentFile == "" {
		t.Fatalf("session should include current file")
	}
}

func newService(t *testing.T) *conflictsvc.Service {
	t.Helper()

	parser, err := ticket.NewParser(config.Default().TicketPattern)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	registry := scm.NewRegistry(gitscm.NewAdapter(parser))
	scanner := repo.NewScanner(registry)
	return conflictsvc.NewService(scanner, registry, parser)
}

func createMergeConflictRepo(t *testing.T) string {
	t.Helper()

	repoRoot := initRepository(t)
	writeFile(t, filepath.Join(repoRoot, "app.txt"), "shared line\n")
	runGit(t, repoRoot, "add", "app.txt")
	runGit(t, repoRoot, "commit", "-m", "BASE-1 seed app file")

	runGit(t, repoRoot, "checkout", "-b", "feature/ABC-123")
	writeFile(t, filepath.Join(repoRoot, "app.txt"), "feature line\n")
	runGit(t, repoRoot, "commit", "-am", "ABC-123 update app behavior")

	runGit(t, repoRoot, "checkout", "main")
	writeFile(t, filepath.Join(repoRoot, "app.txt"), "main line\n")
	runGit(t, repoRoot, "commit", "-am", "OPS-99 tighten validation")

	if output, err := runGitAllowFailure(repoRoot, "merge", "feature/ABC-123"); err == nil {
		t.Fatalf("merge unexpectedly succeeded: %s", output)
	}

	return repoRoot
}

func createRebaseConflictRepo(t *testing.T) string {
	t.Helper()

	repoRoot := initRepository(t)
	writeFile(t, filepath.Join(repoRoot, "app.txt"), "shared line\n")
	runGit(t, repoRoot, "add", "app.txt")
	runGit(t, repoRoot, "commit", "-m", "BASE-1 seed app file")

	runGit(t, repoRoot, "checkout", "-b", "feature/ABC-123")
	writeFile(t, filepath.Join(repoRoot, "app.txt"), "feature line\n")
	runGit(t, repoRoot, "commit", "-am", "ABC-123 update app behavior")

	runGit(t, repoRoot, "checkout", "main")
	writeFile(t, filepath.Join(repoRoot, "app.txt"), "main line\n")
	runGit(t, repoRoot, "commit", "-am", "OPS-99 tighten validation")

	runGit(t, repoRoot, "checkout", "feature/ABC-123")
	if output, err := runGitAllowFailure(repoRoot, "rebase", "main"); err == nil {
		t.Fatalf("rebase unexpectedly succeeded: %s", output)
	}

	return repoRoot
}

func createCherryPickConflictRepo(t *testing.T) string {
	t.Helper()

	repoRoot := initRepository(t)
	writeFile(t, filepath.Join(repoRoot, "app.txt"), "shared line\n")
	runGit(t, repoRoot, "add", "app.txt")
	runGit(t, repoRoot, "commit", "-m", "BASE-1 seed app file")

	runGit(t, repoRoot, "checkout", "-b", "feature/ABC-123")
	writeFile(t, filepath.Join(repoRoot, "app.txt"), "feature line\n")
	runGit(t, repoRoot, "commit", "-am", "ABC-123 update app behavior")
	featureHash := strings.TrimSpace(runGit(t, repoRoot, "rev-parse", "HEAD"))

	runGit(t, repoRoot, "checkout", "main")
	writeFile(t, filepath.Join(repoRoot, "app.txt"), "main line\n")
	runGit(t, repoRoot, "commit", "-am", "OPS-99 tighten validation")

	if output, err := runGitAllowFailure(repoRoot, "cherry-pick", featureHash); err == nil {
		t.Fatalf("cherry-pick unexpectedly succeeded: %s", output)
	}

	return repoRoot
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

func runGitAllowFailure(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
