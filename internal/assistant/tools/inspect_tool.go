package tools

import (
	"context"
)

const inspectToolName = "gig_inspect"

type inspectTool struct {
	handler func(context.Context, InspectRequest) (any, error)
}

func newInspectTool(handler func(context.Context, InspectRequest) (any, error)) Tool {
	if handler == nil {
		return nil
	}
	return inspectTool{handler: handler}
}

func (t inspectTool) Definition() Definition {
	return Definition{
		Name:        inspectToolName,
		Description: "Refresh deterministic inspection evidence for the current saved gig session without changing repository state.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"focus": map[string]any{
					"type":        "string",
					"description": "Optional focus such as overview, repositories, risks, dependencies, evidence, or active-conflict.",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Optional repository name or root filter when the current session spans multiple repositories.",
				},
			},
			"additionalProperties": false,
		},
	}
}

func (t inspectTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
	payload, err := t.handler(ctx, InspectRequest{
		Focus:      stringArg(args, "focus"),
		Repository: stringArg(args, "repository"),
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Name: inspectToolName, Payload: payload}, nil
}
