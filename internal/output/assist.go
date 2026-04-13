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
