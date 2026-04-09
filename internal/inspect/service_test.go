package inspect

import "testing"

func TestInferRiskSignalsClassifiesKnownFiles(t *testing.T) {
	t.Parallel()

	signals := inferRiskSignals(map[string][]string{
		"a": {
			"db/migrations/001_add_column.sql",
			"config/application.yml",
			"mendix/app.mpr",
		},
	})

	if len(signals) != 3 {
		t.Fatalf("inferRiskSignals() returned %d signals, want 3", len(signals))
	}
}

func TestDeriveEnvironmentState(t *testing.T) {
	t.Parallel()

	statuses := []EnvironmentResult{
		{CommitCount: 2},
		{CommitCount: 1, MissingFromPrevious: 1},
		{CommitCount: 0, MissingFromPrevious: 1},
	}

	if got := deriveEnvironmentState(statuses, 0); got != EnvStatePresent {
		t.Fatalf("deriveEnvironmentState(first) = %q, want %q", got, EnvStatePresent)
	}
	if got := deriveEnvironmentState(statuses, 1); got != EnvStateBehind {
		t.Fatalf("deriveEnvironmentState(second) = %q, want %q", got, EnvStateBehind)
	}
	if got := deriveEnvironmentState(statuses, 2); got != EnvStateBehind {
		t.Fatalf("deriveEnvironmentState(third) = %q, want %q", got, EnvStateBehind)
	}
}
