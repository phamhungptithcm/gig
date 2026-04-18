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

func TestAdapterProviderEvidenceIncludesCheckStatuses(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(`\b[A-Z][A-Z0-9]+-\d+\b`)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	adapter := NewAdapter(parser)
	adapter.run = fakeGitHubRunner(map[string]string{
		"repos/acme/payments/commits/abc123456789/pulls?per_page=100&page=1":   `[]`,
		"repos/acme/payments/deployments?sha=abc123456789&per_page=100&page=1": `[]`,
		"repos/acme/payments/commits/abc123456789/status":                      `{"statuses":[{"context":"build","state":"failure","target_url":"https://ci.example.com/build/1"},{"context":"unit","state":"success","target_url":"https://ci.example.com/unit/1"}]}`,
		"repos/acme/payments/releases?per_page=1&page=1":                       `[]`,
	})

	evidence, err := adapter.ProviderEvidence(context.Background(), "github:acme/payments", scm.EvidenceQuery{
		TicketID: "ABC-123",
		Commits:  []scm.Commit{{Hash: "abc123456789"}},
	})
	if err != nil {
		t.Fatalf("ProviderEvidence() error = %v", err)
	}

	if len(evidence.Checks) != 2 {
		t.Fatalf("len(Checks) = %d, want 2", len(evidence.Checks))
	}
	if evidence.Checks[0].Context != "build" || evidence.Checks[0].State != "failure" {
		t.Fatalf("Checks[0] = %#v, want failing build check", evidence.Checks[0])
	}
	if evidence.Checks[1].Context != "unit" || evidence.Checks[1].State != "success" {
		t.Fatalf("Checks[1] = %#v, want passing unit check", evidence.Checks[1])
	}
}

func TestAdapterProviderEvidenceIncludesLinkedIssuesAndLatestRelease(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(`\b[A-Z][A-Z0-9]+-\d+\b`)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	adapter := NewAdapter(parser)
	adapter.run = fakeGitHubRunner(map[string]string{
		"repos/acme/payments/commits/abc123456789/pulls?per_page=100&page=1":   `[{"number":42,"title":"ABC-123 payments release","body":"Fixes #77\nRelates to #88","state":"closed","merged_at":"2026-04-10T01:02:03Z","html_url":"https://github.com/acme/payments/pull/42","head":{"ref":"staging"},"base":{"ref":"main"}}]`,
		"repos/acme/payments/issues/77":                                        `{"number":77,"title":"Prod validation","state":"open","html_url":"https://github.com/acme/payments/issues/77","labels":[{"name":"sev1"},{"name":"release"}]}`,
		"repos/acme/payments/issues/88":                                        `{"number":88,"title":"Audit follow-up","state":"closed","html_url":"https://github.com/acme/payments/issues/88","labels":[{"name":"release"}]}`,
		"repos/acme/payments/deployments?sha=abc123456789&per_page=100&page=1": `[]`,
		"repos/acme/payments/commits/abc123456789/status":                      `{"statuses":[]}`,
		"repos/acme/payments/releases?per_page=1&page=1":                       `[{"id":99,"tag_name":"v1.9.0","name":"Release v1.9.0","body":"Included ABC-123 and XYZ-999","html_url":"https://github.com/acme/payments/releases/tag/v1.9.0","target_commitish":"main","published_at":"2026-04-08T10:00:00Z"}]`,
	})

	evidence, err := adapter.ProviderEvidence(context.Background(), "github:acme/payments", scm.EvidenceQuery{
		TicketID: "ABC-123",
		Commits:  []scm.Commit{{Hash: "abc123456789"}},
	})
	if err != nil {
		t.Fatalf("ProviderEvidence() error = %v", err)
	}

	if len(evidence.PullRequests) != 1 {
		t.Fatalf("len(PullRequests) = %d, want 1", len(evidence.PullRequests))
	}
	if len(evidence.PullRequests[0].LinkedIssues) != 2 {
		t.Fatalf("len(LinkedIssues) = %d, want 2", len(evidence.PullRequests[0].LinkedIssues))
	}
	if len(evidence.Issues) != 2 {
		t.Fatalf("len(Issues) = %d, want 2", len(evidence.Issues))
	}
	if evidence.Issues[0].ID != "#77" || evidence.Issues[0].State != "open" {
		t.Fatalf("Issues[0] = %#v, want open linked issue metadata", evidence.Issues[0])
	}
	if len(evidence.Releases) != 1 {
		t.Fatalf("len(Releases) = %d, want 1", len(evidence.Releases))
	}
	if evidence.Releases[0].Tag != "v1.9.0" {
		t.Fatalf("Releases[0] = %#v, want latest release tag", evidence.Releases[0])
	}
	if len(evidence.Releases[0].TicketIDs) != 2 {
		t.Fatalf("Releases[0].TicketIDs = %#v, want parsed ticket ids", evidence.Releases[0].TicketIDs)
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
