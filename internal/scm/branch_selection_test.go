package scm_test

import (
	"reflect"
	"testing"

	"gig/internal/scm"
)

func TestSelectRemoteAuditBranchesOrdersLikelyPromotionBranches(t *testing.T) {
	t.Parallel()

	branches := scm.SelectRemoteAuditBranches([]string{
		"feature/test",
		"refs/heads/staging",
		"refs/heads/main",
		"develop",
		"develop",
	}, "main")

	want := []string{"develop", "staging", "main"}
	if !reflect.DeepEqual(branches, want) {
		t.Fatalf("SelectRemoteAuditBranches() = %#v, want %#v", branches, want)
	}
}

func TestSelectRemoteAuditBranchesKeepsDefaultBranchFallback(t *testing.T) {
	t.Parallel()

	branches := scm.SelectRemoteAuditBranches([]string{"feature/payments"}, "main")

	want := []string{"main"}
	if !reflect.DeepEqual(branches, want) {
		t.Fatalf("SelectRemoteAuditBranches() = %#v, want %#v", branches, want)
	}
}
