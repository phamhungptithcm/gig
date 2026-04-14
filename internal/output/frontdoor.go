package output

import (
	"fmt"
	"io"

	"gig/internal/workarea"
)

type FrontDoorState struct {
	Current   *workarea.Definition  `json:"current,omitempty"`
	Workareas []workarea.Definition `json:"workareas,omitempty"`
}

func RenderFrontDoor(w io.Writer, state FrontDoorState) error {
	ui := NewConsole(w)

	if err := ui.Title("gig"); err != nil {
		return err
	}
	if err := ui.Subtitle("Remote-first release audit CLI"); err != nil {
		return err
	}
	if err := ui.Blank(); err != nil {
		return err
	}

	if state.Current != nil {
		if err := ui.Section("Current project"); err != nil {
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
		if err := ui.Section("Core workflows"); err != nil {
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
		if err := ui.Section("Start with GitHub"); err != nil {
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

func formatFrontDoorPromotion(definition workarea.Definition) string {
	if definition.FromBranch == "" && definition.ToBranch == "" {
		return ""
	}
	return fmt.Sprintf("%s -> %s", blankAsAuto(definition.FromBranch), blankAsAuto(definition.ToBranch))
}
