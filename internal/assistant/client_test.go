package assistant

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	assisttools "gig/internal/assistant/tools"
	conflictsvc "gig/internal/conflict"
	"gig/internal/scm"
)

func TestBuildAuditPromptIncludesGuardrailsAndBundle(t *testing.T) {
	t.Parallel()

	prompt, err := buildAuditPrompt(AuditBundle{
		ScopeLabel: "github:acme/payments",
		TicketID:   "ABC-123",
		FromBranch: "staging",
		ToBranch:   "main",
		Hints: BundleHints{
			CommandTarget: "--repo github:acme/payments",
		},
	}, AudienceReleaseManager)
	if err != nil {
		t.Fatalf("buildAuditPrompt() error = %v", err)
	}

	if !strings.Contains(prompt, "Use only the facts in the JSON bundle below.") {
		t.Fatalf("prompt = %q, want guardrail text", prompt)
	}
	if !strings.Contains(prompt, "\"ticketId\": \"ABC-123\"") {
		t.Fatalf("prompt = %q, want ticket JSON", prompt)
	}
	if !strings.Contains(prompt, "## Recommended Next gig Commands") {
		t.Fatalf("prompt = %q, want output sections", prompt)
	}
}

func TestBuildReleasePromptUsesAudienceSections(t *testing.T) {
	t.Parallel()

	prompt, err := buildReleasePrompt(ReleaseBundle{
		ScopeLabel: "workspace",
		ReleaseID:  "rel-2026-04-09",
	}, AudienceQA)
	if err != nil {
		t.Fatalf("buildReleasePrompt() error = %v", err)
	}

	if !strings.Contains(prompt, "for the qa audience") {
		t.Fatalf("prompt = %q, want qa audience text", prompt)
	}
	if !strings.Contains(prompt, "## QA Release Summary") {
		t.Fatalf("prompt = %q, want QA release sections", prompt)
	}
	if !strings.Contains(prompt, "\"releaseId\": \"rel-2026-04-09\"") {
		t.Fatalf("prompt = %q, want release JSON", prompt)
	}
}

func TestBuildResolvePromptUsesReadableConflictBlock(t *testing.T) {
	t.Parallel()

	prompt, err := buildResolvePrompt(ResolveBundle{
		ScopeLabel: "workspace/a-service",
		Status:     conflictsStatusForTest(),
		ActiveConflict: &ResolveActiveConflict{
			File: conflictsFileForTest(),
			Block: ResolveBlock{
				Index:     0,
				StartLine: 1,
				EndLine:   5,
				Current:   "main line",
				Incoming:  "feature line",
			},
		},
		Hints: BundleHints{
			CommandTarget: "--path /tmp/workspace/a-service",
		},
	}, AudienceReleaseManager)
	if err != nil {
		t.Fatalf("buildResolvePrompt() error = %v", err)
	}

	if !strings.Contains(prompt, "Git conflict-resolution briefing") {
		t.Fatalf("prompt = %q, want resolve intro", prompt)
	}
	if !strings.Contains(prompt, "\"current\": \"main line\"") {
		t.Fatalf("prompt = %q, want readable current block", prompt)
	}
	if !strings.Contains(prompt, "## Active Conflict Recommendation") {
		t.Fatalf("prompt = %q, want resolve sections", prompt)
	}
}

