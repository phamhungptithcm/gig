package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	assisttools "gig/internal/assistant/tools"
)

const (
	defaultDeerFlowURL = "http://localhost:2026"
	leadAgentID        = "lead_agent"
	maxToolRoundTrips  = 3
)

type RunMode string

const (
	ModeFlash    RunMode = "flash"
	ModeStandard RunMode = "standard"
	ModePro      RunMode = "pro"
	ModeUltra    RunMode = "ultra"
)

type ClientConfig struct {
	BaseURL      string
	GatewayURL   string
	LangGraphURL string
	HTTPClient   *http.Client
}

type AnalyzeOptions struct {
	Mode     RunMode
	Audience Audience
	Bridge   *assisttools.Bridge
}

type PromptOptions struct {
	ThreadID string
	Mode     RunMode
	Bridge   *assisttools.Bridge
}

type AnalysisResponse struct {
	ThreadID string
	Response string
}

type DeerFlowClient struct {
	gatewayURL   string
	langGraphURL string
	httpClient   *http.Client
}

func NewDeerFlowClient(config ClientConfig) *DeerFlowClient {
	gatewayURL, langGraphURL := resolveClientURLs(config)
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 2 * time.Minute}
	}

	return &DeerFlowClient{
		gatewayURL:   gatewayURL,
		langGraphURL: langGraphURL,
		httpClient:   httpClient,
	}
}

func ParseRunMode(raw string) (RunMode, error) {
	switch RunMode(strings.ToLower(strings.TrimSpace(raw))) {
	case ModeFlash:
		return ModeFlash, nil
	case ModeStandard:
		return ModeStandard, nil
	case ModePro, "":
		return ModePro, nil
	case ModeUltra:
		return ModeUltra, nil
	default:
		return "", fmt.Errorf("unsupported mode %q", raw)
	}
}

func defaultRunMode(mode RunMode) RunMode {
	if mode == "" {
		return ModePro
	}
	return mode
}

func (c *DeerFlowClient) AnalyzeAudit(ctx context.Context, bundle AuditBundle, options AnalyzeOptions) (AnalysisResponse, error) {
	prompt, err := buildAuditPrompt(bundle, defaultAudience(options.Audience))
	if err != nil {
		return AnalysisResponse{}, err
	}
	return c.analyzePrompt(ctx, prompt, PromptOptions{Mode: options.Mode, Bridge: options.Bridge})
}

func (c *DeerFlowClient) AnalyzeRelease(ctx context.Context, bundle ReleaseBundle, options AnalyzeOptions) (AnalysisResponse, error) {
	prompt, err := buildReleasePrompt(bundle, defaultAudience(options.Audience))
	if err != nil {
		return AnalysisResponse{}, err
	}
	return c.analyzePrompt(ctx, prompt, PromptOptions{Mode: options.Mode, Bridge: options.Bridge})
}

func (c *DeerFlowClient) AnalyzeResolve(ctx context.Context, bundle ResolveBundle, options AnalyzeOptions) (AnalysisResponse, error) {
	prompt, err := buildResolvePrompt(bundle, defaultAudience(options.Audience))
	if err != nil {
		return AnalysisResponse{}, err
	}
	return c.analyzePrompt(ctx, prompt, PromptOptions{Mode: options.Mode, Bridge: options.Bridge})
}

func (c *DeerFlowClient) AnalyzeFollowUp(ctx context.Context, prompt string, options PromptOptions) (AnalysisResponse, error) {
	return c.analyzePrompt(ctx, prompt, options)
}

