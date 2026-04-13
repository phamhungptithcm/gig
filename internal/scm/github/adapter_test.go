package github

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
	adapter.run = fakeGitHubRunner(map[string]string{
		"repos/acme/payments/branches?protected=true&per_page=100&page=1": `[{"name":"staging","protected":true},{"name":"main","protected":true}]`,
		"repos/acme/payments/commits?sha=staging&per_page=100&page=1":     `[{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments\n\nDepends-On: XYZ-456"}},{"sha":"def123456789","commit":{"message":"OPS-9 chore release"}}]`,
		"repos/acme/payments/commits?sha=main&per_page=100&page=1":        `[{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments\n\nDepends-On: XYZ-456"}}]`,
	})

	commits, err := adapter.SearchCommits(context.Background(), "github:acme/payments", scm.SearchQuery{TicketID: "ABC-123"})
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
	adapter.run = fakeGitHubRunner(map[string]string{
		"repos/acme/payments/branches/staging":                        `{"name":"staging","protected":true}`,
		"repos/acme/payments/branches/main":                           `{"name":"main","protected":true}`,
		"repos/acme/payments/commits?sha=staging&per_page=100&page=1": `[{"sha":"abc123456789","commit":{"message":"ABC-123 fix payments"}}]`,
		"repos/acme/payments/commits?sha=main&per_page=100&page=1":    `[]`,
	})

	result, err := adapter.CompareBranches(context.Background(), "github:acme/payments", scm.CompareQuery{
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

func fakeGitHubRunner(outputs map[string]string) commandRunner {
	return func(_ context.Context, args ...string) (string, error) {
		if len(args) == 0 {
			return "", fmt.Errorf("missing gh arguments")
		}
		if args[0] != "api" {
			return "", fmt.Errorf("unsupported gh command %v", args)
		}
		endpoint := args[len(args)-1]
		output, ok := outputs[endpoint]
		if !ok {
			return "", fmt.Errorf("unexpected endpoint %s", endpoint)
		}
		return output, nil
	}
}
