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
	ui := NewConsole(w)

	if err := ui.Title(fmt.Sprintf("Plan %s", promotionPlan.TicketID)); err != nil {
		return err
	}
	if err := ui.Rows(
		KeyValue{Label: "Promotion", Value: fmt.Sprintf("%s -> %s", promotionPlan.FromBranch, promotionPlan.ToBranch)},
		KeyValue{Label: "Verdict", Value: ui.Verdict(string(promotionPlan.Verdict))},
		KeyValue{Label: "Environments", Value: formatEnvironments(promotionPlan.Environments)},
		KeyValue{Label: "Repositories", Value: fmt.Sprintf("%d touched / %d scanned", promotionPlan.Summary.TouchedRepositories, promotionPlan.Summary.ScannedRepositories)},
		KeyValue{Label: "Commits", Value: pluralizeCount(promotionPlan.Summary.TotalCommitsToPromote, "commit to include", "commits to include")},
		KeyValue{Label: "Manual steps", Value: pluralizeCount(promotionPlan.Summary.TotalManualSteps, "manual step", "manual steps")},
	); err != nil {
		return err
	}
	if err := ui.Blank(); err != nil {
		return err
	}

	if len(promotionPlan.Notes) > 0 {
		if err := ui.Section("Release summary"); err != nil {
			return err
		}
		if err := ui.Bullets(promotionPlan.Notes...); err != nil {
			return err
		}
		if err := ui.Bullets(
			fmt.Sprintf("%d ready repos", promotionPlan.Summary.ReadyRepositories),
			fmt.Sprintf("%d warning repos", promotionPlan.Summary.WarningRepositories),
			fmt.Sprintf("%d blocked repos", promotionPlan.Summary.BlockedRepositories),
			pluralizeCount(promotionPlan.Summary.TotalCommitsToPromote, "commit to promote", "commits to promote"),
			pluralizeCount(promotionPlan.Summary.TotalManualSteps, "manual step", "manual steps"),
		); err != nil {
			return err
		}
		if err := ui.Blank(); err != nil {
			return err
		}
	}

	if len(promotionPlan.Repositories) == 0 {
		_, err := fmt.Fprintln(w, "No repository activity found for this ticket.")
		return err
	}

	if err := ui.Section("Recommended next step"); err != nil {
		return err
	}
	switch promotionPlan.Verdict {
	case plansvc.VerdictSafe:
		if err := ui.Bullets("Run " + ui.Command("gig manifest "+promotionPlan.TicketID) + " with the same scope to prepare the release packet."); err != nil {
			return err
		}
	default:
		if err := ui.Bullets("Review blocked repositories, dependency gaps, and manual steps below before promoting this ticket."); err != nil {
			return err
		}
	}
	if err := ui.Blank(); err != nil {
		return err
	}

	for i, repositoryPlan := range promotionPlan.Repositories {
		if err := ui.Section(fmt.Sprintf("Repository %s", repositoryPlan.Repository.Name)); err != nil {
			return err
		}
		if err := ui.NestedRows(
			KeyValue{Label: "Scope", Value: fmt.Sprintf("%s (%s)", repositoryPlan.Repository.Root, repositoryPlan.Repository.Type)},
			KeyValue{Label: "Verdict", Value: ui.Verdict(string(repositoryPlan.Verdict))},
			KeyValue{Label: "Source commits", Value: fmt.Sprintf("%d on %s", len(repositoryPlan.Compare.SourceCommits), promotionPlan.FromBranch)},
			KeyValue{Label: "Target commits", Value: fmt.Sprintf("%d on %s", len(repositoryPlan.Compare.TargetCommits), promotionPlan.ToBranch)},
			KeyValue{Label: "To include", Value: pluralizeCount(len(repositoryPlan.Compare.MissingCommits), "commit", "commits")},
			KeyValue{Label: "Branches", Value: strings.Join(repositoryPlan.Branches, ", ")},
		); err != nil {
			return err
		}
		if len(repositoryPlan.EnvironmentStatuses) > 0 {
			if err := ui.NestedSection("Environment status"); err != nil {
				return err
			}
			for _, status := range repositoryPlan.EnvironmentStatuses {
				if err := ui.NestedBullets(formatEnvironmentStatus(status)); err != nil {
					return err
				}
			}
		}
		if len(repositoryPlan.RiskSignals) > 0 {
			if err := ui.NestedSection("Risk signals"); err != nil {
				return err
			}
			for _, riskSignal := range repositoryPlan.RiskSignals {
				line := fmt.Sprintf("%s (%s)", riskSignal.Code, riskSignal.Level)
				if riskSignal.Summary != "" {
					line += ": " + riskSignal.Summary
				}
				if err := ui.NestedBullets(line); err != nil {
					return err
				}
			}
		}
		if len(repositoryPlan.DependencyResolutions) > 0 {
			if err := ui.NestedSection("Dependency status"); err != nil {
				return err
			}
			for _, resolution := range repositoryPlan.DependencyResolutions {
				if err := ui.NestedBullets(formatDependencyResolution(resolution, promotionPlan.FromBranch, promotionPlan.ToBranch)); err != nil {
					return err
				}
			}
		}
		if repositoryPlan.ProviderEvidence != nil && (len(repositoryPlan.ProviderEvidence.PullRequests) > 0 || len(repositoryPlan.ProviderEvidence.Deployments) > 0) {
			if err := ui.NestedSection("Provider evidence"); err != nil {
				return err
			}
			if err := renderProviderEvidence(w, *repositoryPlan.ProviderEvidence, "    "); err != nil {
				return err
			}
		}
		if len(repositoryPlan.ManualSteps) > 0 {
			if err := ui.NestedSection("Manual steps"); err != nil {
				return err
			}
			for _, step := range repositoryPlan.ManualSteps {
				if err := ui.NestedBullets(step.Summary); err != nil {
					return err
				}
			}
		}
		if len(repositoryPlan.Actions) > 0 {
			if err := ui.NestedSection("Plan actions"); err != nil {
				return err
			}
			for _, action := range repositoryPlan.Actions {
				if err := ui.NestedBullets(action.Summary); err != nil {
					return err
				}
			}
		}
		if len(repositoryPlan.Notes) > 0 {
			if err := ui.NestedSection("Notes"); err != nil {
				return err
			}
			for _, note := range repositoryPlan.Notes {
				if err := ui.NestedBullets(note); err != nil {
					return err
				}
			}
		}
		if len(repositoryPlan.Compare.MissingCommits) > 0 {
			if err := ui.NestedSection("Commits to include"); err != nil {
				return err
			}
			for _, commit := range repositoryPlan.Compare.MissingCommits {
				if err := ui.NestedBullets(fmt.Sprintf("%s  %s", commit.ShortHash(), commit.Subject)); err != nil {
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
	ui := NewConsole(w)

	if err := ui.Title(fmt.Sprintf("Verify %s", verification.TicketID)); err != nil {
		return err
	}
	if err := ui.Rows(
		KeyValue{Label: "Promotion", Value: fmt.Sprintf("%s -> %s", verification.FromBranch, verification.ToBranch)},
		KeyValue{Label: "Verdict", Value: ui.Verdict(string(verification.Verdict))},
		KeyValue{Label: "Environments", Value: formatEnvironments(verification.Environments)},
		KeyValue{Label: "Repositories", Value: fmt.Sprintf("%d touched / %d scanned", verification.Summary.TouchedRepositories, verification.Summary.ScannedRepositories)},
		KeyValue{Label: "Manual steps", Value: pluralizeCount(totalVerificationManualSteps(verification), "manual step", "manual steps")},
		KeyValue{Label: "Dependencies", Value: pluralizeCount(totalVerificationDependencyFindings(verification), "dependency gap", "dependency gaps")},
	); err != nil {
		return err
	}
	if err := ui.Blank(); err != nil {
		return err
	}

	if len(verification.Reasons) > 0 {
		if err := ui.Section("Why"); err != nil {
			return err
		}
		if err := ui.Bullets(verification.Reasons...); err != nil {
			return err
		}
		if err := ui.Blank(); err != nil {
			return err
		}
	}

	if len(verification.Repositories) == 0 {
		_, err := fmt.Fprintln(w, "No repository activity found for this ticket.")
		return err
	}

	if err := ui.Section("Recommended next step"); err != nil {
		return err
	}
	switch verification.Verdict {
	case plansvc.VerdictSafe:
		if err := ui.Bullets("Run " + ui.Command("gig manifest "+verification.TicketID) + " with the same scope to prepare handoff and release notes."); err != nil {
			return err
		}
	default:
		if err := ui.Bullets("Run " + ui.Command("gig plan "+verification.TicketID) + " with the same scope to review missing commits, dependencies, and manual steps."); err != nil {
			return err
		}
	}
	if err := ui.Blank(); err != nil {
		return err
	}

	for i, repositoryVerification := range verification.Repositories {
		if err := ui.Section(fmt.Sprintf("Repository %s", repositoryVerification.Repository.Name)); err != nil {
			return err
		}
		if err := ui.NestedRows(
			KeyValue{Label: "Scope", Value: fmt.Sprintf("%s (%s)", repositoryVerification.Repository.Root, repositoryVerification.Repository.Type)},
			KeyValue{Label: "Verdict", Value: ui.Verdict(string(repositoryVerification.Verdict))},
			KeyValue{Label: "Checks", Value: pluralizeCount(len(repositoryVerification.Checks), "check", "checks")},
			KeyValue{Label: "Dependencies", Value: pluralizeCount(len(repositoryVerification.DependencyResolutions), "finding", "findings")},
			KeyValue{Label: "Manual steps", Value: pluralizeCount(len(repositoryVerification.ManualSteps), "step", "steps")},
		); err != nil {
			return err
		}
		if err := ui.NestedSection("Checks"); err != nil {
			return err
		}
		for _, check := range repositoryVerification.Checks {
			if err := ui.NestedBullets(check); err != nil {
				return err
			}
		}
		if len(repositoryVerification.DependencyResolutions) > 0 {
			if err := ui.NestedSection("Dependency status"); err != nil {
				return err
			}
			for _, resolution := range repositoryVerification.DependencyResolutions {
				if err := ui.NestedBullets(formatDependencyResolution(resolution, verification.FromBranch, verification.ToBranch)); err != nil {
					return err
				}
			}
		}
		if len(repositoryVerification.ManualSteps) > 0 {
			if err := ui.NestedSection("Manual steps"); err != nil {
				return err
			}
			for _, step := range repositoryVerification.ManualSteps {
				if err := ui.NestedBullets(step.Summary); err != nil {
					return err
				}
			}
		}
		if repositoryVerification.ProviderEvidence != nil && (len(repositoryVerification.ProviderEvidence.PullRequests) > 0 || len(repositoryVerification.ProviderEvidence.Deployments) > 0) {
			if err := ui.NestedSection("Provider evidence"); err != nil {
				return err
			}
			if err := renderProviderEvidence(w, *repositoryVerification.ProviderEvidence, "    "); err != nil {
				return err
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

func totalVerificationManualSteps(verification plansvc.Verification) int {
	total := 0
	for _, repositoryVerification := range verification.Repositories {
		total += len(repositoryVerification.ManualSteps)
	}
	return total
}

func totalVerificationDependencyFindings(verification plansvc.Verification) int {
	total := 0
	for _, repositoryVerification := range verification.Repositories {
		for _, resolution := range repositoryVerification.DependencyResolutions {
			if resolution.Status != depsvc.StatusSatisfied {
				total++
			}
		}
	}
	return total
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
