package cli_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gig/internal/cli"
)

func TestAppScanGolden(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	repoA := filepath.Join(workspace, "a-service")
	repoB := filepath.Join(workspace, "apps", "b-service")

	initRepository(t, repoA)
	initRepository(t, repoB)

	stdout, stderr, exitCode := runApp(t, "scan", "--path", workspace)
	if exitCode != 0 {
		t.Fatalf("scan exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "scan.golden", normalizeOutput(stdout, workspace))
}

func TestAppFindGolden(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	repoRoot := filepath.Join(workspace, "a-service")

	initRepository(t, repoRoot)
	writeFile(t, filepath.Join(repoRoot, "app.txt"), "hello")
	runGit(t, repoRoot, "add", "app.txt")
	runGit(t, repoRoot, "commit", "-m", "ABC-123 | service-a | fix login validation")

	stdout, stderr, exitCode := runApp(t, "find", "abc-123", "--path", workspace)
	if exitCode != 0 {
		t.Fatalf("find exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "find.golden", normalizeOutput(stdout, workspace))
}

func TestAppDiffGolden(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	repoRoot := filepath.Join(workspace, "a-service")

	initRepository(t, repoRoot)
	runGit(t, repoRoot, "checkout", "-b", "dev")
	writeFile(t, filepath.Join(repoRoot, "feature.txt"), "validation fix\n")
	runGit(t, repoRoot, "add", "feature.txt")
	runGit(t, repoRoot, "commit", "-m", "ABC-123 | service-a | add validation fix")
	runGit(t, repoRoot, "checkout", "main")
	runGit(t, repoRoot, "checkout", "-b", "test")

	stdout, stderr, exitCode := runApp(t, "diff", "--ticket", "ABC-123", "--from", "dev", "--to", "test", "--path", workspace)
	if exitCode != 0 {
		t.Fatalf("diff exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "diff.golden", normalizeOutput(stdout, workspace))
}

func TestAppInspectGolden(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	repoRoot := filepath.Join(workspace, "a-service")

	initRepository(t, repoRoot)
	runGit(t, repoRoot, "checkout", "-b", "dev")

	writeFile(t, filepath.Join(repoRoot, "app.txt"), "hello")
	runGit(t, repoRoot, "add", "app.txt")
	runGit(t, repoRoot, "commit", "-m", "ABC-123 | service-a | add validation fix")
	firstHash := strings.TrimSpace(runGit(t, repoRoot, "rev-parse", "HEAD"))

	writeFile(t, filepath.Join(repoRoot, "db", "migrations", "001_add_column.sql"), "alter table demo add column enabled int;\n")
	runGit(t, repoRoot, "add", "db/migrations/001_add_column.sql")
	runGit(t, repoRoot, "commit", "-m", "ABC-123 | service-a | add migration")

	runGit(t, repoRoot, "checkout", "main")
	runGit(t, repoRoot, "checkout", "-b", "test")
	runGit(t, repoRoot, "cherry-pick", firstHash)

	stdout, stderr, exitCode := runApp(t, "inspect", "abc-123", "--path", workspace)
	if exitCode != 0 {
		t.Fatalf("inspect exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "inspect.golden", normalizeOutput(stdout, workspace))
}

func TestAppEnvStatusGolden(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	repoRoot := filepath.Join(workspace, "a-service")

	initRepository(t, repoRoot)
	runGit(t, repoRoot, "checkout", "-b", "dev")

	writeFile(t, filepath.Join(repoRoot, "app.txt"), "hello")
	runGit(t, repoRoot, "add", "app.txt")
	runGit(t, repoRoot, "commit", "-m", "ABC-123 | service-a | add validation fix")
	firstHash := strings.TrimSpace(runGit(t, repoRoot, "rev-parse", "HEAD"))

	writeFile(t, filepath.Join(repoRoot, "db", "migrations", "001_add_column.sql"), "alter table demo add column enabled int;\n")
	runGit(t, repoRoot, "add", "db/migrations/001_add_column.sql")
	runGit(t, repoRoot, "commit", "-m", "ABC-123 | service-a | add migration")

	runGit(t, repoRoot, "checkout", "main")
	runGit(t, repoRoot, "checkout", "-b", "test")
	runGit(t, repoRoot, "cherry-pick", firstHash)

	stdout, stderr, exitCode := runApp(t, "env", "status", "ABC-123", "--path", workspace, "--envs", "dev=dev,test=test,prod=main")
	if exitCode != 0 {
		t.Fatalf("env status exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "env_status.golden", normalizeOutput(stdout, workspace))
}

func TestAppSubcommandHelpReturnsZero(t *testing.T) {
	t.Parallel()

	stdout, stderr, exitCode := runApp(t, "scan", "--help")
	if exitCode != 0 {
		t.Fatalf("scan --help exit code = %d, want 0", exitCode)
	}
	if stdout != "" {
		t.Fatalf("scan --help stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "Usage: gig scan --path .") {
		t.Fatalf("scan --help stderr = %q, want usage", stderr)
	}
}

func TestAppVersionReturnsBuildInfo(t *testing.T) {
	t.Parallel()

	stdout, stderr, exitCode := runApp(t, "version")
	if exitCode != 0 {
		t.Fatalf("version exit code = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("version stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "gig dev") {
		t.Fatalf("version stdout = %q, want build summary", stdout)
	}
}

func runApp(t *testing.T, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()

	var out bytes.Buffer
	var errOut bytes.Buffer

	app, err := cli.NewApp(&out, &errOut)
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	exitCode = app.Run(context.Background(), args)
	return out.String(), errOut.String(), exitCode
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()

	goldenPath := filepath.Join("testdata", name)
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", goldenPath, err)
	}

	if got != string(want) {
		t.Fatalf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, string(want))
	}
}

func normalizeOutput(output, workspace string) string {
	normalized := strings.ReplaceAll(output, workspace, "<WORKSPACE>")
	normalized = filepath.ToSlash(normalized)

	hashPattern := regexp.MustCompile(`\b[0-9a-f]{8,40}\b`)
	return hashPattern.ReplaceAllString(normalized, "<HASH>")
}

func initRepository(t *testing.T, repoRoot string) {
	t.Helper()

	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", repoRoot, err)
	}

	runGit(t, repoRoot, "init")
	runGit(t, repoRoot, "config", "user.name", "Gig Test")
	runGit(t, repoRoot, "config", "user.email", "gig@example.com")
	writeFile(t, filepath.Join(repoRoot, "README.md"), "seed")
	runGit(t, repoRoot, "add", "README.md")
	runGit(t, repoRoot, "commit", "-m", "initial commit")
	runGit(t, repoRoot, "branch", "-m", "main")
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

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
