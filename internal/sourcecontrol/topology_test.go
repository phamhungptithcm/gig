package sourcecontrol_test

import (
	"strings"
	"testing"

	sourcecontrol "gig/internal/sourcecontrol"
)

func TestInferProtectedBranchTopologyBuildsClearPromotionPath(t *testing.T) {
	t.Parallel()

	inference := sourcecontrol.InferProtectedBranchTopology([]string{"main", "feature/test", "staging", "develop"})
	if inference.Confidence != sourcecontrol.TopologyConfidenceHigh {
		t.Fatalf("Confidence = %s, want high", inference.Confidence)
	}
	if inference.FromBranch != "staging" || inference.ToBranch != "main" {
		t.Fatalf("promotion = %s -> %s, want staging -> main", inference.FromBranch, inference.ToBranch)
	}
	if got := inference.Environments; len(got) != 3 || got[0].Branch != "develop" || got[1].Branch != "staging" || got[2].Branch != "main" {
		t.Fatalf("environments = %#v, want develop -> staging -> main", got)
	}
}

func TestInferProtectedBranchTopologyRejectsUnknownProtectedBranches(t *testing.T) {
	t.Parallel()

	inference := sourcecontrol.InferProtectedBranchTopology([]string{"integration/payments", "main"})
	if inference.Confidence != sourcecontrol.TopologyConfidenceLow {
		t.Fatalf("Confidence = %s, want low", inference.Confidence)
	}
	if !strings.Contains(strings.Join(inference.Reasons, " "), "Unrecognized protected branches") {
		t.Fatalf("Reasons = %#v, want unknown branch warning", inference.Reasons)
	}
}

func TestInferProtectedBranchTopologyFlagsAmbiguousReleaseCandidates(t *testing.T) {
	t.Parallel()

	inference := sourcecontrol.InferProtectedBranchTopology([]string{"release/2026.04", "rc/2026.04", "main"})
	if inference.Confidence != sourcecontrol.TopologyConfidenceMedium {
		t.Fatalf("Confidence = %s, want medium", inference.Confidence)
	}
	if !strings.Contains(strings.Join(inference.Reasons, " "), "last pre-production stage") {
		t.Fatalf("Reasons = %#v, want ambiguous predecessor warning", inference.Reasons)
	}
}

func TestInferProtectedBranchTopologyTreatsTrunkAsProduction(t *testing.T) {
	t.Parallel()

	inference := sourcecontrol.InferProtectedBranchTopology([]string{"staging", "trunk"})
	if inference.Confidence != sourcecontrol.TopologyConfidenceHigh {
		t.Fatalf("Confidence = %s, want high", inference.Confidence)
	}
	if inference.ToBranch != "trunk" {
		t.Fatalf("ToBranch = %q, want trunk", inference.ToBranch)
	}
}

func TestInferPromotionBranchesUsesPreviousProtectedBranch(t *testing.T) {
	t.Parallel()

	environments := sourcecontrol.InferEnvironments([]string{"develop", "staging", "main"})
	fromBranch, toBranch, err := sourcecontrol.InferPromotionBranches(environments, "", "")
	if err != nil {
		t.Fatalf("InferPromotionBranches() error = %v", err)
	}

	if fromBranch != "staging" || toBranch != "main" {
		t.Fatalf("branches = %s -> %s, want staging -> main", fromBranch, toBranch)
	}
}
