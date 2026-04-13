package snapshot

import (
	"context"
	"errors"
	"testing"
	"time"

	"gig/internal/config"
	inspectsvc "gig/internal/inspect"
	plansvc "gig/internal/plan"
	"gig/internal/scm"
)

func TestServiceCaptureBuildsBaselineSnapshot(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, time.April, 9, 16, 30, 0, 0, time.UTC)
	repository := scm.Repository{
		Name:          "a-service",
		Root:          "/tmp/workspace/a-service",
		Type:          scm.TypeGit,
		CurrentBranch: "test",
	}

	service := &Service{
		inspector: fakeInspector{
			results: []inspectsvc.RepositoryInspection{
				{
					Repository: repository,
					Commits: []scm.Commit{
						{Hash: "1234567890abcdef", Subject: "ABC-123 | service-a | add validation fix"},
						{Hash: "abcdef1234567890", Subject: "ABC-123 | service-a | add migration"},
					},
				},
			},
			scannedRepositories: 3,
		},
		planner: fakePlanner{
			plan: plansvc.PromotionPlan{
				TicketID:   "ABC-123",
				FromBranch: "test",
				ToBranch:   "main",
				Verdict:    plansvc.VerdictBlocked,
				Summary: plansvc.Summary{
					ScannedRepositories: 3,
					TouchedRepositories: 1,
				},
			},
			verification: plansvc.Verification{
				TicketID:   "ABC-123",
				FromBranch: "test",
				ToBranch:   "main",
				Verdict:    plansvc.VerdictBlocked,
				Repositories: []plansvc.RepositoryVerification{
					{
						Repository: repository,
						Verdict:    plansvc.VerdictBlocked,
					},
				},
			},
		},
		now: func() time.Time { return fixedTime },
	}

	snapshot, err := service.Capture(context.Background(), "/tmp/workspace", config.Loaded{Path: "/tmp/workspace/gig.yaml"}, "ABC-123", "test", "main", []inspectsvc.Environment{
		{Name: "test", Branch: "test"},
		{Name: "prod", Branch: "main"},
	})
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	if snapshot.SchemaVersion != SchemaVersion {
		t.Fatalf("SchemaVersion = %q, want %q", snapshot.SchemaVersion, SchemaVersion)
	}
	if !snapshot.CapturedAt.Equal(fixedTime) {
		t.Fatalf("CapturedAt = %s, want %s", snapshot.CapturedAt, fixedTime)
	}
	if snapshot.Workspace != "/tmp/workspace" {
		t.Fatalf("Workspace = %q, want /tmp/workspace", snapshot.Workspace)
	}
	if snapshot.ConfigPath != "/tmp/workspace/gig.yaml" {
		t.Fatalf("ConfigPath = %q, want /tmp/workspace/gig.yaml", snapshot.ConfigPath)
	}
	if snapshot.Inspection.ScannedRepositories != 3 {
		t.Fatalf("Inspection.ScannedRepositories = %d, want 3", snapshot.Inspection.ScannedRepositories)
	}
	if snapshot.Inspection.TouchedRepositories != 1 {
		t.Fatalf("Inspection.TouchedRepositories = %d, want 1", snapshot.Inspection.TouchedRepositories)
	}
	if snapshot.Inspection.TotalCommits != 2 {
		t.Fatalf("Inspection.TotalCommits = %d, want 2", snapshot.Inspection.TotalCommits)
	}
	if len(snapshot.Inspection.Repositories) != 1 {
		t.Fatalf("len(Inspection.Repositories) = %d, want 1", len(snapshot.Inspection.Repositories))
	}
	if snapshot.Plan.Verdict != plansvc.VerdictBlocked {
		t.Fatalf("Plan.Verdict = %q, want %q", snapshot.Plan.Verdict, plansvc.VerdictBlocked)
	}
	if snapshot.Verification.Verdict != plansvc.VerdictBlocked {
		t.Fatalf("Verification.Verdict = %q, want %q", snapshot.Verification.Verdict, plansvc.VerdictBlocked)
	}
}

func TestServiceCaptureReturnsInspectorError(t *testing.T) {
	t.Parallel()

	service := &Service{
		inspector: fakeInspector{err: errors.New("inspect failed")},
		planner:   fakePlanner{},
		now:       func() time.Time { return time.Unix(0, 0) },
	}

	_, err := service.Capture(context.Background(), "/tmp/workspace", config.Loaded{}, "ABC-123", "test", "main", nil)
	if err == nil || err.Error() != "inspect failed" {
		t.Fatalf("Capture() error = %v, want inspect failed", err)
	}
}

type fakeInspector struct {
	results             []inspectsvc.RepositoryInspection
	scannedRepositories int
	err                 error
}

func (f fakeInspector) Inspect(ctx context.Context, path, ticketID string) ([]inspectsvc.RepositoryInspection, int, error) {
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.results, f.scannedRepositories, nil
}

func (f fakeInspector) InspectInRepositories(ctx context.Context, repositories []scm.Repository, ticketID string) ([]inspectsvc.RepositoryInspection, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.results, nil
}

type fakePlanner struct {
	plan         plansvc.PromotionPlan
	verification plansvc.Verification
	err          error
}

func (f fakePlanner) BuildPromotionPlan(ctx context.Context, path, ticketID, fromBranch, toBranch string, environments []inspectsvc.Environment) (plansvc.PromotionPlan, error) {
	if f.err != nil {
		return plansvc.PromotionPlan{}, f.err
	}
	return f.plan, nil
}

func (f fakePlanner) VerifyPromotion(ctx context.Context, path, ticketID, fromBranch, toBranch string, environments []inspectsvc.Environment) (plansvc.Verification, error) {
	if f.err != nil {
		return plansvc.Verification{}, f.err
	}
	return f.verification, nil
}

func (f fakePlanner) BuildPromotionPlanInRepositories(ctx context.Context, repositories []scm.Repository, ticketID, fromBranch, toBranch string, environments []inspectsvc.Environment) (plansvc.PromotionPlan, error) {
	if f.err != nil {
		return plansvc.PromotionPlan{}, f.err
	}
	return f.plan, nil
}
