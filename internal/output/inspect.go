package output

import (
	"fmt"
	"io"
	"strings"

	inspectsvc "gig/internal/inspect"
)

func RenderInspect(w io.Writer, ticketID, basePath string, scannedRepoCount int, results []inspectsvc.RepositoryInspection) error {
	ui := NewConsole(w)

	totalCommits := 0
	totalRiskSignals := 0
	totalDependencies := 0
	for _, result := range results {
		totalCommits += len(result.Commits)
		totalRiskSignals += len(result.RiskSignals)
		totalDependencies += len(result.DeclaredDependencies)
	}

	if err := ui.Title(fmt.Sprintf("Inspect %s", ticketID)); err != nil {
		return err
	}
	if err := ui.Rows(
		KeyValue{Label: "Scope", Value: basePath},
		KeyValue{Label: "Repositories", Value: fmt.Sprintf("%d touched / %d scanned", len(results), scannedRepoCount)},
		KeyValue{Label: "Commits", Value: pluralizeCount(totalCommits, "ticket commit", "ticket commits")},
		KeyValue{Label: "Risks", Value: pluralizeCount(totalRiskSignals, "risk signal", "risk signals")},
		KeyValue{Label: "Dependencies", Value: pluralizeCount(totalDependencies, "declared dependency", "declared dependencies")},
	); err != nil {
		return err
	}
	if err := ui.Blank(); err != nil {
		return err
	}

	if scannedRepoCount == 0 {
		if err := ui.Section("Recommended next step"); err != nil {
			return err
		}
		if err := ui.Bullets("Check the local path or switch to a remote target such as " + ui.Command("github:owner/name") + "."); err != nil {
			return err
		}
		_, err := fmt.Fprintf(w, "No repositories found under %s.\n", basePath)
		return err
	}
	if len(results) == 0 {
		if err := ui.Section("Recommended next step"); err != nil {
			return err
		}
		if err := ui.Bullets("Confirm the ticket ID or run " + ui.Command("gig verify "+ticketID) + " with the same scope after checking your repo target."); err != nil {
			return err
		}
		_, err := fmt.Fprintf(w, "No commits found for %s.\n", ticketID)
		return err
	}

	if err := ui.Section("Recommended next step"); err != nil {
		return err
	}
	if err := ui.Bullets("Run " + ui.Command("gig verify "+ticketID) + " with the same scope to turn this evidence into a release verdict."); err != nil {
		return err
	}
	if err := ui.Blank(); err != nil {
		return err
	}

	for i, result := range results {
		if err := ui.Section(fmt.Sprintf("Repository %s", result.Repository.Name)); err != nil {
			return err
		}
		if err := ui.NestedRows(
			KeyValue{Label: "Scope", Value: fmt.Sprintf("%s (%s)", result.Repository.Root, result.Repository.Type)},
			KeyValue{Label: "Commits", Value: pluralizeCount(len(result.Commits), "ticket commit", "ticket commits")},
			KeyValue{Label: "Branches", Value: strings.Join(result.Branches, ", ")},
			KeyValue{Label: "Risks", Value: pluralizeCount(len(result.RiskSignals), "signal", "signals")},
			KeyValue{Label: "Dependencies", Value: pluralizeCount(len(result.DeclaredDependencies), "declared dependency", "declared dependencies")},
		); err != nil {
			return err
		}
		if len(result.RiskSignals) > 0 {
			if err := ui.NestedSection("Risk signals"); err != nil {
				return err
			}
			for _, signal := range result.RiskSignals {
				line := fmt.Sprintf("%s (%s)", signal.Code, signal.Level)
				if signal.Summary != "" {
					line += ": " + signal.Summary
				}
				if len(signal.Examples) > 0 {
					line += " [" + strings.Join(signal.Examples, ", ") + "]"
				}
				if err := ui.NestedBullets(line); err != nil {
					return err
				}
			}
		}
		if len(result.DeclaredDependencies) > 0 {
			if err := ui.NestedSection("Declared dependencies"); err != nil {
				return err
			}
			for _, dependency := range result.DeclaredDependencies {
				if err := ui.NestedBullets(fmt.Sprintf("%s (declared by %s  %s)", dependency.DependsOn, shortHash(dependency.CommitHash), dependency.CommitSubject)); err != nil {
					return err
				}
			}
		}
		if result.ProviderEvidence != nil && (len(result.ProviderEvidence.PullRequests) > 0 || len(result.ProviderEvidence.Deployments) > 0) {
			if err := ui.NestedSection("Provider evidence"); err != nil {
				return err
			}
			if err := renderProviderEvidence(w, *result.ProviderEvidence, "    "); err != nil {
				return err
			}
		}

		if err := ui.NestedSection("Commits"); err != nil {
			return err
		}
		for _, commit := range result.Commits {
			if err := ui.NestedBullets(fmt.Sprintf("%s  %s", commit.ShortHash(), commit.Subject)); err != nil {
				return err
			}
			if len(commit.Branches) > 0 {
				if _, err := fmt.Fprintf(w, "      %s  %s\n", ui.Muted("branches"), strings.Join(commit.Branches, ", ")); err != nil {
					return err
				}
			}
		}

		if i < len(results)-1 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}

	return nil
}

