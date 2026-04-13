package cli

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBootstrapDeerFlowCopiesConfigAndReportsNextCommand(t *testing.T) {
	root := t.TempDir()
	deerFlowRoot := filepath.Join(root, "deer-flow")

	mustMkdir(t, filepath.Join(deerFlowRoot, "backend"))
	mustMkdir(t, filepath.Join(deerFlowRoot, "frontend"))
	mustWrite(t, filepath.Join(deerFlowRoot, "Makefile"), "config:\n\t@echo ok\n")
	mustWrite(t, filepath.Join(deerFlowRoot, "config.example.yaml"), "models:\n#  - name: gpt-5\n#    api_key: $OPENAI_API_KEY\n")
	mustWrite(t, filepath.Join(deerFlowRoot, ".env.example"), "OPENAI_API_KEY=\n")
	mustWrite(t, filepath.Join(deerFlowRoot, "frontend", ".env.example"), "NEXT_PUBLIC_DEERFLOW_URL=http://localhost:2026\n")

	result, err := bootstrapDeerFlow(root, deerFlowSetupDeps{
		lookPath: func(file string) (string, error) {
			switch file {
			case "make", "docker":
				return "/usr/bin/" + file, nil
			}
			return "", errors.New("not found")
		},
	})
	if err != nil {
		t.Fatalf("bootstrapDeerFlow() error = %v", err)
	}

	if result.Root != deerFlowRoot {
		t.Fatalf("result.Root = %q, want %q", result.Root, deerFlowRoot)
	}
	if result.ConfigPath != filepath.Join(deerFlowRoot, "config.yaml") {
		t.Fatalf("result.ConfigPath = %q, want config.yaml", result.ConfigPath)
	}
	if len(result.CreatedFiles) != 3 {
		t.Fatalf("len(result.CreatedFiles) = %d, want 3", len(result.CreatedFiles))
	}
	if !result.DockerAvailable {
		t.Fatalf("result.DockerAvailable = false, want true")
	}
	if !strings.Contains(result.RecommendedStart, "make docker-start") {
		t.Fatalf("result.RecommendedStart = %q, want docker start command", result.RecommendedStart)
	}
	if len(result.Remaining) == 0 || !strings.Contains(result.Remaining[0], "add at least one model") {
		t.Fatalf("result.Remaining = %#v, want model reminder", result.Remaining)
	}

	if _, err := os.Stat(filepath.Join(deerFlowRoot, ".env")); err != nil {
		t.Fatalf("expected .env to be created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(deerFlowRoot, "frontend", ".env")); err != nil {
		t.Fatalf("expected frontend/.env to be created: %v", err)
	}
}

func TestBootstrapDeerFlowSkipsMissingOptionalEnvExamples(t *testing.T) {
	root := t.TempDir()
	deerFlowRoot := filepath.Join(root, "deer-flow")

	mustMkdir(t, filepath.Join(deerFlowRoot, "backend"))
	mustMkdir(t, filepath.Join(deerFlowRoot, "frontend"))
	mustWrite(t, filepath.Join(deerFlowRoot, "Makefile"), "config:\n\t@echo ok\n")
	mustWrite(t, filepath.Join(deerFlowRoot, "config.example.yaml"), "models:\n#  - name: gpt-5\n")
	mustWrite(t, filepath.Join(deerFlowRoot, "frontend", ".env.example"), "NEXT_PUBLIC_DEERFLOW_URL=http://localhost:2026\n")

	result, err := bootstrapDeerFlow(root, deerFlowSetupDeps{
		lookPath: func(file string) (string, error) {
			if file == "make" {
				return "/usr/bin/make", nil
			}
			return "", errors.New("not found")
		},
	})
	if err != nil {
		t.Fatalf("bootstrapDeerFlow() error = %v", err)
	}

	if len(result.CreatedFiles) != 2 {
		t.Fatalf("len(result.CreatedFiles) = %d, want 2", len(result.CreatedFiles))
	}
	if result.DockerAvailable {
		t.Fatalf("result.DockerAvailable = true, want false")
	}
	if result.RecommendedStart != "" {
		t.Fatalf("result.RecommendedStart = %q, want empty without a startable toolchain", result.RecommendedStart)
	}
}

func TestInspectDeerFlowReady(t *testing.T) {
	root := t.TempDir()
	deerFlowRoot := filepath.Join(root, "deer-flow")

	mustMkdir(t, filepath.Join(deerFlowRoot, "backend"))
	mustMkdir(t, filepath.Join(deerFlowRoot, "frontend"))
	mustWrite(t, filepath.Join(deerFlowRoot, "Makefile"), "dev:\n\t@echo ok\n")
	mustWrite(t, filepath.Join(deerFlowRoot, "config.example.yaml"), "models:\n#  - name: gpt-5\n")
	mustWrite(t, filepath.Join(deerFlowRoot, "config.yaml"), "models:\n  - name: gpt-5\n    api_key: $OPENAI_API_KEY\n")
	mustWrite(t, filepath.Join(deerFlowRoot, ".env"), "OPENAI_API_KEY=secret\n")
	mustWrite(t, filepath.Join(deerFlowRoot, "frontend", ".env"), "NEXT_PUBLIC_DEERFLOW_URL=http://localhost:2026\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()

	result, err := inspectDeerFlow(context.Background(), root, server.URL, deerFlowDoctorDeps{
		lookPath: func(file string) (string, error) {
			switch file {
			case "make", "docker":
				return "/usr/bin/" + file, nil
			default:
				return "", errors.New("not found")
			}
		},
		getenv: func(string) string { return "" },
		healthCheck: func(ctx context.Context, url string) error {
			return checkDeerFlowHealth(ctx, url)
		},
	})
	if err != nil {
		t.Fatalf("inspectDeerFlow() error = %v", err)
	}

	if result.Readiness != "ready" {
		t.Fatalf("result.Readiness = %q, want ready", result.Readiness)
	}
	if !result.GatewayHealthy {
		t.Fatalf("result.GatewayHealthy = false, want true")
	}
	if !result.ModelConfigured {
		t.Fatalf("result.ModelConfigured = false, want true")
	}
	if len(result.Credentials) != 1 || !result.Credentials[0].Present {
		t.Fatalf("result.Credentials = %#v, want present credential", result.Credentials)
	}
	if !strings.Contains(result.RecommendedStart, "make docker-start") {
		t.Fatalf("result.RecommendedStart = %q, want docker start command", result.RecommendedStart)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
