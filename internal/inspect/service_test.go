package inspect

import (
	"context"
	"testing"

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

func (f fakeAdapter) Type() scm.Type                                        { return f.repoType }
func (f fakeAdapter) DetectRoot(string) (string, bool, error)               { return "", false, nil }
func (f fakeAdapter) IsRepository(string) (bool, error)                     { return false, nil }
func (f fakeAdapter) CurrentBranch(context.Context, string) (string, error) { return "", nil }
func (f fakeAdapter) PrepareCherryPick(context.Context, string, scm.CherryPickPlan) error {
	return scm.ErrUnsupported
}

func (f fakeAdapter) SearchCommits(_ context.Context, repoRoot string, query scm.SearchQuery) ([]scm.Commit, error) {
	if byBranch, ok := f.search[repoRoot]; ok {
		if commits, ok := byBranch[query.Branch]; ok {
			return commits, nil
		}
	}
	return nil, nil
}

func (f fakeAdapter) CompareBranches(_ context.Context, repoRoot string, query scm.CompareQuery) (scm.CompareResult, error) {
	key := repoRoot + "|" + query.FromBranch + "|" + query.ToBranch
	if result, ok := f.compare[key]; ok {
		return result, nil
	}
	return scm.CompareResult{}, nil
}

func (f fakeAdapter) RefExists(_ context.Context, repoRoot, ref string) (bool, error) {
	return f.existingRef[repoRoot][ref], nil
}

func TestInferRiskSignalsClassifiesKnownFiles(t *testing.T) {
	t.Parallel()

	signals := inferRiskSignals(map[string][]string{
		"a": {
			"db/migrations/001_add_column.sql",
			"config/application.yml",
			"mendix/app.mpr",
		},
	})

	if len(signals) != 3 {
		t.Fatalf("inferRiskSignals() returned %d signals, want 3", len(signals))
	}
}

func TestDeriveEnvironmentState(t *testing.T) {
	t.Parallel()

	statuses := []EnvironmentResult{
		{CommitCount: 2},
		{CommitCount: 1, MissingFromPrevious: 1},
		{CommitCount: 0, MissingFromPrevious: 1},
	}

	if got := deriveEnvironmentState(statuses, 0); got != EnvStatePresent {
		t.Fatalf("deriveEnvironmentState(first) = %q, want %q", got, EnvStatePresent)
	}
	if got := deriveEnvironmentState(statuses, 1); got != EnvStateBehind {
		t.Fatalf("deriveEnvironmentState(second) = %q, want %q", got, EnvStateBehind)
	}
	if got := deriveEnvironmentState(statuses, 2); got != EnvStateBehind {
		t.Fatalf("deriveEnvironmentState(third) = %q, want %q", got, EnvStateBehind)
	}
}

func TestEnvironmentStatusUsesCompareEvidenceWhenTargetHasNoTicketCommitMessage(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(`\b[A-Z][A-Z0-9]+-\d+\b`)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	repositories := []scm.Repository{
		{Name: "repo-a", Root: "/workspace/repo-a", Type: scm.TypeSVN},
	}

	service := NewService(
		fakeDiscoverer{repositories: repositories},
		fakeAdapterProvider{
			adapters: map[scm.Type]scm.Adapter{
				scm.TypeSVN: fakeAdapter{
					repoType: scm.TypeSVN,
					search: map[string]map[string][]scm.Commit{
						"/workspace/repo-a": {
							"": {
								{Hash: "r101", Subject: "ABC-123 initial fix", Branches: []string{"dev"}},
							},
							"dev": {
								{Hash: "r101", Subject: "ABC-123 initial fix", Branches: []string{"dev"}},
							},
							"test": {},
						},
					},
					compare: map[string]scm.CompareResult{
						"/workspace/repo-a|dev|test": {
							FromBranch: "dev",
							ToBranch:   "test",
							SourceCommits: []scm.Commit{
								{Hash: "r101", Subject: "ABC-123 initial fix", Branches: []string{"dev"}},
							},
							TargetCommits:  nil,
							MissingCommits: nil,
						},
					},
					existingRef: map[string]map[string]bool{
						"/workspace/repo-a": {
							"dev":  true,
							"test": true,
						},
					},
				},
			},
		},
		parser,
	)

	results, err := service.EnvironmentStatusInRepositories(context.Background(), repositories, "ABC-123", []Environment{
		{Name: "dev", Branch: "dev"},
		{Name: "test", Branch: "test"},
	})
	if err != nil {
		t.Fatalf("EnvironmentStatusInRepositories() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if len(results[0].Statuses) != 2 {
		t.Fatalf("len(statuses) = %d, want 2", len(results[0].Statuses))
	}
	if got := results[0].Statuses[1].State; got != EnvStateAligned {
		t.Fatalf("statuses[1].State = %q, want %q", got, EnvStateAligned)
	}
	if got := results[0].Statuses[1].CommitCount; got != 1 {
		t.Fatalf("statuses[1].CommitCount = %d, want 1", got)
	}
}
