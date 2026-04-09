package ticket

import (
	"context"
	"errors"
	"sort"

	"gig/internal/repo"
	"gig/internal/scm"
)

type adapterProvider interface {
	For(repoType scm.Type) (scm.Adapter, bool)
}

type Service struct {
	discoverer repo.Discoverer
	adapters   adapterProvider
	parser     Parser
}

type SearchResult struct {
	Repository scm.Repository
	Commits    []scm.Commit
}

func NewService(discoverer repo.Discoverer, adapters adapterProvider, parser Parser) *Service {
	return &Service{
		discoverer: discoverer,
		adapters:   adapters,
		parser:     parser,
	}
}

func (s *Service) Find(ctx context.Context, path, ticketID string) ([]SearchResult, error) {
	if err := s.parser.Validate(ticketID); err != nil {
		return nil, err
	}

	repositories, err := s.discoverer.Discover(ctx, path)
	if err != nil {
		return nil, err
	}

	return s.FindInRepositories(ctx, repositories, ticketID)
}

func (s *Service) FindInRepositories(ctx context.Context, repositories []scm.Repository, ticketID string) ([]SearchResult, error) {
	if err := s.parser.Validate(ticketID); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(repositories))
	for _, repository := range repositories {
		adapter, ok := s.adapters.For(repository.Type)
		if !ok {
			continue
		}

		commits, err := adapter.SearchCommits(ctx, repository.Root, scm.SearchQuery{TicketID: ticketID})
		if err != nil {
			if errors.Is(err, scm.ErrUnsupported) {
				continue
			}
			return nil, err
		}

		if len(commits) == 0 {
			continue
		}

		results = append(results, SearchResult{
			Repository: repository,
			Commits:    commits,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Repository.Root < results[j].Repository.Root
	})

	return results, nil
}
