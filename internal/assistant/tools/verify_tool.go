package tools

import (
	"context"
)

const verifyToolName = "gig_verify"

type verifyTool struct {
	handler func(context.Context, VerifyRequest) (any, error)
}

func newVerifyTool(handler func(context.Context, VerifyRequest) (any, error)) Tool {
	if handler == nil {
		return nil
	}
	return verifyTool{handler: handler}
}

func (t verifyTool) Definition() Definition {
	return Definition{
		Name:        verifyToolName,
		Description: "Refresh deterministic verification or release-readiness evidence for the current saved gig session.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"focus": map[string]any{
					"type":        "string",
					"description": "Optional focus such as summary, repositories, manual-steps, checks, or reasons.",
				},
			},
			"additionalProperties": false,
		},
	}
}

func (t verifyTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
	payload, err := t.handler(ctx, VerifyRequest{
		Focus: stringArg(args, "focus"),
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Name: verifyToolName, Payload: payload}, nil
}
