package dependency

import (
	"strings"

	"gig/internal/ticket"
)

type Parser struct {
	tickets ticket.Parser
}

func NewParser(tickets ticket.Parser) Parser {
	return Parser{tickets: tickets}
}

func (p Parser) ParseCommitMessage(ticketID, commitHash, commitSubject, message string) ([]DeclaredDependency, error) {
	if err := p.tickets.Validate(ticketID); err != nil {
		return nil, err
	}

	normalizedTicketID := strings.ToUpper(strings.TrimSpace(ticketID))
	commitHash = strings.TrimSpace(commitHash)
	commitSubject = strings.TrimSpace(commitSubject)

	dependencies := make([]DeclaredDependency, 0)
	seen := make(map[string]struct{})

	for _, line := range strings.Split(message, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), TrailerDependsOn) {
			continue
		}

		for _, rawValue := range strings.Split(value, ",") {
			dependsOn := strings.ToUpper(strings.TrimSpace(rawValue))
			if dependsOn == "" || dependsOn == normalizedTicketID {
				continue
			}
			if err := p.tickets.Validate(dependsOn); err != nil {
				continue
			}
			if _, ok := seen[dependsOn]; ok {
				continue
			}

			seen[dependsOn] = struct{}{}
			dependencies = append(dependencies, DeclaredDependency{
				TicketID:      normalizedTicketID,
				DependsOn:     dependsOn,
				CommitHash:    commitHash,
				CommitSubject: commitSubject,
				TrailerKey:    TrailerDependsOn,
			})
		}
	}

	return dependencies, nil
}