func TestDeerFlowClientAnalyzeAudit(t *testing.T) {
	t.Parallel()

	var runRequestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/langgraph/threads":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"thread_id":"thread-123"}`))
		case "/api/langgraph/threads/thread-123/runs/stream":
			body, _ := io.ReadAll(r.Body)
			runRequestBody = string(body)
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: values\n"))
			_, _ = w.Write([]byte("data: {\"messages\":[{\"type\":\"human\",\"content\":\"ignored\"},{\"type\":\"ai\",\"content\":\"Blocked because main is missing one dependency.\"}]}\n\n"))
			_, _ = w.Write([]byte("event: end\n"))
			_, _ = w.Write([]byte("data: {}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewDeerFlowClient(ClientConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	response, err := client.AnalyzeAudit(context.Background(), AuditBundle{
		ScopeLabel: "github:acme/payments",
		TicketID:   "ABC-123",
		FromBranch: "staging",
		ToBranch:   "main",
	}, AnalyzeOptions{Mode: ModePro, Audience: AudienceClient})
	if err != nil {
		t.Fatalf("AnalyzeAudit() error = %v", err)
	}

	if response.ThreadID != "thread-123" {
		t.Fatalf("ThreadID = %q, want thread-123", response.ThreadID)
	}
	if response.Response != "Blocked because main is missing one dependency." {
		t.Fatalf("Response = %q", response.Response)
	}
	if !strings.Contains(runRequestBody, "\"assistant_id\":\"lead_agent\"") {
		t.Fatalf("run request = %q, want assistant id", runRequestBody)
	}
	if !strings.Contains(runRequestBody, "ABC-123") {
		t.Fatalf("run request = %q, want bundled ticket id", runRequestBody)
	}
	if !strings.Contains(runRequestBody, "for the client audience") {
		t.Fatalf("run request = %q, want audience prompt", runRequestBody)
	}
}

func TestDeerFlowClientAnalyzeRelease(t *testing.T) {
	t.Parallel()

	var runRequestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/langgraph/threads":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"thread_id":"thread-rel-123"}`))
		case "/api/langgraph/threads/thread-rel-123/runs/stream":
			body, _ := io.ReadAll(r.Body)
			runRequestBody = string(body)
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: values\n"))
			_, _ = w.Write([]byte("data: {\"messages\":[{\"type\":\"ai\",\"content\":\"Release remains blocked because one snapshot is blocked.\"}]}\n\n"))
			_, _ = w.Write([]byte("event: end\n"))
			_, _ = w.Write([]byte("data: {}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewDeerFlowClient(ClientConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	response, err := client.AnalyzeRelease(context.Background(), ReleaseBundle{
		ScopeLabel: "workspace",
		ReleaseID:  "rel-2026-04-09",
	}, AnalyzeOptions{Mode: ModeUltra, Audience: AudienceReleaseManager})
	if err != nil {
		t.Fatalf("AnalyzeRelease() error = %v", err)
	}

	if response.ThreadID != "thread-rel-123" {
		t.Fatalf("ThreadID = %q, want thread-rel-123", response.ThreadID)
	}
	if response.Response != "Release remains blocked because one snapshot is blocked." {
		t.Fatalf("Response = %q", response.Response)
	}
	if !strings.Contains(runRequestBody, "rel-2026-04-09") {
		t.Fatalf("run request = %q, want release id", runRequestBody)
	}
	if !strings.Contains(runRequestBody, "## Release Overview") {
		t.Fatalf("run request = %q, want release-manager sections", runRequestBody)
	}
}

func TestDeerFlowClientAnalyzeResolve(t *testing.T) {
	t.Parallel()

	var runRequestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/langgraph/threads":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"thread_id":"thread-resolve-123"}`))
		case "/api/langgraph/threads/thread-resolve-123/runs/stream":
			body, _ := io.ReadAll(r.Body)
			runRequestBody = string(body)
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: values\n"))
			_, _ = w.Write([]byte("data: {\"messages\":[{\"type\":\"ai\",\"content\":\"Accept incoming, then re-check validation paths before staging.\"}]}\n\n"))
			_, _ = w.Write([]byte("event: end\n"))
			_, _ = w.Write([]byte("data: {}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewDeerFlowClient(ClientConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	response, err := client.AnalyzeResolve(context.Background(), ResolveBundle{
		ScopeLabel: "workspace/a-service",
		Status:     conflictsStatusForTest(),
		ActiveConflict: &ResolveActiveConflict{
			File: conflictsFileForTest(),
			Block: ResolveBlock{
				Index:     0,
				StartLine: 1,
				EndLine:   5,
				Current:   "main line",
				Incoming:  "feature line",
			},
		},
	}, AnalyzeOptions{Mode: ModePro, Audience: AudienceQA})
	if err != nil {
		t.Fatalf("AnalyzeResolve() error = %v", err)
	}

	if response.ThreadID != "thread-resolve-123" {
		t.Fatalf("ThreadID = %q, want thread-resolve-123", response.ThreadID)
	}
	if response.Response != "Accept incoming, then re-check validation paths before staging." {
		t.Fatalf("Response = %q", response.Response)
	}
	if !strings.Contains(runRequestBody, "feature line") {
		t.Fatalf("run request = %q, want readable conflict content", runRequestBody)
	}
	if !strings.Contains(runRequestBody, "## QA Conflict Summary") {
		t.Fatalf("run request = %q, want QA resolve sections", runRequestBody)
	}
}

