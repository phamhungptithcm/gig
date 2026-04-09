package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var releaseIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func NormalizeReleaseID(releaseID string) (string, error) {
	releaseID = strings.TrimSpace(releaseID)
	if releaseID == "" {
		return "", fmt.Errorf("release ID is required")
	}
	if !releaseIDPattern.MatchString(releaseID) {
		return "", fmt.Errorf("release ID %q must use only letters, numbers, dot, dash, or underscore", releaseID)
	}
	return releaseID, nil
}

func DefaultReleaseSnapshotDir(workspacePath, releaseID string) string {
	return filepath.Join(workspacePath, ".gig", "releases", releaseID, "snapshots")
}

func DefaultReleaseSnapshotPath(workspacePath, releaseID, ticketID string) string {
	return filepath.Join(DefaultReleaseSnapshotDir(workspacePath, releaseID), strings.ToLower(strings.TrimSpace(ticketID))+".json")
}

func LoadReleaseSnapshots(workspacePath, releaseID string) ([]TicketSnapshot, string, error) {
	releaseID, err := NormalizeReleaseID(releaseID)
	if err != nil {
		return nil, "", err
	}

	snapshotDir := DefaultReleaseSnapshotDir(workspacePath, releaseID)
	entries, err := os.ReadDir(snapshotDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, snapshotDir, fmt.Errorf("no snapshots found for release %s in %s", releaseID, snapshotDir)
		}
		return nil, snapshotDir, err
	}

	snapshots := make([]TicketSnapshot, 0, len(entries))
	seenTickets := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(snapshotDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, snapshotDir, err
		}

		var snapshot TicketSnapshot
		if err := json.Unmarshal(content, &snapshot); err != nil {
			return nil, snapshotDir, fmt.Errorf("parse snapshot %s: %w", path, err)
		}
		if snapshot.SchemaVersion != SchemaVersion {
			return nil, snapshotDir, fmt.Errorf("snapshot %s uses schema version %q, expected %q", path, snapshot.SchemaVersion, SchemaVersion)
		}
		if snapshot.ReleaseID != "" && snapshot.ReleaseID != releaseID {
			return nil, snapshotDir, fmt.Errorf("snapshot %s belongs to release %s, not %s", path, snapshot.ReleaseID, releaseID)
		}
		if strings.TrimSpace(snapshot.TicketID) == "" {
			return nil, snapshotDir, fmt.Errorf("snapshot %s is missing ticket ID", path)
		}
		if _, ok := seenTickets[snapshot.TicketID]; ok {
			return nil, snapshotDir, fmt.Errorf("duplicate ticket snapshot %s in release %s", snapshot.TicketID, releaseID)
		}
		seenTickets[snapshot.TicketID] = struct{}{}
		snapshots = append(snapshots, snapshot)
	}

	if len(snapshots) == 0 {
		return nil, snapshotDir, fmt.Errorf("no snapshots found for release %s in %s", releaseID, snapshotDir)
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].TicketID < snapshots[j].TicketID
	})

	return snapshots, snapshotDir, nil
}
