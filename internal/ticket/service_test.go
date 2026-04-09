package ticket_test

import (
	"context"
	"reflect"
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
	repoType      scm.Type
	searchResults map[string][]scm.Commit
}

func (f fakeAdapter) Type() scm.Type { return f.repoType }

func (f fakeAdapter) DetectRoot(string) (string, bool, error) { return "", false, nil }

func (f fakeAdapter) IsRepository(string) (bool, error) { return false, nil }

func (f fakeAdapter) CurrentBranch(context.Context, string) (string, error) { return "", nil }

func (f fakeAdapter) SearchCommits(_ context.Context, repoRoot string, _ scm.SearchQuery) ([]scm.Commit, error) {
	return f.searchResults[repoRoot], nil
}

func (f fakeAdapter) CompareBranches(context.Context, string, scm.CompareQuery) (scm.CompareResult, error) {
	return scm.CompareResult{}, scm.ErrUnsupported
}

func (f fakeAdapter) PrepareCherryPick(context.Context, string, scm.CherryPickPlan) error {
	return scm.ErrUnsupported
}

func TestServiceFindInRepositoriesFiltersEmptyResultsAndSortsByRepo(t *testing.T) {
	t.Parallel()

	parser := mustParser(t)
	service := ticket.NewService(
		fakeDiscoverer{
			repositories: []scm.Repository{
				{Name: "repo-b", Root: "/workspace/repo-b", Type: scm.TypeGit},
				{Name: "repo-a", Root: "/workspace/repo-a", Type: scm.TypeGit},
			},
		},
		fakeAdapterProvider{
			adapters: map[scm.Type]scm.Adapter{
				scm.TypeGit: fakeAdapter{
					repoType: scm.TypeGit,
					searchResults: map[string][]scm.Commit{
						"/workspace/repo-b": {
							{Hash: "bbb22222", Subject: "ABC-123 adjust validation"},
						},
						"/workspace/repo-a": nil,
					},
				},
			},
		},
		parser,
	)

	results, err := service.FindInRepositories(context.Background(), []scm.Repository{
		{Name: "repo-b", Root: "/workspace/repo-b", Type: scm.TypeGit},
		{Name: "repo-a", Root: "/workspace/repo-a", Type: scm.TypeGit},
	}, "ABC-123")
	if err != nil {
		t.Fatalf("FindInRepositories() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("FindInRepositories() returned %d results, want 1", len(results))
	}

	if results[0].Repository.Root != "/workspace/repo-b" {
		t.Fatalf("results[0].Repository.Root = %q, want %q", results[0].Repository.Root, "/workspace/repo-b")
	}

	wantCommits := []scm.Commit{
		{Hash: "bbb22222", Subject: "ABC-123 adjust validation"},
	}
	if !reflect.DeepEqual(results[0].Commits, wantCommits) {
		t.Fatalf("results[0].Commits = %#v, want %#v", results[0].Commits, wantCommits)
	}
}

func TestServiceFindRejectsInvalidTicketID(t *testing.T) {
	t.Parallel()

	parser := mustParser(t)
	service := ticket.NewService(fakeDiscoverer{}, fakeAdapterProvider{}, parser)

	if _, err := service.FindInRepositories(context.Background(), nil, "bad ticket"); err == nil {
		t.Fatal("FindInRepositories() expected validation error")
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
