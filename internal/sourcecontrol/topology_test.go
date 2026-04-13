package sourcecontrol_test

import (
	"testing"

	sourcecontrol "gig/internal/sourcecontrol"
)

func TestInferEnvironmentsOrdersProtectedBranches(t *testing.T) {
	t.Parallel()

	environments := sourcecontrol.InferEnvironments([]string{"main", "feature/test", "staging", "develop"})
	if len(environments) != 3 {
		t.Fatalf("len(environments) = %d, want 3", len(environments))
	}

	if environments[0].Branch != "develop" || environments[1].Branch != "staging" || environments[2].Branch != "main" {
		t.Fatalf("branches = %#v, want develop -> staging -> main", environments)
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
