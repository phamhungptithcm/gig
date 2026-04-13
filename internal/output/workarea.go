package output

import (
	"fmt"
	"io"
	"strings"

	"gig/internal/workarea"
)

func RenderWorkareaList(w io.Writer, current string, workareas []workarea.Definition) error {
	if len(workareas) == 0 {
		if _, err := fmt.Fprintln(w, "No workareas saved yet."); err != nil {
			return err
		}
		_, err := fmt.Fprintln(w, "Try: gig workarea add payments --repo github:acme/payments --use")
		return err
	}

	if _, err := fmt.Fprintln(w, "Workareas"); err != nil {
		return err
	}
	for index, definition := range workareas {
		marker := " "
		if strings.EqualFold(definition.Name, current) {
			marker = "*"
		}
		if _, err := fmt.Fprintf(w, "%s %s\n", marker, definition.Name); err != nil {
			return err
		}
		if target := formatWorkareaTarget(definition); target != "" {
			if _, err := fmt.Fprintf(w, "  target: %s\n", target); err != nil {
				return err
			}
		}
		if definition.Path != "" {
			if _, err := fmt.Fprintf(w, "  path: %s\n", definition.Path); err != nil {
				return err
			}
		}
		if definition.FromBranch != "" || definition.ToBranch != "" {
			if _, err := fmt.Fprintf(w, "  promotion: %s -> %s\n", blankAsAuto(definition.FromBranch), blankAsAuto(definition.ToBranch)); err != nil {
				return err
			}
		}
		if definition.EnvironmentSpec != "" {
			if _, err := fmt.Fprintf(w, "  envs: %s\n", definition.EnvironmentSpec); err != nil {
				return err
			}
		}
		if index < len(workareas)-1 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}

	return nil
}

func RenderWorkareaDetail(w io.Writer, definition workarea.Definition, scopePath string, current bool) error {
	if _, err := fmt.Fprintf(w, "Workarea %s\n", definition.Name); err != nil {
		return err
	}
	if current {
		if _, err := fmt.Fprintln(w, "Current: yes"); err != nil {
			return err
		}
	}
	if target := formatWorkareaTarget(definition); target != "" {
		if _, err := fmt.Fprintf(w, "Target: %s\n", target); err != nil {
			return err
		}
	}
	if definition.Path != "" {
		if _, err := fmt.Fprintf(w, "Path: %s\n", definition.Path); err != nil {
			return err
		}
	} else if scopePath != "" {
		if _, err := fmt.Fprintf(w, "Home: %s\n", scopePath); err != nil {
			return err
		}
	}
	if definition.ConfigPath != "" {
		if _, err := fmt.Fprintf(w, "Config: %s\n", definition.ConfigPath); err != nil {
			return err
		}
	}
	if definition.FromBranch != "" || definition.ToBranch != "" {
		if _, err := fmt.Fprintf(w, "Promotion: %s -> %s\n", blankAsAuto(definition.FromBranch), blankAsAuto(definition.ToBranch)); err != nil {
			return err
		}
	}
	if definition.EnvironmentSpec != "" {
		if _, err := fmt.Fprintf(w, "Environments: %s\n", definition.EnvironmentSpec); err != nil {
			return err
		}
	}
	return nil
}

func formatWorkareaTarget(definition workarea.Definition) string {
	switch {
	case definition.RepoTarget != "" && definition.Path != "":
		return definition.RepoTarget + " (path fallback available)"
	case definition.RepoTarget != "":
		return definition.RepoTarget
	case definition.Path != "":
		return "local workspace"
	default:
		return ""
	}
}

func blankAsAuto(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "auto"
	}
	return value
}
