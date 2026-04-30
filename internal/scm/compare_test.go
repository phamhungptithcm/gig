package scm

import "testing"

func TestMissingCommitsByEvidenceUsesHash(t *testing.T) {
	t.Parallel()

	missing := MissingCommitsByEvidence(
		[]Commit{{Hash: "abc123", Subject: "ABC-123 fix payment"}},
		[]Commit{{Hash: "abc123", Subject: "ABC-123 cherry-picked payment"}},
	)
	if len(missing) != 0 {
		t.Fatalf("len(missing) = %d, want 0", len(missing))
	}
}

func TestMissingCommitsByEvidenceUsesSubjectForCherryPick(t *testing.T) {
	t.Parallel()

	missing := MissingCommitsByEvidence(
		[]Commit{{Hash: "abc123", Subject: "ABC-123 fix payment"}},
		[]Commit{{Hash: "def456", Subject: " ABC-123   fix payment "}},
	)
	if len(missing) != 0 {
		t.Fatalf("len(missing) = %d, want 0 for equivalent cherry-pick subject", len(missing))
	}
}

func TestMissingCommitsByEvidenceReturnsUnmatchedSource(t *testing.T) {
	t.Parallel()

	missing := MissingCommitsByEvidence(
		[]Commit{{Hash: "abc123", Subject: "ABC-123 fix payment"}},
		[]Commit{{Hash: "def456", Subject: "ABC-123 unrelated follow-up"}},
	)
	if len(missing) != 1 || missing[0].Hash != "abc123" {
		t.Fatalf("missing = %#v, want abc123", missing)
	}
}
