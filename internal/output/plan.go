package output

import (
	"fmt"
	"io"
	"strings"

	depsvc "gig/internal/dependency"
	inspectsvc "gig/internal/inspect"
	plansvc "gig/internal/plan"
)

func RenderPromotionPlan(w io.Writer, promotionPlan plansvc.PromotionPlan) error {
	if _, err := fmt.Fprintf(w, "Ticket %s\n", promotionPlan.TicketID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Promotion: %s -> %s\n", promotionPlan.FromBranch, promotionPlan.ToBranch); err != nil {
		return err
	}
	if len(promotionPlan.Environments) > 0 {
		if _, err := fmt.Fprintf(w, "Environments: %s\n", formatEnvironments(promotionPlan.Environments)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "Verdict: %s\n", promotionPlan.Verdict); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Scanned repositories: %d\n", promotionPlan.Summary.ScannedRepositories); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Touched repositories: %d\n\n", promotionPlan.Summary.TouchedRepositories); err != nil {
		return err
	}

	if len(promotionPlan.Notes) > 0 {
		if _, err := fmt.Fprintln(w, "Summary"); err != nil {
			return err
		}
		for _, note := range promotionPlan.Notes {
			if _, err := fmt.Fprintf(w, "  - %s\n", note); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "  - ready repositories: %d\n", promotionPlan.Summary.ReadyRepositories); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  - warning repositories: %d\n", promotionPlan.Summary.WarningRepositories); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  - blocked repositories: %d\n", promotionPlan.Summary.BlockedRepositories); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  - commits to promote: %d\n", promotionPlan.Summary.TotalCommitsToPromote); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  - manual steps: %d\n", promotionPlan.Summary.TotalManualSteps); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	if len(promotionPlan.Repositories) == 0 {
		_, err := fmt.Fprintln(w, "No repository activity found for this ticket.")
		return err
	}

	for i, repositoryPlan := range promotionPlan.Repositories {
		if _, err := fmt.Fprintf(w, "[%s] %s (%s)\n", repositoryPlan.Repository.Name, repositoryPlan.Repository.Root, repositoryPlan.Repository.Type); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  verdict: %s\n", repositoryPlan.Verdict); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  source commits on %s: %d\n", promotionPlan.FromBranch, len(repositoryPlan.Compare.SourceCommits)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  target commits on %s: %d\n", promotionPlan.ToBranch, len(repositoryPlan.Compare.TargetCommits)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  commits to include: %d\n", len(repositoryPlan.Compare.MissingCommits)); err != nil {
			return err
		}
		if len(repositoryPlan.Branches) > 0 {
			if _, err := fmt.Fprintf(w, "  branches seen: %s\n", strings.Join(repositoryPlan.Branches, ", ")); err != nil {
				return err
			}
		}
		if len(repositoryPlan.EnvironmentStatuses) > 0 {
			if _, err := fmt.Fprintln(w, "  environment status:"); err != nil {
				return err
			}
			for _, status := range repositoryPlan.EnvironmentStatuses {
				if _, err := fmt.Fprintf(w, "    - %s\n", formatEnvironmentStatus(status)); err != nil {
					return err
				}
			}
		}
		if len(repositoryPlan.RiskSignals) > 0 {
			if _, err := fmt.Fprintln(w, "  risk signals:"); err != nil {
				return err
			}
			for _, riskSignal := range repositoryPlan.RiskSignals {
				line := fmt.Sprintf("    - %s (%s)", riskSignal.Code, riskSignal.Level)
				if riskSignal.Summary != "" {
					line += ": " + riskSignal.Summary
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
				if _, err := fmt.Fprintf(w, "    - %s\n", formatDependencyResolution(resolution, promotionPlan.FromBranch, promotionPlan.ToBranch)); err != nil {
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
			if _, err := fmt.Fprintln(w, "  plan actions:"); err != nil {
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
		if len(repositoryPlan.Compare.MissingCommits) > 0 {
			if _, err := fmt.Fprintln(w, "  commits to include:"); err != nil {
				return err
			}
			for _, commit := range repositoryPlan.Compare.MissingCommits {
				if _, err := fmt.Fprintf(w, "    - %s  %s\n", commit.ShortHash(), commit.Subject); err != nil {
					return err
				}
			}
		}

		if i < len(promotionPlan.Repositories)-1 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}

	return nil
}

func RenderVerification(w io.Writer, verification plansvc.Verification) error {
	if _, err := fmt.Fprintf(w, "Ticket %s\n", verification.TicketID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Verification: %s -> %s\n", verification.FromBranch, verification.ToBranch); err != nil {
		return err
	}
	if len(verification.Environments) > 0 {
		if _, err := fmt.Fprintf(w, "Environments: %s\n", formatEnvironments(verification.Environments)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "Verdict: %s\n", verification.Verdict); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Scanned repositories: %d\n", verification.Summary.ScannedRepositories); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Touched repositories: %d\n\n", verification.Summary.TouchedRepositories); err != nil {
		return err
	}

	if len(verification.Reasons) > 0 {
		if _, err := fmt.Fprintln(w, "Why"); err != nil {
			return err
		}
		for _, reason := range verification.Reasons {
			if _, err := fmt.Fprintf(w, "  - %s\n", reason); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	if len(verification.Repositories) == 0 {
		_, err := fmt.Fprintln(w, "No repository activity found for this ticket.")
		return err
	}

	for i, repositoryVerification := range verification.Repositories {
		if _, err := fmt.Fprintf(w, "[%s] %s (%s)\n", repositoryVerification.Repository.Name, repositoryVerification.Repository.Root, repositoryVerification.Repository.Type); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  verdict: %s\n", repositoryVerification.Verdict); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "  checks:"); err != nil {
			return err
		}
		for _, check := range repositoryVerification.Checks {
			if _, err := fmt.Fprintf(w, "    - %s\n", check); err != nil {
				return err
			}
		}
		if len(repositoryVerification.DependencyResolutions) > 0 {
			if _, err := fmt.Fprintln(w, "  dependency status:"); err != nil {
				return err
			}
			for _, resolution := range repositoryVerification.DependencyResolutions {
				if _, err := fmt.Fprintf(w, "    - %s\n", formatDependencyResolution(resolution, verification.FromBranch, verification.ToBranch)); err != nil {
					return err
				}
			}
		}
		if len(repositoryVerification.ManualSteps) > 0 {
			if _, err := fmt.Fprintln(w, "  manual steps:"); err != nil {
				return err
			}
			for _, step := range repositoryVerification.ManualSteps {
				if _, err := fmt.Fprintf(w, "    - %s\n", step.Summary); err != nil {
					return err
				}
			}
		}

		if i < len(verification.Repositories)-1 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}

	return nil
}

func formatEnvironmentStatus(status inspectsvc.EnvironmentResult) string {
	switch status.State {
	case inspectsvc.EnvStateBehind:
		return fmt.Sprintf("%s (%s): behind by %d commit(s), commits here %d", status.Environment.Name, status.Environment.Branch, status.MissingFromPrevious, status.CommitCount)
	case inspectsvc.EnvStateAligned:
		return fmt.Sprintf("%s (%s): aligned, commits here %d", status.Environment.Name, status.Environment.Branch, status.CommitCount)
	case inspectsvc.EnvStatePresent:
		return fmt.Sprintf("%s (%s): present, commits here %d", status.Environment.Name, status.Environment.Branch, status.CommitCount)
	case inspectsvc.EnvStateNotPresent:
		return fmt.Sprintf("%s (%s): not present", status.Environment.Name, status.Environment.Branch)
	case inspectsvc.EnvStateBranchMissing:
		return fmt.Sprintf("%s (%s): branch not found", status.Environment.Name, status.Environment.Branch)
	default:
		return fmt.Sprintf("%s (%s): unknown", status.Environment.Name, status.Environment.Branch)
	}
}

func formatDependencyResolution(resolution depsvc.Resolution, fromBranch, toBranch string) string {
	switch resolution.Status {
	case depsvc.StatusSatisfied:
		return fmt.Sprintf("%s: already present in %s", resolution.DependsOn, toBranch)
	case depsvc.StatusMissingTarget:
		return fmt.Sprintf("%s: present in %s but missing from %s", resolution.DependsOn, fromBranch, toBranch)
	case depsvc.StatusUnresolved:
		return fmt.Sprintf("%s: could not be confirmed in %s or %s", resolution.DependsOn, fromBranch, toBranch)
	default:
		return fmt.Sprintf("%s: unknown dependency status", resolution.DependsOn)
	}
}
