package azuredevops

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"gig/internal/scm"
	"gig/internal/ticket"
)

func TestAdapterSearchCommitsAcrossDetectedBranches(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(`\b[A-Z][A-Z0-9]+-\d+\b`)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	adapter := NewAdapter(parser)
	adapter.run = fakeAzureRunner(map[string]string{
		"account show": `{}`,
		"account get-access-token --resource 499b84ac-1321-427f-aa17-267ca6975798 --query accessToken --output tsv": "token-123",
	})
	adapter.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path + "?" + request.URL.RawQuery {
		case "/acme/Payments/_apis/git/repositories/release-audit?api-version=7.1":
			return jsonResponse(`{"defaultBranch":"refs/heads/main"}`), nil
		case "/acme/Payments/_apis/git/repositories/release-audit/refs?filter=heads/&api-version=7.1":
			return jsonResponse(`{"count":4,"value":[{"name":"refs/heads/feature/test"},{"name":"refs/heads/staging"},{"name":"refs/heads/main"},{"name":"refs/heads/develop"}]}`), nil
		case "/acme/Payments/_apis/git/repositories/release-audit/commits?searchCriteria.$top=100&searchCriteria.itemVersion.versionType=branch&searchCriteria.itemVersion.version=main&api-version=7.1":
			return jsonResponse(`{"count":1,"value":[{"commitId":"abc123","comment":"ABC-123 fix payments"}]}`), nil
		case "/acme/Payments/_apis/git/repositories/release-audit/commits?searchCriteria.$top=100&searchCriteria.itemVersion.versionType=branch&searchCriteria.itemVersion.version=staging&api-version=7.1":
			return jsonResponse(`{"count":2,"value":[{"commitId":"abc123","comment":"ABC-123 fix payments"},{"commitId":"zzz999","comment":"OPS-1 maintenance"}]}`), nil
		case "/acme/Payments/_apis/git/repositories/release-audit/commits?searchCriteria.$top=100&searchCriteria.itemVersion.versionType=branch&searchCriteria.itemVersion.version=develop&api-version=7.1":
			return jsonResponse(`{"count":1,"value":[{"commitId":"abc123","comment":"ABC-123 fix payments"}]}`), nil
		default:
			t.Fatalf("unexpected request %s?%s", request.URL.Path, request.URL.RawQuery)
			return nil, nil
		}
	})}

	commits, err := adapter.SearchCommits(context.Background(), "azure-devops:acme/Payments/release-audit", scm.SearchQuery{TicketID: "ABC-123"})
	if err != nil {
		t.Fatalf("SearchCommits() error = %v", err)
	}

	if len(commits) != 1 {
		t.Fatalf("len(commits) = %d, want 1", len(commits))
	}
	if len(commits[0].Branches) != 3 {
		t.Fatalf("branches = %#v, want commit to be seen in develop, staging, and main", commits[0].Branches)
	}
}

