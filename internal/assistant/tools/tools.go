package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const CallFence = "gig_tool_call"

type Definition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type Call struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type Result struct {
	Name    string `json:"name"`
	Payload any    `json:"payload"`
}

type InspectRequest struct {
	Focus      string `json:"focus,omitempty"`
	Repository string `json:"repository,omitempty"`
}

type VerifyRequest struct {
	Focus string `json:"focus,omitempty"`
}

type ManifestRequest struct {
	Audience string `json:"audience,omitempty"`
}

type Runtime struct {
	Inspect  func(context.Context, InspectRequest) (any, error)
	Verify   func(context.Context, VerifyRequest) (any, error)
	Manifest func(context.Context, ManifestRequest) (any, error)
}

type Tool interface {
	Definition() Definition
	Execute(ctx context.Context, args map[string]any) (Result, error)
}

type Bridge struct {
	tools       map[string]Tool
	definitions []Definition
}

func NewBridge(runtime Runtime) *Bridge {
	bridge := &Bridge{
		tools:       map[string]Tool{},
		definitions: []Definition{},
	}

	bridge.register(newInspectTool(runtime.Inspect))
	bridge.register(newVerifyTool(runtime.Verify))
	bridge.register(newManifestTool(runtime.Manifest))
	if len(bridge.tools) == 0 {
		return nil
	}
	return bridge
}

func (b *Bridge) Definitions() []Definition {
	if b == nil || len(b.definitions) == 0 {
		return nil
	}
	definitions := make([]Definition, len(b.definitions))
	copy(definitions, b.definitions)
	return definitions
}

func (b *Bridge) Instructions() string {
	if b == nil || len(b.definitions) == 0 {
		return ""
	}

	payload, err := json.MarshalIndent(b.definitions, "", "  ")
	if err != nil {
		return ""
	}

	return fmt.Sprintf(
		"You can request fresh deterministic gig evidence through a read-only tool bridge managed by gig.\n\n"+
			"Use a tool only when the current thread context is not enough.\n"+
			"When you need a tool, emit exactly one fenced block and nothing else:\n\n"+
			"```%s\n"+
			"{\"name\":\"gig_verify\",\"arguments\":{\"focus\":\"summary\"}}\n"+
			"```\n\n"+
			"Supported tools:\n\n"+
			"%s\n\n"+
			"Rules:\n"+
			"- Only use the tool names above.\n"+
			"- Keep arguments small and JSON-serializable.\n"+
			"- Never ask for network access, repository mutation, or any capability outside this read-only bridge.\n"+
			"- After gig returns a tool result, continue the same answer using only that result and the existing thread context.\n",
		CallFence,
		string(payload),
	)
}

func (b *Bridge) Execute(ctx context.Context, call Call) (Result, error) {
	if b == nil {
		return Result{}, fmt.Errorf("no gig follow-up tools are available")
	}

	name := strings.TrimSpace(call.Name)
	tool, ok := b.tools[name]
	if !ok {
		return Result{}, fmt.Errorf("unsupported gig follow-up tool %q", name)
	}

	return tool.Execute(ctx, call.Arguments)
}

func ParseCall(text string) (Call, bool, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Call{}, false, nil
	}

	prefix := "```" + CallFence
	start := strings.Index(text, prefix)
	if start < 0 {
		return Call{}, false, nil
	}

	body := text[start+len(prefix):]
	if strings.HasPrefix(body, "\r\n") {
		body = body[2:]
	} else if strings.HasPrefix(body, "\n") {
		body = body[1:]
	}

	end := strings.Index(body, "```")
	if end < 0 {
		return Call{}, false, fmt.Errorf("gig tool block is missing a closing fence")
	}

	block := strings.TrimSpace(body[:end])
	if block == "" {
		return Call{}, false, fmt.Errorf("gig tool block is empty")
	}

	var call Call
	if err := json.Unmarshal([]byte(block), &call); err != nil {
		return Call{}, false, fmt.Errorf("parse gig tool block: %w", err)
	}
	call.Name = strings.TrimSpace(call.Name)
	if call.Name == "" {
		return Call{}, false, fmt.Errorf("gig tool block is missing a tool name")
	}
	if call.Arguments == nil {
		call.Arguments = map[string]any{}
	}
	return call, true, nil
}

func FormatResultMessage(result Result) (string, error) {
	payload, err := json.MarshalIndent(map[string]any{
		"name":   result.Name,
		"result": result.Payload,
	}, "", "  ")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"gig returned a deterministic read-only tool result.\n\n"+
			"Continue the same answer using only this result and the existing thread context.\n"+
			"If you still need one more gig tool, emit exactly one %s fenced block and no narrative text.\n\n"+
			"Tool result:\n\n"+
			"```json\n%s\n```",
		CallFence,
		string(payload),
	), nil
}

func FormatErrorMessage(call Call, err error) string {
	payload, marshalErr := json.MarshalIndent(map[string]any{
		"name":  strings.TrimSpace(call.Name),
		"error": err.Error(),
	}, "", "  ")
	if marshalErr != nil {
		return fmt.Sprintf("gig tool %s failed: %v", strings.TrimSpace(call.Name), err)
	}

	return fmt.Sprintf(
		"gig could not complete the requested read-only tool call.\n\n"+
			"Do not guess past this error. Either answer with the evidence already available, or request a different gig tool if one would help.\n\n"+
			"Tool error:\n\n"+
			"```json\n%s\n```",
		string(payload),
	)
}

func (b *Bridge) register(tool Tool) {
	if b == nil || tool == nil {
		return
	}
	definition := tool.Definition()
	name := strings.TrimSpace(definition.Name)
	if name == "" {
		return
	}
	b.tools[name] = tool
	b.definitions = append(b.definitions, definition)
}

func stringArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	value, _ := args[key]
	text, _ := value.(string)
	return strings.TrimSpace(text)
}
