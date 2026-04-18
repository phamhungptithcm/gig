package sourcecontrol_test

import (
	"testing"

	"gig/internal/scm"
	sourcecontrol "gig/internal/sourcecontrol"
)

func TestProviderCapabilitiesDescribeGitHubAsDeepEvidence(t *testing.T) {
	t.Parallel()

	capability := sourcecontrol.ProviderCapabilities(scm.TypeGitHub)
	if !capability.Checks || !capability.LinkedIssues || !capability.Releases {
		t.Fatalf("GitHub capability = %#v, want deep evidence flags", capability)
	}
	if capability.EvidenceTier() != "deep release evidence" {
		t.Fatalf("EvidenceTier() = %q, want deep release evidence", capability.EvidenceTier())
	}
}

func TestProviderCapabilitiesDescribeGitLabAndAzureAsDeepEvidence(t *testing.T) {
	t.Parallel()

	for _, provider := range []scm.Type{scm.TypeGitLab, scm.TypeAzureDevOps} {
		capability := sourcecontrol.ProviderCapabilities(provider)
		if !capability.Checks || !capability.LinkedIssues {
			t.Fatalf("%s capability = %#v, want check and linked issue evidence", provider, capability)
		}
		if capability.EvidenceTier() != "deep release evidence" {
			t.Fatalf("%s EvidenceTier() = %q, want deep release evidence", provider, capability.EvidenceTier())
		}
	}
}

func TestProviderCapabilitiesDescribeSVNAsAuditOnly(t *testing.T) {
	t.Parallel()

	capability := sourcecontrol.ProviderCapabilities(scm.TypeRemoteSVN)
	if capability.PullRequests || capability.Deployments || capability.Checks {
		t.Fatalf("SVN capability = %#v, want no release evidence flags", capability)
	}
	if capability.EvidenceTier() != "audit topology only" {
		t.Fatalf("EvidenceTier() = %q, want audit topology only", capability.EvidenceTier())
	}
}
