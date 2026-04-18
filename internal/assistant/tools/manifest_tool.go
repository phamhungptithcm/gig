package tools

import (
	"context"
)

const manifestToolName = "gig_manifest"

type manifestTool struct {
	handler func(context.Context, ManifestRequest) (any, error)
}

func newManifestTool(handler func(context.Context, ManifestRequest) (any, error)) Tool {
	if handler == nil {
		return nil
	}
	return manifestTool{handler: handler}
}

func (t manifestTool) Definition() Definition {
	return Definition{
		Name:        manifestToolName,
		Description: "Refresh deterministic release-packet or audience handoff evidence for the current saved gig session.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"audience": map[string]any{
					"type":        "string",
					"description": "Optional audience section to focus on, such as qa, client, or release-manager.",
				},
			},
			"additionalProperties": false,
		},
	}
}

func (t manifestTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
	payload, err := t.handler(ctx, ManifestRequest{
		Audience: stringArg(args, "audience"),
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Name: manifestToolName, Payload: payload}, nil
}
