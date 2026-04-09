package conflict

import (
	"bytes"
	"path/filepath"
	"strings"

	"gig/internal/scm"
)

func assessRisk(path string, block Block) RiskAssessment {
	lowerPath := strings.ToLower(path)

	duplicates := duplicateLines(block.Current, block.Incoming)
	if isHighRiskPath(lowerPath) {
		notes := []string{"Review the final merged content line by line before staging."}
		if len(duplicates) > 0 {
			notes = append(notes, "Drop duplicate lines manually after using an accept-both action.")
		}
		return RiskAssessment{
			Severity:       SeverityHigh,
			Summary:        "High-risk file type. Prefer manual review even when one side looks obviously correct.",
			ReviewNotes:    notes,
			DuplicateLines: duplicates,
		}
	}

	if len(duplicates) > 0 || isMediumRiskPath(lowerPath) || looksLikeImportConflict(block) {
		notes := []string{"Check for duplicates, ordering issues, and config key collisions after combine."}
		if len(duplicates) > 0 {
			notes = append(notes, "The same line exists on both sides and may need to be kept only once.")
		}
		return RiskAssessment{
			Severity:       SeverityMedium,
			Summary:        "Combine can work, but you should inspect the final block before staging.",
			ReviewNotes:    notes,
			DuplicateLines: duplicates,
		}
	}

	return RiskAssessment{
		Severity:    SeverityLow,
		Summary:     "This looks like a straightforward line conflict. Accept one side or combine, then verify the result.",
		ReviewNotes: []string{"Confirm the final block still compiles or parses as expected."},
	}
}

func deriveScopeWarnings(scopeTicketID string, block Block, operation scm.ConflictOperationState) []string {
	if scopeTicketID == "" {
		return nil
	}

	warnings := make([]string, 0, 2)
	currentHasScope := containsTicket(operation.CurrentSide.TicketIDs, scopeTicketID)
	incomingHasScope := containsTicket(operation.IncomingSide.TicketIDs, scopeTicketID)

	switch {
	case incomingHasScope && !currentHasScope:
		warnings = append(warnings, "Accept current would drop the only visible scoped ticket on this conflict side.")
	case currentHasScope && !incomingHasScope:
		warnings = append(warnings, "Accept incoming would replace a scoped ticket side with changes from another ticket or branch.")
	}

	if incomingHasScope && len(operation.IncomingSide.TicketIDs) > 1 {
		warnings = append(warnings, "Incoming side also references other tickets. Double-check whether they should travel together.")
	}

	return warnings
}

func duplicateLines(current, incoming []byte) []string {
	currentLines := normalizedNonEmptyLines(current)
	incomingLines := normalizedNonEmptyLines(incoming)
	if len(currentLines) == 0 || len(incomingLines) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(currentLines))
	for _, line := range currentLines {
		seen[line] = struct{}{}
	}

	duplicates := make([]string, 0)
	added := make(map[string]struct{})
	for _, line := range incomingLines {
		if _, ok := seen[line]; !ok {
			continue
		}
		if _, ok := added[line]; ok {
			continue
		}
		added[line] = struct{}{}
		duplicates = append(duplicates, line)
	}

	return duplicates
}

func normalizedNonEmptyLines(content []byte) []string {
	lines := bytes.Split(content, []byte{'\n'})
	values := make([]string, 0, len(lines))
	for _, line := range lines {
		text := strings.TrimSpace(strings.TrimSuffix(string(line), "\r"))
		if text == "" {
			continue
		}
		values = append(values, text)
	}
	return values
}

func looksLikeImportConflict(block Block) bool {
	for _, line := range normalizedNonEmptyLines(append(append([]byte{}, block.Current...), block.Incoming...)) {
		switch {
		case strings.HasPrefix(line, "import "),
			strings.HasPrefix(line, "from "),
			strings.HasPrefix(line, "using "),
			strings.HasPrefix(line, "require("):
			return true
		}
	}
	return false
}

func isHighRiskPath(lowerPath string) bool {
	base := filepath.Base(lowerPath)
	switch {
	case strings.HasSuffix(lowerPath, ".sql"),
		strings.Contains(lowerPath, "/db/"),
		strings.Contains(lowerPath, "/migrations/"),
		strings.Contains(lowerPath, "/migration/"),
		base == "package-lock.json",
		base == "yarn.lock",
		base == "pnpm-lock.yaml",
		base == "cargo.lock",
		base == "go.sum",
		base == "composer.lock",
		base == "poetry.lock":
		return true
	default:
		return false
	}
}

func isMediumRiskPath(lowerPath string) bool {
	base := filepath.Base(lowerPath)
	switch {
	case strings.HasSuffix(lowerPath, ".json"),
		strings.HasSuffix(lowerPath, ".yaml"),
		strings.HasSuffix(lowerPath, ".yml"),
		strings.HasSuffix(lowerPath, ".toml"),
		strings.HasSuffix(lowerPath, ".ini"),
		strings.HasSuffix(lowerPath, ".env"),
		strings.Contains(lowerPath, "config"),
		strings.HasSuffix(lowerPath, ".md"),
		base == "dockerfile":
		return true
	default:
		return false
	}
}

func containsTicket(ticketIDs []string, ticketID string) bool {
	for _, current := range ticketIDs {
		if current == ticketID {
			return true
		}
	}
	return false
}