func (c *DeerFlowClient) analyzePrompt(ctx context.Context, prompt string, options PromptOptions) (AnalysisResponse, error) {
	if err := c.checkHealth(ctx); err != nil {
		return AnalysisResponse{}, err
	}

	threadID := strings.TrimSpace(options.ThreadID)
	var err error
	if threadID == "" {
		threadID, err = c.createThread(ctx)
		if err != nil {
			return AnalysisResponse{}, err
		}
	}

	bridge := options.Bridge
	if instructions := bridgeInstructions(bridge); instructions != "" {
		prompt = strings.TrimSpace(prompt) + "\n\n" + instructions
	}

	nextPrompt := prompt
	for roundTrip := 0; roundTrip <= maxToolRoundTrips; roundTrip++ {
		run, err := c.streamAuditRun(ctx, threadID, nextPrompt, defaultRunMode(options.Mode))
		if err != nil {
			return AnalysisResponse{}, err
		}
		if run.ToolCall != nil {
			if bridge == nil {
				return AnalysisResponse{}, fmt.Errorf("deerflow requested %q but no gig follow-up bridge is available", run.ToolCall.Name)
			}
			if roundTrip == maxToolRoundTrips {
				return AnalysisResponse{}, fmt.Errorf("deerflow requested more than %d gig follow-up tool round(s)", maxToolRoundTrips)
			}

			result, err := bridge.Execute(ctx, *run.ToolCall)
			if err != nil {
				nextPrompt = assisttools.FormatErrorMessage(*run.ToolCall, err)
				continue
			}

			nextPrompt, err = assisttools.FormatResultMessage(result)
			if err != nil {
				return AnalysisResponse{}, fmt.Errorf("format gig tool result: %w", err)
			}
			continue
		}
		if strings.TrimSpace(run.Response) == "" {
			return AnalysisResponse{}, fmt.Errorf("deerflow returned no assistant response")
		}

		return AnalysisResponse{
			ThreadID: threadID,
			Response: run.Response,
		}, nil
	}

	return AnalysisResponse{
		ThreadID: threadID,
	}, fmt.Errorf("deerflow returned no assistant response")
}

func (c *DeerFlowClient) checkHealth(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.gatewayURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("deerflow is not reachable at %s: %w", c.gatewayURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("deerflow health check failed at %s: %s", c.gatewayURL, formatHTTPError(resp.StatusCode, body))
	}

	return nil
}

