package diagnostics

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmitAppendsStructuredEvent(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "gig-diagnostics.jsonl")
	logger := NewFromEnv(func(key string) (string, bool) {
		if key == envDiagnosticsFile {
			return filePath, true
		}
		return "", false
	})

	ctx := WithLogger(context.Background(), logger)
	Emit(ctx, "info", "topology.resolve", "promotion path resolved", Meta{
		Command:    "verify",
		Repo:       "github:acme/payments",
		SCM:        "github",
		FromBranch: "staging",
		ToBranch:   "main",
	}, nil)

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", filePath, err)
	}
	text := string(data)
	if !strings.Contains(text, `"operation":"topology.resolve"`) {
		t.Fatalf("diagnostics = %q, want operation field", text)
	}
	if !strings.Contains(text, `"command":"verify"`) || !strings.Contains(text, `"repo":"github:acme/payments"`) {
		t.Fatalf("diagnostics = %q, want metadata fields", text)
	}
}
