package output

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	assistsvc "gig/internal/assistant"
)

func TestRenderAssistantReleaseGolden(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		audience assistsvc.Audience
		golden   string
	}{
		{name: "qa", audience: assistsvc.AudienceQA, golden: "assist_release_qa.golden"},
		{name: "client", audience: assistsvc.AudienceClient, golden: "assist_release_client.golden"},
		{name: "release-manager", audience: assistsvc.AudienceReleaseManager, golden: "assist_release_release_manager.golden"},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var buffer bytes.Buffer
			err := RenderAssistantRelease(&buffer, assistsvc.ReleaseResult{
				Audience: test.audience,
				Bundle: assistsvc.ReleaseBundle{
					ReleaseID:  "rel-2026-04-17",
					ScopeLabel: "github:acme/payments",
					FromBranch: "staging",
					ToBranch:   "main",
					EvidenceSummary: assistsvc.ReleaseEvidenceSummary{
						PullRequests:                 2,
						Deployments:                  1,
						Checks:                       3,
						FailingChecks:                1,
						LinkedIssues:                 2,
						Releases:                     1,
						OverlappingTickets:           1,
						NewTicketsSinceLatestRelease: 1,
						Hotspots:                     1,
					},
					ExecutiveSummary: []string{
						"Verdict blocked across 2 ticket(s): 1 blocked, 1 warning, 0 safe.",
						"1 linked issue(s) add delivery context for this release.",
					},
					OperatorSummary: []string{
						"github:acme/payments: 1 non-green check(s); latest release v1.9.0; new since latest release ABC-123.",
						"1 repository overlap(s) carry multiple release tickets in the same scope.",
					},
				},
				Response: "Release still blocked until the build check is green.",
			})
			if err != nil {
				t.Fatalf("RenderAssistantRelease() error = %v", err)
			}

			assertGolden(t, test.golden, buffer.String())
		})
	}
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("testdata", name)
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if got != string(want) {
		t.Fatalf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, string(want))
	}
}
