package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"gig/internal/cli"
	"gig/internal/workarea"
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
	runGit(t, repoRoot, "commit", "-m", "ABC-123 | service-a | add validation fix", "-m", "Depends-On: XYZ-456")
	firstHash := strings.TrimSpace(runGit(t, repoRoot, "rev-parse", "HEAD"))

	writeFile(t, filepath.Join(repoRoot, "db", "migrations", "001_add_column.sql"), "alter table demo add column enabled int;\n")
	runGit(t, repoRoot, "add", "db/migrations/001_add_column.sql")
	runGit(t, repoRoot, "commit", "-m", "ABC-123 | service-a | add migration")

	runGit(t, repoRoot, "checkout", "-b", "test", firstHash)

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

	runGit(t, repoRoot, "checkout", "-b", "test", firstHash)

	stdout, stderr, exitCode := runApp(t, "env", "status", "ABC-123", "--path", workspace, "--envs", "dev=dev,test=test,prod=main")
	if exitCode != 0 {
		t.Fatalf("env status exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "env_status.golden", normalizeOutput(stdout, workspace))
}

func TestAppPlanGolden(t *testing.T) {
	t.Parallel()

	workspace := createPromotionFixture(t)

	stdout, stderr, exitCode := runApp(t, "plan", "--ticket", "ABC-123", "--from", "test", "--to", "main", "--path", workspace, "--envs", "dev=dev,test=test,prod=main")
	if exitCode != 0 {
		t.Fatalf("plan exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "plan.golden", normalizeOutput(stdout, workspace))
}

func TestAppPlanJSONGolden(t *testing.T) {
	t.Parallel()

	workspace := createPromotionFixture(t)

	stdout, stderr, exitCode := runApp(t, "plan", "--ticket", "ABC-123", "--from", "test", "--to", "main", "--path", workspace, "--envs", "dev=dev,test=test,prod=main", "--format", "json")
	if exitCode != 0 {
		t.Fatalf("plan json exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "plan_json.golden", normalizeOutput(stdout, workspace))
}

func TestAppVerifyGolden(t *testing.T) {
	t.Parallel()

	workspace := createPromotionFixture(t)

	stdout, stderr, exitCode := runApp(t, "verify", "--ticket", "ABC-123", "--from", "test", "--to", "main", "--path", workspace, "--envs", "dev=dev,test=test,prod=main")
	if exitCode != 0 {
		t.Fatalf("verify exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "verify.golden", normalizeOutput(stdout, workspace))
}

func TestAppVerifyTicketFileGolden(t *testing.T) {
	t.Parallel()

	workspace := createPromotionFixture(t)
	ticketFile := filepath.Join(workspace, "tickets.txt")
	writeFile(t, ticketFile, `
# release candidates
abc-123

XYZ-999
ABC-123
`)

	stdout, stderr, exitCode := runApp(t, "verify", "--ticket-file", ticketFile, "--from", "test", "--to", "main", "--path", workspace, "--envs", "dev=dev,test=test,prod=main")
	if exitCode != 0 {
		t.Fatalf("verify ticket file exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "verify_ticket_file.golden", normalizeOutput(stdout, workspace))
}

func TestAppVerifyTicketFileJSONGolden(t *testing.T) {
	t.Parallel()

	workspace := createPromotionFixture(t)
	ticketFile := filepath.Join(workspace, "tickets.txt")
	writeFile(t, ticketFile, `
ABC-123
XYZ-999
`)

	stdout, stderr, exitCode := runApp(t, "verify", "--ticket-file", ticketFile, "--from", "test", "--to", "main", "--path", workspace, "--envs", "dev=dev,test=test,prod=main", "--format", "json")
	if exitCode != 0 {
		t.Fatalf("verify ticket file json exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "verify_ticket_file_json.golden", normalizeOutput(stdout, workspace))
}

func TestAppManifestGenerateGolden(t *testing.T) {
	t.Parallel()

	workspace := createPromotionFixture(t)
	writeFile(t, filepath.Join(workspace, "gig.yaml"), `
ticketPattern: '\b[A-Z][A-Z0-9]+-\d+\b'
environments:
  - name: build
    branch: dev
  - name: qa
    branch: test
  - name: prod
    branch: main
repositories:
  - path: a-service
    service: Accounts API
    owner: Backend Team
    kind: app
    notes:
      - Verify login and billing summary.
`)

	stdout, stderr, exitCode := runApp(t, "manifest", "generate", "--ticket", "ABC-123", "--from", "test", "--to", "main", "--path", workspace)
	if exitCode != 0 {
		t.Fatalf("manifest generate exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "manifest_generate.golden", normalizeOutput(stdout, workspace))
}

func TestAppRootHelpGroupsCommonFlows(t *testing.T) {
	t.Parallel()

	stdout, stderr, exitCode := runApp(t, "--help")
	if exitCode != 0 {
		t.Fatalf("help exit code = %d, stdout = %q, stderr = %q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "First-time users") || !strings.Contains(stderr, "Core workflows") || !strings.Contains(stderr, "Commands") {
		t.Fatalf("stderr = %q, want grouped help sections", stderr)
	}
	if !strings.Contains(stderr, "gig inspect ABC-123 --repo github:owner/name") {
		t.Fatalf("stderr = %q, want remote-first example", stderr)
	}
}

func TestAppInspectHelpShowsExamplesAndNextCommands(t *testing.T) {
	t.Parallel()

	stdout, stderr, exitCode := runApp(t, "inspect", "--help")
	if exitCode != 0 {
		t.Fatalf("inspect help exit code = %d, stdout = %q, stderr = %q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "Start here") || !strings.Contains(stderr, "Common flags") || !strings.Contains(stderr, "Next commands") {
		t.Fatalf("stderr = %q, want structured inspect help", stderr)
	}
	if !strings.Contains(stderr, "gig plan --ticket ABC-123") {
		t.Fatalf("stderr = %q, want next-command guidance", stderr)
	}
}

func TestAppVerifyUsageErrorSuggestsNextCommand(t *testing.T) {
	t.Parallel()

	stdout, stderr, exitCode := runApp(t, "verify", "ABC-123")
	if exitCode != 2 {
		t.Fatalf("verify exit code = %d, stdout = %q, stderr = %q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "verify failed: use flags instead of positional arguments.") {
		t.Fatalf("stderr = %q, want friendlier verify error", stderr)
	}
	if !strings.Contains(stderr, "Try next") || !strings.Contains(stderr, "gig verify --ticket ABC-123") {
		t.Fatalf("stderr = %q, want next-step examples", stderr)
	}
}

func TestAppWorkareaAddUseAndShow(t *testing.T) {
	workareaFile := filepath.Join(t.TempDir(), "workareas.json")

	stdout, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "", workareaFile, "workarea", "add", "payments", "--repo", "github:acme/payments", "--from", "develop", "--to", "staging", "--envs", "dev=develop,test=staging,prod=main", "--use")
	if exitCode != 0 {
		t.Fatalf("workarea add exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "Workarea payments") || !strings.Contains(stdout, "Current: yes") {
		t.Fatalf("stdout = %q, want saved workarea summary", stdout)
	}

	stdout, stderr, exitCode = runAppWithInputAndWorkareaFile(t, "", workareaFile, "workarea", "show")
	if exitCode != 0 {
		t.Fatalf("workarea show exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "Target: github:acme/payments") || !strings.Contains(stdout, "Promotion: develop -> staging") {
		t.Fatalf("stdout = %q, want current workarea detail", stdout)
	}
}

func TestAppWorkareaUseInteractive(t *testing.T) {
	workareaFile := filepath.Join(t.TempDir(), "workareas.json")

	if _, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "", workareaFile, "workarea", "add", "payments", "--repo", "github:acme/payments"); exitCode != 0 {
		t.Fatalf("seed payments exit code = %d, stderr = %q", exitCode, stderr)
	}
	if _, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "", workareaFile, "workarea", "add", "billing", "--repo", "github:acme/billing"); exitCode != 0 {
		t.Fatalf("seed billing exit code = %d, stderr = %q", exitCode, stderr)
	}

	stdout, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "bill\n", workareaFile, "workarea", "use")
	if exitCode != 0 {
		t.Fatalf("workarea use exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "Select a workarea:") || !strings.Contains(stdout, "Choice or filter text") || !strings.Contains(stdout, "Workarea billing") {
		t.Fatalf("stdout = %q, want interactive selection output", stdout)
	}
}

func TestAppFrontDoorWithoutWorkarea(t *testing.T) {
	workareaFile := filepath.Join(t.TempDir(), "workareas.json")

	stdout, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "", workareaFile)
	if exitCode != 0 {
		t.Fatalf("front door exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "googling in git") || !strings.Contains(stdout, "status:  no project selected yet") || !strings.Contains(stdout, "input:   ticket, command, or Enter for picker") {
		t.Fatalf("stdout = %q, want branded hero state", stdout)
	}
	if !strings.Contains(stdout, "Run `gig` in a real terminal and use ↑/↓ then Enter") {
		t.Fatalf("stdout = %q, want guided picker hint", stdout)
	}
	if !strings.Contains(stdout, "gig remembers a successful remote repo as your current project automatically") {
		t.Fatalf("stdout = %q, want implicit project-memory hint", stdout)
	}
	if !strings.Contains(stdout, "Ask gig to") || !strings.Contains(stdout, "Still local?") {
		t.Fatalf("stdout = %q, want focused workflow sections", stdout)
	}
	if !strings.Contains(stdout, "gig assist doctor") {
		t.Fatalf("stdout = %q, want DeerFlow doctor suggestion", stdout)
	}
	if !strings.Contains(stdout, "gig assist setup") {
		t.Fatalf("stdout = %q, want DeerFlow setup suggestion", stdout)
	}
}

func TestAppFrontDoorQuickStartInspectsRepoTarget(t *testing.T) {
	workareaFile := filepath.Join(t.TempDir(), "workareas.json")

	ghDir := installFakeGitHubCLI(t, map[string]string{
		"repos/acme/payments": `{"default_branch":"main"}`,
		"repos/acme/payments/branches?protected=true&per_page=100&page=1":      `[{"name":"staging","protected":true},{"name":"main","protected":true}]`,
		"repos/acme/payments/commits?sha=staging&per_page=100&page=1":          `[{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments"}}]`,
		"repos/acme/payments/commits?sha=main&per_page=100&page=1":             `[]`,
		"repos/acme/payments/commits/abc123456789":                             `{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments"},"files":[{"filename":"service/app.txt"}]}`,
		"repos/acme/payments/commits/abc123456789/pulls?per_page=100&page=1":   `[{"number":42,"title":"ABC-123 payments release","state":"closed","merged_at":"2026-04-10T01:02:03Z","html_url":"https://github.com/acme/payments/pull/42","head":{"ref":"staging"},"base":{"ref":"main"}}]`,
		"repos/acme/payments/deployments?sha=abc123456789&per_page=100&page=1": `[{"id":1001,"sha":"abc123456789","ref":"main","environment":"production"}]`,
		"repos/acme/payments/deployments/1001/statuses?per_page=100&page=1":    `[{"state":"success","environment":"production","log_url":"https://deploy.example.com/github/1001"}]`,
	})
	t.Setenv("PATH", ghDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	stdout, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "2\ngithub:acme/payments\nABC-123\n", workareaFile)
	if exitCode != 0 {
		t.Fatalf("front door quick-start exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "ask gig >") || !strings.Contains(stdout, "Repository target:") || !strings.Contains(stdout, "Ticket ID:") {
		t.Fatalf("stdout = %q, want palette and quick-start prompts", stdout)
	}
	if !strings.Contains(stdout, "github:acme/payments") || !strings.Contains(stdout, "Provider evidence") {
		t.Fatalf("stdout = %q, want inspect output after quick start", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	store := workarea.NewStoreAt(workareaFile)
	current, ok, err := store.Current()
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if !ok {
		t.Fatal("Current() ok = false, want true")
	}
	if current.RepoTarget != "github:acme/payments" {
		t.Fatalf("current.RepoTarget = %q, want github:acme/payments", current.RepoTarget)
	}
}

func TestAppFrontDoorWithCurrentWorkarea(t *testing.T) {
	workareaFile := filepath.Join(t.TempDir(), "workareas.json")

	if _, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "", workareaFile, "workarea", "add", "payments", "--repo", "github:acme/payments", "--from", "staging", "--to", "main", "--use"); exitCode != 0 {
		t.Fatalf("seed workarea exit code = %d, stderr = %q", exitCode, stderr)
	}

	stdout, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "", workareaFile)
	if exitCode != 0 {
		t.Fatalf("front door exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "Ready now") || !strings.Contains(stdout, "Workarea") || !strings.Contains(stdout, "payments") || !strings.Contains(stdout, "input:   ticket or command palette") {
		t.Fatalf("stdout = %q, want current project summary", stdout)
	}
	if !strings.Contains(stdout, "Quick commands") || !strings.Contains(stdout, "gig manifest generate --ticket ABC-123") {
		t.Fatalf("stdout = %q, want guided core workflows", stdout)
	}
	if !strings.Contains(stdout, "Run `gig` in a real terminal and use ↑/↓ then Enter") {
		t.Fatalf("stdout = %q, want interactive action hint", stdout)
	}
}

func TestAppFrontDoorWithCurrentWorkareaPromptsActionAndInspects(t *testing.T) {
	workareaFile := filepath.Join(t.TempDir(), "workareas.json")

	ghDir := installFakeGitHubCLI(t, map[string]string{
		"repos/acme/payments": `{"default_branch":"main"}`,
		"repos/acme/payments/branches?protected=true&per_page=100&page=1":      `[{"name":"staging","protected":true},{"name":"main","protected":true}]`,
		"repos/acme/payments/commits?sha=staging&per_page=100&page=1":          `[{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments"}}]`,
		"repos/acme/payments/commits?sha=main&per_page=100&page=1":             `[]`,
		"repos/acme/payments/commits/abc123456789":                             `{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments"},"files":[{"filename":"service/app.txt"}]}`,
		"repos/acme/payments/commits/abc123456789/pulls?per_page=100&page=1":   `[{"number":42,"title":"ABC-123 payments release","state":"closed","merged_at":"2026-04-10T01:02:03Z","html_url":"https://github.com/acme/payments/pull/42","head":{"ref":"staging"},"base":{"ref":"main"}}]`,
		"repos/acme/payments/deployments?sha=abc123456789&per_page=100&page=1": `[{"id":1001,"sha":"abc123456789","ref":"main","environment":"production"}]`,
		"repos/acme/payments/deployments/1001/statuses?per_page=100&page=1":    `[{"state":"success","environment":"production","log_url":"https://deploy.example.com/github/1001"}]`,
	})
	t.Setenv("PATH", ghDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if _, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "", workareaFile, "workarea", "add", "payments", "--repo", "github:acme/payments", "--use"); exitCode != 0 {
		t.Fatalf("seed workarea exit code = %d, stderr = %q", exitCode, stderr)
	}

	stdout, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "1\nABC-123\n", workareaFile)
	if exitCode != 0 {
		t.Fatalf("front door inspect exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "ask gig >") || !strings.Contains(stdout, "Ticket ID:") {
		t.Fatalf("stdout = %q, want command palette and ticket prompt", stdout)
	}
	if !strings.Contains(stdout, "github:acme/payments") || !strings.Contains(stdout, "Provider evidence") {
		t.Fatalf("stdout = %q, want inspect output", stdout)
	}
	if !strings.Contains(stderr, "Using workarea payments (github:acme/payments)") {
		t.Fatalf("stderr = %q, want workarea hint", stderr)
	}
}

func TestAppFrontDoorWithCurrentWorkareaPromptsActionAndVerifies(t *testing.T) {
	workareaFile := filepath.Join(t.TempDir(), "workareas.json")

	ghDir := installFakeGitHubCLI(t, map[string]string{
		"repos/acme/payments": `{"default_branch":"main"}`,
		"repos/acme/payments/branches?protected=true&per_page=100&page=1":      `[{"name":"staging","protected":true},{"name":"main","protected":true}]`,
		"repos/acme/payments/branches/staging":                                 `{"name":"staging","protected":true}`,
		"repos/acme/payments/branches/main":                                    `{"name":"main","protected":true}`,
		"repos/acme/payments/commits?sha=staging&per_page=100&page=1":          `[{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments"}}]`,
		"repos/acme/payments/commits?sha=main&per_page=100&page=1":             `[]`,
		"repos/acme/payments/commits/abc123456789":                             `{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments"},"files":[{"filename":"service/app.txt"}]}`,
		"repos/acme/payments/commits/abc123456789/pulls?per_page=100&page=1":   `[{"number":42,"title":"ABC-123 payments release","state":"closed","merged_at":"2026-04-10T01:02:03Z","html_url":"https://github.com/acme/payments/pull/42","head":{"ref":"staging"},"base":{"ref":"main"}}]`,
		"repos/acme/payments/deployments?sha=abc123456789&per_page=100&page=1": `[{"id":1001,"sha":"abc123456789","ref":"main","environment":"production"}]`,
		"repos/acme/payments/deployments/1001/statuses?per_page=100&page=1":    `[{"state":"success","environment":"production","log_url":"https://deploy.example.com/github/1001"}]`,
	})
	t.Setenv("PATH", ghDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if _, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "", workareaFile, "workarea", "add", "payments", "--repo", "github:acme/payments", "--use"); exitCode != 0 {
		t.Fatalf("seed workarea exit code = %d, stderr = %q", exitCode, stderr)
	}

	stdout, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "2\nABC-123\n", workareaFile)
	if exitCode != 0 {
		t.Fatalf("front door verify exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "ask gig >") || !strings.Contains(stdout, "Ticket ID:") {
		t.Fatalf("stdout = %q, want command palette and ticket prompt", stdout)
	}
	if !strings.Contains(stdout, "Promotion") || !strings.Contains(stdout, "staging -> main") || !strings.Contains(stdout, "SAFE") {
		t.Fatalf("stdout = %q, want verification output", stdout)
	}
	if !strings.Contains(stderr, "Using workarea payments (github:acme/payments)") {
		t.Fatalf("stderr = %q, want workarea hint", stderr)
	}
}

func TestAppInspectUsesCurrentRemoteWorkarea(t *testing.T) {
	workareaFile := filepath.Join(t.TempDir(), "workareas.json")

	ghDir := installFakeGitHubCLI(t, map[string]string{
		"repos/acme/payments": `{"default_branch":"main"}`,
		"repos/acme/payments/branches?protected=true&per_page=100&page=1":      `[{"name":"staging","protected":true},{"name":"main","protected":true}]`,
		"repos/acme/payments/commits?sha=staging&per_page=100&page=1":          `[{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments"}}]`,
		"repos/acme/payments/commits?sha=main&per_page=100&page=1":             `[]`,
		"repos/acme/payments/commits/abc123456789":                             `{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments"},"files":[{"filename":"service/app.txt"}]}`,
		"repos/acme/payments/commits/abc123456789/pulls?per_page=100&page=1":   `[{"number":42,"title":"ABC-123 payments release","state":"closed","merged_at":"2026-04-10T01:02:03Z","html_url":"https://github.com/acme/payments/pull/42","head":{"ref":"staging"},"base":{"ref":"main"}}]`,
		"repos/acme/payments/deployments?sha=abc123456789&per_page=100&page=1": `[{"id":1001,"sha":"abc123456789","ref":"main","environment":"production"}]`,
		"repos/acme/payments/deployments/1001/statuses?per_page=100&page=1":    `[{"state":"success","environment":"production","log_url":"https://deploy.example.com/github/1001"}]`,
	})
	t.Setenv("PATH", ghDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if _, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "", workareaFile, "workarea", "add", "payments", "--repo", "github:acme/payments", "--from", "staging", "--to", "main", "--use"); exitCode != 0 {
		t.Fatalf("workarea add exit code = %d, stderr = %q", exitCode, stderr)
	}

	stdout, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "", workareaFile, "inspect", "ABC-123")
	if exitCode != 0 {
		t.Fatalf("inspect via workarea exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stderr, "Using workarea payments (github:acme/payments)") {
		t.Fatalf("stderr = %q, want workarea selection hint", stderr)
	}
	if !strings.Contains(stdout, "github:acme/payments") || !strings.Contains(stdout, "Provider evidence") {
		t.Fatalf("stdout = %q, want remote inspect output", stdout)
	}
}

func TestAppWorkareaAddDiscoversGitHubRepository(t *testing.T) {
	workareaFile := filepath.Join(t.TempDir(), "workareas.json")
	stateFile := filepath.Join(t.TempDir(), "gh-auth-state")

	ghDir := installFakeGitHubCLI(t, map[string]string{
		"user/repos?sort=updated&per_page=100&page=1": `[{"name":"payments","full_name":"acme/payments","archived":false,"disabled":false,"owner":{"login":"acme"}}]`,
		"user/repos?sort=updated&per_page=100&page=2": `[]`,
	})
	t.Setenv("PATH", ghDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_GH_STATE", stateFile)
	t.Setenv("FAKE_GH_REQUIRE_LOGIN", "1")

	stdout, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "1\n", workareaFile, "workarea", "add", "--provider", "github", "--use")
	if exitCode != 0 {
		t.Fatalf("workarea add discovery exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stderr, "Starting gh auth login") {
		t.Fatalf("stderr = %q, want auto-login message", stderr)
	}
	if !strings.Contains(stdout, "Select a GitHub repository:") || !strings.Contains(stdout, "Workarea payments") || !strings.Contains(stdout, "Target: github:acme/payments") {
		t.Fatalf("stdout = %q, want discovered github workarea", stdout)
	}
}

func TestAppWorkareaAddDiscoveryPrefersRecentRepository(t *testing.T) {
	workareaFile := filepath.Join(t.TempDir(), "workareas.json")

	if _, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "", workareaFile, "workarea", "add", "billing", "--repo", "github:acme/billing", "--use"); exitCode != 0 {
		t.Fatalf("seed billing exit code = %d, stderr = %q", exitCode, stderr)
	}

	stateFile := filepath.Join(t.TempDir(), "gh-auth-state")
	ghDir := installFakeGitHubCLI(t, map[string]string{
		"user/repos?sort=updated&per_page=100&page=1": `[
			{"name":"payments","full_name":"acme/payments","archived":false,"disabled":false,"owner":{"login":"acme"}},
			{"name":"billing","full_name":"acme/billing","archived":false,"disabled":false,"owner":{"login":"acme"}}
		]`,
		"user/repos?sort=updated&per_page=100&page=2": `[]`,
	})
	t.Setenv("PATH", ghDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_GH_STATE", stateFile)

	stdout, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "1\n", workareaFile, "workarea", "add", "release-audit", "--provider", "github")
	if exitCode != 0 {
		t.Fatalf("workarea add discovery exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "billing [recent]") || !strings.Contains(stdout, "Target: github:acme/billing") {
		t.Fatalf("stdout = %q, want recent repository promoted and selected", stdout)
	}
}

func TestAppWorkareaAddDiscoveryFiltersRepositoryChoices(t *testing.T) {
	workareaFile := filepath.Join(t.TempDir(), "workareas.json")
	stateFile := filepath.Join(t.TempDir(), "gh-auth-state")

	ghDir := installFakeGitHubCLI(t, map[string]string{
		"user/repos?sort=updated&per_page=100&page=1": `[
			{"name":"payments","full_name":"acme/payments","archived":false,"disabled":false,"owner":{"login":"acme"}},
			{"name":"billing","full_name":"acme/billing","archived":false,"disabled":false,"owner":{"login":"acme"}}
		]`,
		"user/repos?sort=updated&per_page=100&page=2": `[]`,
	})
	t.Setenv("PATH", ghDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_GH_STATE", stateFile)

	stdout, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "pay\n", workareaFile, "workarea", "add", "--provider", "github")
	if exitCode != 0 {
		t.Fatalf("workarea add discovery exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "Choice or filter text") || !strings.Contains(stdout, "Workarea payments") || !strings.Contains(stdout, "Target: github:acme/payments") {
		t.Fatalf("stdout = %q, want filtered repository selection output", stdout)
	}
}

func TestAppWorkareaAddDiscoversAzureRepository(t *testing.T) {
	workareaFile := filepath.Join(t.TempDir(), "workareas.json")
	stateFile := filepath.Join(t.TempDir(), "az-auth-state")

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path + "?" + request.URL.RawQuery {
		case "/acme/_apis/projects?$top=100&api-version=7.1":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"value":[{"name":"Payments"}]}`))
		case "/acme/Payments/_apis/git/repositories?api-version=7.1":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"value":[{"name":"release-audit"}]}`))
		default:
			t.Fatalf("unexpected azure discovery request %s?%s", request.URL.Path, request.URL.RawQuery)
		}
	}))
	defer server.Close()

	azDir := installFakeAzureCLI(t)
	t.Setenv("PATH", azDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_AZ_STATE", stateFile)
	t.Setenv("FAKE_AZ_REQUIRE_LOGIN", "1")
	t.Setenv("FAKE_AZ_ACCESS_TOKEN", "token-123")
	t.Setenv("GIG_AZURE_DEVOPS_BASE_URL", server.URL)

	stdout, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "acme\n1\n", workareaFile, "workarea", "add", "--provider", "azure-devops", "--use")
	if exitCode != 0 {
		t.Fatalf("workarea add azure discovery exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stderr, "Starting az login") {
		t.Fatalf("stderr = %q, want auto-login message", stderr)
	}
	if !strings.Contains(stdout, "Azure DevOps organization:") || !strings.Contains(stdout, "Workarea release-audit") || !strings.Contains(stdout, "Target: azure-devops:acme/Payments/release-audit") {
		t.Fatalf("stdout = %q, want discovered azure workarea", stdout)
	}
}

func TestAppDoctorGolden(t *testing.T) {
	t.Parallel()

	workspace := createPromotionFixture(t)
	writeFile(t, filepath.Join(workspace, "gig.yaml"), `
ticketPattern: '\b[A-Z][A-Z0-9]+-\d+\b'
environments:
  - name: build
    branch: dev
  - name: qa
    branch: qa
  - name: prod
    branch: main
repositories:
  - path: a-service
    service: Accounts API
    owner: Backend Team
  - path: apps/missing-ui
    service: Admin Web
    owner: Frontend Team
    kind: app
`)

	stdout, stderr, exitCode := runApp(t, "doctor", "--path", workspace)
	if exitCode != 0 {
		t.Fatalf("doctor exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "doctor.golden", normalizeOutput(stdout, workspace))
}

func TestAppDoctorWithoutConfigUsesInferenceAndBuiltIns(t *testing.T) {
	t.Parallel()

	workspace := createPromotionFixture(t)

	stdout, stderr, exitCode := runApp(t, "doctor", "--path", workspace)
	if exitCode != 0 {
		t.Fatalf("doctor exit code = %d, stderr = %q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("doctor stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Overrides: none (using built-in inference and defaults)") {
		t.Fatalf("stdout = %q, want configless overrides message", stdout)
	}
	if !strings.Contains(stdout, "Verdict: ok") {
		t.Fatalf("stdout = %q, want ok verdict", stdout)
	}
	if strings.Contains(stdout, "No gig config file was found") {
		t.Fatalf("stdout = %q, want no config-missing warning", stdout)
	}
}

func TestAppSnapshotCreateGolden(t *testing.T) {
	t.Parallel()

	workspace := createPromotionFixture(t)
	outputPath := filepath.Join(workspace, ".gig", "snapshots", "abc-123.json")

	stdout, stderr, exitCode := runApp(t, "snapshot", "create", "--ticket", "ABC-123", "--from", "test", "--to", "main", "--path", workspace, "--envs", "dev=dev,test=test,prod=main", "--output", outputPath)
	if exitCode != 0 {
		t.Fatalf("snapshot create exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "snapshot_create.golden", normalizeOutput(stdout, workspace))
}

func TestAppSnapshotCreateJSONGolden(t *testing.T) {
	t.Parallel()

	workspace := createPromotionFixture(t)

	stdout, stderr, exitCode := runApp(t, "snapshot", "create", "--ticket", "ABC-123", "--from", "test", "--to", "main", "--path", workspace, "--envs", "dev=dev,test=test,prod=main", "--format", "json")
	if exitCode != 0 {
		t.Fatalf("snapshot create json exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "snapshot_create_json.golden", normalizeOutput(stdout, workspace))
}

func TestAppSnapshotCreateWritesJSONFile(t *testing.T) {
	t.Parallel()

	workspace := createPromotionFixture(t)
	outputPath := filepath.Join(workspace, ".gig", "snapshots", "abc-123.json")

	stdout, stderr, exitCode := runApp(t, "snapshot", "create", "--ticket", "ABC-123", "--from", "test", "--to", "main", "--path", workspace, "--envs", "dev=dev,test=test,prod=main", "--output", outputPath)
	if exitCode != 0 {
		t.Fatalf("snapshot create with output exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stdout, outputPath) {
		t.Fatalf("snapshot create stdout = %q, want output path %q", stdout, outputPath)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", outputPath, err)
	}

	var payload struct {
		SchemaVersion string `json:"schemaVersion"`
		TicketID      string `json:"ticketId"`
		FromBranch    string `json:"fromBranch"`
		ToBranch      string `json:"toBranch"`
		Inspection    struct {
			ScannedRepositories int `json:"scannedRepositories"`
		} `json:"inspection"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		t.Fatalf("Unmarshal(snapshot) error = %v", err)
	}

	if payload.SchemaVersion != "1" {
		t.Fatalf("SchemaVersion = %q, want 1", payload.SchemaVersion)
	}
	if payload.TicketID != "ABC-123" {
		t.Fatalf("TicketID = %q, want ABC-123", payload.TicketID)
	}
	if payload.FromBranch != "test" || payload.ToBranch != "main" {
		t.Fatalf("branches = %s -> %s, want test -> main", payload.FromBranch, payload.ToBranch)
	}
	if payload.Inspection.ScannedRepositories != 1 {
		t.Fatalf("Inspection.ScannedRepositories = %d, want 1", payload.Inspection.ScannedRepositories)
	}
}

func TestAppSnapshotCreateReleaseWritesDefaultPath(t *testing.T) {
	t.Parallel()

	workspace := createPromotionFixture(t)
	releaseID := "rel-2026-04-09"

	stdout, stderr, exitCode := runApp(t, "snapshot", "create", "--ticket", "ABC-123", "--from", "test", "--to", "main", "--path", workspace, "--release", releaseID)
	if exitCode != 0 {
		t.Fatalf("snapshot create release exit code = %d, stderr = %q", exitCode, stderr)
	}

	expectedPath := filepath.Join(workspace, ".gig", "releases", releaseID, "snapshots", "abc-123.json")
	if !strings.Contains(stdout, expectedPath) {
		t.Fatalf("snapshot create stdout = %q, want output path %q", stdout, expectedPath)
	}

	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", expectedPath, err)
	}

	var payload struct {
		ReleaseID string `json:"releaseId"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		t.Fatalf("Unmarshal(snapshot release) error = %v", err)
	}
	if payload.ReleaseID != releaseID {
		t.Fatalf("ReleaseID = %q, want %q", payload.ReleaseID, releaseID)
	}
}

func TestAppAssistAuditLocalPath(t *testing.T) {
	t.Parallel()

	workspace := createPromotionFixture(t)
	var runRequestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/langgraph/threads":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"thread_id":"thread-assist-123"}`))
		case "/api/langgraph/threads/thread-assist-123/runs/stream":
			body, _ := io.ReadAll(r.Body)
			runRequestBody = string(body)
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: values\n"))
			_, _ = w.Write([]byte("data: {\"messages\":[{\"type\":\"human\",\"content\":\"ignored\"},{\"type\":\"ai\",\"content\":\"Blocked because main is missing one dependency.\"}]}\n\n"))
			_, _ = w.Write([]byte("event: end\n"))
			_, _ = w.Write([]byte("data: {}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	stdout, stderr, exitCode := runApp(
		t,
		"assist",
		"audit",
		"--ticket", "ABC-123",
		"--from", "test",
		"--to", "main",
		"--path", workspace,
		"--envs", "dev=dev,test=test,prod=main",
		"--audience", "client",
		"--url", server.URL,
	)
	if exitCode != 0 {
		t.Fatalf("assist audit exit code = %d, stderr = %q", exitCode, stderr)
	}

	if !strings.Contains(stdout, "Ticket Brief: ABC-123") {
		t.Fatalf("stdout = %q, want ticket brief heading", stdout)
	}
	if !strings.Contains(stdout, "Audience: client") {
		t.Fatalf("stdout = %q, want client audience", stdout)
	}
	if strings.Contains(stdout, "Thread:") || strings.Contains(stdout, "Mode:") {
		t.Fatalf("stdout = %q, want polished human output without debug metadata", stdout)
	}
	if !strings.Contains(stdout, "Blocked because main is missing one dependency.") {
		t.Fatalf("stdout = %q, want assistant response", stdout)
	}
	if !strings.Contains(runRequestBody, "ABC-123") {
		t.Fatalf("run request = %q, want bundled ticket id", runRequestBody)
	}
	if !strings.Contains(runRequestBody, "missing-dependency") {
		t.Fatalf("run request = %q, want promotion-plan evidence", runRequestBody)
	}
	if !strings.Contains(runRequestBody, "for the client audience") {
		t.Fatalf("run request = %q, want client audience prompt", runRequestBody)
	}
}

func TestAppAssistAuditJSON(t *testing.T) {
	t.Parallel()

	workspace := createPromotionFixture(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/langgraph/threads":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"thread_id":"thread-assist-json"}`))
		case "/api/langgraph/threads/thread-assist-json/runs/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: values\n"))
			_, _ = w.Write([]byte("data: {\"messages\":[{\"type\":\"ai\",\"content\":\"Warning: review DB migration manually.\"}]}\n\n"))
			_, _ = w.Write([]byte("event: end\n"))
			_, _ = w.Write([]byte("data: {}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	stdout, stderr, exitCode := runApp(
		t,
		"assist",
		"audit",
		"--ticket", "ABC-123",
		"--from", "test",
		"--to", "main",
		"--path", workspace,
		"--envs", "dev=dev,test=test,prod=main",
		"--audience", "qa",
		"--url", server.URL,
		"--format", "json",
	)
	if exitCode != 0 {
		t.Fatalf("assist audit json exit code = %d, stderr = %q", exitCode, stderr)
	}

	var payload struct {
		Command string `json:"command"`
		Scope   string `json:"scope"`
		Result  struct {
			ThreadID string `json:"threadId"`
			Audience string `json:"audience"`
			Response string `json:"response"`
			Bundle   struct {
				TicketID           string   `json:"ticketId"`
				ManifestHighlights []string `json:"manifestHighlights"`
			} `json:"bundle"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("Unmarshal(assist audit json) error = %v", err)
	}

	if payload.Command != "assist audit" {
		t.Fatalf("command = %q, want assist audit", payload.Command)
	}
	if payload.Result.ThreadID != "thread-assist-json" {
		t.Fatalf("threadId = %q, want thread-assist-json", payload.Result.ThreadID)
	}
	if payload.Result.Audience != "qa" {
		t.Fatalf("audience = %q, want qa", payload.Result.Audience)
	}
	if payload.Result.Bundle.TicketID != "ABC-123" {
		t.Fatalf("ticketId = %q, want ABC-123", payload.Result.Bundle.TicketID)
	}
	if len(payload.Result.Bundle.ManifestHighlights) == 0 {
		t.Fatalf("manifest highlights = %#v, want at least one highlight", payload.Result.Bundle.ManifestHighlights)
	}
}

func TestAppAssistReleaseLocalPath(t *testing.T) {
	t.Parallel()

	workspace := createReleaseFixture(t)
	releaseID := "rel-2026-04-09"
	createReleaseSnapshots(t, workspace, releaseID)

	var runRequestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/langgraph/threads":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"thread_id":"thread-release-123"}`))
		case "/api/langgraph/threads/thread-release-123/runs/stream":
			body, _ := io.ReadAll(r.Body)
			runRequestBody = string(body)
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: values\n"))
			_, _ = w.Write([]byte("data: {\"messages\":[{\"type\":\"ai\",\"content\":\"Release is still blocked because ABC-123 has unresolved evidence gaps.\"}]}\n\n"))
			_, _ = w.Write([]byte("event: end\n"))
			_, _ = w.Write([]byte("data: {}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	stdout, stderr, exitCode := runApp(
		t,
		"assist",
		"release",
		"--release", releaseID,
		"--path", workspace,
		"--audience", "release-manager",
		"--url", server.URL,
	)
	if exitCode != 0 {
		t.Fatalf("assist release exit code = %d, stderr = %q", exitCode, stderr)
	}

	if !strings.Contains(stdout, "Release Brief: "+releaseID) {
		t.Fatalf("stdout = %q, want release brief heading", stdout)
	}
	if !strings.Contains(stdout, "Audience: release-manager") {
		t.Fatalf("stdout = %q, want release-manager audience", stdout)
	}
	if strings.Contains(stdout, "Thread:") || strings.Contains(stdout, "Mode:") {
		t.Fatalf("stdout = %q, want polished human output without debug metadata", stdout)
	}
	if !strings.Contains(stdout, "Release is still blocked because ABC-123 has unresolved evidence gaps.") {
		t.Fatalf("stdout = %q, want assistant response", stdout)
	}
	if !strings.Contains(runRequestBody, releaseID) {
		t.Fatalf("run request = %q, want release id", runRequestBody)
	}
	if !strings.Contains(runRequestBody, "ABC-123") || !strings.Contains(runRequestBody, "XYZ-999") {
		t.Fatalf("run request = %q, want bundled ticket snapshots", runRequestBody)
	}
	if !strings.Contains(runRequestBody, "## Release Overview") {
		t.Fatalf("run request = %q, want release-manager sections", runRequestBody)
	}
}

func TestAppAssistReleaseJSON(t *testing.T) {
	t.Parallel()

	workspace := createReleaseFixture(t)
	releaseID := "rel-2026-04-09"
	createReleaseSnapshots(t, workspace, releaseID)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/langgraph/threads":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"thread_id":"thread-release-json"}`))
		case "/api/langgraph/threads/thread-release-json/runs/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: values\n"))
			_, _ = w.Write([]byte("data: {\"messages\":[{\"type\":\"ai\",\"content\":\"Client-ready summary is available, but one ticket still needs manual confirmation.\"}]}\n\n"))
			_, _ = w.Write([]byte("event: end\n"))
			_, _ = w.Write([]byte("data: {}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	stdout, stderr, exitCode := runApp(
		t,
		"assist",
		"release",
		"--release", releaseID,
		"--path", workspace,
		"--audience", "client",
		"--url", server.URL,
		"--format", "json",
	)
	if exitCode != 0 {
		t.Fatalf("assist release json exit code = %d, stderr = %q", exitCode, stderr)
	}

	var payload struct {
		Command string `json:"command"`
		Scope   string `json:"scope"`
		Result  struct {
			ThreadID string `json:"threadId"`
			Audience string `json:"audience"`
			Response string `json:"response"`
			Bundle   struct {
				ReleaseID string `json:"releaseId"`
				Snapshots []struct {
					TicketID string `json:"ticketId"`
				} `json:"snapshots"`
				Packets []struct {
					TicketID string `json:"ticketId"`
				} `json:"packets"`
			} `json:"bundle"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("Unmarshal(assist release json) error = %v", err)
	}

	if payload.Command != "assist release" {
		t.Fatalf("command = %q, want assist release", payload.Command)
	}
	if payload.Result.ThreadID != "thread-release-json" {
		t.Fatalf("threadId = %q, want thread-release-json", payload.Result.ThreadID)
	}
	if payload.Result.Audience != "client" {
		t.Fatalf("audience = %q, want client", payload.Result.Audience)
	}
	if payload.Result.Bundle.ReleaseID != releaseID {
		t.Fatalf("releaseId = %q, want %q", payload.Result.Bundle.ReleaseID, releaseID)
	}
	if len(payload.Result.Bundle.Snapshots) != 2 {
		t.Fatalf("snapshots = %#v, want 2 snapshots", payload.Result.Bundle.Snapshots)
	}
	if len(payload.Result.Bundle.Packets) != 2 {
		t.Fatalf("packets = %#v, want 2 packets", payload.Result.Bundle.Packets)
	}
}

func TestAppAssistReleaseRemoteTicketFile(t *testing.T) {
	workspace := t.TempDir()
	ticketFile := filepath.Join(workspace, "tickets.txt")
	writeFile(t, ticketFile, "ABC-123\nXYZ-999\n")

	ghDir := installFakeGitHubCLI(t, map[string]string{
		"repos/acme/payments": `{"default_branch":"main"}`,
		"repos/acme/payments/branches?protected=true&per_page=100&page=1":      `[{"name":"staging","protected":true},{"name":"main","protected":true}]`,
		"repos/acme/payments/branches/staging":                                 `{"name":"staging","protected":true}`,
		"repos/acme/payments/branches/main":                                    `{"name":"main","protected":true}`,
		"repos/acme/payments/commits?sha=staging&per_page=100&page=1":          `[{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments\n\nDepends-On: XYZ-456"}},{"sha":"xyz987654321","commit":{"message":"XYZ-999 add migration"}}]`,
		"repos/acme/payments/commits?sha=main&per_page=100&page=1":             `[{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments\n\nDepends-On: XYZ-456"}}]`,
		"repos/acme/payments/commits/abc123456789":                             `{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments\n\nDepends-On: XYZ-456"},"files":[{"filename":"service/app.txt"}]}`,
		"repos/acme/payments/commits/abc123456789/pulls?per_page=100&page=1":   `[{"number":42,"title":"ABC-123 payments release","state":"closed","merged_at":"2026-04-10T01:02:03Z","html_url":"https://github.com/acme/payments/pull/42","head":{"ref":"staging"},"base":{"ref":"main"}}]`,
		"repos/acme/payments/deployments?sha=abc123456789&per_page=100&page=1": `[{"id":1001,"sha":"abc123456789","ref":"main","environment":"production"}]`,
		"repos/acme/payments/deployments/1001/statuses?per_page=100&page=1":    `[{"state":"success","environment":"production","log_url":"https://deploy.example.com/github/1001"}]`,
		"repos/acme/payments/commits/xyz987654321":                             `{"sha":"xyz987654321","commit":{"message":"XYZ-999 add migration"},"files":[{"filename":"db/migrations/001_add_column.sql"}]}`,
		"repos/acme/payments/commits/xyz987654321/pulls?per_page=100&page=1":   `[]`,
		"repos/acme/payments/deployments?sha=xyz987654321&per_page=100&page=1": `[]`,
	})
	t.Setenv("PATH", ghDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var runRequestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/langgraph/threads":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"thread_id":"thread-release-remote"}`))
		case "/api/langgraph/threads/thread-release-remote/runs/stream":
			body, _ := io.ReadAll(r.Body)
			runRequestBody = string(body)
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: values\n"))
			_, _ = w.Write([]byte("data: {\"messages\":[{\"type\":\"ai\",\"content\":\"Remote release review is ready, but XYZ-999 is still missing from main.\"}]}\n\n"))
			_, _ = w.Write([]byte("event: end\n"))
			_, _ = w.Write([]byte("data: {}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	stdout, stderr, exitCode := runApp(
		t,
		"assist",
		"release",
		"--release", "rel-2026-04-09",
		"--ticket-file", ticketFile,
		"--repo", "github:acme/payments",
		"--path", workspace,
		"--audience", "release-manager",
		"--url", server.URL,
	)
	if exitCode != 0 {
		t.Fatalf("assist release remote exit code = %d, stderr = %q", exitCode, stderr)
	}

	if !strings.Contains(stdout, "Release Brief: rel-2026-04-09") {
		t.Fatalf("stdout = %q, want release brief heading", stdout)
	}
	if !strings.Contains(stdout, "Scope: github:acme/payments") {
		t.Fatalf("stdout = %q, want remote scope", stdout)
	}
	if !strings.Contains(stdout, "Remote release review is ready, but XYZ-999 is still missing from main.") {
		t.Fatalf("stdout = %q, want assistant response", stdout)
	}
	if !strings.Contains(runRequestBody, "rel-2026-04-09") {
		t.Fatalf("run request = %q, want release id", runRequestBody)
	}
	if !strings.Contains(runRequestBody, "ABC-123") || !strings.Contains(runRequestBody, "XYZ-999") {
		t.Fatalf("run request = %q, want bundled ticket ids", runRequestBody)
	}
	if !strings.Contains(runRequestBody, "github:acme/payments") {
		t.Fatalf("run request = %q, want remote scope label", runRequestBody)
	}
}

func TestAppAssistResolve(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	repoRoot := createMergeConflictFixture(t, workspace)

	var runRequestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/langgraph/threads":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"thread_id":"thread-resolve-cli"}`))
		case "/api/langgraph/threads/thread-resolve-cli/runs/stream":
			body, _ := io.ReadAll(r.Body)
			runRequestBody = string(body)
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: values\n"))
			_, _ = w.Write([]byte("data: {\"messages\":[{\"type\":\"ai\",\"content\":\"Accept incoming for this block, then stage the file after checking validation order.\"}]}\n\n"))
			_, _ = w.Write([]byte("event: end\n"))
			_, _ = w.Write([]byte("data: {}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	stdout, stderr, exitCode := runApp(
		t,
		"assist",
		"resolve",
		"--path", repoRoot,
		"--ticket", "ABC-123",
		"--audience", "release-manager",
		"--url", server.URL,
	)
	if exitCode != 0 {
		t.Fatalf("assist resolve exit code = %d, stderr = %q", exitCode, stderr)
	}

	if !strings.Contains(stdout, "Conflict Brief: "+repoRoot) {
		t.Fatalf("stdout = %q, want conflict brief heading", stdout)
	}
	if !strings.Contains(stdout, "Operation: merge") {
		t.Fatalf("stdout = %q, want merge operation", stdout)
	}
	if !strings.Contains(stdout, "Scoped ticket: ABC-123") {
		t.Fatalf("stdout = %q, want scoped ticket", stdout)
	}
	if strings.Contains(stdout, "Thread:") || strings.Contains(stdout, "Mode:") {
		t.Fatalf("stdout = %q, want polished human output without debug metadata", stdout)
	}
	if !strings.Contains(stdout, "Accept incoming for this block, then stage the file after checking validation order.") {
		t.Fatalf("stdout = %q, want assistant response", stdout)
	}
	if !strings.Contains(runRequestBody, "main line") || !strings.Contains(runRequestBody, "feature line") {
		t.Fatalf("run request = %q, want readable conflict block text", runRequestBody)
	}
	if !strings.Contains(runRequestBody, "## Active Conflict Recommendation") {
		t.Fatalf("run request = %q, want resolve prompt sections", runRequestBody)
	}
}

func TestAppAssistResolveJSON(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	repoRoot := createMergeConflictFixture(t, workspace)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/langgraph/threads":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"thread_id":"thread-resolve-json"}`))
		case "/api/langgraph/threads/thread-resolve-json/runs/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: values\n"))
			_, _ = w.Write([]byte("data: {\"messages\":[{\"type\":\"ai\",\"content\":\"QA should re-run validation checks after choosing the incoming side.\"}]}\n\n"))
			_, _ = w.Write([]byte("event: end\n"))
			_, _ = w.Write([]byte("data: {}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	stdout, stderr, exitCode := runApp(
		t,
		"assist",
		"resolve",
		"--path", repoRoot,
		"--ticket", "ABC-123",
		"--audience", "qa",
		"--url", server.URL,
		"--format", "json",
	)
	if exitCode != 0 {
		t.Fatalf("assist resolve json exit code = %d, stderr = %q", exitCode, stderr)
	}

	var payload struct {
		Command string `json:"command"`
		Scope   string `json:"scope"`
		Result  struct {
			ThreadID string `json:"threadId"`
			Audience string `json:"audience"`
			Response string `json:"response"`
			Bundle   struct {
				ScopedTicketID string `json:"scopedTicketId"`
				Status         struct {
					Operation struct {
						Type string `json:"type"`
					} `json:"operation"`
				} `json:"status"`
				ActiveConflict struct {
					File struct {
						Path string `json:"path"`
					} `json:"file"`
					Block struct {
						Current  string `json:"current"`
						Incoming string `json:"incoming"`
					} `json:"block"`
				} `json:"activeConflict"`
				SupportedActions []struct {
					Key string `json:"key"`
				} `json:"supportedActions"`
			} `json:"bundle"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("Unmarshal(assist resolve json) error = %v", err)
	}

	if payload.Command != "assist resolve" {
		t.Fatalf("command = %q, want assist resolve", payload.Command)
	}
	if payload.Scope != repoRoot {
		t.Fatalf("scope = %q, want %q", payload.Scope, repoRoot)
	}
	if payload.Result.ThreadID != "thread-resolve-json" {
		t.Fatalf("threadId = %q, want thread-resolve-json", payload.Result.ThreadID)
	}
	if payload.Result.Audience != "qa" {
		t.Fatalf("audience = %q, want qa", payload.Result.Audience)
	}
	if payload.Result.Bundle.ScopedTicketID != "ABC-123" {
		t.Fatalf("scopedTicketId = %q, want ABC-123", payload.Result.Bundle.ScopedTicketID)
	}
	if payload.Result.Bundle.Status.Operation.Type != "merge" {
		t.Fatalf("operation type = %q, want merge", payload.Result.Bundle.Status.Operation.Type)
	}
	if payload.Result.Bundle.ActiveConflict.File.Path != "app.txt" {
		t.Fatalf("active file = %q, want app.txt", payload.Result.Bundle.ActiveConflict.File.Path)
	}
	if payload.Result.Bundle.ActiveConflict.Block.Current == "" || payload.Result.Bundle.ActiveConflict.Block.Incoming == "" {
		t.Fatalf("active block = %#v, want readable conflict block", payload.Result.Bundle.ActiveConflict.Block)
	}
	if len(payload.Result.Bundle.SupportedActions) < 4 {
		t.Fatalf("supported actions = %#v, want keyboard action list", payload.Result.Bundle.SupportedActions)
	}
}

func TestAppAssistDoctor(t *testing.T) {
	root := t.TempDir()
	deerFlowRoot := filepath.Join(root, "deer-flow")
	writeFile(t, filepath.Join(deerFlowRoot, "Makefile"), "dev:\n\t@echo ok\n")
	if err := os.MkdirAll(filepath.Join(deerFlowRoot, "backend"), 0o755); err != nil {
		t.Fatalf("MkdirAll(backend) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(deerFlowRoot, "frontend"), 0o755); err != nil {
		t.Fatalf("MkdirAll(frontend) error = %v", err)
	}
	writeFile(t, filepath.Join(deerFlowRoot, "config.example.yaml"), "models:\n#  - name: gpt-5\n")
	writeFile(t, filepath.Join(deerFlowRoot, "config.yaml"), "models:\n  - name: gpt-5\n    api_key: $OPENAI_API_KEY\n")
	writeFile(t, filepath.Join(deerFlowRoot, ".env"), "OPENAI_API_KEY=secret\n")
	writeFile(t, filepath.Join(deerFlowRoot, "frontend", ".env"), "NEXT_PUBLIC_DEERFLOW_URL=http://localhost:2026\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()

	fakeBin := installFakeCommands(t, "make", "docker")
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	stdout, stderr, exitCode := runApp(
		t,
		"assist",
		"doctor",
		"--path", root,
		"--url", server.URL,
	)
	if exitCode != 0 {
		t.Fatalf("assist doctor exit code = %d, stderr = %q", exitCode, stderr)
	}

	if !strings.Contains(stdout, "DeerFlow doctor") {
		t.Fatalf("stdout = %q, want doctor heading", stdout)
	}
	if !strings.Contains(stdout, "Readiness: ready") {
		t.Fatalf("stdout = %q, want ready readiness", stdout)
	}
	if !strings.Contains(stdout, "Gateway: reachable") {
		t.Fatalf("stdout = %q, want reachable gateway", stdout)
	}
	if !strings.Contains(stdout, "Credentials:") || !strings.Contains(stdout, "OPENAI_API_KEY: present") {
		t.Fatalf("stdout = %q, want credential status", stdout)
	}
}

func TestAppPlanReleaseGolden(t *testing.T) {
	t.Parallel()

	workspace := createReleaseFixture(t)
	releaseID := "rel-2026-04-09"
	createReleaseSnapshots(t, workspace, releaseID)

	stdout, stderr, exitCode := runApp(t, "plan", "--release", releaseID, "--path", workspace)
	if exitCode != 0 {
		t.Fatalf("plan release exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "plan_release.golden", normalizeOutput(stdout, workspace))
}

func TestAppPlanReleaseJSONGolden(t *testing.T) {
	t.Parallel()

	workspace := createReleaseFixture(t)
	releaseID := "rel-2026-04-09"
	createReleaseSnapshots(t, workspace, releaseID)

	stdout, stderr, exitCode := runApp(t, "plan", "--release", releaseID, "--path", workspace, "--format", "json")
	if exitCode != 0 {
		t.Fatalf("plan release json exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "plan_release_json.golden", normalizeOutput(stdout, workspace))
}

func TestAppVerifyReleaseGolden(t *testing.T) {
	t.Parallel()

	workspace := createReleaseFixture(t)
	releaseID := "rel-2026-04-09"
	createReleaseSnapshots(t, workspace, releaseID)

	stdout, stderr, exitCode := runApp(t, "verify", "--release", releaseID, "--path", workspace)
	if exitCode != 0 {
		t.Fatalf("verify release exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "verify_release.golden", normalizeOutput(stdout, workspace))
}

func TestAppVerifyReleaseJSONGolden(t *testing.T) {
	t.Parallel()

	workspace := createReleaseFixture(t)
	releaseID := "rel-2026-04-09"
	createReleaseSnapshots(t, workspace, releaseID)

	stdout, stderr, exitCode := runApp(t, "verify", "--release", releaseID, "--path", workspace, "--format", "json")
	if exitCode != 0 {
		t.Fatalf("verify release json exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "verify_release_json.golden", normalizeOutput(stdout, workspace))
}

func TestAppManifestGenerateReleaseGolden(t *testing.T) {
	t.Parallel()

	workspace := createReleaseFixture(t)
	releaseID := "rel-2026-04-09"
	createReleaseSnapshots(t, workspace, releaseID)

	stdout, stderr, exitCode := runApp(t, "manifest", "generate", "--release", releaseID, "--path", workspace)
	if exitCode != 0 {
		t.Fatalf("manifest generate release exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "manifest_generate_release.golden", normalizeOutput(stdout, workspace))
}

func TestAppManifestGenerateReleaseJSONGolden(t *testing.T) {
	t.Parallel()

	workspace := createReleaseFixture(t)
	releaseID := "rel-2026-04-09"
	createReleaseSnapshots(t, workspace, releaseID)

	stdout, stderr, exitCode := runApp(t, "manifest", "generate", "--release", releaseID, "--path", workspace, "--format", "json")
	if exitCode != 0 {
		t.Fatalf("manifest generate release json exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "manifest_generate_release_json.golden", normalizeOutput(stdout, workspace))
}

func TestAppInspectRemoteGitHubAutoLogin(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "gh-auth-state")
	ghDir := installFakeGitHubCLI(t, map[string]string{
		"repos/acme/payments": `{"default_branch":"main"}`,
		"repos/acme/payments/branches?protected=true&per_page=100&page=1":      `[{"name":"staging","protected":true},{"name":"main","protected":true}]`,
		"repos/acme/payments/commits?sha=staging&per_page=100&page=1":          `[{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments\n\nDepends-On: XYZ-456"}}]`,
		"repos/acme/payments/commits?sha=main&per_page=100&page=1":             `[]`,
		"repos/acme/payments/commits/abc123456789":                             `{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments\n\nDepends-On: XYZ-456"},"files":[{"filename":"db/migrations/001_add_column.sql"}]}`,
		"repos/acme/payments/commits/abc123456789/pulls?per_page=100&page=1":   `[{"number":42,"title":"ABC-123 payments release","state":"closed","merged_at":"2026-04-10T01:02:03Z","html_url":"https://github.com/acme/payments/pull/42","head":{"ref":"staging"},"base":{"ref":"main"}}]`,
		"repos/acme/payments/deployments?sha=abc123456789&per_page=100&page=1": `[{"id":1001,"sha":"abc123456789","ref":"main","environment":"production"}]`,
		"repos/acme/payments/deployments/1001/statuses?per_page=100&page=1":    `[{"state":"success","environment":"production","log_url":"https://deploy.example.com/github/1001"}]`,
	})
	t.Setenv("PATH", ghDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_GH_STATE", stateFile)
	t.Setenv("FAKE_GH_REQUIRE_LOGIN", "1")

	stdout, stderr, exitCode := runApp(t, "inspect", "ABC-123", "--repo", "github:acme/payments")
	if exitCode != 0 {
		t.Fatalf("inspect remote exit code = %d, stderr = %q", exitCode, stderr)
	}

	if !strings.Contains(stderr, "Starting gh auth login") {
		t.Fatalf("stderr = %q, want auto-login message", stderr)
	}
	if !strings.Contains(stdout, "github:acme/payments") {
		t.Fatalf("stdout = %q, want remote repository label", stdout)
	}
	if !strings.Contains(stdout, "Declared dependencies") {
		t.Fatalf("stdout = %q, want declared dependency output", stdout)
	}
	if !strings.Contains(stdout, "Provider evidence") || !strings.Contains(stdout, "#42") || !strings.Contains(stdout, "production") {
		t.Fatalf("stdout = %q, want provider evidence output", stdout)
	}
}

func TestAppVerifyRemoteGitHubInfersProtectedBranches(t *testing.T) {
	ghDir := installFakeGitHubCLI(t, map[string]string{
		"repos/acme/payments": `{"default_branch":"main"}`,
		"repos/acme/payments/branches?protected=true&per_page=100&page=1":      `[{"name":"staging","protected":true},{"name":"main","protected":true}]`,
		"repos/acme/payments/branches/staging":                                 `{"name":"staging","protected":true}`,
		"repos/acme/payments/branches/main":                                    `{"name":"main","protected":true}`,
		"repos/acme/payments/commits?sha=staging&per_page=100&page=1":          `[{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments"}}]`,
		"repos/acme/payments/commits?sha=main&per_page=100&page=1":             `[]`,
		"repos/acme/payments/commits/abc123456789":                             `{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments"},"files":[{"filename":"service/app.txt"}]}`,
		"repos/acme/payments/commits/abc123456789/pulls?per_page=100&page=1":   `[{"number":42,"title":"ABC-123 payments release","state":"closed","merged_at":"2026-04-10T01:02:03Z","html_url":"https://github.com/acme/payments/pull/42","head":{"ref":"staging"},"base":{"ref":"main"}}]`,
		"repos/acme/payments/deployments?sha=abc123456789&per_page=100&page=1": `[{"id":1001,"sha":"abc123456789","ref":"main","environment":"production"}]`,
		"repos/acme/payments/deployments/1001/statuses?per_page=100&page=1":    `[{"state":"success","environment":"production","log_url":"https://deploy.example.com/github/1001"}]`,
	})
	t.Setenv("PATH", ghDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	stdout, stderr, exitCode := runApp(t, "verify", "--ticket", "ABC-123", "--repo", "github:acme/payments")
	if exitCode != 0 {
		t.Fatalf("verify remote exit code = %d, stderr = %q", exitCode, stderr)
	}

	if !strings.Contains(stdout, "Promotion") || !strings.Contains(stdout, "staging -> main") {
		t.Fatalf("stdout = %q, want inferred staging -> main verification", stdout)
	}
}

func TestAppVerifyRemoteWorkareaRemembersInferredTopology(t *testing.T) {
	workareaFile := filepath.Join(t.TempDir(), "workareas.json")

	ghDir := installFakeGitHubCLI(t, map[string]string{
		"repos/acme/payments": `{"default_branch":"main"}`,
		"repos/acme/payments/branches?protected=true&per_page=100&page=1":      `[{"name":"staging","protected":true},{"name":"main","protected":true}]`,
		"repos/acme/payments/branches/staging":                                 `{"name":"staging","protected":true}`,
		"repos/acme/payments/branches/main":                                    `{"name":"main","protected":true}`,
		"repos/acme/payments/commits?sha=staging&per_page=100&page=1":          `[{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments"}}]`,
		"repos/acme/payments/commits?sha=main&per_page=100&page=1":             `[]`,
		"repos/acme/payments/commits/abc123456789":                             `{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments"},"files":[{"filename":"service/app.txt"}]}`,
		"repos/acme/payments/commits/abc123456789/pulls?per_page=100&page=1":   `[{"number":42,"title":"ABC-123 payments release","state":"closed","merged_at":"2026-04-10T01:02:03Z","html_url":"https://github.com/acme/payments/pull/42","head":{"ref":"staging"},"base":{"ref":"main"}}]`,
		"repos/acme/payments/deployments?sha=abc123456789&per_page=100&page=1": `[{"id":1001,"sha":"abc123456789","ref":"main","environment":"production"}]`,
		"repos/acme/payments/deployments/1001/statuses?per_page=100&page=1":    `[{"state":"success","environment":"production","log_url":"https://deploy.example.com/github/1001"}]`,
	})
	t.Setenv("PATH", ghDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if _, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "", workareaFile, "workarea", "add", "payments", "--repo", "github:acme/payments", "--use"); exitCode != 0 {
		t.Fatalf("workarea add exit code = %d, stderr = %q", exitCode, stderr)
	}

	stdout, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "", workareaFile, "verify", "--ticket", "ABC-123")
	if exitCode != 0 {
		t.Fatalf("verify via workarea exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "Promotion") || !strings.Contains(stdout, "staging -> main") {
		t.Fatalf("stdout = %q, want inferred verification", stdout)
	}
	if !strings.Contains(stderr, "Using workarea payments (github:acme/payments)") {
		t.Fatalf("stderr = %q, want workarea hint", stderr)
	}

	store := workarea.NewStoreAt(workareaFile)
	current, ok, err := store.Current()
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if !ok {
		t.Fatal("Current() ok = false, want true")
	}
	if current.FromBranch != "staging" || current.ToBranch != "main" {
		t.Fatalf("stored branches = %s -> %s, want staging -> main", current.FromBranch, current.ToBranch)
	}
	if current.EnvironmentSpec != "staging=staging,prod=main" {
		t.Fatalf("stored envs = %q, want inferred env mapping", current.EnvironmentSpec)
	}

	stdout, stderr, exitCode = runAppWithInputAndWorkareaFile(t, "", workareaFile)
	if exitCode != 0 {
		t.Fatalf("front door exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "Promotion") || !strings.Contains(stdout, "staging -> main") {
		t.Fatalf("stdout = %q, want remembered promotion on front door", stdout)
	}
}

func TestAppInspectRemoteRepoAutoCreatesCurrentProject(t *testing.T) {
	workareaFile := filepath.Join(t.TempDir(), "workareas.json")

	ghDir := installFakeGitHubCLI(t, map[string]string{
		"repos/acme/payments": `{"default_branch":"main"}`,
		"repos/acme/payments/branches?protected=true&per_page=100&page=1":      `[{"name":"staging","protected":true},{"name":"main","protected":true}]`,
		"repos/acme/payments/commits?sha=staging&per_page=100&page=1":          `[{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments"}}]`,
		"repos/acme/payments/commits?sha=main&per_page=100&page=1":             `[]`,
		"repos/acme/payments/commits/abc123456789":                             `{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments"},"files":[{"filename":"service/app.txt"}]}`,
		"repos/acme/payments/commits/abc123456789/pulls?per_page=100&page=1":   `[{"number":42,"title":"ABC-123 payments release","state":"closed","merged_at":"2026-04-10T01:02:03Z","html_url":"https://github.com/acme/payments/pull/42","head":{"ref":"staging"},"base":{"ref":"main"}}]`,
		"repos/acme/payments/deployments?sha=abc123456789&per_page=100&page=1": `[{"id":1001,"sha":"abc123456789","ref":"main","environment":"production"}]`,
		"repos/acme/payments/deployments/1001/statuses?per_page=100&page=1":    `[{"state":"success","environment":"production","log_url":"https://deploy.example.com/github/1001"}]`,
	})
	t.Setenv("PATH", ghDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	stdout, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "", workareaFile, "inspect", "ABC-123", "--repo", "github:acme/payments")
	if exitCode != 0 {
		t.Fatalf("inspect remote exit code = %d, stderr = %q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("inspect remote stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "github:acme/payments") {
		t.Fatalf("stdout = %q, want remote scope label", stdout)
	}

	store := workarea.NewStoreAt(workareaFile)
	current, ok, err := store.Current()
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if !ok {
		t.Fatal("Current() ok = false, want true")
	}
	if current.Name != "payments" || current.RepoTarget != "github:acme/payments" {
		t.Fatalf("current = %#v, want payments/github:acme/payments", current)
	}

	stdout, stderr, exitCode = runAppWithInputAndWorkareaFile(t, "", workareaFile)
	if exitCode != 0 {
		t.Fatalf("front door exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "Ready now") || !strings.Contains(stdout, "Workarea") || !strings.Contains(stdout, "payments") {
		t.Fatalf("stdout = %q, want current project summary", stdout)
	}
}

func TestAppFrontDoorPaletteRepoCommandInspects(t *testing.T) {
	workareaFile := filepath.Join(t.TempDir(), "workareas.json")

	ghDir := installFakeGitHubCLI(t, map[string]string{
		"repos/acme/payments": `{"default_branch":"main"}`,
		"repos/acme/payments/branches?protected=true&per_page=100&page=1":      `[{"name":"staging","protected":true},{"name":"main","protected":true}]`,
		"repos/acme/payments/commits?sha=staging&per_page=100&page=1":          `[{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments"}}]`,
		"repos/acme/payments/commits?sha=main&per_page=100&page=1":             `[]`,
		"repos/acme/payments/commits/abc123456789":                             `{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments"},"files":[{"filename":"service/app.txt"}]}`,
		"repos/acme/payments/commits/abc123456789/pulls?per_page=100&page=1":   `[{"number":42,"title":"ABC-123 payments release","state":"closed","merged_at":"2026-04-10T01:02:03Z","html_url":"https://github.com/acme/payments/pull/42","head":{"ref":"staging"},"base":{"ref":"main"}}]`,
		"repos/acme/payments/deployments?sha=abc123456789&per_page=100&page=1": `[{"id":1001,"sha":"abc123456789","ref":"main","environment":"production"}]`,
		"repos/acme/payments/deployments/1001/statuses?per_page=100&page=1":    `[{"state":"success","environment":"production","log_url":"https://deploy.example.com/github/1001"}]`,
	})
	t.Setenv("PATH", ghDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	stdout, stderr, exitCode := runAppWithInputAndWorkareaFile(t, "repo github:acme/payments ABC-123\n", workareaFile)
	if exitCode != 0 {
		t.Fatalf("front door palette exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "ask gig >") || !strings.Contains(stdout, "Provider evidence") {
		t.Fatalf("stdout = %q, want palette output and inspect results", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestAppInspectRemoteGitLabAutoLogin(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "glab-auth-state")
	glabDir := installFakeGitLabCLI(t, map[string]string{
		"projects/acme%2Fplatform%2Fpayments":                                                          `{"default_branch":"main"}`,
		"projects/acme%2Fplatform%2Fpayments/repository/branches?per_page=100&page=1":                  `[{"name":"staging","protected":true},{"name":"main","protected":true}]`,
		"projects/acme%2Fplatform%2Fpayments/repository/commits?ref_name=staging&per_page=100&page=1":  `[{"id":"abc123456789","message":"ABC-123 fix payments\n\nDepends-On: XYZ-456"}]`,
		"projects/acme%2Fplatform%2Fpayments/repository/commits?ref_name=main&per_page=100&page=1":     `[]`,
		"projects/acme%2Fplatform%2Fpayments/repository/commits/abc123456789":                          `{"id":"abc123456789","message":"ABC-123 fix payments\n\nDepends-On: XYZ-456"}`,
		"projects/acme%2Fplatform%2Fpayments/repository/commits/abc123456789/diff?per_page=100&page=1": `[{"new_path":"db/migrations/001_add_column.sql"}]`,
		"projects/acme%2Fplatform%2Fpayments/repository/commits/abc123456789/merge_requests":           `[{"iid":42,"title":"ABC-123 payments release","state":"merged","web_url":"https://gitlab.com/acme/platform/payments/-/merge_requests/42","source_branch":"staging","target_branch":"main","merged_at":"2026-04-10T01:02:03Z"}]`,
		"projects/acme%2Fplatform%2Fpayments/deployments?per_page=100&page=1":                          `[{"id":1001,"iid":1,"ref":"main","sha":"abc123456789","status":"success","environment":{"name":"production","external_url":"https://deploy.example.com/gitlab/1001"}}]`,
	})
	t.Setenv("PATH", glabDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_GLAB_STATE", stateFile)
	t.Setenv("FAKE_GLAB_REQUIRE_LOGIN", "1")

	stdout, stderr, exitCode := runApp(t, "inspect", "ABC-123", "--repo", "gitlab:acme/platform/payments")
	if exitCode != 0 {
		t.Fatalf("inspect remote exit code = %d, stderr = %q", exitCode, stderr)
	}

	if !strings.Contains(stderr, "Starting glab auth login") {
		t.Fatalf("stderr = %q, want auto-login message", stderr)
	}
	if !strings.Contains(stdout, "gitlab:acme/platform/payments") {
		t.Fatalf("stdout = %q, want remote repository label", stdout)
	}
	if !strings.Contains(stdout, "Declared dependencies") {
		t.Fatalf("stdout = %q, want declared dependency output", stdout)
	}
	if !strings.Contains(stdout, "Provider evidence") || !strings.Contains(stdout, "!42") || !strings.Contains(stdout, "production") {
		t.Fatalf("stdout = %q, want provider evidence output", stdout)
	}
}

func TestAppVerifyRemoteGitLabInfersProtectedBranches(t *testing.T) {
	glabDir := installFakeGitLabCLI(t, map[string]string{
		"projects/acme%2Fplatform%2Fpayments":                                                          `{"default_branch":"main"}`,
		"projects/acme%2Fplatform%2Fpayments/repository/branches?per_page=100&page=1":                  `[{"name":"staging","protected":true},{"name":"main","protected":true}]`,
		"projects/acme%2Fplatform%2Fpayments/repository/branches/staging":                              `{"name":"staging","protected":true}`,
		"projects/acme%2Fplatform%2Fpayments/repository/branches/main":                                 `{"name":"main","protected":true}`,
		"projects/acme%2Fplatform%2Fpayments/repository/commits?ref_name=staging&per_page=100&page=1":  `[{"id":"abc123456789","message":"ABC-123 fix payments"}]`,
		"projects/acme%2Fplatform%2Fpayments/repository/commits?ref_name=main&per_page=100&page=1":     `[]`,
		"projects/acme%2Fplatform%2Fpayments/repository/commits/abc123456789":                          `{"id":"abc123456789","message":"ABC-123 fix payments"}`,
		"projects/acme%2Fplatform%2Fpayments/repository/commits/abc123456789/diff?per_page=100&page=1": `[{"new_path":"service/app.txt"}]`,
		"projects/acme%2Fplatform%2Fpayments/repository/commits/abc123456789/merge_requests":           `[{"iid":42,"title":"ABC-123 payments release","state":"merged","web_url":"https://gitlab.com/acme/platform/payments/-/merge_requests/42","source_branch":"staging","target_branch":"main","merged_at":"2026-04-10T01:02:03Z"}]`,
		"projects/acme%2Fplatform%2Fpayments/deployments?per_page=100&page=1":                          `[{"id":1001,"iid":1,"ref":"main","sha":"abc123456789","status":"success","environment":{"name":"production","external_url":"https://deploy.example.com/gitlab/1001"}}]`,
	})
	t.Setenv("PATH", glabDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	stdout, stderr, exitCode := runApp(t, "verify", "--ticket", "ABC-123", "--repo", "gitlab:acme/platform/payments")
	if exitCode != 0 {
		t.Fatalf("verify remote exit code = %d, stderr = %q", exitCode, stderr)
	}

	if !strings.Contains(stdout, "Promotion") || !strings.Contains(stdout, "staging -> main") {
		t.Fatalf("stdout = %q, want inferred staging -> main verification", stdout)
	}
}

func TestAppInspectRemoteAzureDevOpsAutoLogin(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "az-auth-state")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path + "?" + r.URL.RawQuery {
		case "/acme/Payments/_apis/git/repositories/release-audit?api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"defaultBranch":"refs/heads/main"}`))
		case "/acme/Payments/_apis/git/repositories/release-audit/refs?filter=heads/&api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"count":3,"value":[{"name":"refs/heads/feature/test"},{"name":"refs/heads/staging"},{"name":"refs/heads/main"}]}`))
		case "/acme/Payments/_apis/git/repositories/release-audit/commits?searchCriteria.$top=100&searchCriteria.itemVersion.versionType=branch&searchCriteria.itemVersion.version=staging&api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"count":1,"value":[{"commitId":"abc123456789","comment":"ABC-123 fix payments\n\nDepends-On: XYZ-456"}]}`))
		case "/acme/Payments/_apis/git/repositories/release-audit/commits?searchCriteria.$top=100&searchCriteria.itemVersion.versionType=branch&searchCriteria.itemVersion.version=main&api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"count":0,"value":[]}`))
		case "/acme/Payments/_apis/git/repositories/release-audit/commits/abc123456789?api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"commitId":"abc123456789","comment":"ABC-123 fix payments\n\nDepends-On: XYZ-456"}`))
		case "/acme/Payments/_apis/git/repositories/release-audit/commits/abc123456789/changes?$top=100&api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"changes":[{"item":{"path":"/db/migrations/001_add_column.sql"}}]}`))
		case "/acme/Payments/_apis/git/repositories/release-audit/pullrequests?searchCriteria.sourceRefName=refs%2Fheads%2Fstaging&searchCriteria.status=all&$top=100&api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"count":1,"value":[{"pullRequestId":42,"title":"ABC-123 payments release","status":"completed","sourceRefName":"refs/heads/staging","targetRefName":"refs/heads/main","url":"https://dev.azure.com/acme/Payments/_apis/git/repositories/release-audit/pullRequests/42"}]}`))
		case "/acme/Payments/_apis/release/deployments?$top=100&api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"count":1,"value":[{"id":1001,"deploymentStatus":"succeeded","releaseEnvironment":{"name":"production"},"_links":{"web":{"href":"https://dev.azure.com/acme/Payments/_releaseProgress?releaseId=1&environmentId=1"}},"release":{"name":"Release-1","artifacts":[{"definitionReference":{"version":{"name":"abc123456789"}}}]}}]}`))
		default:
			t.Fatalf("unexpected request %s?%s", r.URL.Path, r.URL.RawQuery)
		}
	}))
	defer server.Close()

	azDir := installFakeAzureCLI(t)
	t.Setenv("PATH", azDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_AZ_STATE", stateFile)
	t.Setenv("FAKE_AZ_REQUIRE_LOGIN", "1")
	t.Setenv("FAKE_AZ_ACCESS_TOKEN", "token-123")
	t.Setenv("GIG_AZURE_DEVOPS_BASE_URL", server.URL)

	stdout, stderr, exitCode := runApp(t, "inspect", "ABC-123", "--repo", "azure-devops:acme/Payments/release-audit")
	if exitCode != 0 {
		t.Fatalf("inspect remote exit code = %d, stderr = %q", exitCode, stderr)
	}

	if !strings.Contains(stderr, "Starting az login") {
		t.Fatalf("stderr = %q, want auto-login message", stderr)
	}
	if !strings.Contains(stdout, "azure-devops:acme/Payments/release-audit") {
		t.Fatalf("stdout = %q, want remote repository label", stdout)
	}
	if !strings.Contains(stdout, "Declared dependencies") {
		t.Fatalf("stdout = %q, want declared dependency output", stdout)
	}
	if !strings.Contains(stdout, "Provider evidence") || !strings.Contains(stdout, "#42") || !strings.Contains(stdout, "production") {
		t.Fatalf("stdout = %q, want provider evidence output", stdout)
	}
}

func TestAppVerifyRemoteAzureDevOpsInfersProtectedBranches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path + "?" + r.URL.RawQuery {
		case "/acme/Payments/_apis/git/repositories/release-audit?api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"defaultBranch":"refs/heads/main"}`))
		case "/acme/Payments/_apis/git/repositories/release-audit/refs?filter=heads/&api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"count":2,"value":[{"name":"refs/heads/staging"},{"name":"refs/heads/main"}]}`))
		case "/acme/Payments/_apis/git/repositories/release-audit/refs?filter=heads/staging&api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"count":1,"value":[{"name":"refs/heads/staging"}]}`))
		case "/acme/Payments/_apis/git/repositories/release-audit/refs?filter=heads/main&api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"count":1,"value":[{"name":"refs/heads/main"}]}`))
		case "/acme/Payments/_apis/git/repositories/release-audit/commits?searchCriteria.$top=100&searchCriteria.itemVersion.versionType=branch&searchCriteria.itemVersion.version=staging&api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"count":1,"value":[{"commitId":"abc123456789","comment":"ABC-123 fix payments"}]}`))
		case "/acme/Payments/_apis/git/repositories/release-audit/commits?searchCriteria.$top=100&searchCriteria.itemVersion.versionType=branch&searchCriteria.itemVersion.version=main&api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"count":0,"value":[]}`))
		case "/acme/Payments/_apis/git/repositories/release-audit/commits/abc123456789?api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"commitId":"abc123456789","comment":"ABC-123 fix payments"}`))
		case "/acme/Payments/_apis/git/repositories/release-audit/commits/abc123456789/changes?$top=100&api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"changes":[{"item":{"path":"/service/app.txt"}}]}`))
		case "/acme/Payments/_apis/git/repositories/release-audit/pullrequests?searchCriteria.sourceRefName=refs%2Fheads%2Fstaging&searchCriteria.status=all&$top=100&api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"count":1,"value":[{"pullRequestId":42,"title":"ABC-123 payments release","status":"completed","sourceRefName":"refs/heads/staging","targetRefName":"refs/heads/main","url":"https://dev.azure.com/acme/Payments/_apis/git/repositories/release-audit/pullRequests/42"}]}`))
		case "/acme/Payments/_apis/release/deployments?$top=100&api-version=7.1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"count":1,"value":[{"id":1001,"deploymentStatus":"succeeded","releaseEnvironment":{"name":"production"},"_links":{"web":{"href":"https://dev.azure.com/acme/Payments/_releaseProgress?releaseId=1&environmentId=1"}},"release":{"name":"Release-1","artifacts":[{"definitionReference":{"version":{"name":"abc123456789"}}}]}}]}`))
		default:
			t.Fatalf("unexpected request %s?%s", r.URL.Path, r.URL.RawQuery)
		}
	}))
	defer server.Close()

	azDir := installFakeAzureCLI(t)
	t.Setenv("PATH", azDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_AZ_ACCESS_TOKEN", "token-123")
	t.Setenv("GIG_AZURE_DEVOPS_BASE_URL", server.URL)

	stdout, stderr, exitCode := runApp(t, "verify", "--ticket", "ABC-123", "--repo", "azure-devops:acme/Payments/release-audit")
	if exitCode != 0 {
		t.Fatalf("verify remote exit code = %d, stderr = %q", exitCode, stderr)
	}

	if !strings.Contains(stdout, "Promotion") || !strings.Contains(stdout, "staging -> main") {
		t.Fatalf("stdout = %q, want inferred staging -> main verification", stdout)
	}
}

func TestAppInspectRemoteSVNAutoLogin(t *testing.T) {
	authFile := filepath.Join(t.TempDir(), "svn-auth.json")
	svnDir := installFakeSVNCLI(t, map[string]string{
		"info --xml https://svn.example.com/repos/app/branches/staging/HorizonCRM": `<info><entry><url>https://svn.example.com/repos/app/branches/staging/HorizonCRM</url><relative-url>^/branches/staging/HorizonCRM</relative-url><repository><root>https://svn.example.com/repos/app</root></repository></entry></info>`,
		"log --xml --verbose https://svn.example.com/repos/app/branches/staging/HorizonCRM": `<log><logentry revision="101"><msg>ABC-123 stage release fix

Depends-On: XYZ-456</msg><paths><path>/branches/staging/HorizonCRM/db/migrations/001_add_column.sql</path></paths></logentry></log>`,
		"log --xml --verbose -r 101 https://svn.example.com/repos/app/branches/staging/HorizonCRM": `<log><logentry revision="101"><msg>ABC-123 stage release fix

Depends-On: XYZ-456</msg><paths><path>/branches/staging/HorizonCRM/db/migrations/001_add_column.sql</path></paths></logentry></log>`,
		"log --xml -r 101 https://svn.example.com/repos/app/branches/staging/HorizonCRM": `<log><logentry revision="101"><msg>ABC-123 stage release fix

Depends-On: XYZ-456</msg></logentry></log>`,
	})
	t.Setenv("PATH", svnDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GIG_SVN_AUTH_FILE", authFile)

	stdout, stderr, exitCode := runAppWithInput(t, "\ndemo\nsecret\n", "inspect", "ABC-123", "--repo", "svn:https://svn.example.com/repos/app/branches/staging/HorizonCRM")
	if exitCode != 0 {
		t.Fatalf("inspect remote svn exit code = %d, stderr = %q", exitCode, stderr)
	}

	if !strings.Contains(stderr, "Starting interactive SVN login") {
		t.Fatalf("stderr = %q, want auto-login message", stderr)
	}
	if !strings.Contains(stdout, "svn:https://svn.example.com/repos/app/branches/staging/HorizonCRM") {
		t.Fatalf("stdout = %q, want remote svn repository label", stdout)
	}
	if !strings.Contains(stdout, "Declared dependencies") {
		t.Fatalf("stdout = %q, want declared dependency output", stdout)
	}
	if _, err := os.Stat(authFile); err != nil {
		t.Fatalf("Stat(%q) error = %v", authFile, err)
	}
}

func TestAppVerifyRemoteSVNInfersBranchTopology(t *testing.T) {
	svnDir := installFakeSVNCLI(t, map[string]string{
		"info --xml https://svn.example.com/repos/app/branches/staging/HorizonCRM":                                                                                   `<info><entry><url>https://svn.example.com/repos/app/branches/staging/HorizonCRM</url><relative-url>^/branches/staging/HorizonCRM</relative-url><repository><root>https://svn.example.com/repos/app</root></repository></entry></info>`,
		"info --xml https://svn.example.com/repos/app/trunk":                                                                                                         `<info><entry><url>https://svn.example.com/repos/app/trunk</url><relative-url>^/trunk</relative-url><repository><root>https://svn.example.com/repos/app</root></repository></entry></info>`,
		"info --xml https://svn.example.com/repos/app/trunk/HorizonCRM":                                                                                              `<info><entry><url>https://svn.example.com/repos/app/trunk/HorizonCRM</url><relative-url>^/trunk/HorizonCRM</relative-url><repository><root>https://svn.example.com/repos/app</root></repository></entry></info>`,
		"list --xml https://svn.example.com/repos/app/branches":                                                                                                      `<lists><list path="https://svn.example.com/repos/app/branches"><entry kind="dir"><name>develop</name></entry><entry kind="dir"><name>staging</name></entry><entry kind="dir"><name>main</name></entry></list></lists>`,
		"info --xml https://svn.example.com/repos/app/branches/develop/HorizonCRM":                                                                                   `<info><entry><url>https://svn.example.com/repos/app/branches/develop/HorizonCRM</url><relative-url>^/branches/develop/HorizonCRM</relative-url><repository><root>https://svn.example.com/repos/app</root></repository></entry></info>`,
		"info --xml https://svn.example.com/repos/app/branches/main/HorizonCRM":                                                                                      `<info><entry><url>https://svn.example.com/repos/app/branches/main/HorizonCRM</url><relative-url>^/branches/main/HorizonCRM</relative-url><repository><root>https://svn.example.com/repos/app</root></repository></entry></info>`,
		"log --xml --verbose https://svn.example.com/repos/app/branches/develop/HorizonCRM":                                                                          `<log></log>`,
		"log --xml --verbose https://svn.example.com/repos/app/trunk/HorizonCRM":                                                                                     `<log></log>`,
		"log --xml --verbose https://svn.example.com/repos/app/branches/staging/HorizonCRM":                                                                          `<log><logentry revision="101"><msg>ABC-123 stage release fix</msg><paths><path>/branches/staging/HorizonCRM/service/app.txt</path></paths></logentry></log>`,
		"log --xml --verbose https://svn.example.com/repos/app/branches/main/HorizonCRM":                                                                             `<log></log>`,
		"mergeinfo --show-revs eligible https://svn.example.com/repos/app/branches/develop/HorizonCRM https://svn.example.com/repos/app/branches/staging/HorizonCRM": ``,
		"mergeinfo --show-revs eligible https://svn.example.com/repos/app/branches/staging/HorizonCRM https://svn.example.com/repos/app/branches/main/HorizonCRM":    `r101`,
		"log --xml --verbose -r 101 https://svn.example.com/repos/app/branches/staging/HorizonCRM":                                                                   `<log><logentry revision="101"><msg>ABC-123 stage release fix</msg><paths><path>/branches/staging/HorizonCRM/service/app.txt</path></paths></logentry></log>`,
		"log --xml -r 101 https://svn.example.com/repos/app/branches/staging/HorizonCRM":                                                                             `<log><logentry revision="101"><msg>ABC-123 stage release fix</msg></logentry></log>`,
	})
	t.Setenv("PATH", svnDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GIG_SVN_USERNAME", "demo")
	t.Setenv("GIG_SVN_PASSWORD", "secret")

	stdout, stderr, exitCode := runApp(t, "verify", "--ticket", "ABC-123", "--repo", "svn:https://svn.example.com/repos/app/branches/staging/HorizonCRM")
	if exitCode != 0 {
		t.Fatalf("verify remote svn exit code = %d, stderr = %q", exitCode, stderr)
	}

	if !strings.Contains(stdout, "Promotion") || !strings.Contains(stdout, "staging -> main") {
		t.Fatalf("stdout = %q, want inferred staging -> main verification", stdout)
	}
}

func TestAppLoginBitbucketInteractive(t *testing.T) {
	authFile := filepath.Join(t.TempDir(), "bitbucket-auth.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repositories" || r.URL.RawQuery != "pagelen=1" {
			t.Fatalf("unexpected request %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "demo@example.com" || password != "secret-token" {
			t.Fatalf("basic auth = %q/%q, want demo@example.com/secret-token", username, password)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"values":[]}`))
	}))
	defer server.Close()

	t.Setenv("GIG_BITBUCKET_AUTH_FILE", authFile)
	t.Setenv("GIG_BITBUCKET_BASE_URL", server.URL)

	stdout, stderr, exitCode := runAppWithInput(t, "demo@example.com\nsecret-token\n", "login", "bitbucket")
	if exitCode != 0 {
		t.Fatalf("login bitbucket exit code = %d, stderr = %q", exitCode, stderr)
	}

	if !strings.Contains(stdout, "Bitbucket authentication is ready.") {
		t.Fatalf("stdout = %q, want ready message", stdout)
	}
	if !strings.Contains(stderr, "Bitbucket email:") || !strings.Contains(stderr, "Bitbucket API token:") {
		t.Fatalf("stderr = %q, want interactive prompts", stderr)
	}
	if _, err := os.Stat(authFile); err != nil {
		t.Fatalf("Stat(%q) error = %v", authFile, err)
	}
}

func TestAppInspectRemoteBitbucketAutoLogin(t *testing.T) {
	authFile := filepath.Join(t.TempDir(), "bitbucket-auth.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "demo@example.com" || password != "secret-token" {
			t.Fatalf("basic auth = %q/%q, want demo@example.com/secret-token", username, password)
		}

		switch r.URL.Path + "?" + r.URL.RawQuery {
		case "/repositories?pagelen=1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[]}`))
		case "/repositories/acme/payments?":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"mainbranch":{"name":"main"}}`))
		case "/repositories/acme/payments/branch-restrictions?pagelen=100&page=1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[{"branch_match_kind":"branching_model","branch_type":"development"},{"branch_match_kind":"branching_model","branch_type":"production"}]}`))
		case "/repositories/acme/payments/effective-branching-model?":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"development":{"branch":{"name":"staging"}},"production":{"branch":{"name":"main"}}}`))
		case "/repositories/acme/payments/commits/staging?pagelen=100&page=1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[{"hash":"abc123456789","message":"ABC-123 fix payments\n\nDepends-On: XYZ-456"}]}`))
		case "/repositories/acme/payments/commits/main?pagelen=100&page=1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[]}`))
		case "/repositories/acme/payments/commit/abc123456789?":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"hash":"abc123456789","message":"ABC-123 fix payments\n\nDepends-On: XYZ-456"}`))
		case "/repositories/acme/payments/diffstat/abc123456789?pagelen=100&page=1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[{"new":{"path":"db/migrations/001_add_column.sql"}}]}`))
		case "/repositories/acme/payments/commit/abc123456789/pullrequests?pagelen=100&page=1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[{"id":42,"title":"ABC-123 payments release","state":"FULFILLED","links":{"html":{"href":"https://bitbucket.org/acme/payments/pull-requests/42"}},"source":{"branch":{"name":"staging"}},"destination":{"branch":{"name":"main"}}}]}`))
		case "/repositories/acme/payments/deployments?pagelen=100&page=1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[{"uuid":"{deploy-1}","state":{"name":"COMPLETED"},"environment":{"name":"production"},"release":{"commit":{"hash":"abc123456789"},"ref_name":"main"},"links":{"html":{"href":"https://bitbucket.org/acme/payments/deployments/1"}}}]}`))
		default:
			t.Fatalf("unexpected request %s?%s", r.URL.Path, r.URL.RawQuery)
		}
	}))
	defer server.Close()

	t.Setenv("GIG_BITBUCKET_AUTH_FILE", authFile)
	t.Setenv("GIG_BITBUCKET_BASE_URL", server.URL)

	stdout, stderr, exitCode := runAppWithInput(t, "demo@example.com\nsecret-token\n", "inspect", "ABC-123", "--repo", "bitbucket:acme/payments")
	if exitCode != 0 {
		t.Fatalf("inspect remote exit code = %d, stderr = %q", exitCode, stderr)
	}

	if !strings.Contains(stderr, "Starting interactive API token login") {
		t.Fatalf("stderr = %q, want auto-login message", stderr)
	}
	if !strings.Contains(stdout, "bitbucket:acme/payments") {
		t.Fatalf("stdout = %q, want remote repository label", stdout)
	}
	if !strings.Contains(stdout, "Declared dependencies") {
		t.Fatalf("stdout = %q, want declared dependency output", stdout)
	}
	if !strings.Contains(stdout, "Provider evidence") || !strings.Contains(stdout, "#42") || !strings.Contains(stdout, "production") {
		t.Fatalf("stdout = %q, want provider evidence output", stdout)
	}
}

func TestAppVerifyRemoteBitbucketInfersProtectedBranches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "demo@example.com" || password != "secret-token" {
			t.Fatalf("basic auth = %q/%q, want demo@example.com/secret-token", username, password)
		}

		switch r.URL.Path + "?" + r.URL.RawQuery {
		case "/repositories?pagelen=1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[]}`))
		case "/repositories/acme/payments?":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"mainbranch":{"name":"main"}}`))
		case "/repositories/acme/payments/branch-restrictions?pagelen=100&page=1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[{"branch_match_kind":"branching_model","branch_type":"development"},{"branch_match_kind":"branching_model","branch_type":"production"}]}`))
		case "/repositories/acme/payments/effective-branching-model?":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"development":{"branch":{"name":"staging"}},"production":{"branch":{"name":"main"}}}`))
		case "/repositories/acme/payments/refs/branches/staging?":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"staging"}`))
		case "/repositories/acme/payments/refs/branches/main?":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"main"}`))
		case "/repositories/acme/payments/commits/staging?pagelen=100&page=1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[{"hash":"abc123456789","message":"ABC-123 fix payments"}]}`))
		case "/repositories/acme/payments/commits/main?pagelen=100&page=1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[]}`))
		case "/repositories/acme/payments/commit/abc123456789?":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"hash":"abc123456789","message":"ABC-123 fix payments"}`))
		case "/repositories/acme/payments/diffstat/abc123456789?pagelen=100&page=1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[{"new":{"path":"service/app.txt"}}]}`))
		case "/repositories/acme/payments/commit/abc123456789/pullrequests?pagelen=100&page=1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[{"id":42,"title":"ABC-123 payments release","state":"FULFILLED","links":{"html":{"href":"https://bitbucket.org/acme/payments/pull-requests/42"}},"source":{"branch":{"name":"staging"}},"destination":{"branch":{"name":"main"}}}]}`))
		case "/repositories/acme/payments/deployments?pagelen=100&page=1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[{"uuid":"{deploy-1}","state":{"name":"COMPLETED"},"environment":{"name":"production"},"release":{"commit":{"hash":"abc123456789"},"ref_name":"main"},"links":{"html":{"href":"https://bitbucket.org/acme/payments/deployments/1"}}}]}`))
		default:
			t.Fatalf("unexpected request %s?%s", r.URL.Path, r.URL.RawQuery)
		}
	}))
	defer server.Close()

	t.Setenv("GIG_BITBUCKET_BASE_URL", server.URL)
	t.Setenv("GIG_BITBUCKET_EMAIL", "demo@example.com")
	t.Setenv("GIG_BITBUCKET_API_TOKEN", "secret-token")

	stdout, stderr, exitCode := runApp(t, "verify", "--ticket", "ABC-123", "--repo", "bitbucket:acme/payments")
	if exitCode != 0 {
		t.Fatalf("verify remote exit code = %d, stderr = %q", exitCode, stderr)
	}

	if !strings.Contains(stdout, "Promotion") || !strings.Contains(stdout, "staging -> main") {
		t.Fatalf("stdout = %q, want inferred staging -> main verification", stdout)
	}
}

func TestAppResolveStatusGolden(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	repoRoot := createMergeConflictFixture(t, workspace)

	stdout, stderr, exitCode := runApp(t, "resolve", "status", "--path", repoRoot, "--ticket", "ABC-123")
	if exitCode != 0 {
		t.Fatalf("resolve status exit code = %d, stderr = %q", exitCode, stderr)
	}

	assertGolden(t, "resolve_status.golden", normalizeOutput(stdout, workspace))
}

func TestAppResolveStatusJSON(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	repoRoot := createMergeConflictFixture(t, workspace)

	stdout, stderr, exitCode := runApp(t, "resolve", "status", "--path", repoRoot, "--ticket", "ABC-123", "--format", "json")
	if exitCode != 0 {
		t.Fatalf("resolve status json exit code = %d, stderr = %q", exitCode, stderr)
	}

	var payload struct {
		Command string `json:"command"`
		Status  struct {
			ResolvableFiles  int `json:"resolvableFiles"`
			UnsupportedFiles int `json:"unsupportedFiles"`
			Files            []struct {
				Path       string `json:"path"`
				BlockCount int    `json:"blockCount"`
				Supported  bool   `json:"supported"`
			} `json:"files"`
			Operation struct {
				Type string `json:"type"`
			} `json:"operation"`
		} `json:"status"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("Unmarshal(resolve status) error = %v", err)
	}

	if payload.Command != "resolve status" {
		t.Fatalf("command = %q, want resolve status", payload.Command)
	}
	if payload.Status.Operation.Type != "merge" {
		t.Fatalf("operation type = %q, want merge", payload.Status.Operation.Type)
	}
	if payload.Status.ResolvableFiles != 1 || payload.Status.UnsupportedFiles != 0 {
		t.Fatalf("resolve counts = %d/%d, want 1/0", payload.Status.ResolvableFiles, payload.Status.UnsupportedFiles)
	}
	if len(payload.Status.Files) != 1 || !payload.Status.Files[0].Supported || payload.Status.Files[0].BlockCount != 1 {
		t.Fatalf("files payload = %#v, want one supported file with one block", payload.Status.Files)
	}
}

func TestAppResolveStartScriptedResolvesIncoming(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	repoRoot := createMergeConflictFixture(t, workspace)

	stdout, stderr, exitCode := runAppWithInput(t, "2\ns\n", "resolve", "start", "--path", repoRoot, "--ticket", "ABC-123")
	if exitCode != 0 {
		t.Fatalf("resolve start exit code = %d, stderr = %q", exitCode, stderr)
	}
	if !strings.Contains(stdout, "All supported conflict blocks are resolved.") {
		t.Fatalf("resolve start stdout = %q, want completion message", stdout)
	}

	content, err := os.ReadFile(filepath.Join(repoRoot, "app.txt"))
	if err != nil {
		t.Fatalf("ReadFile(app.txt) error = %v", err)
	}
	if strings.Contains(string(content), "<<<<<<<") {
		t.Fatalf("resolved file still contains conflict markers: %q", string(content))
	}
	if got := strings.TrimSpace(string(content)); got != "feature line" {
		t.Fatalf("resolved file content = %q, want feature line", got)
	}
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

func TestAppUpdateHelpReturnsZero(t *testing.T) {
	t.Parallel()

	stdout, stderr, exitCode := runApp(t, "update", "--help")
	if exitCode != 0 {
		t.Fatalf("update --help exit code = %d, want 0", exitCode)
	}
	if stdout != "" {
		t.Fatalf("update --help stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "Usage: gig update") {
		t.Fatalf("update --help stderr = %q, want usage", stderr)
	}
}

func TestAppRootHelpReturnsZero(t *testing.T) {
	t.Parallel()

	stdout, stderr, exitCode := runApp(t, "--help")
	if exitCode != 0 {
		t.Fatalf("--help exit code = %d, want 0", exitCode)
	}
	if stdout != "" {
		t.Fatalf("--help stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "gig [command] [flags]") {
		t.Fatalf("--help stderr = %q, want root usage", stderr)
	}
	if !strings.Contains(stderr, "scan        Find repositories under a local path") {
		t.Fatalf("--help stderr = %q, want command summary", stderr)
	}
	if !strings.Contains(stderr, "workarea    Remember a project so later commands stay short") {
		t.Fatalf("--help stderr = %q, want workarea command summary", stderr)
	}
	if !strings.Contains(stderr, "manifest    Generate a release packet for QA and release review") {
		t.Fatalf("--help stderr = %q, want manifest command summary", stderr)
	}
	if !strings.Contains(stderr, "snapshot    Save a repeatable ticket baseline for audit and re-check") {
		t.Fatalf("--help stderr = %q, want snapshot command summary", stderr)
	}
	if !strings.Contains(stderr, "doctor      Check inferred topology, overrides, and repo health") {
		t.Fatalf("--help stderr = %q, want doctor command summary", stderr)
	}
	if !strings.Contains(stderr, "resolve     Inspect or resolve active Git merge conflicts") {
		t.Fatalf("--help stderr = %q, want resolve command summary", stderr)
	}
	if !strings.Contains(stderr, "update      Install the latest release or a specific version") {
		t.Fatalf("--help stderr = %q, want update command summary", stderr)
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

	return runAppWithInput(t, "", args...)
}

func runAppWithInput(t *testing.T, input string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()

	return runAppWithInputAndWorkareaFile(t, input, filepath.Join(t.TempDir(), "workareas.json"), args...)
}

var appEnvMu sync.Mutex

func runAppWithInputAndWorkareaFile(t *testing.T, input, workareaFile string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()

	appEnvMu.Lock()
	defer appEnvMu.Unlock()

	previousWorkareaFile, hadWorkareaFile := os.LookupEnv("GIG_WORKAREA_FILE")
	if err := os.Setenv("GIG_WORKAREA_FILE", workareaFile); err != nil {
		t.Fatalf("Setenv(GIG_WORKAREA_FILE) error = %v", err)
	}
	defer func() {
		if hadWorkareaFile {
			_ = os.Setenv("GIG_WORKAREA_FILE", previousWorkareaFile)
			return
		}
		_ = os.Unsetenv("GIG_WORKAREA_FILE")
	}()

	var out bytes.Buffer
	var errOut bytes.Buffer

	app, err := cli.NewAppWithIO(strings.NewReader(input), &out, &errOut)
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
	normalized = hashPattern.ReplaceAllString(normalized, "<HASH>")

	timestampPattern := regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})\b`)
	return timestampPattern.ReplaceAllString(normalized, "<TIMESTAMP>")
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

func installFakeGitHubCLI(t *testing.T, endpoints map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "gh")

	var script strings.Builder
	script.WriteString("#!/bin/sh\n")
	script.WriteString("set -eu\n")
	script.WriteString("state=\"${FAKE_GH_STATE:-}\"\n")
	script.WriteString("require_login=\"${FAKE_GH_REQUIRE_LOGIN:-0}\"\n")
	script.WriteString("cmd=\"${1:-}\"\n")
	script.WriteString("sub=\"${2:-}\"\n")
	script.WriteString("if [ \"$cmd\" = \"auth\" ] && [ \"$sub\" = \"status\" ]; then\n")
	script.WriteString("  if [ \"$require_login\" = \"1\" ] && [ ! -f \"$state\" ]; then\n")
	script.WriteString("    echo \"not logged in\" >&2\n")
	script.WriteString("    exit 1\n")
	script.WriteString("  fi\n")
	script.WriteString("  echo \"Logged in\"\n")
	script.WriteString("  exit 0\n")
	script.WriteString("fi\n")
	script.WriteString("if [ \"$cmd\" = \"auth\" ] && [ \"$sub\" = \"login\" ]; then\n")
	script.WriteString("  if [ -n \"$state\" ]; then touch \"$state\"; fi\n")
	script.WriteString("  echo \"Logged in\"\n")
	script.WriteString("  exit 0\n")
	script.WriteString("fi\n")
	script.WriteString("if [ \"$cmd\" = \"api\" ]; then\n")
	script.WriteString("  for last; do true; done\n")
	script.WriteString("  endpoint=\"$last\"\n")
	for endpoint, output := range endpoints {
		script.WriteString("  if [ \"$endpoint\" = ")
		script.WriteString(shellQuote(endpoint))
		script.WriteString(" ]; then\n")
		script.WriteString("    cat <<'EOF'\n")
		script.WriteString(output)
		script.WriteString("\nEOF\n")
		script.WriteString("    exit 0\n")
		script.WriteString("  fi\n")
	}
	script.WriteString("  echo \"unexpected endpoint: $endpoint\" >&2\n")
	script.WriteString("  exit 1\n")
	script.WriteString("fi\n")
	script.WriteString("echo \"unsupported gh invocation: $*\" >&2\n")
	script.WriteString("exit 1\n")

	if err := os.WriteFile(scriptPath, []byte(script.String()), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", scriptPath, err)
	}

	return dir
}

func installFakeCommands(t *testing.T, commands ...string) string {
	t.Helper()

	dir := t.TempDir()
	for _, name := range commands {
		path := filepath.Join(dir, name)
		script := "#!/bin/sh\nexit 0\n"
		if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
	}
	return dir
}

func installFakeGitLabCLI(t *testing.T, endpoints map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "glab")

	var script strings.Builder
	script.WriteString("#!/bin/sh\n")
	script.WriteString("set -eu\n")
	script.WriteString("state=\"${FAKE_GLAB_STATE:-}\"\n")
	script.WriteString("require_login=\"${FAKE_GLAB_REQUIRE_LOGIN:-0}\"\n")
	script.WriteString("cmd=\"${1:-}\"\n")
	script.WriteString("sub=\"${2:-}\"\n")
	script.WriteString("if [ \"$cmd\" = \"auth\" ] && [ \"$sub\" = \"status\" ]; then\n")
	script.WriteString("  if [ \"$require_login\" = \"1\" ] && [ ! -f \"$state\" ]; then\n")
	script.WriteString("    echo \"not logged in\" >&2\n")
	script.WriteString("    exit 1\n")
	script.WriteString("  fi\n")
	script.WriteString("  echo \"Logged in\"\n")
	script.WriteString("  exit 0\n")
	script.WriteString("fi\n")
	script.WriteString("if [ \"$cmd\" = \"auth\" ] && [ \"$sub\" = \"login\" ]; then\n")
	script.WriteString("  if [ -n \"$state\" ]; then touch \"$state\"; fi\n")
	script.WriteString("  echo \"Logged in\"\n")
	script.WriteString("  exit 0\n")
	script.WriteString("fi\n")
	script.WriteString("if [ \"$cmd\" = \"api\" ]; then\n")
	script.WriteString("  for last; do true; done\n")
	script.WriteString("  endpoint=\"$last\"\n")
	for endpoint, output := range endpoints {
		script.WriteString("  if [ \"$endpoint\" = ")
		script.WriteString(shellQuote(endpoint))
		script.WriteString(" ]; then\n")
		script.WriteString("    cat <<'EOF'\n")
		script.WriteString(output)
		script.WriteString("\nEOF\n")
		script.WriteString("    exit 0\n")
		script.WriteString("  fi\n")
	}
	script.WriteString("  echo \"unexpected endpoint: $endpoint\" >&2\n")
	script.WriteString("  exit 1\n")
	script.WriteString("fi\n")
	script.WriteString("echo \"unsupported glab invocation: $*\" >&2\n")
	script.WriteString("exit 1\n")

	if err := os.WriteFile(scriptPath, []byte(script.String()), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", scriptPath, err)
	}

	return dir
}

func installFakeAzureCLI(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "az")

	var script strings.Builder
	script.WriteString("#!/bin/sh\n")
	script.WriteString("set -eu\n")
	script.WriteString("state=\"${FAKE_AZ_STATE:-}\"\n")
	script.WriteString("require_login=\"${FAKE_AZ_REQUIRE_LOGIN:-0}\"\n")
	script.WriteString("cmd=\"${1:-}\"\n")
	script.WriteString("sub=\"${2:-}\"\n")
	script.WriteString("if [ \"$cmd\" = \"account\" ] && [ \"$sub\" = \"show\" ]; then\n")
	script.WriteString("  if [ \"$require_login\" = \"1\" ] && [ ! -f \"$state\" ]; then\n")
	script.WriteString("    echo \"not logged in\" >&2\n")
	script.WriteString("    exit 1\n")
	script.WriteString("  fi\n")
	script.WriteString("  echo '{}'\n")
	script.WriteString("  exit 0\n")
	script.WriteString("fi\n")
	script.WriteString("if [ \"$cmd\" = \"login\" ]; then\n")
	script.WriteString("  if [ -n \"$state\" ]; then touch \"$state\"; fi\n")
	script.WriteString("  echo '[]'\n")
	script.WriteString("  exit 0\n")
	script.WriteString("fi\n")
	script.WriteString("if [ \"$cmd\" = \"account\" ] && [ \"$sub\" = \"get-access-token\" ]; then\n")
	script.WriteString("  if [ \"$require_login\" = \"1\" ] && [ ! -f \"$state\" ]; then\n")
	script.WriteString("    echo \"not logged in\" >&2\n")
	script.WriteString("    exit 1\n")
	script.WriteString("  fi\n")
	script.WriteString("  printf '%s\\n' \"${FAKE_AZ_ACCESS_TOKEN:-token-123}\"\n")
	script.WriteString("  exit 0\n")
	script.WriteString("fi\n")
	script.WriteString("echo \"unsupported az invocation: $*\" >&2\n")
	script.WriteString("exit 1\n")

	if err := os.WriteFile(scriptPath, []byte(script.String()), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", scriptPath, err)
	}

	return dir
}

func installFakeSVNCLI(t *testing.T, outputs map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "svn")

	var script strings.Builder
	script.WriteString("#!/bin/sh\n")
	script.WriteString("set -eu\n")
	script.WriteString("args=\"\"\n")
	script.WriteString("while [ \"$#\" -gt 0 ]; do\n")
	script.WriteString("  case \"$1\" in\n")
	script.WriteString("    --non-interactive)\n")
	script.WriteString("      shift\n")
	script.WriteString("      ;;\n")
	script.WriteString("    --username|--password)\n")
	script.WriteString("      shift 2\n")
	script.WriteString("      ;;\n")
	script.WriteString("    *)\n")
	script.WriteString("      break\n")
	script.WriteString("      ;;\n")
	script.WriteString("  esac\n")
	script.WriteString("done\n")
	script.WriteString("key=\"$*\"\n")
	for endpoint, output := range outputs {
		script.WriteString("if [ \"$key\" = ")
		script.WriteString(shellQuote(endpoint))
		script.WriteString(" ]; then\n")
		script.WriteString("  cat <<'EOF'\n")
		script.WriteString(output)
		script.WriteString("\nEOF\n")
		script.WriteString("  exit 0\n")
		script.WriteString("fi\n")
	}
	script.WriteString("case \"$key\" in\n")
	script.WriteString("  \"log --xml --verbose \"*)\n")
	script.WriteString("    printf '%s\\n' '<log></log>'\n")
	script.WriteString("    exit 0\n")
	script.WriteString("    ;;\n")
	script.WriteString("  \"log --xml -r \"*)\n")
	script.WriteString("    printf '%s\\n' '<log></log>'\n")
	script.WriteString("    exit 0\n")
	script.WriteString("    ;;\n")
	script.WriteString("esac\n")
	script.WriteString("echo \"unexpected svn invocation: $key\" >&2\n")
	script.WriteString("exit 1\n")

	if err := os.WriteFile(scriptPath, []byte(script.String()), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", scriptPath, err)
	}

	return dir
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func createPromotionFixture(t *testing.T) string {
	t.Helper()

	workspace := t.TempDir()
	repoRoot := filepath.Join(workspace, "a-service")

	initRepository(t, repoRoot)
	runGit(t, repoRoot, "checkout", "-b", "dev")

	writeFile(t, filepath.Join(repoRoot, "app.txt"), "hello")
	runGit(t, repoRoot, "add", "app.txt")
	runGit(t, repoRoot, "commit", "-m", "ABC-123 | service-a | add validation fix", "-m", "Depends-On: XYZ-456")

	writeFile(t, filepath.Join(repoRoot, "db", "migrations", "000_prerequisite.sql"), "alter table demo add column prerequisite int;\n")
	runGit(t, repoRoot, "add", "db/migrations/000_prerequisite.sql")
	runGit(t, repoRoot, "commit", "-m", "XYZ-456 | service-a | add prerequisite migration")
	secondHash := strings.TrimSpace(runGit(t, repoRoot, "rev-parse", "HEAD"))

	writeFile(t, filepath.Join(repoRoot, "db", "migrations", "001_add_column.sql"), "alter table demo add column enabled int;\n")
	runGit(t, repoRoot, "add", "db/migrations/001_add_column.sql")
	runGit(t, repoRoot, "commit", "-m", "ABC-123 | service-a | add migration")

	runGit(t, repoRoot, "checkout", "-b", "test", secondHash)

	return workspace
}

func createMergeConflictFixture(t *testing.T, workspace string) string {
	t.Helper()

	repoRoot := filepath.Join(workspace, "a-service")
	initRepository(t, repoRoot)

	writeFile(t, filepath.Join(repoRoot, "app.txt"), "shared line\n")
	runGit(t, repoRoot, "add", "app.txt")
	runGit(t, repoRoot, "commit", "-m", "BASE-1 seed app file")

	runGit(t, repoRoot, "checkout", "-b", "feature/ABC-123")
	writeFile(t, filepath.Join(repoRoot, "app.txt"), "feature line\n")
	runGit(t, repoRoot, "commit", "-am", "ABC-123 update app behavior")

	runGit(t, repoRoot, "checkout", "main")
	writeFile(t, filepath.Join(repoRoot, "app.txt"), "main line\n")
	runGit(t, repoRoot, "commit", "-am", "OPS-99 tighten validation")

	cmd := exec.Command("git", "-C", repoRoot, "merge", "feature/ABC-123")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("git merge unexpectedly succeeded: %s", strings.TrimSpace(string(output)))
	}

	return repoRoot
}

func createReleaseFixture(t *testing.T) string {
	t.Helper()

	workspace := createPromotionFixture(t)
	repoRoot := filepath.Join(workspace, "b-service")

	initRepository(t, repoRoot)
	runGit(t, repoRoot, "checkout", "-b", "dev")

	writeFile(t, filepath.Join(repoRoot, "app.txt"), "release-ready\n")
	runGit(t, repoRoot, "add", "app.txt")
	runGit(t, repoRoot, "commit", "-m", "XYZ-999 | service-b | add release-ready fix")
	commitHash := strings.TrimSpace(runGit(t, repoRoot, "rev-parse", "HEAD"))

	runGit(t, repoRoot, "checkout", "main")
	runGit(t, repoRoot, "cherry-pick", commitHash)
	runGit(t, repoRoot, "checkout", "-b", "test")

	return workspace
}

func createReleaseSnapshots(t *testing.T, workspace, releaseID string) {
	t.Helper()

	tickets := []string{"ABC-123", "XYZ-999"}
	for _, ticketID := range tickets {
		stdout, stderr, exitCode := runApp(t, "snapshot", "create", "--ticket", ticketID, "--from", "test", "--to", "main", "--path", workspace, "--release", releaseID)
		if exitCode != 0 {
			t.Fatalf("snapshot create %s exit code = %d, stderr = %q", ticketID, exitCode, stderr)
		}
		if !strings.Contains(stdout, releaseID) {
			t.Fatalf("snapshot create stdout = %q, want release %q", stdout, releaseID)
		}
	}
}
