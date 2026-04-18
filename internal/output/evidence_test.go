package output

import (
	"bytes"
	"strings"
	"testing"

	"gig/internal/scm"
)

func TestRenderProviderEvidenceIncludesChecksIssuesAndReleases(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	err := renderProviderEvidence(&buffer, scm.ProviderEvidence{
		Checks:   []scm.CheckEvidence{{Context: "build", State: "failed", CommitHash: "abc123456789"}},
		Issues:   []scm.IssueEvidence{{ID: "#77", Title: "Prod validation", State: "open"}},
		Releases: []scm.ReleaseEvidence{{Tag: "v1.9.0", State: "released", TicketIDs: []string{"ABC-123"}}},
	}, "  ")
	if err != nil {
		t.Fatalf("renderProviderEvidence() error = %v", err)
	}

	text := buffer.String()
	for _, fragment := range []string{"checks:", "linked issues:", "releases:", "build | failed", "#77 | Prod validation | open", "v1.9.0 | released | tickets ABC-123"} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("output = %q, want fragment %q", text, fragment)
		}
	}
}

func TestHasProviderEvidenceTreatsAdditiveFieldsAsVisible(t *testing.T) {
	t.Parallel()

	if !hasProviderEvidence(&scm.ProviderEvidence{Checks: []scm.CheckEvidence{{Context: "build"}}}) {
		t.Fatalf("hasProviderEvidence() = false, want true for check-only evidence")
	}
	if hasProviderEvidence(&scm.ProviderEvidence{}) {
		t.Fatalf("hasProviderEvidence() = true, want false for empty evidence")
	}
}
