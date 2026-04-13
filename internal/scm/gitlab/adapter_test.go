package gitlab

import (
	"context"
	"fmt"
	"testing"

	"gig/internal/scm"
	"gig/internal/ticket"
)

func TestAdapterSearchCommitsAcrossProtectedBranches(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(`\b[A-Z][A-Z0-9]+-\d+\b`)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	adapter := NewAdapter(parser)
	adapter.run = fakeGitLabRunner(map[string]string{
		"projects/acme%2Fpayments/repository/branches?per_page=100&page=1":                 `[{"name":"staging","protected":true},{"name":"main","protected":true},{"name":"feature/test","protected":false}]`,
		"projects/acme%2Fpayments/repository/commits?ref_name=staging&per_page=100&page=1": `[{"id":"abc123456789","message":"ABC-123 fix payments\n\nDepends-On: XYZ-456"},{"id":"def123456789","message":"OPS-9 chore release"}]`,
		"projects/acme%2Fpayments/repository/commits?ref_name=main&per_page=100&page=1":    `[{"id":"abc123456789","message":"ABC-123 fix payments\n\nDepends-On: XYZ-456"}]`,
	})

	commits, err := adapter.SearchCommits(context.Background(), "gitlab:acme/payments", scm.SearchQuery{TicketID: "ABC-123"})
	if err != nil {
		t.Fatalf("SearchCommits() error = %v", err)
	}

	if len(commits) != 1 {
		t.Fatalf("len(commits) = %d, want 1", len(commits))
	}
	if len(commits[0].Branches) != 2 {
		t.Fatalf("branches = %#v, want commit to be seen in staging and main", commits[0].Branches)
	}
}

func TestAdapterCompareBranchesReturnsMissingCommits(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(`\b[A-Z][A-Z0-9]+-\d+\b`)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	adapter := NewAdapter(parser)
	adapter.run = fakeGitLabRunner(map[string]string{
		"projects/acme%2Fpayments/repository/branches/staging":                             `{"name":"staging","protected":true}`,
		"projects/acme%2Fpayments/repository/branches/main":                                `{"name":"main","protected":true}`,
		"projects/acme%2Fpayments/repository/commits?ref_name=staging&per_page=100&page=1": `[{"id":"abc123456789","message":"ABC-123 fix payments"}]`,
		"projects/acme%2Fpayments/repository/commits?ref_name=main&per_page=100&page=1":    `[]`,
	})

	result, err := adapter.CompareBranches(context.Background(), "gitlab:acme/payments", scm.CompareQuery{
		TicketID:   "ABC-123",
		FromBranch: "staging",
		ToBranch:   "main",
	})
	if err != nil {
		t.Fatalf("CompareBranches() error = %v", err)
	}

	if len(result.MissingCommits) != 1 {
		t.Fatalf("len(MissingCommits) = %d, want 1", len(result.MissingCommits))
	}
}

func TestAdapterProtectedBranchesFallsBackToDefaultBranch(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(`\b[A-Z][A-Z0-9]+-\d+\b`)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	adapter := NewAdapter(parser)
	adapter.run = fakeGitLabRunner(map[string]string{
		"projects/acme%2Fpayments/repository/branches?per_page=100&page=1": `[]`,
		"projects/acme%2Fpayments": `{"default_branch":"main"}`,
	})

	branches, err := adapter.ProtectedBranches(context.Background(), "gitlab:acme/payments")
	if err != nil {
		t.Fatalf("ProtectedBranches() error = %v", err)
	}

	if len(branches) != 1 || branches[0] != "main" {
		t.Fatalf("branches = %#v, want [main]", branches)
	}
}

func fakeGitLabRunner(outputs map[string]string) commandRunner {
	return func(_ context.Context, args ...string) (string, error) {
		if len(args) == 0 {
			return "", fmt.Errorf("missing glab arguments")
		}
		if args[0] != "api" {
			return "", fmt.Errorf("unsupported glab command %v", args)
		}
		endpoint := args[len(args)-1]
		output, ok := outputs[endpoint]
		if !ok {
			return "", fmt.Errorf("unexpected endpoint %s", endpoint)
		}
		return output, nil
	}
}
