package tools

import (
	"context"
	"testing"
)

func TestBridgeExecuteInspectTool(t *testing.T) {
	t.Parallel()

	var captured InspectRequest
	bridge := NewBridge(Runtime{
		Inspect: func(_ context.Context, request InspectRequest) (any, error) {
			captured = request
			return map[string]any{"ok": true}, nil
		},
	})

	result, err := bridge.Execute(context.Background(), Call{
		Name: inspectToolName,
		Arguments: map[string]any{
			"focus":      "risks",
			"repository": "payments",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if captured.Focus != "risks" || captured.Repository != "payments" {
		t.Fatalf("captured request = %#v", captured)
	}
	if result.Name != inspectToolName {
		t.Fatalf("result.Name = %q, want %q", result.Name, inspectToolName)
	}
}

func TestBridgeExecuteVerifyTool(t *testing.T) {
	t.Parallel()

	var captured VerifyRequest
	bridge := NewBridge(Runtime{
		Verify: func(_ context.Context, request VerifyRequest) (any, error) {
			captured = request
			return map[string]any{"verdict": "blocked"}, nil
		},
	})

	result, err := bridge.Execute(context.Background(), Call{
		Name: verifyToolName,
		Arguments: map[string]any{
			"focus": "summary",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if captured.Focus != "summary" {
		t.Fatalf("captured request = %#v", captured)
	}
	if result.Name != verifyToolName {
		t.Fatalf("result.Name = %q, want %q", result.Name, verifyToolName)
	}
}

func TestBridgeExecuteManifestTool(t *testing.T) {
	t.Parallel()

	var captured ManifestRequest
	bridge := NewBridge(Runtime{
		Manifest: func(_ context.Context, request ManifestRequest) (any, error) {
			captured = request
			return map[string]any{"section": "qa"}, nil
		},
	})

	result, err := bridge.Execute(context.Background(), Call{
		Name: manifestToolName,
		Arguments: map[string]any{
			"audience": "qa",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if captured.Audience != "qa" {
		t.Fatalf("captured request = %#v", captured)
	}
	if result.Name != manifestToolName {
		t.Fatalf("result.Name = %q, want %q", result.Name, manifestToolName)
	}
}