func TestAdapterCompareBranchesReturnsMissingCommits(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(`\b[A-Z][A-Z0-9]+-\d+\b`)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	adapter := NewAdapter(parser)
	adapter.run = fakeAzureRunner(map[string]string{
		"account show": `{}`,
		"account get-access-token --resource 499b84ac-1321-427f-aa17-267ca6975798 --query accessToken --output tsv": "token-123",
	})
	adapter.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path + "?" + request.URL.RawQuery {
		case "/acme/Payments/_apis/git/repositories/release-audit/refs?filter=heads/staging&api-version=7.1":
			return jsonResponse(`{"count":1,"value":[{"name":"refs/heads/staging"}]}`), nil
		case "/acme/Payments/_apis/git/repositories/release-audit/refs?filter=heads/main&api-version=7.1":
			return jsonResponse(`{"count":1,"value":[{"name":"refs/heads/main"}]}`), nil
		case "/acme/Payments/_apis/git/repositories/release-audit/commits?searchCriteria.$top=100&searchCriteria.itemVersion.versionType=branch&searchCriteria.itemVersion.version=staging&api-version=7.1":
			return jsonResponse(`{"count":1,"value":[{"commitId":"abc123","comment":"ABC-123 fix payments"}]}`), nil
		case "/acme/Payments/_apis/git/repositories/release-audit/commits?searchCriteria.$top=100&searchCriteria.itemVersion.versionType=branch&searchCriteria.itemVersion.version=main&api-version=7.1":
			return jsonResponse(`{"count":0,"value":[]}`), nil
		default:
			t.Fatalf("unexpected request %s?%s", request.URL.Path, request.URL.RawQuery)
			return nil, nil
		}
	})}

	result, err := adapter.CompareBranches(context.Background(), "azure-devops:acme/Payments/release-audit", scm.CompareQuery{
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
	adapter.run = fakeAzureRunner(map[string]string{
		"account show": `{}`,
		"account get-access-token --resource 499b84ac-1321-427f-aa17-267ca6975798 --query accessToken --output tsv": "token-123",
	})
	adapter.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path + "?" + request.URL.RawQuery {
		case "/acme/Payments/_apis/git/repositories/release-audit/pullrequests?searchCriteria.sourceRefName=refs%2Fheads%2Fstaging&searchCriteria.status=all&$top=100&api-version=7.1":
			return jsonResponse(`{"count":0,"value":[]}`), nil
		case "/acme/Payments/_apis/git/repositories/release-audit/commits/abc123/statuses?api-version=7.1":
			return jsonResponse(`{"count":2,"value":[{"state":"failed","targetUrl":"https://dev.azure.com/acme/Payments/_build/results?buildId=1","context":{"name":"build","genre":"ci"}},{"state":"succeeded","targetUrl":"https://dev.azure.com/acme/Payments/_build/results?buildId=2","context":{"name":"unit","genre":"ci"}}]}`), nil
		case "/acme/Payments/_apis/release/deployments?$top=100&api-version=7.1":
			return jsonResponse(`{"count":0,"value":[]}`), nil
		default:
			t.Fatalf("unexpected request %s?%s", request.URL.Path, request.URL.RawQuery)
			return nil, nil
		}
	})}

	evidence, err := adapter.ProviderEvidence(context.Background(), "azure-devops:acme/Payments/release-audit", scm.EvidenceQuery{
		TicketID: "ABC-123",
		Commits:  []scm.Commit{{Hash: "abc123", Branches: []string{"staging"}}},
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
	if evidence.Checks[1].Context != "unit" || evidence.Checks[1].State != "succeeded" {
		t.Fatalf("Checks[1] = %#v, want passing unit check", evidence.Checks[1])
	}
}

func TestAdapterProviderEvidenceIncludesLinkedWorkItems(t *testing.T) {
	t.Parallel()

	parser, err := ticket.NewParser(`\b[A-Z][A-Z0-9]+-\d+\b`)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	adapter := NewAdapter(parser)
	adapter.run = fakeAzureRunner(map[string]string{
		"account show": `{}`,
		"account get-access-token --resource 499b84ac-1321-427f-aa17-267ca6975798 --query accessToken --output tsv": "token-123",
	})
	adapter.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path + "?" + request.URL.RawQuery {
		case "/acme/Payments/_apis/git/repositories/release-audit/pullrequests?searchCriteria.sourceRefName=refs%2Fheads%2Fstaging&searchCriteria.status=all&$top=100&api-version=7.1":
			return jsonResponse(`{"count":1,"value":[{"pullRequestId":42,"title":"ABC-123 release","status":"completed","sourceRefName":"refs/heads/staging","targetRefName":"refs/heads/main"}]}`), nil
		case "/acme/Payments/_apis/git/repositories/release-audit/pullRequests/42/workitems?api-version=7.1":
			return jsonResponse(`{"count":2,"value":[{"id":"77"},{"id":"88"}]}`), nil
		case "/acme/Payments/_apis/wit/workitems?ids=77%2C88&fields=System.Title%2CSystem.State%2CSystem.Tags&api-version=7.1":
			return jsonResponse(`{"count":2,"value":[{"id":77,"fields":{"System.Title":"Prod validation","System.State":"Active","System.Tags":"release; sev1"}},{"id":88,"fields":{"System.Title":"Audit follow-up","System.State":"Closed","System.Tags":"release"}}]}`), nil
		case "/acme/Payments/_apis/git/repositories/release-audit/commits/abc123/statuses?api-version=7.1":
			return jsonResponse(`{"count":0,"value":[]}`), nil
		case "/acme/Payments/_apis/release/deployments?$top=100&api-version=7.1":
			return jsonResponse(`{"count":0,"value":[]}`), nil
		default:
			t.Fatalf("unexpected request %s?%s", request.URL.Path, request.URL.RawQuery)
			return nil, nil
		}
	})}

	evidence, err := adapter.ProviderEvidence(context.Background(), "azure-devops:acme/Payments/release-audit", scm.EvidenceQuery{
		TicketID: "ABC-123",
		Commits:  []scm.Commit{{Hash: "abc123", Branches: []string{"staging"}}},
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
	if evidence.Issues[0].ID != "#77" || evidence.Issues[0].State != "Active" {
		t.Fatalf("Issues[0] = %#v, want active linked work item metadata", evidence.Issues[0])
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func fakeAzureRunner(outputs map[string]string) commandRunner {
	return func(_ context.Context, args ...string) (string, error) {
		key := strings.Join(args, " ")
		output, ok := outputs[key]
		if !ok {
			return "", errors.New("unexpected az invocation: " + key)
		}
		return output, nil
	}
}
