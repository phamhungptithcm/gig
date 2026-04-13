package output

import (
	"fmt"
	"io"
	"strings"

	manifestsvc "gig/internal/manifest"
)

func RenderReleasePacketMarkdown(w io.Writer, packet manifestsvc.ReleasePacket) error {
	if _, err := fmt.Fprintf(w, "# Release Packet: %s\n\n", packet.TicketID); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "## Overview"); err != nil {
		return err
	}
	lines := []string{
		fmt.Sprintf("Workspace: `%s`", packet.Workspace),
		fmt.Sprintf("Promotion: `%s -> %s`", packet.FromBranch, packet.ToBranch),
		fmt.Sprintf("Verdict: `%s`", packet.Verdict),
		fmt.Sprintf("Touched repositories: `%d`", packet.Summary.TouchedRepositories),
		fmt.Sprintf("Commits to promote: `%d`", packet.Summary.TotalCommitsToPromote),
		fmt.Sprintf("Manual steps: `%d`", packet.Summary.TotalManualSteps),
	}
	if packet.ConfigPath != "" {
		lines = append(lines, fmt.Sprintf("Config: `%s`", packet.ConfigPath))
	}
	if err := renderMarkdownList(w, lines); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	if len(packet.Highlights) > 0 {
		if _, err := fmt.Fprintln(w, "## Summary"); err != nil {
			return err
		}
		if err := renderMarkdownList(w, packet.Highlights); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	if err := renderAudienceSection(w, packet.QA); err != nil {
		return err
	}
	if err := renderAudienceSection(w, packet.Client); err != nil {
		return err
	}
	if err := renderAudienceSection(w, packet.ReleaseManager); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "## Repository Details"); err != nil {
		return err
	}
	if len(packet.Repositories) == 0 {
		_, err := fmt.Fprintln(w, "No repositories were found for this ticket.")
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	for i, repository := range packet.Repositories {
		if _, err := fmt.Fprintf(w, "### %s\n\n", repository.Repository.Name); err != nil {
			return err
		}

		overview := []string{
			fmt.Sprintf("Repo path: `%s`", repository.Repository.Root),
			fmt.Sprintf("SCM: `%s`", repository.Repository.Type),
			fmt.Sprintf("Verdict: `%s`", repository.Verdict),
		}
		if repository.ConfigEntry != nil {
			if repository.ConfigEntry.Service != "" {
				overview = append(overview, fmt.Sprintf("Service: `%s`", repository.ConfigEntry.Service))
			}
			if repository.ConfigEntry.Owner != "" {
				overview = append(overview, fmt.Sprintf("Owner: `%s`", repository.ConfigEntry.Owner))
			}
			if repository.ConfigEntry.Kind != "" {
				overview = append(overview, fmt.Sprintf("Kind: `%s`", repository.ConfigEntry.Kind))
			}
		}
		if err := renderMarkdownList(w, overview); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}

		if len(repository.EnvironmentStatuses) > 0 {
			if _, err := fmt.Fprintln(w, "Environment status:"); err != nil {
				return err
			}
			statusLines := make([]string, 0, len(repository.EnvironmentStatuses))
			for _, status := range repository.EnvironmentStatuses {
				statusLines = append(statusLines, formatEnvironmentStatus(status))
			}
			if err := renderMarkdownList(w, statusLines); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}

		if len(repository.RiskSignals) > 0 {
			if _, err := fmt.Fprintln(w, "Risk signals:"); err != nil {
				return err
			}
			lines := make([]string, 0, len(repository.RiskSignals))
			for _, signal := range repository.RiskSignals {
				line := fmt.Sprintf("`%s` (%s)", signal.Code, signal.Level)
				if signal.Summary != "" {
					line += ": " + signal.Summary
				}
				if len(signal.Examples) > 0 {
					line += " [" + strings.Join(signal.Examples, ", ") + "]"
				}
				lines = append(lines, line)
			}
			if err := renderMarkdownList(w, lines); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}

		if len(repository.DependencyResolutions) > 0 {
			if _, err := fmt.Fprintln(w, "Dependency status:"); err != nil {
				return err
			}
			lines := make([]string, 0, len(repository.DependencyResolutions))
			for _, resolution := range repository.DependencyResolutions {
				lines = append(lines, formatDependencyResolution(resolution, packet.FromBranch, packet.ToBranch))
			}
			if err := renderMarkdownList(w, lines); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}

		if repository.ProviderEvidence != nil && (len(repository.ProviderEvidence.PullRequests) > 0 || len(repository.ProviderEvidence.Deployments) > 0) {
			if _, err := fmt.Fprintln(w, "Provider evidence:"); err != nil {
				return err
			}
			if err := renderProviderEvidenceMarkdown(w, *repository.ProviderEvidence); err != nil {
				return err
			}
		}

		if len(repository.ManualSteps) > 0 {
			if _, err := fmt.Fprintln(w, "Manual steps:"); err != nil {
				return err
			}
			lines := make([]string, 0, len(repository.ManualSteps))
			for _, step := range repository.ManualSteps {
				lines = append(lines, step.Summary)
			}
			if err := renderMarkdownList(w, lines); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}

		if len(repository.Actions) > 0 {
			if _, err := fmt.Fprintln(w, "Planned actions:"); err != nil {
				return err
			}
			lines := make([]string, 0, len(repository.Actions))
			for _, action := range repository.Actions {
				lines = append(lines, action.Summary)
			}
			if err := renderMarkdownList(w, lines); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}

		if len(repository.Notes) > 0 {
			if _, err := fmt.Fprintln(w, "Notes:"); err != nil {
				return err
			}
			if err := renderMarkdownList(w, repository.Notes); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}

		if len(repository.CommitsToInclude) > 0 {
			if _, err := fmt.Fprintln(w, "Commits to include:"); err != nil {
				return err
			}
			lines := make([]string, 0, len(repository.CommitsToInclude))
			for _, commit := range repository.CommitsToInclude {
				lines = append(lines, fmt.Sprintf("`%s` %s", commit.ShortHash(), commit.Subject))
			}
			if err := renderMarkdownList(w, lines); err != nil {
				return err
			}
			if i < len(packet.Repositories)-1 {
				if _, err := fmt.Fprintln(w); err != nil {
					return err
				}
			}
		}

		if i < len(packet.Repositories)-1 {
			if _, err := fmt.Fprintln(w, "---"); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}

	return nil
}

func renderAudienceSection(w io.Writer, section manifestsvc.AudienceSection) error {
	if _, err := fmt.Fprintf(w, "## %s\n", section.Title); err != nil {
		return err
	}
	if len(section.Summary) > 0 {
		if err := renderMarkdownList(w, section.Summary); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	if len(section.Checklist) > 0 {
		if _, err := fmt.Fprintln(w, "Checklist:"); err != nil {
			return err
		}
		if err := renderMarkdownList(w, section.Checklist); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	return nil
}

func renderMarkdownList(w io.Writer, lines []string) error {
	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "- %s\n", line); err != nil {
			return err
		}
	}
	return nil
}
