package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"gig/internal/output"
	"gig/internal/sourcecontrol"
	"gig/internal/workarea"

	"golang.org/x/term"
)

func (a *App) runFrontDoor(ctx context.Context) int {
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
	if a.frontDoorPromptEnabled() {
		var (
			exitCode int
			err      error
		)
		switch {
		case current != nil:
			exitCode, err = a.runFrontDoorCurrentProjectAction(ctx)
		default:
			exitCode, err = a.runFrontDoorQuickStart(ctx, store, workareas)
		}
		if err != nil {
			fmt.Fprintf(a.stderr, "front door failed: %v\n", err)
			return 1
		}
		if exitCode >= 0 {
			return exitCode
		}
	}

	return 0
}

func (a *App) frontDoorPromptEnabled() bool {
	if file, ok := a.stdin.(*os.File); ok {
		return term.IsTerminal(int(file.Fd()))
	}

	type lenReader interface {
		Len() int
	}
	if reader, ok := a.stdin.(lenReader); ok {
		return reader.Len() > 0
	}

	return false
}

func (a *App) runFrontDoorQuickStart(ctx context.Context, store *workarea.Store, workareas []workarea.Definition) (int, error) {
	reader := bufio.NewReader(a.stdin)

	fmt.Fprintln(a.stdout)
	fmt.Fprintln(a.stdout, "Inspect now?")
	fmt.Fprintln(a.stdout, "  1. Enter a remote repo target")
	fmt.Fprintln(a.stdout, "  2. Discover a repo from a provider")
	if len(workareas) > 0 {
		fmt.Fprintln(a.stdout, "  3. Use a saved workarea")
	}
	fmt.Fprintln(a.stdout, "Press Enter to skip.")
	fmt.Fprint(a.stdout, "Choice: ")

	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return -1, err
	}
	choice := strings.TrimSpace(line)
	if choice == "" {
		return 0, nil
	}

	switch choice {
	case "1", "repo", "target":
		repoTarget, err := a.promptForLine(reader, "Repository target")
		if err != nil {
			return -1, err
		}
		if _, err := sourcecontrol.ParseRepositoryTargets(repoTarget); err != nil {
			return -1, err
		}
		ticketID, err := a.promptForLine(reader, "Ticket ID")
		if err != nil {
			return -1, err
		}
		return a.runInspect(ctx, []string{ticketID, "--repo", repoTarget}), nil
	case "2", "provider", "discover":
		repository, err := a.discoverWorkareaRepositoryWithReader(ctx, reader, "", "", "")
		if err != nil {
			return -1, err
		}
		ticketID, err := a.promptForLine(reader, "Ticket ID")
		if err != nil {
			return -1, err
		}
		return a.runInspect(ctx, []string{ticketID, "--repo", repository.Root}), nil
	case "3", "workarea":
		if len(workareas) == 0 {
			return -1, fmt.Errorf("no saved workareas are available")
		}
		name, err := a.promptForWorkareaSelectionWithReader(reader, store)
		if err != nil {
			return -1, err
		}
		if _, err := store.Use(name); err != nil {
			return -1, err
		}
		ticketID, err := a.promptForLine(reader, "Ticket ID")
		if err != nil {
			return -1, err
		}
		return a.runInspect(ctx, []string{ticketID}), nil
	default:
		return -1, fmt.Errorf("invalid quick-start choice %q", choice)
	}
}

func (a *App) runFrontDoorCurrentProjectAction(ctx context.Context) (int, error) {
	reader := bufio.NewReader(a.stdin)

	fmt.Fprintln(a.stdout)
	fmt.Fprintln(a.stdout, "What do you want to do?")
	fmt.Fprintln(a.stdout, "  1. Inspect ticket")
	fmt.Fprintln(a.stdout, "  2. Verify ticket")
	fmt.Fprintln(a.stdout, "  3. Generate manifest")
	fmt.Fprintln(a.stdout, "  4. AI brief")
	fmt.Fprint(a.stdout, "Choice (press Enter to stay on the dashboard): ")

	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return -1, err
	}
	choice := strings.ToLower(strings.TrimSpace(line))
	if choice == "" {
		return 0, nil
	}

	ticketID, err := a.promptForLine(reader, "Ticket ID")
	if err != nil {
		return -1, err
	}

	switch choice {
	case "1", "inspect", "i":
		return a.runInspect(ctx, []string{ticketID}), nil
	case "2", "verify", "v":
		return a.runVerify(ctx, []string{"--ticket", ticketID}), nil
	case "3", "manifest", "m":
		return a.runManifestGenerate(ctx, []string{"--ticket", ticketID}), nil
	case "4", "assist", "audit", "ai", "brief":
		return a.runAssistAudit(ctx, []string{"--ticket", ticketID}), nil
	default:
		return -1, fmt.Errorf("invalid front-door choice %q", choice)
	}
}
