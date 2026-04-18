package output

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"gig/internal/scm"
)

func renderProviderEvidence(w io.Writer, evidence scm.ProviderEvidence, indent string) error {
	if evidence.IsZero() {
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

	if len(evidence.Checks) > 0 {
		if _, err := fmt.Fprintf(w, "%schecks:\n", indent); err != nil {
			return err
		}
		for _, item := range sortedChecks(evidence.Checks) {
			if _, err := fmt.Fprintf(w, "%s  - %s\n", indent, formatCheckEvidence(item)); err != nil {
				return err
			}
		}
	}

	if len(evidence.Issues) > 0 {
		if _, err := fmt.Fprintf(w, "%slinked issues:\n", indent); err != nil {
			return err
		}
		for _, item := range sortedIssues(evidence.Issues) {
			if _, err := fmt.Fprintf(w, "%s  - %s\n", indent, formatIssueEvidence(item)); err != nil {
				return err
			}
		}
	}

	if len(evidence.Releases) > 0 {
		if _, err := fmt.Fprintf(w, "%sreleases:\n", indent); err != nil {
			return err
		}
		for _, item := range sortedReleases(evidence.Releases) {
			if _, err := fmt.Fprintf(w, "%s  - %s\n", indent, formatReleaseEvidence(item)); err != nil {
				return err
			}
		}
	}

	return nil
}

func renderProviderEvidenceMarkdown(w io.Writer, evidence scm.ProviderEvidence) error {
	if evidence.IsZero() {
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

	if len(evidence.Checks) > 0 {
		if _, err := fmt.Fprintln(w, "Checks:"); err != nil {
			return err
		}
		lines := make([]string, 0, len(evidence.Checks))
		for _, item := range sortedChecks(evidence.Checks) {
			lines = append(lines, formatCheckEvidence(item))
		}
		if err := renderMarkdownList(w, lines); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	if len(evidence.Issues) > 0 {
		if _, err := fmt.Fprintln(w, "Linked issues:"); err != nil {
			return err
		}
		lines := make([]string, 0, len(evidence.Issues))
		for _, item := range sortedIssues(evidence.Issues) {
			lines = append(lines, formatIssueEvidence(item))
		}
		if err := renderMarkdownList(w, lines); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	if len(evidence.Releases) > 0 {
		if _, err := fmt.Fprintln(w, "Releases:"); err != nil {
			return err
		}
		lines := make([]string, 0, len(evidence.Releases))
		for _, item := range sortedReleases(evidence.Releases) {
			lines = append(lines, formatReleaseEvidence(item))
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

func formatCheckEvidence(item scm.CheckEvidence) string {
	parts := make([]string, 0, 4)
	if context := strings.TrimSpace(item.Context); context != "" {
		parts = append(parts, context)
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

func formatIssueEvidence(item scm.IssueEvidence) string {
	parts := make([]string, 0, 5)
	if id := strings.TrimSpace(item.ID); id != "" {
		parts = append(parts, id)
	}
	if title := strings.TrimSpace(item.Title); title != "" {
		parts = append(parts, title)
	}
	if state := strings.TrimSpace(item.State); state != "" {
		parts = append(parts, state)
	}
	if len(item.Labels) > 0 {
		parts = append(parts, "labels "+strings.Join(item.Labels, ", "))
	}
	if link := strings.TrimSpace(item.URL); link != "" {
		parts = append(parts, link)
	}
	return strings.Join(parts, " | ")
}

func formatReleaseEvidence(item scm.ReleaseEvidence) string {
	parts := make([]string, 0, 6)
	if tag := firstNonEmpty(item.Tag, item.ID); tag != "" {
		parts = append(parts, tag)
	}
	if name := strings.TrimSpace(item.Name); name != "" && name != strings.TrimSpace(item.Tag) {
		parts = append(parts, name)
	}
	if state := strings.TrimSpace(item.State); state != "" {
		parts = append(parts, state)
	}
	if target := strings.TrimSpace(item.Target); target != "" {
		parts = append(parts, "target "+target)
	}
	if len(item.TicketIDs) > 0 {
		parts = append(parts, "tickets "+strings.Join(item.TicketIDs, ", "))
	}
	if publishedAt := strings.TrimSpace(item.PublishedAt); publishedAt != "" {
		parts = append(parts, publishedAt)
	}
	if link := strings.TrimSpace(item.URL); link != "" {
		parts = append(parts, link)
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

func sortedChecks(values []scm.CheckEvidence) []scm.CheckEvidence {
	sorted := append([]scm.CheckEvidence(nil), values...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].CommitHash == sorted[j].CommitHash {
			return sorted[i].Context < sorted[j].Context
		}
		return sorted[i].CommitHash < sorted[j].CommitHash
	})
	return sorted
}

func sortedIssues(values []scm.IssueEvidence) []scm.IssueEvidence {
	sorted := append([]scm.IssueEvidence(nil), values...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})
	return sorted
}

func sortedReleases(values []scm.ReleaseEvidence) []scm.ReleaseEvidence {
	sorted := append([]scm.ReleaseEvidence(nil), values...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].PublishedAt == sorted[j].PublishedAt {
			return sorted[i].Tag < sorted[j].Tag
		}
		return sorted[i].PublishedAt > sorted[j].PublishedAt
	})
	return sorted
}

func hasProviderEvidence(evidence *scm.ProviderEvidence) bool {
	return evidence != nil && !evidence.IsZero()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
