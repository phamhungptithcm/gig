package scm

import (
	"context"
	"errors"
)

type Type string

const (
	TypeGit Type = "git"
	TypeSVN Type = "svn"
)

var ErrUnsupported = errors.New("operation not supported")

type Repository struct {
	Name          string
	Root          string
	Type          Type
	CurrentBranch string
}

type Commit struct {
	Hash     string
	Subject  string
	Branches []string
}

func (c Commit) ShortHash() string {
	if len(c.Hash) <= 8 {
		return c.Hash
	}

	return c.Hash[:8]
}

type SearchQuery struct {
	TicketID string
	Branch   string
}

type CompareQuery struct {
	TicketID   string
	FromBranch string
	ToBranch   string
}

type CompareResult struct {
	FromBranch     string
	ToBranch       string
	SourceCommits  []Commit
	TargetCommits  []Commit
	MissingCommits []Commit
}

type CherryPickPlan struct {
	TargetBranch string
	Commits      []Commit
}

type Adapter interface {
	Type() Type
	DetectRoot(path string) (string, bool, error)
	IsRepository(path string) (bool, error)
	CurrentBranch(ctx context.Context, repoRoot string) (string, error)
	SearchCommits(ctx context.Context, repoRoot string, query SearchQuery) ([]Commit, error)
	CompareBranches(ctx context.Context, repoRoot string, query CompareQuery) (CompareResult, error)
	PrepareCherryPick(ctx context.Context, repoRoot string, plan CherryPickPlan) error
}