func TestDeerFlowClientAnalyzeFollowUpUsesGigToolBridge(t *testing.T) {
	t.Parallel()

	runBodies := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/langgraph/threads":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"thread_id":"thread-tools-123"}`))
		case "/api/langgraph/threads/thread-tools-123/runs/stream":
			body, _ := io.ReadAll(r.Body)
			runBodies = append(runBodies, string(body))
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: values\n"))
			if len(runBodies) == 1 {
				_, _ = w.Write([]byte("data: {\"messages\":[{\"type\":\"ai\",\"content\":\"\",\"tool_calls\":[{\"name\":\"gig_verify\",\"args\":{\"focus\":\"summary\"},\"id\":\"tc-1\"}]}]}\n\n"))
			} else {
				_, _ = w.Write([]byte("data: {\"messages\":[{\"type\":\"ai\",\"content\":\"Biggest risk: one dependency is still missing from main.\"}]}\n\n"))
			}
			_, _ = w.Write([]byte("event: end\n"))
			_, _ = w.Write([]byte("data: {}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var captured assisttools.VerifyRequest
	bridge := assisttools.NewBridge(assisttools.Runtime{
		Verify: func(_ context.Context, request assisttools.VerifyRequest) (any, error) {
			captured = request
			return map[string]any{
				"verdict": "blocked",
				"reasons": []string{
					"Dependency XYZ-456 is still missing from main.",
				},
			}, nil
		},
	})

	client := NewDeerFlowClient(ClientConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	response, err := client.AnalyzeFollowUp(context.Background(), "what is the biggest risk?", PromptOptions{
		Mode:   ModePro,
		Bridge: bridge,
	})
	if err != nil {
		t.Fatalf("AnalyzeFollowUp() error = %v", err)
	}

	if captured.Focus != "summary" {
		t.Fatalf("captured tool request = %#v", captured)
	}
	if response.ThreadID != "thread-tools-123" {
		t.Fatalf("ThreadID = %q, want thread-tools-123", response.ThreadID)
	}
	if response.Response != "Biggest risk: one dependency is still missing from main." {
		t.Fatalf("Response = %q", response.Response)
	}
	if len(runBodies) != 2 {
		t.Fatalf("runBodies = %d, want 2 tool bridge turns", len(runBodies))
	}
	if !strings.Contains(runBodies[0], "read-only tool bridge managed by gig") {
		t.Fatalf("first run body = %q, want bridge instructions", runBodies[0])
	}
	if !strings.Contains(runBodies[1], "gig_verify") || !strings.Contains(runBodies[1], "blocked") {
		t.Fatalf("second run body = %q, want deterministic tool result", runBodies[1])
	}
}

func conflictsStatusForTest() conflictsvc.Status {
	return conflictsvc.Status{
		Repository: scm.Repository{
			Root:          "/tmp/workspace/a-service",
			CurrentBranch: "main",
		},
		Operation: scm.ConflictOperationState{
			Type: "merge",
			CurrentSide: scm.ConflictSide{
				Label:   "Current",
				Branch:  "main",
				Subject: "OPS-99 tighten validation",
			},
			IncomingSide: scm.ConflictSide{
				Label:   "Incoming",
				Branch:  "feature/ABC-123",
				Subject: "ABC-123 update app behavior",
			},
		},
		ResolvableFiles: 1,
		SuggestedNext:   "Run `gig resolve start --path /tmp/workspace/a-service` to walk the supported text conflicts.",
	}
}

func conflictsFileForTest() conflictsvc.FileStatus {
	return conflictsvc.FileStatus{
		Path:         "app.txt",
		ConflictCode: "UU",
		BlockCount:   1,
		Supported:    true,
	}
}
