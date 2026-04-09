package releaseplan

import (
	"testing"
	"time"

	depsvc "gig/internal/dependency"
	inspectsvc "gig/internal/inspect"
	plansvc "gig/internal/plan"
	"gig/internal/scm"
	snapshotsvc "gig/internal/snapshot"
)

func TestBuildAggregatesSnapshots(t *testing.T) {
	t.Parallel()

	repository := scm.Repository{
		Name:          "a-service",
		Root:          "/tmp/workspace/a-service",
		Type:          scm.TypeGit,
		CurrentBranch: "test",
	}
	snapshots := []snapshotsvc.TicketSnapshot{
		{
			SchemaVersion: snapshotsvc.SchemaVersion,
			ReleaseID:     "rel-2026-04-09",
			CapturedAt:    time.Date(2026, time.April, 9, 10, 0, 0, 0, time.UTC),
			Workspace:     "/tmp/workspace",
			TicketID:      "ABC-123",
			FromBranch:    "test",
			ToBranch:      "main",
			Environments: []inspectsvc.Environment{
				{Name: "test", Branch: "test"},
				{Name: "prod", Branch: "main"},
			},
			Plan: plansvc.PromotionPlan{
				TicketID:   "ABC-123",
				FromBranch: "test",
				ToBranch:   "main",
				Verdict:    plansvc.VerdictBlocked,
				Summary: plansvc.Summary{
					TouchedRepositories:   1,
					TotalCommitsToPromote: 1,
					TotalManualSteps:      1,
				},
				Repositories: []plansvc.RepositoryPlan{
					{
						Repository: repository,
						Compare: scm.CompareResult{
							MissingCommits: []scm.Commit{{Hash: "aaaabbbb", Subject: "ABC-123 fix"}},
						},
						Verdict: plansvc.VerdictBlocked,
						RiskSignals: []inspectsvc.RiskSignal{
							{Code: "db-change", Level: "manual-review", Summary: "DB review needed"},
						},
						DependencyResolutions: []depsvc.Resolution{
							{TicketID: "ABC-123", DependsOn: "XYZ-456", Status: depsvc.StatusMissingTarget},
						},
						ManualSteps: []plansvc.Action{{Code: "review-db", Summary: "Review DB rollout"}},
						Actions:     []plansvc.Action{{Code: "include-commits", Summary: "Include missing commits"}},
						Notes:       []string{"Target branch is behind."},
					},
				},
			},
		},
		{
			SchemaVersion: snapshotsvc.SchemaVersion,
			ReleaseID:     "rel-2026-04-09",
			CapturedAt:    time.Date(2026, time.April, 9, 11, 0, 0, 0, time.UTC),
			Workspace:     "/tmp/workspace",
			TicketID:      "XYZ-999",
			FromBranch:    "test",
			ToBranch:      "main",
			Environments: []inspectsvc.Environment{
				{Name: "test", Branch: "test"},
				{Name: "prod", Branch: "main"},
			},
			Plan: plansvc.PromotionPlan{
				TicketID:   "XYZ-999",
				FromBranch: "test",
				ToBranch:   "main",
				Verdict:    plansvc.VerdictSafe,
				Summary: plansvc.Summary{
					TouchedRepositories:   1,
					TotalCommitsToPromote: 0,
					TotalManualSteps:      0,
				},
				Repositories: []plansvc.RepositoryPlan{
					{
						Repository: repository,
						Verdict:    plansvc.VerdictSafe,
						Actions:    []plansvc.Action{{Code: "already-aligned", Summary: "No missing commits"}},
					},
				},
			},
		},
	}

	releasePlan, err := Build("rel-2026-04-09", "/tmp/workspace", "/tmp/workspace/.gig/releases/rel-2026-04-09/snapshots", snapshots)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if releasePlan.Verdict != plansvc.VerdictBlocked {
		t.Fatalf("Verdict = %q, want blocked", releasePlan.Verdict)
	}
	if releasePlan.Summary.Tickets != 2 {
		t.Fatalf("Summary.Tickets = %d, want 2", releasePlan.Summary.Tickets)
	}
	if releasePlan.Summary.BlockedTickets != 1 || releasePlan.Summary.SafeTickets != 1 {
		t.Fatalf("ticket verdict counts = %+v, want one blocked and one safe", releasePlan.Summary)
	}
	if len(releasePlan.Repositories) != 1 {
		t.Fatalf("len(Repositories) = %d, want 1", len(releasePlan.Repositories))
	}
	if releasePlan.Repositories[0].CommitsToInclude != 1 {
		t.Fatalf("CommitsToInclude = %d, want 1", releasePlan.Repositories[0].CommitsToInclude)
	}
	if len(releasePlan.Repositories[0].TicketIDs) != 2 {
		t.Fatalf("len(TicketIDs) = %d, want 2", len(releasePlan.Repositories[0].TicketIDs))
	}
	if got := releasePlan.Repositories[0].ManualSteps[0].Summary; got != "ABC-123: Review DB rollout" {
		t.Fatalf("ManualSteps[0] = %q, want ticket-prefixed summary", got)
	}
}

func TestBuildBlocksMismatchedSnapshotBaseline(t *testing.T) {
	t.Parallel()

	releasePlan, err := Build("rel-2026-04-09", "/tmp/workspace", "/tmp/workspace/.gig/releases/rel-2026-04-09/snapshots", []snapshotsvc.TicketSnapshot{
		{
			SchemaVersion: snapshotsvc.SchemaVersion,
			Workspace:     "/tmp/workspace",
			TicketID:      "ABC-123",
			FromBranch:    "test",
			ToBranch:      "main",
			Environments:  []inspectsvc.Environment{{Name: "test", Branch: "test"}},
			Plan:          plansvc.PromotionPlan{Verdict: plansvc.VerdictSafe},
		},
		{
			SchemaVersion: snapshotsvc.SchemaVersion,
			Workspace:     "/tmp/workspace",
			TicketID:      "XYZ-999",
			FromBranch:    "uat",
			ToBranch:      "main",
			Environments:  []inspectsvc.Environment{{Name: "uat", Branch: "uat"}},
			Plan:          plansvc.PromotionPlan{Verdict: plansvc.VerdictSafe},
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if releasePlan.Verdict != plansvc.VerdictBlocked {
		t.Fatalf("Verdict = %q, want blocked", releasePlan.Verdict)
	}
	if len(releasePlan.Notes) == 0 {
		t.Fatalf("Notes = %v, want mismatch note", releasePlan.Notes)
	}
}
