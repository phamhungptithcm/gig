package assistant

import (
	"encoding/json"
	"strings"
	"testing"

	inspectsvc "gig/internal/inspect"
	plansvc "gig/internal/plan"
	releaseplansvc "gig/internal/releaseplan"
	"gig/internal/scm"
)

func TestCollectReleaseRepositoryEvidenceIncludesGitHubDelta(t *testing.T) {
	t.Parallel()

	releasePlan := releaseplansvc.ReleasePlan{
		ReleaseID: "rel-2026-04-17",
		Verdict:   plansvc.VerdictBlocked,
		Summary: releaseplansvc.Summary{
			Tickets:        2,
			SafeTickets:    1,
			BlockedTickets: 1,
		},
		Repositories: []releaseplansvc.RepositoryPlan{
			{
				Repository: scm.Repository{Root: "github:acme/payments", Type: scm.TypeGitHub},
				TicketIDs:  []string{"ABC-123", "XYZ-999"},
				Verdict:    plansvc.VerdictBlocked,
				ProviderEvidence: &scm.ProviderEvidence{
					PullRequests: []scm.PullRequestEvidence{{ID: "#42", LinkedIssues: []scm.IssueEvidence{{ID: "#77", State: "open"}}}},
					Checks:       []scm.CheckEvidence{{Context: "build", State: "failure"}},
					Issues:       []scm.IssueEvidence{{ID: "#77", State: "open"}},
					Releases:     []scm.ReleaseEvidence{{ID: "v1.9.0", Tag: "v1.9.0", TicketIDs: []string{"XYZ-999"}}},
				},
				RiskSignals: []inspectsvc.RiskSignal{{Code: "db", Level: "warning", Summary: "DB migration"}},
			},
		},
	}

	repositoryEvidence := collectReleaseRepositoryEvidence(releasePlan)
	if len(repositoryEvidence) != 1 {
		t.Fatalf("len(repositoryEvidence) = %d, want 1", len(repositoryEvidence))
	}
	if got := repositoryEvidence[0].OverlapTickets; len(got) != 1 || got[0] != "XYZ-999" {
		t.Fatalf("OverlapTickets = %#v, want XYZ-999", got)
	}
	if got := repositoryEvidence[0].NewTicketsSinceLatestRelease; len(got) != 1 || got[0] != "ABC-123" {
		t.Fatalf("NewTicketsSinceLatestRelease = %#v, want ABC-123", got)
	}
	if repositoryEvidence[0].LatestRelease == nil || repositoryEvidence[0].LatestRelease.Tag != "v1.9.0" {
		t.Fatalf("LatestRelease = %#v, want v1.9.0", repositoryEvidence[0].LatestRelease)
	}

	overlap := collectReleaseTicketOverlap(releasePlan)
	if len(overlap) != 1 {
		t.Fatalf("len(ticketOverlap) = %d, want 1", len(overlap))
	}
	if len(overlap[0].PullRequestIDs) != 1 || overlap[0].PullRequestIDs[0] != "#42" {
		t.Fatalf("PullRequestIDs = %#v, want #42", overlap[0].PullRequestIDs)
	}

	hotspots := collectReleaseHotspots(releasePlan)
	summary := summarizeReleaseEvidence(repositoryEvidence, hotspots)
	if summary.LinkedIssues != 1 {
		t.Fatalf("LinkedIssues = %d, want 1", summary.LinkedIssues)
	}
	if summary.OverlappingTickets != 1 || summary.NewTicketsSinceLatestRelease != 1 {
		t.Fatalf("summary = %#v, want one overlapping and one new ticket", summary)
	}
}

func TestReleaseBundleJSONContractIsAdditive(t *testing.T) {
	t.Parallel()

	bundle := ReleaseBundle{
		ScopeLabel:      "github:acme/payments",
		ReleaseID:       "rel-2026-04-17",
		EvidenceSummary: ReleaseEvidenceSummary{PullRequests: 1, LinkedIssues: 2},
		RepositoryEvidence: []ReleaseRepositoryEvidence{
			{Repository: scm.Repository{Root: "github:acme/payments", Type: scm.TypeGitHub}},
		},
		TicketOverlap:    []ReleaseTicketOverlap{{Repository: scm.Repository{Root: "github:acme/payments", Type: scm.TypeGitHub}, TicketIDs: []string{"ABC-123", "XYZ-999"}}},
		ExecutiveSummary: []string{"Verdict blocked across two tickets."},
		OperatorSummary:  []string{"github:acme/payments has one non-green check."},
		Hotspots:         []ReleaseHotspot{{Repository: scm.Repository{Root: "github:acme/payments", Type: scm.TypeGitHub}, Severity: "blocked"}},
	}

	payload, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	text := string(payload)
	for _, key := range []string{
		`"evidenceSummary"`,
		`"repositoryEvidence"`,
		`"hotspots"`,
		`"ticketOverlap"`,
		`"executiveSummary"`,
		`"operatorSummary"`,
	} {
		if !strings.Contains(text, key) {
			t.Fatalf("payload = %s, want key %s", text, key)
		}
	}
}
