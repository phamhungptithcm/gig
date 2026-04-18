package output

import (
	"fmt"
	"io"
	"strings"

	assistsvc "gig/internal/assistant"
)

func RenderAssistantAudit(w io.Writer, result assistsvc.AuditResult) error {
	if _, err := fmt.Fprintf(w, "Ticket Brief: %s\n", result.Bundle.TicketID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Scope: %s\n", result.Bundle.ScopeLabel); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Promotion: %s -> %s\n", result.Bundle.FromBranch, result.Bundle.ToBranch); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Audience: %s\n", result.Audience); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, strings.TrimSpace(result.Response))
	return err
}

func RenderAssistantRelease(w io.Writer, result assistsvc.ReleaseResult) error {
	if _, err := fmt.Fprintf(w, "Release Brief: %s\n", result.Bundle.ReleaseID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Scope: %s\n", result.Bundle.ScopeLabel); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Promotion: %s -> %s\n", result.Bundle.FromBranch, result.Bundle.ToBranch); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Audience: %s\n", result.Audience); err != nil {
		return err
	}
	if err := renderReleaseAudienceSummaries(w, result); err != nil {
		return err
	}
	if summary := formatReleaseEvidenceSummary(result.Bundle.EvidenceSummary); summary != "" {
		if _, err := fmt.Fprintf(w, "Evidence: %s\n", summary); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, strings.TrimSpace(result.Response))
	return err
}

func RenderAssistantResolve(w io.Writer, result assistsvc.ResolveResult) error {
	if _, err := fmt.Fprintf(w, "Conflict Brief: %s\n", result.Bundle.Status.Repository.Root); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Scope: %s\n", result.Bundle.ScopeLabel); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Operation: %s\n", result.Bundle.Status.Operation.Type); err != nil {
		return err
	}
	if result.Bundle.ScopedTicketID != "" {
		if _, err := fmt.Fprintf(w, "Scoped ticket: %s\n", result.Bundle.ScopedTicketID); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "Audience: %s\n", result.Audience); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, strings.TrimSpace(result.Response))
	return err
}

func formatReleaseEvidenceSummary(summary assistsvc.ReleaseEvidenceSummary) string {
	parts := make([]string, 0, 8)
	if summary.PullRequests > 0 {
		parts = append(parts, fmt.Sprintf("%d PR(s)", summary.PullRequests))
	}
	if summary.Deployments > 0 {
		parts = append(parts, fmt.Sprintf("%d deployment(s)", summary.Deployments))
	}
	if summary.Checks > 0 {
		value := fmt.Sprintf("%d check(s)", summary.Checks)
		if summary.FailingChecks > 0 {
			value += fmt.Sprintf(", %d not green", summary.FailingChecks)
		}
		parts = append(parts, value)
	}
	if summary.Hotspots > 0 {
		parts = append(parts, fmt.Sprintf("%d hotspot(s)", summary.Hotspots))
	}
	if summary.LinkedIssues > 0 {
		parts = append(parts, fmt.Sprintf("%d linked issue(s)", summary.LinkedIssues))
	}
	if summary.Releases > 0 {
		parts = append(parts, fmt.Sprintf("%d latest release reference(s)", summary.Releases))
	}
	if summary.OverlappingTickets > 0 {
		parts = append(parts, fmt.Sprintf("%d overlapping ticket(s)", summary.OverlappingTickets))
	}
	if summary.NewTicketsSinceLatestRelease > 0 {
		parts = append(parts, fmt.Sprintf("%d new ticket(s) since latest release", summary.NewTicketsSinceLatestRelease))
	}
	return strings.Join(parts, "; ")
}

func renderReleaseAudienceSummaries(w io.Writer, result assistsvc.ReleaseResult) error {
	switch result.Audience {
	case assistsvc.AudienceQA:
		return renderReleaseSummaryBlock(w, "Operator Summary", result.Bundle.OperatorSummary)
	case assistsvc.AudienceClient:
		return renderReleaseSummaryBlock(w, "Executive Summary", result.Bundle.ExecutiveSummary)
	default:
		if err := renderReleaseSummaryBlock(w, "Executive Summary", result.Bundle.ExecutiveSummary); err != nil {
			return err
		}
		return renderReleaseSummaryBlock(w, "Operator Summary", result.Bundle.OperatorSummary)
	}
}

func renderReleaseSummaryBlock(w io.Writer, heading string, lines []string) error {
	if len(lines) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(w, "%s:\n", heading); err != nil {
		return err
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "- %s\n", strings.TrimSpace(line)); err != nil {
			return err
		}
	}
	return nil
}
