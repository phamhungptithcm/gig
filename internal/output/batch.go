package output

import (
	"fmt"
	"io"

	manifestsvc "gig/internal/manifest"
	plansvc "gig/internal/plan"
)

func RenderPromotionPlanBatch(w io.Writer, plans []plansvc.PromotionPlan) error {
	safeCount, warningCount, blockedCount := summarizePlanVerdicts(plans)
	if _, err := fmt.Fprintln(w, "Batch promotion plan"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Tickets: %d\n", len(plans)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Safe: %d\n", safeCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Warning: %d\n", warningCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Blocked: %d\n\n", blockedCount); err != nil {
		return err
	}

	for i, plan := range plans {
		if i > 0 {
			if _, err := fmt.Fprint(w, "\n---\n"); err != nil {
				return err
			}
		}
		if err := RenderPromotionPlan(w, plan); err != nil {
			return err
		}
	}

	return nil
}

func RenderVerificationBatch(w io.Writer, verifications []plansvc.Verification) error {
	safeCount, warningCount, blockedCount := summarizeVerificationVerdicts(verifications)
	if _, err := fmt.Fprintln(w, "Batch verification"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Tickets: %d\n", len(verifications)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Safe: %d\n", safeCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Warning: %d\n", warningCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Blocked: %d\n\n", blockedCount); err != nil {
		return err
	}

	for i, verification := range verifications {
		if i > 0 {
			if _, err := fmt.Fprint(w, "\n---\n"); err != nil {
				return err
			}
		}
		if err := RenderVerification(w, verification); err != nil {
			return err
		}
	}

	return nil
}

func RenderReleasePacketBundleMarkdown(w io.Writer, packets []manifestsvc.ReleasePacket) error {
	safeCount, warningCount, blockedCount := summarizePacketVerdicts(packets)
	if _, err := fmt.Fprintln(w, "# Release Packet Bundle"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	lines := []string{
		fmt.Sprintf("Tickets: `%d`", len(packets)),
		fmt.Sprintf("Safe: `%d`", safeCount),
		fmt.Sprintf("Warning: `%d`", warningCount),
		fmt.Sprintf("Blocked: `%d`", blockedCount),
	}
	if err := renderMarkdownList(w, lines); err != nil {
		return err
	}

	for _, packet := range packets {
		if _, err := fmt.Fprint(w, "\n---\n\n"); err != nil {
			return err
		}
		if err := RenderReleasePacketMarkdown(w, packet); err != nil {
			return err
		}
	}

	return nil
}

func summarizePlanVerdicts(plans []plansvc.PromotionPlan) (safeCount, warningCount, blockedCount int) {
	for _, plan := range plans {
		switch plan.Verdict {
		case plansvc.VerdictSafe:
			safeCount++
		case plansvc.VerdictWarning:
			warningCount++
		case plansvc.VerdictBlocked:
			blockedCount++
		}
	}
	return safeCount, warningCount, blockedCount
}

func summarizeVerificationVerdicts(verifications []plansvc.Verification) (safeCount, warningCount, blockedCount int) {
	for _, verification := range verifications {
		switch verification.Verdict {
		case plansvc.VerdictSafe:
			safeCount++
		case plansvc.VerdictWarning:
			warningCount++
		case plansvc.VerdictBlocked:
			blockedCount++
		}
	}
	return safeCount, warningCount, blockedCount
}

func summarizePacketVerdicts(packets []manifestsvc.ReleasePacket) (safeCount, warningCount, blockedCount int) {
	for _, packet := range packets {
		switch packet.Verdict {
		case plansvc.VerdictSafe:
			safeCount++
		case plansvc.VerdictWarning:
			warningCount++
		case plansvc.VerdictBlocked:
			blockedCount++
		}
	}
	return safeCount, warningCount, blockedCount
}
