package diff_test

import (
	"context"
	"reflect"
	"testing"

	diffsvc "gig/internal/diff"
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
	compareResults map[string]scm.CompareResult
}

func (f fakeAdapter) Type() scm.Type { return f.repoType }

func (f fakeAdapter) DetectRoot(string) (string, bool, error) { return "", false, nil }

func (f fakeAdapter) IsRepository(string) (bool, error) { return false, nil }

func (f fakeAdapter) CurrentBranch(context.Context, string) (string, error) { return "", nil }

func (f fakeAdapter) SearchCommits(context.Context, string, scm.SearchQuery) ([]scm.Commit, error) {
	return nil, scm.ErrUnsupported
}

func (f fakeAdapter) CompareBranches(_ context.Context, repoRoot string, _ scm.CompareQuery) (scm.CompareResult, error) {
	result, ok := f.compareResults[repoRoot]
	if !ok {
		return scm.CompareResult{}, nil
	}
	return result, nil
}

func (f fakeAdapter) PrepareCherryPick(context.Context, string, scm.CherryPickPlan) error {
	return scm.ErrUnsupported
}

func TestServiceCompareTicketFiltersRepositoriesWithoutTicketActivity(t *testing.T) {
	t.Parallel()

	parser := mustParser(t)
	service := diffsvc.NewService(
		fakeDiscoverer{
			repositories: []scm.Repository{
				{Name: "repo-a", Root: "/workspace/repo-a", Type: scm.TypeGit},
				{Name: "repo-b", Root: "/workspace/repo-b", Type: scm.TypeGit},
			},
		},
		fakeAdapterProvider{
			adapters: map[scm.Type]scm.Adapter{
				scm.TypeGit: fakeAdapter{
					repoType: scm.TypeGit,
					compareResults: map[string]scm.CompareResult{
						"/workspace/repo-a": {
							FromBranch: "dev",
							ToBranch:   "test",
							SourceCommits: []scm.Commit{
								{Hash: "abc12345", Subject: "ABC-123 fix login"},
							},
							MissingCommits: []scm.Commit{
								{Hash: "abc12345", Subject: "ABC-123 fix login"},
							},
						},
						"/workspace/repo-b": {},
					},
				},
			},
		},
		parser,
	)

	results, err := service.CompareTicket(context.Background(), ".", "ABC-123", "dev", "test")
	if err != nil {
		t.Fatalf("CompareTicket() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("CompareTicket() returned %d results, want 1", len(results))
	}
	if results[0].Repository.Root != "/workspace/repo-a" {
		t.Fatalf("results[0].Repository.Root = %q, want %q", results[0].Repository.Root, "/workspace/repo-a")
	}
}

func TestServiceCompareTicketReturnsMissingCommits(t *testing.T) {
	t.Parallel()

	parser := mustParser(t)
	wantMissing := []scm.Commit{
		{Hash: "abc12345", Subject: "ABC-123 fix login"},
		{Hash: "def67890", Subject: "ABC-123 adjust validation"},
	}

	service := diffsvc.NewService(
		fakeDiscoverer{
			repositories: []scm.Repository{
				{Name: "repo-a", Root: "/workspace/repo-a", Type: scm.TypeGit},
			},
		},
		fakeAdapterProvider{
			adapters: map[scm.Type]scm.Adapter{
				scm.TypeGit: fakeAdapter{
					repoType: scm.TypeGit,
					compareResults: map[string]scm.CompareResult{
						"/workspace/repo-a": {
							FromBranch: "dev",
							ToBranch:   "test",
							SourceCommits: []scm.Commit{
								{Hash: "abc12345", Subject: "ABC-123 fix login"},
								{Hash: "def67890", Subject: "ABC-123 adjust validation"},
							},
							TargetCommits: []scm.Commit{
								{Hash: "99999999", Subject: "ABC-123 previous fix"},
							},
							MissingCommits: wantMissing,
						},
					},
				},
			},
		},
		parser,
	)

	results, err := service.CompareTicket(context.Background(), ".", "ABC-123", "dev", "test")
	if err != nil {
		t.Fatalf("CompareTicket() error = %v", err)
	}

	if !reflect.DeepEqual(results[0].Compare.MissingCommits, wantMissing) {
		t.Fatalf("MissingCommits = %#v, want %#v", results[0].Compare.MissingCommits, wantMissing)
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
