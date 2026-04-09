package plan_test

import (
	"context"
	"strings"
	"testing"

	depsvc "gig/internal/dependency"
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
	repoType       scm.Type
	search         map[string]map[string][]scm.Commit
	searchByTicket map[string]map[string]map[string][]scm.Commit
	compare        map[string]scm.CompareResult
	existingRef    map[string]map[string]bool
	commitMessages map[string]map[string]string
}

func (f fakeAdapter) Type() scm.Type { return f.repoType }

func (f fakeAdapter) DetectRoot(string) (string, bool, error) { return "", false, nil }

func (f fakeAdapter) IsRepository(string) (bool, error) { return false, nil }

func (f fakeAdapter) CurrentBranch(context.Context, string) (string, error) { return "", nil }

func (f fakeAdapter) SearchCommits(_ context.Context, repoRoot string, query scm.SearchQuery) ([]scm.Commit, error) {
	if byTicket, ok := f.searchByTicket[repoRoot]; ok {
		if byBranch, ok := byTicket[query.TicketID]; ok {
			if commits, ok := byBranch[query.Branch]; ok {
				return commits, nil
			}
		}
	}
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

func (f fakeAdapter) CommitMessages(_ context.Context, repoRoot string, hashes []string) (map[string]string, error) {
	messages := map[string]string{}
	byHash := f.commitMessages[repoRoot]
	for _, hash := range hashes {
		if message, ok := byHash[hash]; ok {
			messages[hash] = message
		}
	}
	return messages, nil
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

func TestBuildPromotionPlanBlocksWhenDependencyIsMissingFromTarget(t *testing.T) {
	t.Parallel()

	parser := mustParser(t)
	service := plansvc.NewService(
		fakeDiscoverer{
			repositories: []scm.Repository{
				{Name: "repo-a", Root: "/workspace/repo-a", Type: scm.TypeGit},
				{Name: "repo-db", Root: "/workspace/repo-db", Type: scm.TypeGit},
			},
		},
		fakeAdapterProvider{
			adapters: map[scm.Type]scm.Adapter{
				scm.TypeGit: fakeAdapter{
					repoType: scm.TypeGit,
					searchByTicket: map[string]map[string]map[string][]scm.Commit{
						"/workspace/repo-a": {
							"ABC-123": {
								"": {
									{Hash: "abc12345", Subject: "ABC-123 app fix", Branches: []string{"test"}},
								},
								"test": {
									{Hash: "abc12345", Subject: "ABC-123 app fix", Branches: []string{"test"}},
								},
								"main": {},
							},
						},
						"/workspace/repo-db": {
							"XYZ-456": {
								"": {
									{Hash: "dep12345", Subject: "XYZ-456 db prerequisite", Branches: []string{"test"}},
								},
								"test": {
									{Hash: "dep12345", Subject: "XYZ-456 db prerequisite", Branches: []string{"test"}},
								},
								"main": {},
							},
						},
					},
					compare: map[string]scm.CompareResult{
						"/workspace/repo-a": {
							FromBranch: "test",
							ToBranch:   "main",
							SourceCommits: []scm.Commit{
								{Hash: "abc12345", Subject: "ABC-123 app fix"},
							},
							MissingCommits: []scm.Commit{
								{Hash: "abc12345", Subject: "ABC-123 app fix"},
							},
						},
						"/workspace/repo-db": {},
					},
					existingRef: map[string]map[string]bool{
						"/workspace/repo-a":  {"test": true, "main": true},
						"/workspace/repo-db": {"test": true, "main": true},
					},
					commitMessages: map[string]map[string]string{
						"/workspace/repo-a": {
							"abc12345": "ABC-123 app fix\n\nDepends-On: XYZ-456\n",
						},
					},
				},
			},
		},
		parser,
	)

	plan, err := service.BuildPromotionPlan(context.Background(), ".", "ABC-123", "test", "main", nil)
	if err != nil {
		t.Fatalf("BuildPromotionPlan() error = %v", err)
	}

	if plan.Verdict != plansvc.VerdictBlocked {
		t.Fatalf("plan.Verdict = %q, want %q", plan.Verdict, plansvc.VerdictBlocked)
	}
	if len(plan.Repositories) != 1 {
		t.Fatalf("len(plan.Repositories) = %d, want 1", len(plan.Repositories))
	}
	if len(plan.Repositories[0].DependencyResolutions) != 1 {
		t.Fatalf("len(DependencyResolutions) = %d, want 1", len(plan.Repositories[0].DependencyResolutions))
	}
	if got := plan.Repositories[0].DependencyResolutions[0].Status; got != depsvc.StatusMissingTarget {
		t.Fatalf("DependencyResolutions[0].Status = %q, want %q", got, depsvc.StatusMissingTarget)
	}
	if got := plan.Repositories[0].RiskSignals[0].Code; got != "missing-dependency" && !containsRiskCode(plan.Repositories[0].RiskSignals, "missing-dependency") {
		t.Fatalf("expected missing-dependency risk signal, got %#v", plan.Repositories[0].RiskSignals)
	}
	if !containsAction(plan.Repositories[0].Actions, "include-missing-dependency") {
		t.Fatalf("expected include-missing-dependency action, got %#v", plan.Repositories[0].Actions)
	}
}

func TestBuildPromotionPlanWarnsWhenDependencyIsUnresolved(t *testing.T) {
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
					searchByTicket: map[string]map[string]map[string][]scm.Commit{
						"/workspace/repo-a": {
							"ABC-123": {
								"": {
									{Hash: "abc12345", Subject: "ABC-123 app fix", Branches: []string{"test"}},
								},
								"test": {
									{Hash: "abc12345", Subject: "ABC-123 app fix", Branches: []string{"test"}},
								},
								"main": {},
							},
						},
					},
					compare: map[string]scm.CompareResult{
						"/workspace/repo-a": {
							FromBranch: "test",
							ToBranch:   "main",
							SourceCommits: []scm.Commit{
								{Hash: "abc12345", Subject: "ABC-123 app fix"},
							},
							MissingCommits: []scm.Commit{
								{Hash: "abc12345", Subject: "ABC-123 app fix"},
							},
						},
					},
					existingRef: map[string]map[string]bool{
						"/workspace/repo-a": {"test": true, "main": true},
					},
					commitMessages: map[string]map[string]string{
						"/workspace/repo-a": {
							"abc12345": "ABC-123 app fix\n\nDepends-On: OPS-99\n",
						},
					},
				},
			},
		},
		parser,
	)

	plan, err := service.BuildPromotionPlan(context.Background(), ".", "ABC-123", "test", "main", nil)
	if err != nil {
		t.Fatalf("BuildPromotionPlan() error = %v", err)
	}

	if plan.Verdict != plansvc.VerdictWarning {
		t.Fatalf("plan.Verdict = %q, want %q", plan.Verdict, plansvc.VerdictWarning)
	}
	if got := plan.Repositories[0].DependencyResolutions[0].Status; got != depsvc.StatusUnresolved {
		t.Fatalf("DependencyResolutions[0].Status = %q, want %q", got, depsvc.StatusUnresolved)
	}
	if !containsRiskCode(plan.Repositories[0].RiskSignals, "unresolved-dependency") {
		t.Fatalf("expected unresolved-dependency risk signal, got %#v", plan.Repositories[0].RiskSignals)
	}
	if !containsAction(plan.Repositories[0].Actions, "verify-dependency-scope") {
		t.Fatalf("expected verify-dependency-scope action, got %#v", plan.Repositories[0].Actions)
	}
	if !strings.Contains(strings.Join(plan.Notes, "\n"), "could not be confirmed") {
		t.Fatalf("expected dependency warning note, got %#v", plan.Notes)
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

func containsRiskCode(riskSignals []inspectsvc.RiskSignal, code string) bool {
	for _, riskSignal := range riskSignals {
		if riskSignal.Code == code {
			return true
		}
	}
	return false
}

func containsAction(actions []plansvc.Action, code string) bool {
	for _, action := range actions {
		if action.Code == code {
			return true
		}
	}
	return false
}
