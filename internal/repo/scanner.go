package repo

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"gig/internal/scm"
)

type Discoverer interface {
	Discover(ctx context.Context, path string) ([]scm.Repository, error)
}

type Scanner struct {
	registry *scm.Registry
}

func NewScanner(registry *scm.Registry) *Scanner {
	return &Scanner{registry: registry}
}

func (s *Scanner) Discover(ctx context.Context, path string) ([]scm.Repository, error) {
	basePath, err := normalizePath(path)
	if err != nil {
		return nil, err
	}

	if repo, ok, err := s.detectEnclosingRepository(ctx, basePath); err != nil {
		return nil, err
	} else if ok {
		return []scm.Repository{repo}, nil
	}

	repositories := make([]scm.Repository, 0)
	seen := map[string]struct{}{}

	walkErr := filepath.WalkDir(basePath, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if !entry.IsDir() {
			return nil
		}

		switch entry.Name() {
		case ".git", ".svn":
			return filepath.SkipDir
		}

		for _, adapter := range s.registry.All() {
			ok, err := adapter.IsRepository(current)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}

			if _, exists := seen[current]; exists {
				break
			}

			seen[current] = struct{}{}
			repositories = append(repositories, s.describeRepository(ctx, adapter, current))
			break
		}

		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	sort.Slice(repositories, func(i, j int) bool {
		return repositories[i].Root < repositories[j].Root
	})

	return repositories, nil
}

func (s *Scanner) detectEnclosingRepository(ctx context.Context, path string) (scm.Repository, bool, error) {
	var (
		foundRepo scm.Repository
		found     bool
	)

	for _, adapter := range s.registry.All() {
		root, ok, err := adapter.DetectRoot(path)
		if err != nil {
			return scm.Repository{}, false, err
		}
		if !ok {
			continue
		}

		if !found || len(root) > len(foundRepo.Root) {
			foundRepo = s.describeRepository(ctx, adapter, root)
			found = true
		}
	}

	return foundRepo, found, nil
}

func (s *Scanner) describeRepository(ctx context.Context, adapter scm.Adapter, root string) scm.Repository {
	repository := scm.Repository{
		Name: filepath.Base(root),
		Root: root,
		Type: adapter.Type(),
	}

	branch, err := adapter.CurrentBranch(ctx, root)
	if err == nil {
		repository.CurrentBranch = branch
	}

	return repository
}

func normalizePath(path string) (string, error) {
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return absPath, nil
	}

	return filepath.Dir(absPath), nil
}