func (c *DeerFlowClient) createThread(ctx context.Context) (string, error) {
	req, err := c.newJSONRequest(ctx, http.MethodPost, c.langGraphURL+"/threads", map[string]any{})
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("create deerflow thread: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read deerflow thread response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("create deerflow thread failed: %s", formatHTTPError(resp.StatusCode, body))
	}

	var payload struct {
		ThreadID string `json:"thread_id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("decode deerflow thread response: %w", err)
	}
	if strings.TrimSpace(payload.ThreadID) == "" {
		return "", fmt.Errorf("deerflow thread response did not include thread_id")
	}

	return strings.TrimSpace(payload.ThreadID), nil
}

type streamRunResult struct {
	Response string
	ToolCall *assisttools.Call
}

func (c *DeerFlowClient) streamAuditRun(ctx context.Context, threadID, prompt string, mode RunMode) (streamRunResult, error) {
	body := map[string]any{
		"assistant_id": leadAgentID,
		"input": map[string]any{
			"messages": []map[string]any{
				{
					"type": "human",
					"content": []map[string]any{
						{
							"type": "text",
							"text": prompt,
						},
					},
				},
			},
		},
		"stream_mode":      []string{"values", "messages-tuple"},
		"stream_subgraphs": true,
		"config": map[string]any{
			"recursion_limit": 1000,
		},
		"context": modeContext(mode, threadID),
	}

	req, err := c.newJSONRequest(ctx, http.MethodPost, c.langGraphURL+"/threads/"+threadID+"/runs/stream", body)
	if err != nil {
		return streamRunResult{}, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return streamRunResult{}, fmt.Errorf("stream deerflow audit run: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return streamRunResult{}, fmt.Errorf("read deerflow audit stream: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return streamRunResult{}, fmt.Errorf("deerflow audit run failed: %s", formatHTTPError(resp.StatusCode, raw))
	}

	events := parseSSE(string(raw))
	if run := extractRunResultFromEvents(events); run.ToolCall != nil || strings.TrimSpace(run.Response) != "" {
		return run, nil
	}

	if errorMessage := extractErrorFromEvents(events); errorMessage != "" {
		return streamRunResult{}, fmt.Errorf("deerflow returned an error: %s", errorMessage)
	}

	return streamRunResult{}, fmt.Errorf("deerflow returned no assistant response")
}

func (c *DeerFlowClient) newJSONRequest(ctx context.Context, method, url string, body any) (*http.Request, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func resolveClientURLs(config ClientConfig) (string, string) {
	baseURL := normalizeURL(firstNonEmpty(config.BaseURL, os.Getenv("DEERFLOW_URL"), defaultDeerFlowURL))
	gatewayURL := normalizeURL(firstNonEmpty(config.GatewayURL, os.Getenv("DEERFLOW_GATEWAY_URL"), baseURL))
	langGraphURL := normalizeURL(firstNonEmpty(config.LangGraphURL, os.Getenv("DEERFLOW_LANGGRAPH_URL"), baseURL+"/api/langgraph"))
	return gatewayURL, langGraphURL
}

func normalizeURL(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func modeContext(mode RunMode, threadID string) map[string]any {
	context := map[string]any{
		"thread_id":        threadID,
		"thinking_enabled": true,
		"is_plan_mode":     true,
		"subagent_enabled": false,
	}

	switch defaultRunMode(mode) {
	case ModeFlash:
		context["thinking_enabled"] = false
		context["is_plan_mode"] = false
	case ModeStandard:
		context["is_plan_mode"] = false
	case ModeUltra:
		context["subagent_enabled"] = true
	}

	return context
}

type sseEvent struct {
	Type string
	Data string
}

func parseSSE(raw string) []sseEvent {
	lines := strings.Split(raw, "\n")
	events := make([]sseEvent, 0)
	currentType := ""
	currentData := make([]string, 0)

	flush := func() {
		if currentType == "" {
			return
		}
		events = append(events, sseEvent{
			Type: currentType,
			Data: strings.Join(currentData, "\n"),
		})
		currentType = ""
		currentData = currentData[:0]
	}

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "event:"):
			flush()
			currentType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			currentData = append(currentData, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		case strings.TrimSpace(line) == "":
			flush()
		}
	}

	flush()
	return events
}

func bridgeInstructions(bridge *assisttools.Bridge) string {
	if bridge == nil {
		return ""
	}
	return strings.TrimSpace(bridge.Instructions())
}

func extractRunResultFromEvents(events []sseEvent) streamRunResult {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type != "values" {
			continue
		}

		var payload struct {
			Messages []map[string]any `json:"messages"`
		}
		if err := json.Unmarshal([]byte(events[i].Data), &payload); err != nil {
			continue
		}

		if call, ok := extractToolCall(payload.Messages); ok {
			return streamRunResult{ToolCall: &call}
		}
		if response := extractResponseText(payload.Messages); response != "" {
			return streamRunResult{Response: response}
		}
	}

	return streamRunResult{}
}

func extractErrorFromEvents(events []sseEvent) string {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type == "error" && strings.TrimSpace(events[i].Data) != "" {
			return strings.TrimSpace(events[i].Data)
		}
	}
	return ""
}

func extractResponseText(messages []map[string]any) string {
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]

		if messageType, _ := message["type"].(string); messageType == "tool" {
			if name, _ := message["name"].(string); name == "ask_clarification" {
				if text := extractContentText(message["content"]); text != "" {
					return text
				}
			}
		}

		if messageType, _ := message["type"].(string); messageType == "ai" {
			if text := extractContentText(message["content"]); text != "" {
				return text
			}
		}
	}

	return ""
}

func extractToolCall(messages []map[string]any) (assisttools.Call, bool) {
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		if messageType, _ := message["type"].(string); messageType != "ai" {
			continue
		}

		if call, ok := extractStructuredToolCall(message["tool_calls"]); ok {
			return call, true
		}

		if text := extractContentText(message["content"]); text != "" {
			call, ok, err := assisttools.ParseCall(text)
			if err == nil && ok {
				return call, true
			}
		}
	}

	return assisttools.Call{}, false
}

func extractStructuredToolCall(raw any) (assisttools.Call, bool) {
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return assisttools.Call{}, false
	}

	first, ok := items[0].(map[string]any)
	if !ok {
		return assisttools.Call{}, false
	}

	name, _ := first["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return assisttools.Call{}, false
	}

	args, _ := first["args"].(map[string]any)
	if args == nil {
		args = map[string]any{}
	}

	return assisttools.Call{Name: name, Arguments: args}, true
}

func extractContentText(content any) string {
	switch value := content.(type) {
	case string:
		return strings.TrimSpace(value)
	case []any:
		parts := make([]string, 0, len(value))
		for _, item := range value {
			switch typed := item.(type) {
			case string:
				if strings.TrimSpace(typed) != "" {
					parts = append(parts, strings.TrimSpace(typed))
				}
			case map[string]any:
				if itemType, _ := typed["type"].(string); itemType == "text" {
					if text, _ := typed["text"].(string); strings.TrimSpace(text) != "" {
						parts = append(parts, strings.TrimSpace(text))
					}
				}
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}

func formatHTTPError(statusCode int, body []byte) string {
	message := strings.TrimSpace(string(body))
	if message == "" {
		return fmt.Sprintf("http %d", statusCode)
	}
	return fmt.Sprintf("http %d: %s", statusCode, message)
}

func buildAuditPrompt(bundle AuditBundle, audience Audience) (string, error) {
	payload, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`You are helping gig produce a professional ticket-level release-audit briefing for the %s audience.

Use only the facts in the JSON bundle below.
Do not invent repositories, commits, branch state, deployments, approvals, or release outcomes.
If the deterministic bundle says the verdict is warning or blocked, preserve that severity.
If an evidence gap is unresolved in the bundle, call it out instead of guessing.

Tailor the summary to this audience:
%s

Return concise markdown with these sections:
%s

When suggesting commands, prefer real gig commands that match the ticket, branches, and command target hints in the bundle.
If there are no evidence gaps confirmed by the bundle, say so plainly.

JSON bundle:

%s
`, audience, audienceGuidance(audience, "audit"), audienceSections(audience, "audit"), string(payload)), nil
}

func buildReleasePrompt(bundle ReleaseBundle, audience Audience) (string, error) {
	payload, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`You are helping gig produce a professional release-level audit briefing for the %s audience.

Use only the facts in the JSON bundle below.
Do not invent repositories, commits, branch state, deployments, approvals, release readiness, or ticket outcomes.
Preserve the deterministic severity of the release plan and any blocked or warning tickets.
Call out cross-ticket patterns only when the bundle actually supports them.
Use evidenceSummary, repositoryEvidence, ticketOverlap, executiveSummary, operatorSummary, and hotspots when the bundle provides concrete GitHub pull request, issue, deployment, release, or check context.

Tailor the summary to this audience:
%s

Return concise markdown with these sections:
%s

When suggesting commands, prefer real gig commands that match the release ID, workspace scope, and command target hints in the bundle.
If the release still contains blocked tickets or inconsistent snapshots, say so directly.

JSON bundle:

%s
`, audience, audienceGuidance(audience, "release"), audienceSections(audience, "release"), string(payload)), nil
}

func buildResolvePrompt(bundle ResolveBundle, audience Audience) (string, error) {
	payload, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`You are helping gig produce a professional Git conflict-resolution briefing for the %s audience.

Use only the facts in the JSON bundle below.
Do not invent repository state, resolved file content, branch outcomes, or Git commands that have not already happened.
Do not tell the user that the merge, rebase, or cherry-pick is complete unless the bundle proves it.
If activeConflict is present, focus first on that conflict block and the safest next action.
If activeConflict is absent, explain that gig only found unsupported or manually resolvable files.

Tailor the summary to this audience:
%s

Return concise markdown with these sections:
%s

When suggesting commands, prefer real gig commands that match the command target hints in the bundle.
Prefer gig resolve start, gig resolve status, gig inspect, and gig verify over generic Git advice unless the bundle proves the user is already past that point.

JSON bundle:

%s
`, audience, audienceGuidance(audience, "resolve"), audienceSections(audience, "resolve"), string(payload)), nil
}

func buildAuditFollowUpPrompt(bundle AuditBundle, audience Audience, question string) (string, error) {
	return buildFollowUpPrompt("ticket-level audit", bundle, audience, question)
}

func buildReleaseFollowUpPrompt(bundle ReleaseBundle, audience Audience, question string) (string, error) {
	return buildFollowUpPrompt("release-level audit", bundle, audience, question)
}

func buildResolveFollowUpPrompt(bundle ResolveBundle, audience Audience, question string) (string, error) {
	return buildFollowUpPrompt("conflict-resolution", bundle, audience, question)
}

func buildFollowUpPrompt(scope string, bundle any, audience Audience, question string) (string, error) {
	payload, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`You are continuing an existing gig %s conversation for the %s audience.

Use only the facts in the refreshed JSON bundle below.
Answer the follow-up question directly.
Do not repeat the full brief unless the question explicitly asks for it.
If the question asks for information the bundle does not prove, say that the evidence is not present.
When recommending next steps, prefer real gig commands that match the current bundle hints.

Return concise markdown with these sections:
## Answer
## Evidence Gaps
## Recommended Next gig Commands

Follow-up question:
%s

Refreshed JSON bundle:

%s
`, scope, audience, strings.TrimSpace(question), string(payload)), nil
}

func audienceGuidance(audience Audience, scope string) string {
	switch defaultAudience(audience) {
	case AudienceQA:
		if scope == "resolve" {
			return "Focus on risky files, manual validation after conflict choices, and what QA should re-check before sign-off."
		}
		if scope == "release" {
			return "Focus on regression priorities, manual validation hotspots, risky repositories, and where QA should spend time first."
		}
		return "Focus on regression priorities, manual validation hotspots, and what QA should re-check before sign-off."
	case AudienceClient:
		if scope == "resolve" {
			return "Focus on delivery impact, communication-ready risk language, and what still needs confirmation before this conflict can be considered safely handled."
		}
		if scope == "release" {
			return "Focus on scope clarity, delivery status, communication-ready risks, and what still needs confirmation before client approval."
		}
		return "Focus on scope clarity, delivery status, communication-ready risks, and what still needs confirmation before client approval."
	default:
		if scope == "resolve" {
			return "Focus on the safest next action, scope risk, why a conflict choice is risky or safe, and concrete next gig commands."
		}
		if scope == "release" {
			return "Focus on release readiness, blockers, cross-ticket patterns, concrete next commands, and decision support for promotion."
		}
		return "Focus on release readiness, blockers, concrete next commands, and decision support for promotion."
	}
}

func audienceSections(audience Audience, scope string) string {
	switch defaultAudience(audience) {
	case AudienceQA:
		if scope == "resolve" {
			return "## QA Conflict Summary\n## Highest Risk Files\n## Manual Checks\n## Evidence Gaps\n## Recommended Next gig Commands"
		}
		if scope == "release" {
			return "## QA Release Summary\n## Highest Risk Tickets\n## Regression Focus\n## Evidence Gaps\n## Recommended Next gig Commands"
		}
		return "## QA Summary\n## Regression Focus\n## Manual Checks\n## Evidence Gaps\n## Recommended Next gig Commands"
	case AudienceClient:
		if scope == "resolve" {
			return "## Client Conflict Summary\n## Delivery Risk\n## What Still Needs Confirmation\n## Recommended Next gig Commands"
		}
		if scope == "release" {
			return "## Client Release Summary\n## Included Tickets And Status\n## Risks To Communicate\n## Open Evidence Gaps\n## Recommended Next gig Commands"
		}
		return "## Client Summary\n## Scope And Status\n## Risks To Communicate\n## Open Evidence Gaps\n## Recommended Next gig Commands"
	default:
		if scope == "resolve" {
			return "## Resolve Overview\n## Active Conflict Recommendation\n## Risks And Scope Warnings\n## Recommended Next gig Commands\n## Resolver Note"
		}
		if scope == "release" {
			return "## Release Overview\n## Blockers And Warnings\n## Cross-Ticket Hotspots\n## Recommended Next gig Commands\n## Release Manager Note"
		}
		return "## Executive Summary\n## Risks And Blockers\n## Evidence Gaps\n## Recommended Next gig Commands\n## Release Manager Note"
	}
}
