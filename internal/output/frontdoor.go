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
	if _, err := fmt.Fprintln(w, "gig"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "Remote-first release audit CLI"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	if state.Current != nil {
		if _, err := fmt.Fprintln(w, "Current project"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  Workarea: %s\n", state.Current.Name); err != nil {
			return err
		}
		if target := formatWorkareaTarget(*state.Current); target != "" {
			if _, err := fmt.Fprintf(w, "  Target: %s\n", target); err != nil {
				return err
			}
		}
		if state.Current.FromBranch != "" || state.Current.ToBranch != "" {
			if _, err := fmt.Fprintf(w, "  Promotion: %s -> %s\n", blankAsAuto(state.Current.FromBranch), blankAsAuto(state.Current.ToBranch)); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "Next commands"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "  gig inspect ABC-123"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "  gig verify --ticket ABC-123"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "  gig manifest generate --ticket ABC-123"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "  gig assist audit --ticket ABC-123 --audience release-manager"); err != nil {
			return err
		}
		if len(state.Workareas) > 1 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w, "Need a different project?"); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w, "  gig workarea use"); err != nil {
				return err
			}
		}
	} else {
		if _, err := fmt.Fprintln(w, "No workarea selected yet."); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "Start here"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "  gig login github"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "  gig workarea add --provider github --use"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "  gig inspect ABC-123 --repo github:owner/name"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "  gig verify --ticket ABC-123 --repo github:owner/name"); err != nil {
			return err
		}
		if len(state.Workareas) > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w, "Saved workareas"); err != nil {
				return err
			}
			for _, definition := range state.Workareas {
				if _, err := fmt.Fprintf(w, "  %s\n", definition.Name); err != nil {
					return err
				}
				if target := formatWorkareaTarget(definition); target != "" {
					if _, err := fmt.Fprintf(w, "    target: %s\n", target); err != nil {
						return err
					}
				}
			}
			if _, err := fmt.Fprintln(w, "  gig workarea use"); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "Optional AI sidecar"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "  gig assist doctor"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "  gig assist setup"); err != nil {
		return err
	}
	if state.Current != nil {
		if _, err := fmt.Fprintln(w, "  gig assist release --release rel-2026-04-09 --path ."); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "More help"); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, "  gig --help")
	return err
}
