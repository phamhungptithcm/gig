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

func TestAdapterProviderEvidenceIncludesCheckStatuses(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(`\b[A-Z][A-Z0-9]+-\d+\b`)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	adapter := NewAdapter(parser)
	adapter.run = fakeGitLabRunner(map[string]string{
		"projects/acme%2Fpayments/repository/commits/abc123456789/merge_requests":               `[]`,
		"projects/acme%2Fpayments/repository/commits/abc123456789/statuses?per_page=100&page=1": `[{"name":"build","status":"failed","target_url":"https://ci.example.com/build/1"},{"name":"unit","status":"success","target_url":"https://ci.example.com/unit/1"}]`,
		"projects/acme%2Fpayments/deployments?per_page=100&page=1":                              `[]`,
		"projects/acme%2Fpayments/releases?per_page=1&page=1":                                   `[]`,
	})

	evidence, err := adapter.ProviderEvidence(context.Background(), "gitlab:acme/payments", scm.EvidenceQuery{
		TicketID: "ABC-123",
		Commits:  []scm.Commit{{Hash: "abc123456789"}},
	})
	if err != nil {
		t.Fatalf("ProviderEvidence() error = %v", err)
	}

	if len(evidence.Checks) != 2 {
		t.Fatalf("len(Checks) = %d, want 2", len(evidence.Checks))
	}
	if evidence.Checks[0].Context != "build" || evidence.Checks[0].State != "failed" {
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
	adapter.run = fakeGitLabRunner(map[string]string{
		"projects/acme%2Fpayments/repository/commits/abc123456789/merge_requests":               `[{"iid":42,"title":"ABC-123 payments release","description":"Closes #77\nRelates to #88","state":"merged","web_url":"https://gitlab.example.com/acme/payments/-/merge_requests/42","source_branch":"staging","target_branch":"main","merged_at":"2026-04-10T01:02:03Z"}]`,
		"projects/acme%2Fpayments/issues/77":                                                    `{"iid":77,"title":"Prod validation","state":"opened","web_url":"https://gitlab.example.com/acme/payments/-/issues/77","labels":["release","sev1"]}`,
		"projects/acme%2Fpayments/issues/88":                                                    `{"iid":88,"title":"Audit follow-up","state":"closed","web_url":"https://gitlab.example.com/acme/payments/-/issues/88","labels":["release"]}`,
		"projects/acme%2Fpayments/repository/commits/abc123456789/statuses?per_page=100&page=1": `[]`,
		"projects/acme%2Fpayments/deployments?per_page=100&page=1":                              `[]`,
		"projects/acme%2Fpayments/releases?per_page=1&page=1":                                   `[{"tag_name":"v1.9.0","name":"Release v1.9.0","description":"Included ABC-123 and XYZ-999","released_at":"2026-04-08T10:00:00Z","commit":{"id":"main"},"_links":{"self":"https://gitlab.example.com/acme/payments/-/releases/v1.9.0"}}]`,
	})

	evidence, err := adapter.ProviderEvidence(context.Background(), "gitlab:acme/payments", scm.EvidenceQuery{
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
	if evidence.Issues[0].ID != "#77" || evidence.Issues[0].State != "opened" {
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
