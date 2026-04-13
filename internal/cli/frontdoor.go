package cli

import (
	"context"
	"fmt"

	"gig/internal/output"
	"gig/internal/workarea"
)

func (a *App) runFrontDoor(ctx context.Context) int {
	_ = ctx

	store, err := workarea.NewStore()
	if err != nil {
		fmt.Fprintf(a.stderr, "front door failed: %v\n", err)
		a.printRootUsage()
		return 1
	}

	workareas, _, err := store.List()
	if err != nil {
		fmt.Fprintf(a.stderr, "front door failed: %v\n", err)
		a.printRootUsage()
		return 1
	}

	var current *workarea.Definition
	if definition, ok, err := store.Current(); err == nil && ok {
		current = &definition
	}

	if err := output.RenderFrontDoor(a.stdout, output.FrontDoorState{
		Current:   current,
		Workareas: workareas,
	}); err != nil {
		fmt.Fprintf(a.stderr, "render failed: %v\n", err)
		return 1
	}

	return 0
}
