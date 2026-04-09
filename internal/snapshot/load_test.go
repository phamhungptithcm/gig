package snapshot

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadReleaseSnapshots(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	releaseID := "rel-2026-04-09"
	snapshotPath := DefaultReleaseSnapshotPath(workspace, releaseID, "ABC-123")

	if err := os.MkdirAll(filepath.Dir(snapshotPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	content := `{
  "schemaVersion": "1",
  "releaseId": "rel-2026-04-09",
  "capturedAt": "2026-04-09T12:00:00Z",
  "toolVersion": "gig dev",
  "workspace": "` + workspace + `",
  "ticketId": "ABC-123",
  "fromBranch": "test",
  "toBranch": "main",
  "inspection": {
    "scannedRepositories": 1,
    "touchedRepositories": 1,
    "totalCommits": 1
  },
  "plan": {
    "ticketId": "ABC-123",
    "fromBranch": "test",
    "toBranch": "main",
    "summary": {
      "scannedRepositories": 1,
      "touchedRepositories": 1,
      "readyRepositories": 1,
      "warningRepositories": 0,
      "blockedRepositories": 0,
      "totalCommitsToPromote": 0,
      "totalManualSteps": 0
    },
    "verdict": "safe"
  },
  "verification": {
    "ticketId": "ABC-123",
    "fromBranch": "test",
    "toBranch": "main",
    "summary": {
      "scannedRepositories": 1,
      "touchedRepositories": 1,
      "readyRepositories": 1,
      "warningRepositories": 0,
      "blockedRepositories": 0,
      "totalCommitsToPromote": 0,
      "totalManualSteps": 0
    },
    "verdict": "safe",
    "reasons": ["ready"]
  }
}`
	if err := os.WriteFile(snapshotPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	snapshots, dir, err := LoadReleaseSnapshots(workspace, releaseID)
	if err != nil {
		t.Fatalf("LoadReleaseSnapshots() error = %v", err)
	}

	if dir != DefaultReleaseSnapshotDir(workspace, releaseID) {
		t.Fatalf("dir = %q, want %q", dir, DefaultReleaseSnapshotDir(workspace, releaseID))
	}
	if len(snapshots) != 1 {
		t.Fatalf("len(snapshots) = %d, want 1", len(snapshots))
	}
	if snapshots[0].ReleaseID != releaseID {
		t.Fatalf("ReleaseID = %q, want %q", snapshots[0].ReleaseID, releaseID)
	}
	if !snapshots[0].CapturedAt.Equal(time.Date(2026, time.April, 9, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("CapturedAt = %s, want fixed timestamp", snapshots[0].CapturedAt)
	}
}
