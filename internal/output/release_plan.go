package output

import (
	"fmt"
	"io"
	"strings"

	releaseplansvc "gig/internal/releaseplan"
)

func RenderReleasePlan(w io.Writer, releasePlan releaseplansvc.ReleasePlan) error {
	if _, err := fmt.Fprintf(w, "Release %s\n", releasePlan.ReleaseID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Workspace: %s\n", releasePlan.Workspace); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Snapshot directory: %s\n", releasePlan.SnapshotDir); err != nil {
		return err
	}
	if releasePlan.FromBranch != "" || releasePlan.ToBranch != "" {
		if _, err := fmt.Fprintf(w, "Promotion baseline: %s -> %s\n", releasePlan.FromBranch, releasePlan.ToBranch); err != nil {
			return err
		}
	}
	if len(releasePlan.Environments) > 0 {
		if _, err := fmt.Fprintf(w, "Environments: %s\n", formatEnvironments(releasePlan.Environments)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "Verdict: %s\n", releasePlan.Verdict); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Tickets: %d\n", releasePlan.Summary.Tickets); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Touched repositories: %d\n\n", releasePlan.Summary.TouchedRepositories); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "Summary"); err != nil {
		return err
	}
	lines := []string{
		fmt.Sprintf("safe tickets: %d", releasePlan.Summary.SafeTickets),
		fmt.Sprintf("warning tickets: %d", releasePlan.Summary.WarningTickets),
		fmt.Sprintf("blocked tickets: %d", releasePlan.Summary.BlockedTickets),
		fmt.Sprintf("commits to promote: %d", releasePlan.Summary.TotalCommitsToPromote),
		fmt.Sprintf("manual steps: %d", releasePlan.Summary.TotalManualSteps),
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "  - %s\n", line); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	if len(releasePlan.Notes) > 0 {
		if _, err := fmt.Fprintln(w, "Notes"); err != nil {
			return err
		}
		for _, note := range releasePlan.Notes {
			if _, err := fmt.Fprintf(w, "  - %s\n", note); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	if len(releasePlan.Tickets) > 0 {
		if _, err := fmt.Fprintln(w, "Ticket baselines"); err != nil {
			return err
		}
		for _, ticket := range releasePlan.Tickets {
			if _, err := fmt.Fprintf(w, "  - %s: %s, captured %s, repos %d, commits %d, manual steps %d\n", ticket.TicketID, ticket.Verdict, ticket.CapturedAt.Format("2006-01-02T15:04:05Z07:00"), ticket.TouchedRepositories, ticket.CommitsToPromote, ticket.ManualSteps); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	if len(releasePlan.Repositories) == 0 {
		_, err := fmt.Fprintln(w, "No repository activity found in loaded snapshots.")
		return err
	}

	for i, repositoryPlan := range releasePlan.Repositories {
		if _, err := fmt.Fprintf(w, "[%s] %s (%s)\n", repositoryPlan.Repository.Name, repositoryPlan.Repository.Root, repositoryPlan.Repository.Type); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  tickets: %s\n", strings.Join(repositoryPlan.TicketIDs, ", ")); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  verdict: %s\n", repositoryPlan.Verdict); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  commits to include: %d\n", repositoryPlan.CommitsToInclude); err != nil {
			return err
		}
		if len(repositoryPlan.RiskSignals) > 0 {
			if _, err := fmt.Fprintln(w, "  risk signals:"); err != nil {
				return err
			}
			for _, signal := range repositoryPlan.RiskSignals {
				line := fmt.Sprintf("    - %s (%s)", signal.Code, signal.Level)
				if signal.Summary != "" {
					line += ": " + signal.Summary
				}
				if _, err := fmt.Fprintln(w, line); err != nil {
					return err
				}
			}
		}
		if len(repositoryPlan.DependencyResolutions) > 0 {
			if _, err := fmt.Fprintln(w, "  dependency status:"); err != nil {
				return err
			}
			for _, resolution := range repositoryPlan.DependencyResolutions {
				if _, err := fmt.Fprintf(w, "    - %s\n", formatDependencyResolution(resolution, releasePlan.FromBranch, releasePlan.ToBranch)); err != nil {
					return err
				}
			}
		}
		if len(repositoryPlan.ManualSteps) > 0 {
			if _, err := fmt.Fprintln(w, "  manual steps:"); err != nil {
				return err
			}
			for _, step := range repositoryPlan.ManualSteps {
				if _, err := fmt.Fprintf(w, "    - %s\n", step.Summary); err != nil {
					return err
				}
			}
		}
		if len(repositoryPlan.Actions) > 0 {
			if _, err := fmt.Fprintln(w, "  planned actions:"); err != nil {
				return err
			}
			for _, action := range repositoryPlan.Actions {
				if _, err := fmt.Fprintf(w, "    - %s\n", action.Summary); err != nil {
					return err
				}
			}
		}
		if len(repositoryPlan.Notes) > 0 {
			if _, err := fmt.Fprintln(w, "  notes:"); err != nil {
				return err
			}
			for _, note := range repositoryPlan.Notes {
				if _, err := fmt.Fprintf(w, "    - %s\n", note); err != nil {
					return err
				}
			}
		}

		if i < len(releasePlan.Repositories)-1 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}

	return nil
}