func pluralizeCount(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

func shortHash(hash string) string {
	hash = strings.TrimSpace(hash)
	if len(hash) <= 8 {
		return hash
	}
	return hash[:8]
}

func RenderEnvironmentStatus(w io.Writer, ticketID, basePath string, environments []inspectsvc.Environment, scannedRepoCount int, results []inspectsvc.RepositoryEnvironmentStatus) error {
	if _, err := fmt.Fprintf(w, "Ticket %s\n", ticketID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Workspace: %s\n", basePath); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Environments: %s\n", formatEnvironments(environments)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Scanned repositories: %d\n", scannedRepoCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Touched repositories: %d\n\n", len(results)); err != nil {
		return err
	}

	if scannedRepoCount == 0 {
		_, err := fmt.Fprintf(w, "No repositories found under %s.\n", basePath)
		return err
	}
	if len(results) == 0 {
		_, err := fmt.Fprintf(w, "No commits found for %s.\n", ticketID)
		return err
	}

	summary := summarizeEnvironmentStates(results, environments)
	if _, err := fmt.Fprintln(w, "Summary"); err != nil {
		return err
	}
	for _, line := range summary {
		if _, err := fmt.Fprintf(w, "  %s\n", line); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	for i, result := range results {
		if _, err := fmt.Fprintf(w, "[%s] %s (%s)\n", result.Repository.Name, result.Repository.Root, result.Repository.Type); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  ticket commits: %d\n", len(result.Commits)); err != nil {
			return err
		}
		if len(result.Branches) > 0 {
			if _, err := fmt.Fprintf(w, "  branches seen: %s\n", strings.Join(result.Branches, ", ")); err != nil {
				return err
			}
		}
		if len(result.RiskSignals) > 0 {
			if _, err := fmt.Fprintln(w, "  risk signals:"); err != nil {
				return err
			}
			for _, signal := range result.RiskSignals {
				if _, err := fmt.Fprintf(w, "    - %s (%s)\n", signal.Code, signal.Level); err != nil {
					return err
				}
			}
		}

		for _, status := range result.Statuses {
			switch status.State {
			case inspectsvc.EnvStateBehind:
				if _, err := fmt.Fprintf(w, "  %s (%s): behind by %d commit(s), commits here %d\n", status.Environment.Name, status.Environment.Branch, status.MissingFromPrevious, status.CommitCount); err != nil {
					return err
				}
			case inspectsvc.EnvStateAligned:
				if _, err := fmt.Fprintf(w, "  %s (%s): aligned, commits here %d\n", status.Environment.Name, status.Environment.Branch, status.CommitCount); err != nil {
					return err
				}
			case inspectsvc.EnvStatePresent:
				if _, err := fmt.Fprintf(w, "  %s (%s): present, commits here %d\n", status.Environment.Name, status.Environment.Branch, status.CommitCount); err != nil {
					return err
				}
			case inspectsvc.EnvStateNotPresent:
				if _, err := fmt.Fprintf(w, "  %s (%s): not present\n", status.Environment.Name, status.Environment.Branch); err != nil {
					return err
				}
			case inspectsvc.EnvStateBranchMissing:
				if _, err := fmt.Fprintf(w, "  %s (%s): branch not found\n", status.Environment.Name, status.Environment.Branch); err != nil {
					return err
				}
			}
		}

		if i < len(results)-1 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}

	return nil
}

func formatEnvironments(environments []inspectsvc.Environment) string {
	parts := make([]string, 0, len(environments))
	for _, environment := range environments {
		parts = append(parts, fmt.Sprintf("%s=%s", environment.Name, environment.Branch))
	}
	return strings.Join(parts, " -> ")
}

func summarizeEnvironmentStates(results []inspectsvc.RepositoryEnvironmentStatus, environments []inspectsvc.Environment) []string {
	lines := make([]string, 0, len(environments))
	for index, environment := range environments {
		counts := map[inspectsvc.EnvironmentState]int{}
		for _, result := range results {
			if index >= len(result.Statuses) {
				continue
			}
			counts[result.Statuses[index].State]++
		}

		switch {
		case counts[inspectsvc.EnvStateBehind] > 0:
			lines = append(lines, fmt.Sprintf("%s (%s): behind in %d repo(s)", environment.Name, environment.Branch, counts[inspectsvc.EnvStateBehind]))
		case counts[inspectsvc.EnvStateAligned] > 0:
			lines = append(lines, fmt.Sprintf("%s (%s): aligned in %d repo(s)", environment.Name, environment.Branch, counts[inspectsvc.EnvStateAligned]))
		case counts[inspectsvc.EnvStatePresent] > 0:
			lines = append(lines, fmt.Sprintf("%s (%s): present in %d repo(s)", environment.Name, environment.Branch, counts[inspectsvc.EnvStatePresent]))
		case counts[inspectsvc.EnvStateNotPresent] > 0:
			lines = append(lines, fmt.Sprintf("%s (%s): not present in %d repo(s)", environment.Name, environment.Branch, counts[inspectsvc.EnvStateNotPresent]))
		case counts[inspectsvc.EnvStateBranchMissing] > 0:
			lines = append(lines, fmt.Sprintf("%s (%s): branch missing in %d repo(s)", environment.Name, environment.Branch, counts[inspectsvc.EnvStateBranchMissing]))
		default:
			lines = append(lines, fmt.Sprintf("%s (%s): no data", environment.Name, environment.Branch))
		}
	}

	return lines
}
