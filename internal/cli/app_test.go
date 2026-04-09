package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
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
	if !strings.Contains(stderr, "gig <command> [flags]") {
		t.Fatalf("--help stderr = %q, want root usage", stderr)
	}
	if !strings.Contains(stderr, "scan        Find repositories under a path") {
		t.Fatalf("--help stderr = %q, want command summary", stderr)
	}
	if !strings.Contains(stderr, "manifest    Generate a release packet for QA, client, and release review") {
		t.Fatalf("--help stderr = %q, want manifest command summary", stderr)
	}
	if !strings.Contains(stderr, "snapshot    Save a repeatable ticket baseline for audit and re-check") {
		t.Fatalf("--help stderr = %q, want snapshot command summary", stderr)
	}
	if !strings.Contains(stderr, "doctor      Check config coverage, env mappings, and repo catalog health") {
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
