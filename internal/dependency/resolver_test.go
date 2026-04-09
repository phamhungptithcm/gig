package dependency_test

import (
	"context"
	"reflect"
	"testing"

	"gig/internal/config"
	"gig/internal/dependency"
	"gig/internal/scm"
	"gig/internal/ticket"
)

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
	existingRef    map[string]map[string]bool
}

func (f fakeAdapter) Type() scm.Type                                        { return f.repoType }
func (f fakeAdapter) DetectRoot(string) (string, bool, error)               { return "", false, nil }
func (f fakeAdapter) IsRepository(string) (bool, error)                     { return false, nil }
func (f fakeAdapter) CurrentBranch(context.Context, string) (string, error) { return "", nil }
func (f fakeAdapter) PrepareCherryPick(context.Context, string, scm.CherryPickPlan) error {
	return scm.ErrUnsupported
}
func (f fakeAdapter) CompareBranches(context.Context, string, scm.CompareQuery) (scm.CompareResult, error) {
	return scm.CompareResult{}, nil
}
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
func (f fakeAdapter) RefExists(_ context.Context, repoRoot, ref string) (bool, error) {
	return f.existingRef[repoRoot][ref], nil
}

func TestResolverResolveInRepositoriesMarksMissingTarget(t *testing.T) {
	t.Parallel()

	resolver := newResolver(t, fakeAdapterProvider{
		adapters: map[scm.Type]scm.Adapter{
			scm.TypeGit: fakeAdapter{
				repoType: scm.TypeGit,
				searchByTicket: map[string]map[string]map[string][]scm.Commit{
					"/workspace/repo-a": {
						"XYZ-456": {
							"test": {{Hash: "dep1", Subject: "XYZ-456 prerequisite"}},
							"main": {},
						},
					},
				},
				existingRef: map[string]map[string]bool{
					"/workspace/repo-a": {"test": true, "main": true},
				},
			},
		},
	})

	got, err := resolver.ResolveInRepositories(context.Background(), []scm.Repository{
		{Name: "repo-a", Root: "/workspace/repo-a", Type: scm.TypeGit},
	}, []dependency.DeclaredDependency{
		{TicketID: "ABC-123", DependsOn: "XYZ-456", CommitHash: "abc12345"},
	}, "test", "main")
	if err != nil {
		t.Fatalf("ResolveInRepositories() error = %v", err)
	}

	want := []dependency.Resolution{
		{
			TicketID:        "ABC-123",
			DependsOn:       "XYZ-456",
			Status:          dependency.StatusMissingTarget,
			FoundInSource:   true,
			FoundInTarget:   false,
			EvidenceCommits: []string{"abc12345"},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveInRepositories() = %#v, want %#v", got, want)
	}
}

func TestResolverResolveInRepositoriesMarksUnresolvedAndSatisfied(t *testing.T) {
	t.Parallel()

	resolver := newResolver(t, fakeAdapterProvider{
		adapters: map[scm.Type]scm.Adapter{
			scm.TypeGit: fakeAdapter{
				repoType: scm.TypeGit,
				searchByTicket: map[string]map[string]map[string][]scm.Commit{
					"/workspace/repo-a": {
						"OPS-99": {
							"test": {{Hash: "dep1", Subject: "OPS-99 prerequisite"}},
							"main": {{Hash: "dep1", Subject: "OPS-99 prerequisite"}},
						},
					},
				},
				existingRef: map[string]map[string]bool{
					"/workspace/repo-a": {"test": true, "main": true},
				},
			},
		},
	})

	got, err := resolver.ResolveInRepositories(context.Background(), []scm.Repository{
		{Name: "repo-a", Root: "/workspace/repo-a", Type: scm.TypeGit},
	}, []dependency.DeclaredDependency{
		{TicketID: "ABC-123", DependsOn: "OPS-99", CommitHash: "abc12345"},
		{TicketID: "ABC-123", DependsOn: "DB-7", CommitHash: "abc12345"},
		{TicketID: "ABC-123", DependsOn: "DB-7", CommitHash: "abc12345"},
	}, "test", "main")
	if err != nil {
		t.Fatalf("ResolveInRepositories() error = %v", err)
	}

	want := []dependency.Resolution{
		{
			TicketID:        "ABC-123",
			DependsOn:       "DB-7",
			Status:          dependency.StatusUnresolved,
			FoundInSource:   false,
			FoundInTarget:   false,
			EvidenceCommits: []string{"abc12345"},
		},
		{
			TicketID:        "ABC-123",
			DependsOn:       "OPS-99",
			Status:          dependency.StatusSatisfied,
			FoundInSource:   true,
			FoundInTarget:   true,
			EvidenceCommits: []string{"abc12345"},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveInRepositories() = %#v, want %#v", got, want)
	}
}

func newResolver(t *testing.T, adapters fakeAdapterProvider) *dependency.Resolver {
	t.Helper()

	tickets, err := ticket.NewParser(config.Default().TicketPattern)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	return dependency.NewResolver(adapters, tickets)
}
