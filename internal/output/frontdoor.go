package output

import (
	"fmt"
	"io"
	"strings"

	"gig/internal/workarea"
)

type FrontDoorState struct {
	Current   *workarea.Definition  `json:"current,omitempty"`
	Workareas []workarea.Definition `json:"workareas,omitempty"`
	Version   string                `json:"version,omitempty"`
}

func RenderFrontDoor(w io.Writer, state FrontDoorState) error {
	ui := NewConsole(w)

	if err := renderFrontDoorHero(w, ui, state); err != nil {
		return err
	}
	if err := ui.Blank(); err != nil {
		return err
	}

	if state.Current != nil {
		if err := ui.Section("Ready now"); err != nil {
			return err
		}
		if err := ui.Rows(
			KeyValue{Label: "Workarea", Value: state.Current.Name},
			KeyValue{Label: "Target", Value: formatWorkareaTarget(*state.Current)},
			KeyValue{Label: "Promotion", Value: formatFrontDoorPromotion(*state.Current)},
		); err != nil {
			return err
		}
		if err := ui.Blank(); err != nil {
			return err
		}
		if err := ui.Section("Ask gig to"); err != nil {
			return err
		}
		if err := ui.Bullets(
			"inspect one ticket across branches and repos",
			"verify whether the next release move is safe",
			"generate a release packet for QA or release review",
		); err != nil {
			return err
		}
		if err := ui.Blank(); err != nil {
			return err
		}
		if err := ui.Section("Quick commands"); err != nil {
			return err
		}
		if err := ui.Commands(
			"gig inspect ABC-123",
			"gig verify --ticket ABC-123",
			"gig manifest generate --ticket ABC-123",
		); err != nil {
			return err
		}
		if err := ui.Note("Run `gig` in a real terminal and use ↑/↓ then Enter to choose the next action."); err != nil {
			return err
		}
		if len(state.Workareas) > 1 {
			if err := ui.Blank(); err != nil {
				return err
			}
			if err := ui.Section("Need a different project?"); err != nil {
				return err
			}
			if err := ui.Commands("gig workarea use"); err != nil {
				return err
			}
		}
	} else {
		if err := ui.Section("Start here"); err != nil {
			return err
		}
		if err := ui.Bullets(
			"pick a GitHub repository",
			"paste a repository target",
			"use the current folder if the repo is already checked out",
		); err != nil {
			return err
		}
		if err := ui.Blank(); err != nil {
			return err
		}
		if err := ui.Section("Fastest path"); err != nil {
			return err
		}
		if err := ui.Commands(
			"gig",
			"gig login github",
			"gig inspect ABC-123 --repo github:owner/name",
		); err != nil {
			return err
		}
		if err := ui.Note("Run `gig` in a real terminal and use ↑/↓ then Enter if you want gig to guide you to the right repo first."); err != nil {
			return err
		}
		if err := ui.Note("gig remembers a successful remote repo as your current project automatically."); err != nil {
			return err
		}
		if err := ui.Blank(); err != nil {
			return err
		}
		if err := ui.Section("Ask gig to"); err != nil {
			return err
		}
		if err := ui.Bullets(
			"inspect one ticket",
			"verify release readiness",
			"generate a release packet",
		); err != nil {
			return err
		}
		if err := ui.Blank(); err != nil {
			return err
		}
		if err := ui.Section("Core workflows"); err != nil {
			return err
		}
		if err := ui.Commands(
			"gig inspect ABC-123 --repo github:owner/name",
			"gig verify --ticket ABC-123 --repo github:owner/name",
			"gig manifest generate --ticket ABC-123 --repo github:owner/name",
		); err != nil {
			return err
		}
		if err := ui.Note("These three commands are the main path. Most teams do not need to learn the rest on day one."); err != nil {
			return err
		}
		if err := ui.Blank(); err != nil {
			return err
		}
		if err := ui.Section("Still local?"); err != nil {
			return err
		}
		if err := ui.Commands(
			"gig inspect ABC-123 --path .",
			"gig verify --ticket ABC-123 --path .",
		); err != nil {
			return err
		}
		if err := ui.Note("Local Git and SVN still work when you already have a repo checked out."); err != nil {
			return err
		}
		if len(state.Workareas) > 0 {
			if err := ui.Blank(); err != nil {
				return err
			}
			if err := ui.Section("Saved workareas"); err != nil {
				return err
			}
			for _, definition := range state.Workareas {
				if err := ui.Bullets(definition.Name); err != nil {
					return err
				}
				if target := formatWorkareaTarget(definition); target != "" {
					if _, err := fmt.Fprintf(w, "    %s  %s\n", ui.Muted("target"), target); err != nil {
						return err
					}
				}
			}
			if err := ui.Commands("gig workarea use"); err != nil {
				return err
			}
		}
	}

	if err := ui.Blank(); err != nil {
		return err
	}
	if err := ui.Section("Optional AI sidecar"); err != nil {
		return err
	}
	if err := ui.Commands("gig assist doctor", "gig assist setup"); err != nil {
		return err
	}
	if state.Current != nil {
		if err := ui.Commands("gig assist audit --ticket ABC-123 --audience release-manager"); err != nil {
			return err
		}
	}

	if err := ui.Blank(); err != nil {
		return err
	}
	if err := ui.Section("More help"); err != nil {
		return err
	}
	return ui.Commands("gig --help")
}

func renderFrontDoorHero(w io.Writer, ui Console, state FrontDoorState) error {
	lines := []string{
		fmt.Sprintf(">_ gig  (%s)", blankAsDefault(state.Version, "dev")),
		"googling in git",
	}

	if state.Current != nil {
		lines = append(lines,
			fmt.Sprintf("project: %s", state.Current.Name),
			fmt.Sprintf("target:  %s", formatWorkareaTarget(*state.Current)),
			"focus:   inspect | verify | manifest",
		)
	} else {
		lines = append(lines,
			"mode:    guided terminal front door",
			"focus:   inspect | verify | manifest",
			"status:  no project selected yet",
		)
	}

	return writeFrontDoorBox(w, ui, lines)
}

func writeFrontDoorBox(w io.Writer, ui Console, lines []string) error {
	width := 0
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, " ")
		normalized = append(normalized, line)
		if len(line) > width {
			width = len(line)
		}
	}

	border := "+" + strings.Repeat("-", width+2) + "+"
	if _, err := fmt.Fprintln(w, ui.Emphasis(border)); err != nil {
		return err
	}
	for index, line := range normalized {
		padded := line + strings.Repeat(" ", width-len(line))
		value := padded
		switch {
		case index == 1:
			value = ui.Command(padded)
		case strings.HasPrefix(line, "status:"):
			value = ui.Muted(padded)
		}
		if _, err := fmt.Fprintf(w, "%s %s %s\n", ui.Emphasis("|"), value, ui.Emphasis("|")); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w, ui.Emphasis(border))
	return err
}

func formatFrontDoorPromotion(definition workarea.Definition) string {
	if definition.FromBranch == "" && definition.ToBranch == "" {
		return ""
	}
	return fmt.Sprintf("%s -> %s", blankAsAuto(definition.FromBranch), blankAsAuto(definition.ToBranch))
}

func blankAsDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
