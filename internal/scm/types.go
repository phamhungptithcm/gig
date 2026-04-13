package scm

import (
	"context"
	"errors"
)

type Type string

const (
	TypeGit         Type = "git"
	TypeGitHub      Type = "github"
	TypeGitLab      Type = "gitlab"
	TypeBitbucket   Type = "bitbucket"
	TypeAzureDevOps Type = "azure-devops"
	TypeSVN         Type = "svn"
	TypeRemoteSVN   Type = "remote-svn"
)

func (t Type) IsRemote() bool {
	switch t {
	case TypeGitHub, TypeGitLab, TypeBitbucket, TypeAzureDevOps, TypeRemoteSVN:
		return true
	default:
		return false
	}
}

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

type EvidenceQuery struct {
	TicketID string
	Commits  []Commit
	Branches []string
}

type PullRequestEvidence struct {
	ID           string `json:"id"`
	Title        string `json:"title,omitempty"`
	State        string `json:"state,omitempty"`
	SourceBranch string `json:"sourceBranch,omitempty"`
	TargetBranch string `json:"targetBranch,omitempty"`
	URL          string `json:"url,omitempty"`
	CommitHash   string `json:"commitHash,omitempty"`
}

type DeploymentEvidence struct {
	ID          string `json:"id"`
	Environment string `json:"environment,omitempty"`
	State       string `json:"state,omitempty"`
	Ref         string `json:"ref,omitempty"`
	URL         string `json:"url,omitempty"`
	CommitHash  string `json:"commitHash,omitempty"`
}

type ProviderEvidence struct {
	PullRequests []PullRequestEvidence `json:"pullRequests,omitempty"`
	Deployments  []DeploymentEvidence  `json:"deployments,omitempty"`
}

func (e ProviderEvidence) IsZero() bool {
	return len(e.PullRequests) == 0 && len(e.Deployments) == 0
}

func NormalizeProviderEvidence(e *ProviderEvidence) *ProviderEvidence {
	if e == nil || e.IsZero() {
		return nil
	}
	return e
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

type ProtectedBranchProvider interface {
	ProtectedBranches(ctx context.Context, repoRoot string) ([]string, error)
}

type ProviderEvidenceProvider interface {
	ProviderEvidence(ctx context.Context, repoRoot string, query EvidenceQuery) (ProviderEvidence, error)
}
