package dependency

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"gig/internal/scm"
	"gig/internal/ticket"
)

type adapterProvider interface {
	For(repoType scm.Type) (scm.Adapter, bool)
}

type refChecker interface {
	RefExists(ctx context.Context, repoRoot, ref string) (bool, error)
}

type Resolver struct {
	adapters adapterProvider
	tickets  ticket.Parser
}

func NewResolver(adapters adapterProvider, tickets ticket.Parser) *Resolver {
	return &Resolver{
		adapters: adapters,
		tickets:  tickets,
	}
}

func (r *Resolver) ResolveInRepositories(ctx context.Context, repositories []scm.Repository, declared []DeclaredDependency, fromBranch, toBranch string) ([]Resolution, error) {
	if strings.TrimSpace(fromBranch) == "" || strings.TrimSpace(toBranch) == "" {
		return nil, fmt.Errorf("both --from and --to branches are required")
	}

	aggregated := make(map[string]*Resolution)
	for _, dependency := range declared {
		if err := r.tickets.Validate(dependency.DependsOn); err != nil {
			continue
		}

		entry, ok := aggregated[dependency.DependsOn]
		if !ok {
			entry = &Resolution{
				TicketID:        dependency.TicketID,
				DependsOn:       dependency.DependsOn,
				EvidenceCommits: make([]string, 0, 2),
			}
			aggregated[dependency.DependsOn] = entry
		}

		if dependency.TicketID != "" {
			entry.TicketID = dependency.TicketID
		}
		if !contains(entry.EvidenceCommits, dependency.CommitHash) {
			entry.EvidenceCommits = append(entry.EvidenceCommits, dependency.CommitHash)
		}
	}

	resolutions := make([]Resolution, 0, len(aggregated))
	for dependsOn, resolution := range aggregated {
		foundInSource, foundInTarget, err := r.findTicketPresence(ctx, repositories, dependsOn, fromBranch, toBranch)
		if err != nil {
			return nil, err
		}

		resolution.FoundInSource = foundInSource
		resolution.FoundInTarget = foundInTarget
		switch {
		case foundInTarget:
			resolution.Status = StatusSatisfied
		case foundInSource:
			resolution.Status = StatusMissingTarget
		default:
			resolution.Status = StatusUnresolved
		}

		sort.Strings(resolution.EvidenceCommits)
		resolutions = append(resolutions, *resolution)
	}

	sort.Slice(resolutions, func(i, j int) bool {
		return resolutions[i].DependsOn < resolutions[j].DependsOn
	})

	return resolutions, nil
}

func (r *Resolver) findTicketPresence(ctx context.Context, repositories []scm.Repository, ticketID, fromBranch, toBranch string) (bool, bool, error) {
	foundInSource := false
	foundInTarget := false

	for _, repository := range repositories {
		adapter, ok := r.adapters.For(repository.Type)
		if !ok {
			continue
		}

		if !foundInSource {
			present, err := ticketPresentOnBranch(ctx, adapter, repository.Root, fromBranch, ticketID)
			if err != nil {
				return false, false, err
			}
			foundInSource = present
		}

		if !foundInTarget {
			present, err := ticketPresentOnBranch(ctx, adapter, repository.Root, toBranch, ticketID)
			if err != nil {
				return false, false, err
			}
			foundInTarget = present
		}

		if foundInSource && foundInTarget {
			break
		}
	}

	return foundInSource, foundInTarget, nil
}

func ticketPresentOnBranch(ctx context.Context, adapter scm.Adapter, repoRoot, branch, ticketID string) (bool, error) {
	if checker, ok := adapter.(refChecker); ok {
		exists, err := checker.RefExists(ctx, repoRoot, branch)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, nil
		}
	}

	commits, err := adapter.SearchCommits(ctx, repoRoot, scm.SearchQuery{
		TicketID: ticketID,
		Branch:   branch,
	})
	if err != nil {
		return false, err
	}

	return len(commits) > 0, nil
}

func contains(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
