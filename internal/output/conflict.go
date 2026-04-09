package output

import (
	"fmt"
	"io"
	"strings"

	conflictsvc "gig/internal/conflict"
	"gig/internal/scm"
)

func RenderConflictStatus(w io.Writer, status conflictsvc.Status) error {
	if _, err := fmt.Fprintf(w, "Repository: %s\n", status.Repository.Root); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Operation: %s\n", status.Operation.Type); err != nil {
		return err
	}
	if status.Repository.CurrentBranch != "" {
		if _, err := fmt.Fprintf(w, "Branch: %s\n", status.Repository.CurrentBranch); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "Current side: %s\n", formatConflictSide(status.Operation.CurrentSide)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Incoming side: %s\n", formatConflictSide(status.Operation.IncomingSide)); err != nil {
		return err
	}
	if status.Operation.SequenceBranch != "" {
		if _, err := fmt.Fprintf(w, "Sequence branch: %s\n", status.Operation.SequenceBranch); err != nil {
			return err
		}
	}
	if status.Operation.TargetBranch != "" {
		if _, err := fmt.Fprintf(w, "Target branch: %s\n", status.Operation.TargetBranch); err != nil {
			return err
		}
	}
	if status.ScopeTicketID != "" {
		if _, err := fmt.Fprintf(w, "Scoped ticket: %s\n", status.ScopeTicketID); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "Resolvable files: %d\n", status.ResolvableFiles); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Unsupported files: %d\n", status.UnsupportedFiles); err != nil {
		return err
	}

	if len(status.Files) == 0 {
		_, err := fmt.Fprintln(w, "\nNo unresolved files were reported by Git.")
		return err
	}

	if _, err := fmt.Fprintln(w, "\nFiles:"); err != nil {
		return err
	}
	for _, file := range status.Files {
		support := "supported"
		if !file.Supported {
			support = "manual"
		}
		if _, err := fmt.Fprintf(w, "  - %s [%s, %s]", file.Path, file.ConflictCode, support); err != nil {
			return err
		}
		if file.BlockCount > 0 {
			if _, err := fmt.Fprintf(w, " (%d block(s))", file.BlockCount); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		if file.UnsupportedReason != "" {
			if _, err := fmt.Fprintf(w, "    reason: %s\n", file.UnsupportedReason); err != nil {
				return err
			}
		}
		for _, warning := range file.Warnings {
			if _, err := fmt.Fprintf(w, "    note: %s\n", warning); err != nil {
				return err
			}
		}
	}

	if status.SuggestedNext != "" {
		if _, err := fmt.Fprintf(w, "\nNext: %s\n", status.SuggestedNext); err != nil {
			return err
		}
	}

	return nil
}

func formatConflictSide(side scm.ConflictSide) string {
	parts := make([]string, 0, 4)
	if side.Label != "" {
		parts = append(parts, side.Label)
	}
	if side.Branch != "" {
		parts = append(parts, side.Branch)
	}
	if short := side.ShortHash(); short != "" {
		parts = append(parts, short)
	}
	if side.Subject != "" {
		parts = append(parts, side.Subject)
	}
	if len(side.TicketIDs) > 0 {
		parts = append(parts, "tickets="+strings.Join(side.TicketIDs, ","))
	}
	return strings.Join(parts, " | ")
}
