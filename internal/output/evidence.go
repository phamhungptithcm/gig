package output

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"gig/internal/scm"
)

func renderProviderEvidence(w io.Writer, evidence scm.ProviderEvidence, indent string) error {
	if len(evidence.PullRequests) == 0 && len(evidence.Deployments) == 0 {
		return nil
	}

	if len(evidence.PullRequests) > 0 {
		if _, err := fmt.Fprintf(w, "%spull requests:\n", indent); err != nil {
			return err
		}
		for _, item := range sortedPullRequests(evidence.PullRequests) {
			if _, err := fmt.Fprintf(w, "%s  - %s\n", indent, formatPullRequestEvidence(item)); err != nil {
				return err
			}
		}
	}

	if len(evidence.Deployments) > 0 {
		if _, err := fmt.Fprintf(w, "%sdeployments:\n", indent); err != nil {
			return err
		}
		for _, item := range sortedDeployments(evidence.Deployments) {
			if _, err := fmt.Fprintf(w, "%s  - %s\n", indent, formatDeploymentEvidence(item)); err != nil {
				return err
			}
		}
	}

	return nil
}

func renderProviderEvidenceMarkdown(w io.Writer, evidence scm.ProviderEvidence) error {
	if len(evidence.PullRequests) == 0 && len(evidence.Deployments) == 0 {
		return nil
	}

	if len(evidence.PullRequests) > 0 {
		if _, err := fmt.Fprintln(w, "Pull requests:"); err != nil {
			return err
		}
		lines := make([]string, 0, len(evidence.PullRequests))
		for _, item := range sortedPullRequests(evidence.PullRequests) {
			lines = append(lines, formatPullRequestEvidence(item))
		}
		if err := renderMarkdownList(w, lines); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	if len(evidence.Deployments) > 0 {
		if _, err := fmt.Fprintln(w, "Deployments:"); err != nil {
			return err
		}
		lines := make([]string, 0, len(evidence.Deployments))
		for _, item := range sortedDeployments(evidence.Deployments) {
			lines = append(lines, formatDeploymentEvidence(item))
		}
		if err := renderMarkdownList(w, lines); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	return nil
}

func formatPullRequestEvidence(item scm.PullRequestEvidence) string {
	parts := make([]string, 0, 5)
	if strings.TrimSpace(item.ID) != "" {
		parts = append(parts, item.ID)
	}

	title := strings.TrimSpace(item.Title)
	if title != "" {
		parts = append(parts, title)
	}

	branchPair := formatBranchPair(item.SourceBranch, item.TargetBranch)
	if branchPair != "" {
		parts = append(parts, branchPair)
	}
	if state := strings.TrimSpace(item.State); state != "" {
		parts = append(parts, state)
	}
	if commit := shortHash(item.CommitHash); commit != "" {
		parts = append(parts, "commit "+commit)
	}
	if link := strings.TrimSpace(item.URL); link != "" {
		parts = append(parts, link)
	}

	return strings.Join(parts, " | ")
}

func formatDeploymentEvidence(item scm.DeploymentEvidence) string {
	parts := make([]string, 0, 5)
	if strings.TrimSpace(item.Environment) != "" {
		parts = append(parts, item.Environment)
	}
	if state := strings.TrimSpace(item.State); state != "" {
		parts = append(parts, state)
	}
	if ref := strings.TrimSpace(item.Ref); ref != "" {
		parts = append(parts, "ref "+ref)
	}
	if commit := shortHash(item.CommitHash); commit != "" {
		parts = append(parts, "commit "+commit)
	}
	if link := strings.TrimSpace(item.URL); link != "" {
		parts = append(parts, link)
	}
	if len(parts) == 0 {
		return strings.TrimSpace(item.ID)
	}
	if id := strings.TrimSpace(item.ID); id != "" {
		return id + " | " + strings.Join(parts, " | ")
	}
	return strings.Join(parts, " | ")
}

func formatBranchPair(source, target string) string {
	source = strings.TrimSpace(source)
	target = strings.TrimSpace(target)
	switch {
	case source != "" && target != "":
		return source + " -> " + target
	case source != "":
		return "source " + source
	case target != "":
		return "target " + target
	default:
		return ""
	}
}

func sortedPullRequests(values []scm.PullRequestEvidence) []scm.PullRequestEvidence {
	sorted := append([]scm.PullRequestEvidence(nil), values...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].ID == sorted[j].ID {
			return sorted[i].CommitHash < sorted[j].CommitHash
		}
		return sorted[i].ID < sorted[j].ID
	})
	return sorted
}

func sortedDeployments(values []scm.DeploymentEvidence) []scm.DeploymentEvidence {
	sorted := append([]scm.DeploymentEvidence(nil), values...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Environment == sorted[j].Environment {
			return sorted[i].ID < sorted[j].ID
		}
		return sorted[i].Environment < sorted[j].Environment
	})
	return sorted
}
