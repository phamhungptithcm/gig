package ticket

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type Parser struct {
	pattern string
	regex   *regexp.Regexp
}

func NewParser(pattern string) (Parser, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return Parser{}, fmt.Errorf("invalid ticket pattern: %w", err)
	}

	return Parser{
		pattern: pattern,
		regex:   regex,
	}, nil
}

func (p Parser) Pattern() string {
	return p.pattern
}

func (p Parser) FindAll(text string) []string {
	matches := p.regex.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}

	unique := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		unique[strings.ToUpper(strings.TrimSpace(match))] = struct{}{}
	}

	tickets := make([]string, 0, len(unique))
	for ticketID := range unique {
		tickets = append(tickets, ticketID)
	}

	sort.Strings(tickets)
	return tickets
}

func (p Parser) Matches(ticketID, text string) bool {
	normalizedTicketID := strings.ToUpper(strings.TrimSpace(ticketID))
	for _, match := range p.FindAll(strings.ToUpper(text)) {
		if match == normalizedTicketID {
			return true
		}
	}

	return false
}

func (p Parser) Validate(ticketID string) error {
	normalizedTicketID := strings.ToUpper(strings.TrimSpace(ticketID))
	if normalizedTicketID == "" {
		return fmt.Errorf("ticket ID is required")
	}
	matches := p.FindAll(normalizedTicketID)
	if len(matches) != 1 || matches[0] != normalizedTicketID {
		return fmt.Errorf("ticket ID %q does not match expected pattern", ticketID)
	}

	return nil
}

func RegexPattern(ticketID string) string {
	return regexp.QuoteMeta(strings.ToUpper(strings.TrimSpace(ticketID)))
}
