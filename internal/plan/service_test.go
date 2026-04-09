package plan_test

import (
	"context"
	"testing"

	inspectsvc "gig/internal/inspect"
	plansvc "gig/internal/plan"
	"gig/internal/scm"
	"gig/internal/ticket"
)

type fakeDiscoverer struct {
	repositories []scm.Repository
}

func (f fakeDiscoverer) Discover(context.Context, string) ([]scm.Repository, error) {
	return f.repositories, nil
}

type fakeAdapterProvider struct {
	adapters map[scm.Type]scm.Adapter
}

func (f fakeAdapterProvider) For(repoType scm.Type) (scm.Adapter, bool) {
	adapter, ok := f.adapters[repoType]
	return adapter, ok
}

type fakeAdapter struct {
	repoType    scm.Type
	search      map[string]map[string][]scm.Commit
	compare     map[string]scm.CompareResult
	existingRef map[string]map[string]bool
}

func (f fakeAdapter) Type() scm.Type { return f.repoType }

func (f fakeAdapter) DetectRoot(string) (string, bool, error) { return "", false, nil }

func (f fakeAdapter) IsRepository(string) (bool, error) { return false, nil }

func (f fakeAdapter) CurrentBranch(context.Context, string) (string, error) { return "", nil }

func (f fakeAdapter) SearchCommits(_ context.Context, repoRoot string, query scm.SearchQuery) ([]scm.Commit, error) {
	if byBranch, ok := f.search[repoRoot]; ok {
		if commits, ok := byBranch[query.Branch]; ok {
			return commits, nil
		}
	}
	return nil, nil
}

func (f fakeAdapter) CompareBranches(_ context.Context, repoRoot string, _ scm.CompareQuery) (scm.CompareResult, error) {
	if result, ok := f.compare[repoRoot]; ok {
		return result, nil
	}
	return scm.CompareResult{}, nil
}

func (f fakeAdapter) PrepareCherryPick(context.Context, string, scm.CherryPickPlan) error {
	return scm.ErrUnsupported
}

func (f fakeAdapter) RefExists(_ context.Context, repoRoot, ref string) (bool, error) {
	return f.existingRef[repoRoot][ref], nil
}

func TestBuildPromotionPlanBlocksWhenSourceEnvIsBehind(t *testing.T) {
	t.Parallel()

	parser := mustParser(t)
	service := plansvc.NewService(
		fakeDiscoverer{
			repositories: []scm.Repository{
				{Name: "repo-a", Root: "/workspace/repo-a", Type: scm.TypeGit},
			},
		},
		fakeAdapterProvider{
			adapters: map[scm.Type]scm.Adapter{
				scm.TypeGit: fakeAdapter{
					repoType: scm.TypeGit,
					search: map[string]map[string][]scm.Commit{
						"/workspace/repo-a": {
							"": {
								{Hash: "abc12345", Subject: "ABC-123 initial fix", Branches: []string{"dev", "test"}},
								{Hash: "def67890", Subject: "ABC-123 follow-up fix", Branches: []string{"dev"}},
							},
							"dev": {
								{Hash: "abc12345", Subject: "ABC-123 initial fix", Branches: []string{"dev"}},
								{Hash: "def67890", Subject: "ABC-123 follow-up fix", Branches: []string{"dev"}},
							},
							"test": {
								{Hash: "abc12345", Subject: "ABC-123 initial fix", Branches: []string{"test"}},
							},
							"main": {},
						},
					},
					compare: map[string]scm.CompareResult{
						"/workspace/repo-a": {
							FromBranch: "test",
							ToBranch:   "main",
							SourceCommits: []scm.Commit{
								{Hash: "abc12345", Subject: "ABC-123 initial fix"},
							},
							MissingCommits: []scm.Commit{
								{Hash: "abc12345", Subject: "ABC-123 initial fix"},
							},
						},
					},
					existingRef: map[string]map[string]bool{
						"/workspace/repo-a": {
							"dev":  true,
							"test": true,
							"main": true,
						},
					},
				},
			},
		},
		parser,
	)

	promotionPlan, err := service.BuildPromotionPlan(
		context.Background(),
		".",
		"ABC-123",
		"test",
		"main",
		[]inspectsvc.Environment{
			{Name: "dev", Branch: "dev"},
			{Name: "test", Branch: "test"},
			{Name: "prod", Branch: "main"},
		},
	)
	if err != nil {
		t.Fatalf("BuildPromotionPlan() error = %v", err)
	}

	if promotionPlan.Verdict != plansvc.VerdictBlocked {
		t.Fatalf("promotionPlan.Verdict = %q, want %q", promotionPlan.Verdict, plansvc.VerdictBlocked)
	}
	if promotionPlan.Summary.BlockedRepositories != 1 {
		t.Fatalf("promotionPlan.Summary.BlockedRepositories = %d, want 1", promotionPlan.Summary.BlockedRepositories)
	}
	if len(promotionPlan.Repositories) != 1 {
		t.Fatalf("len(promotionPlan.Repositories) = %d, want 1", len(promotionPlan.Repositories))
	}
	if len(promotionPlan.Repositories[0].Actions) == 0 {
		t.Fatalf("expected repository plan actions to be populated")
	}
}

func TestBuildVerificationWarnsForManualSteps(t *testing.T) {
	t.Parallel()

	verification := plansvc.BuildVerification(plansvc.PromotionPlan{
		TicketID:   "ABC-123",
		FromBranch: "test",
		ToBranch:   "main",
		Summary: plansvc.Summary{
			ScannedRepositories:   1,
			TouchedRepositories:   1,
			WarningRepositories:   1,
			TotalCommitsToPromote: 1,
			TotalManualSteps:      1,
		},
		Verdict: plansvc.VerdictWarning,
		Repositories: []plansvc.RepositoryPlan{
			{
				Repository: scm.Repository{Name: "repo-a", Root: "/workspace/repo-a", Type: scm.TypeGit},
				Compare: scm.CompareResult{
					FromBranch: "test",
					ToBranch:   "main",
					SourceCommits: []scm.Commit{
						{Hash: "abc12345", Subject: "ABC-123 initial fix"},
					},
					MissingCommits: []scm.Commit{
						{Hash: "abc12345", Subject: "ABC-123 initial fix"},
					},
				},
				RiskSignals: []inspectsvc.RiskSignal{
					{Code: "db-change", Level: "manual-review"},
				},
				ManualSteps: []plansvc.Action{
					{Code: "review-db-rollout", Summary: "Review DB migration ordering, rollback steps, and deployment timing before promotion."},
				},
				Verdict: plansvc.VerdictWarning,
			},
		},
	})

	if verification.Verdict != plansvc.VerdictWarning {
		t.Fatalf("verification.Verdict = %q, want %q", verification.Verdict, plansvc.VerdictWarning)
	}
	if len(verification.Reasons) == 0 {
		t.Fatalf("expected verification reasons to be populated")
	}
	if len(verification.Repositories) != 1 {
		t.Fatalf("len(verification.Repositories) = %d, want 1", len(verification.Repositories))
	}
	if len(verification.Repositories[0].Checks) == 0 {
		t.Fatalf("expected repository checks to be populated")
	}
}

func mustParser(t *testing.T) ticket.Parser {
	t.Helper()

	parser, err := ticket.NewParser(`\b[A-Z][A-Z0-9]+-\d+\b`)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	return parser
}
