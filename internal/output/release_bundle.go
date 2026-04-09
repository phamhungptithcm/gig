package output

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	manifestsvc "gig/internal/manifest"
	plansvc "gig/internal/plan"
)

func RenderReleaseVerificationBatch(w io.Writer, releaseID, snapshotDir string, verifications []plansvc.Verification) error {
	safeCount, warningCount, blockedCount := summarizeVerificationVerdicts(verifications)
	if _, err := fmt.Fprintf(w, "Release verification %s\n", releaseID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Snapshot directory: %s\n", snapshotDir); err != nil {
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

func RenderReleasePacketBundleMarkdownForRelease(w io.Writer, releaseID, snapshotDir string, packets []manifestsvc.ReleasePacket) error {
	safeCount, warningCount, blockedCount := summarizePacketVerdicts(packets)
	if _, err := fmt.Fprintf(w, "# Release Packet Bundle: %s\n\n", releaseID); err != nil {
		return err
	}
	lines := []string{
		fmt.Sprintf("Snapshot directory: `%s`", snapshotDir),
		fmt.Sprintf("Tickets: `%d`", len(packets)),
		fmt.Sprintf("Safe: `%d`", safeCount),
		fmt.Sprintf("Warning: `%d`", warningCount),
		fmt.Sprintf("Blocked: `%d`", blockedCount),
	}
	if err := renderMarkdownList(w, lines); err != nil {
		return err
	}

	for i, packet := range packets {
		separator := "\n---\n\n"
		if i > 0 {
			separator = "\n\n---\n\n"
		}
		if _, err := fmt.Fprint(w, separator); err != nil {
			return err
		}
		var rendered bytes.Buffer
		if err := RenderReleasePacketMarkdown(&rendered, packet); err != nil {
			return err
		}
		if _, err := io.WriteString(w, strings.TrimRight(rendered.String(), "\n")); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(w)
	return err
}
