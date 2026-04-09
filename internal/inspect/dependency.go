package inspect

import (
	"context"
	"sort"
	"strings"

	depsvc "gig/internal/dependency"
	"gig/internal/scm"
)

type commitMessageProvider interface {
	CommitMessages(ctx context.Context, repoRoot string, hashes []string) (map[string]string, error)
}

func (s *Service) loadDeclaredDependencies(ctx context.Context, adapter scm.Adapter, repoRoot, ticketID string, commits []scm.Commit) ([]depsvc.DeclaredDependency, error) {
	messageProvider, ok := adapter.(commitMessageProvider)
	if !ok {
		return nil, nil
	}

	hashes := make([]string, 0, len(commits))
	for _, commit := range commits {
		hash := strings.TrimSpace(commit.Hash)
		if hash == "" {
			continue
		}
		hashes = append(hashes, hash)
	}

	messagesByHash, err := messageProvider.CommitMessages(ctx, repoRoot, hashes)
	if err != nil {
		return nil, err
	}

	parser := depsvc.NewParser(s.parser)
	dependencies := make([]depsvc.DeclaredDependency, 0)
	seen := make(map[string]struct{})

	for _, commit := range commits {
		message, ok := messagesByHash[commit.Hash]
		if !ok {
			continue
		}

		declared, err := parser.ParseCommitMessage(ticketID, commit.Hash, commit.Subject, message)
		if err != nil {
			return nil, err
		}

		for _, dependency := range declared {
			key := dependency.DependsOn + "|" + dependency.CommitHash
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			dependencies = append(dependencies, dependency)
		}
	}

	sort.Slice(dependencies, func(i, j int) bool {
		if dependencies[i].DependsOn == dependencies[j].DependsOn {
			return dependencies[i].CommitHash < dependencies[j].CommitHash
		}
		return dependencies[i].DependsOn < dependencies[j].DependsOn
	})

	return dependencies, nil
}
