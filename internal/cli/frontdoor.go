package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"

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

	items := []pickerItem{
		{
			Value:    "discover-github",
			Title:    "Pick a GitHub repository",
			Subtitle: "Recommended first run. gig can browse your GitHub repos after login.",
			Keywords: []string{"github", "recommended", "remote"},
		},
		{
			Value:    "enter-target",
			Title:    "Paste a repository target",
			Subtitle: "Use a target like github:owner/name when you already know the repo.",
			Keywords: []string{"repo", "target", "github:owner/name"},
		},
		{
			Value:    "use-current-folder",
			Title:    "Use the current folder",
			Subtitle: "Local Git or SVN fallback if you already have the code checked out.",
			Keywords: []string{"local", "path", "folder", "workspace"},
		},
		{
			Value:    "discover-other-provider",
			Title:    "Pick from another provider",
			Subtitle: "GitLab, Bitbucket, Azure DevOps, or another supported remote source.",
			Keywords: []string{"gitlab", "bitbucket", "azure", "provider"},
		},
	}
	if len(workareas) > 0 {
		items = append(items, pickerItem{
			Value:    "saved-workarea",
			Title:    "Use a saved project",
			Subtitle: "Reuse a remembered repo so you can skip provider and branch setup.",
			Keywords: []string{"workarea", "saved", "project"},
			Recent:   true,
		})
	}

	fmt.Fprintln(a.stdout)
	selected, err := a.runPicker(reader, "How do you want to start?", items)
	if err != nil {
		if errors.Is(err, errPickerCancelled) {
			return 0, nil
		}
		return -1, err
	}

	switch selected.Value {
	case "discover-github":
		repository, err := a.discoverWorkareaRepositoryWithReader(ctx, reader, "github", "", "")
		if err != nil {
			if errors.Is(err, errPickerCancelled) {
				return 0, nil
			}
			return -1, err
		}
		ticketID, err := a.promptForLine(reader, "Ticket ID")
		if err != nil {
			return -1, err
		}
		return a.runInspect(ctx, []string{ticketID, "--repo", repository.Root}), nil
	case "enter-target":
		fmt.Fprintln(a.stdout, "Repository target example: github:owner/name")
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
	case "use-current-folder":
		ticketID, err := a.promptForLine(reader, "Ticket ID")
		if err != nil {
			return -1, err
		}
		return a.runInspect(ctx, []string{ticketID, "--path", "."}), nil
	case "discover-other-provider":
		repository, err := a.discoverWorkareaRepositoryWithReader(ctx, reader, "", "", "")
		if err != nil {
			if errors.Is(err, errPickerCancelled) {
				return 0, nil
			}
			return -1, err
		}
		ticketID, err := a.promptForLine(reader, "Ticket ID")
		if err != nil {
			return -1, err
		}
		return a.runInspect(ctx, []string{ticketID, "--repo", repository.Root}), nil
	case "saved-workarea":
		name, err := a.promptForWorkareaSelectionWithReader(reader, store)
		if err != nil {
			if errors.Is(err, errPickerCancelled) {
				return 0, nil
			}
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
		return -1, fmt.Errorf("invalid quick-start choice %q", selected.Value)
	}
}

func (a *App) runFrontDoorCurrentProjectAction(ctx context.Context) (int, error) {
	reader := bufio.NewReader(a.stdin)

	store, err := workarea.NewStore()
	if err != nil {
		return -1, err
	}
	workareas, _, err := store.List()
	if err != nil {
		return -1, err
	}

	items := []pickerItem{
		{
			Value:    "inspect",
			Title:    "Inspect one ticket",
			Subtitle: "See every commit, branch, PR, and risk hint for a ticket.",
			Keywords: []string{"inspect", "ticket", "audit"},
		},
		{
			Value:    "verify",
			Title:    "Verify release readiness",
			Subtitle: "Get a safe, warning, or blocked verdict for the next move.",
			Keywords: []string{"verify", "safe", "blocked", "warning"},
		},
		{
			Value:    "manifest",
			Title:    "Generate a release packet",
			Subtitle: "Export Markdown or JSON for QA, client review, and handoff.",
			Keywords: []string{"manifest", "packet", "release"},
		},
	}
	if len(workareas) > 1 {
		items = append(items, pickerItem{
			Value:    "switch-workarea",
			Title:    "Switch saved project",
			Subtitle: "Change the current project before you run the next ticket command.",
			Keywords: []string{"switch", "workarea", "project"},
			Recent:   true,
		})
	}

	fmt.Fprintln(a.stdout)
	selected, err := a.runPicker(reader, "What do you want to do next?", items)
	if err != nil {
		if errors.Is(err, errPickerCancelled) {
			return 0, nil
		}
		return -1, err
	}

	if selected.Value == "switch-workarea" {
		name, err := a.promptForWorkareaSelectionWithReader(reader, store)
		if err != nil {
			if errors.Is(err, errPickerCancelled) {
				return 0, nil
			}
			return -1, err
		}
		if _, err := store.Use(name); err != nil {
			return -1, err
		}
		fmt.Fprintf(a.stdout, "Current project switched to %s.\n", name)
		return 0, nil
	}

	ticketID, err := a.promptForLine(reader, "Ticket ID")
	if err != nil {
		return -1, err
	}

	switch selected.Value {
	case "inspect":
		return a.runInspect(ctx, []string{ticketID}), nil
	case "verify":
		return a.runVerify(ctx, []string{"--ticket", ticketID}), nil
	case "manifest":
		return a.runManifestGenerate(ctx, []string{"--ticket", ticketID}), nil
	default:
		return -1, fmt.Errorf("invalid front-door choice %q", selected.Value)
	}
}
