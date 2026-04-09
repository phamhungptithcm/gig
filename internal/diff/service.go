package diff

import (
	"context"
	"errors"
	"sort"
	"strings"

	"gig/internal/repo"
	"gig/internal/scm"
	"gig/internal/ticket"
)

type adapterProvider interface {
	For(repoType scm.Type) (scm.Adapter, bool)
}

type Service struct {
	discoverer repo.Discoverer
	adapters   adapterProvider
	parser     ticket.Parser
}

type Result struct {
	Repository scm.Repository
	Compare    scm.CompareResult
}

func NewService(discoverer repo.Discoverer, adapters adapterProvider, parser ticket.Parser) *Service {
	return &Service{
		discoverer: discoverer,
		adapters:   adapters,
		parser:     parser,
	}
}

func (s *Service) CompareTicket(ctx context.Context, path, ticketID, fromBranch, toBranch string) ([]Result, error) {
	if err := s.parser.Validate(ticketID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(fromBranch) == "" || strings.TrimSpace(toBranch) == "" {
		return nil, errors.New("both --from and --to branches are required")
	}

	repositories, err := s.discoverer.Discover(ctx, path)
	if err != nil {
		return nil, err
	}

	return s.CompareTicketInRepositories(ctx, repositories, ticketID, fromBranch, toBranch)
}

func (s *Service) CompareTicketInRepositories(ctx context.Context, repositories []scm.Repository, ticketID, fromBranch, toBranch string) ([]Result, error) {
	if err := s.parser.Validate(ticketID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(fromBranch) == "" || strings.TrimSpace(toBranch) == "" {
		return nil, errors.New("both --from and --to branches are required")
	}

	results := make([]Result, 0, len(repositories))
	for _, repository := range repositories {
		adapter, ok := s.adapters.For(repository.Type)
		if !ok {
			continue
		}

		compareResult, err := adapter.CompareBranches(ctx, repository.Root, scm.CompareQuery{
			TicketID:   ticketID,
			FromBranch: fromBranch,
			ToBranch:   toBranch,
		})
		if err != nil {
			if errors.Is(err, scm.ErrUnsupported) {
				continue
			}
			return nil, err
		}

		if len(compareResult.SourceCommits) == 0 && len(compareResult.TargetCommits) == 0 && len(compareResult.MissingCommits) == 0 {
			continue
		}

		results = append(results, Result{
			Repository: repository,
			Compare:    compareResult,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Repository.Root < results[j].Repository.Root
	})

	return results, nil
}
