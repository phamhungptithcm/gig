package inspect

import (
	"context"
	"errors"
	"sort"
	"strings"

	depsvc "gig/internal/dependency"
	"gig/internal/repo"
	"gig/internal/scm"
	"gig/internal/ticket"
)

type adapterProvider interface {
	For(repoType scm.Type) (scm.Adapter, bool)
}

type commitFileProvider interface {
	CommitFiles(ctx context.Context, repoRoot string, hashes []string) (map[string][]string, error)
}

type refChecker interface {
	RefExists(ctx context.Context, repoRoot, ref string) (bool, error)
}

type Service struct {
	discoverer repo.Discoverer
	adapters   adapterProvider
	parser     ticket.Parser
}

type RiskSignal struct {
	Code     string
	Level    string
	Summary  string
	Examples []string
}

type RepositoryInspection struct {
	Repository           scm.Repository
	Commits              []scm.Commit
	Branches             []string
	RiskSignals          []RiskSignal
	DeclaredDependencies []depsvc.DeclaredDependency
}

type Environment struct {
	Name   string
	Branch string
}

type EnvironmentState string

const (
	EnvStatePresent       EnvironmentState = "present"
	EnvStateAligned       EnvironmentState = "aligned"
	EnvStateBehind        EnvironmentState = "behind"
	EnvStateNotPresent    EnvironmentState = "not-present"
	EnvStateBranchMissing EnvironmentState = "branch-missing"
)

type EnvironmentResult struct {
	Environment         Environment
	State               EnvironmentState
	CommitCount         int
	MissingFromPrevious int
}

type RepositoryEnvironmentStatus struct {
	Repository           scm.Repository
	Commits              []scm.Commit
	Branches             []string
	RiskSignals          []RiskSignal
	DeclaredDependencies []depsvc.DeclaredDependency
	Statuses             []EnvironmentResult
}

func NewService(discoverer repo.Discoverer, adapters adapterProvider, parser ticket.Parser) *Service {
	return &Service{
		discoverer: discoverer,
		adapters:   adapters,
		parser:     parser,
	}
}

func (s *Service) Inspect(ctx context.Context, path, ticketID string) ([]RepositoryInspection, int, error) {
	if err := s.parser.Validate(ticketID); err != nil {
		return nil, 0, err
	}

	repositories, err := s.discoverer.Discover(ctx, path)
	if err != nil {
		return nil, 0, err
	}

	results, err := s.InspectInRepositories(ctx, repositories, ticketID)
	return results, len(repositories), err
}

func (s *Service) InspectInRepositories(ctx context.Context, repositories []scm.Repository, ticketID string) ([]RepositoryInspection, error) {
	if err := s.parser.Validate(ticketID); err != nil {
		return nil, err
	}

	results := make([]RepositoryInspection, 0, len(repositories))
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

		filesByCommit, err := s.loadCommitFiles(ctx, adapter, repository.Root, commits)
		if err != nil {
			return nil, err
		}
		declaredDependencies, err := s.loadDeclaredDependencies(ctx, adapter, repository.Root, ticketID, commits)
		if err != nil {
			return nil, err
		}

		results = append(results, RepositoryInspection{
			Repository:           repository,
			Commits:              commits,
			Branches:             collectBranches(commits),
			RiskSignals:          inferRiskSignals(filesByCommit),
			DeclaredDependencies: declaredDependencies,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Repository.Root < results[j].Repository.Root
	})

	return results, nil
}

func (s *Service) EnvironmentStatus(ctx context.Context, path, ticketID string, environments []Environment) ([]RepositoryEnvironmentStatus, int, error) {
	if err := s.parser.Validate(ticketID); err != nil {
		return nil, 0, err
	}

	repositories, err := s.discoverer.Discover(ctx, path)
	if err != nil {
		return nil, 0, err
	}

	results, err := s.EnvironmentStatusInRepositories(ctx, repositories, ticketID, environments)
	return results, len(repositories), err
}

func (s *Service) EnvironmentStatusInRepositories(ctx context.Context, repositories []scm.Repository, ticketID string, environments []Environment) ([]RepositoryEnvironmentStatus, error) {
	if err := s.parser.Validate(ticketID); err != nil {
		return nil, err
	}

	inspections, err := s.InspectInRepositories(ctx, repositories, ticketID)
	if err != nil {
		return nil, err
	}

	results := make([]RepositoryEnvironmentStatus, 0, len(inspections))
	for _, inspection := range inspections {
		adapter, ok := s.adapters.For(inspection.Repository.Type)
		if !ok {
			continue
		}

		statuses := make([]EnvironmentResult, 0, len(environments))
		for _, environment := range environments {
			exists, err := refExists(ctx, adapter, inspection.Repository.Root, environment.Branch)
			if err != nil {
				return nil, err
			}
			if !exists {
				statuses = append(statuses, EnvironmentResult{
					Environment: environment,
					State:       EnvStateBranchMissing,
				})
				continue
			}

			commits, err := adapter.SearchCommits(ctx, inspection.Repository.Root, scm.SearchQuery{
				TicketID: ticketID,
				Branch:   environment.Branch,
			})
			if err != nil {
				return nil, err
			}

			statuses = append(statuses, EnvironmentResult{
				Environment: environment,
				CommitCount: len(commits),
			})
		}

		for i := range statuses {
			statuses[i].State = deriveEnvironmentState(statuses, i)
			if i == 0 || statuses[i].State == EnvStateBranchMissing || statuses[i-1].State == EnvStateBranchMissing {
				continue
			}

			compareResult, err := adapter.CompareBranches(ctx, inspection.Repository.Root, scm.CompareQuery{
				TicketID:   ticketID,
				FromBranch: statuses[i-1].Environment.Branch,
				ToBranch:   statuses[i].Environment.Branch,
			})
			if err != nil {
				return nil, err
			}

			if statuses[i].CommitCount == 0 && len(compareResult.SourceCommits) > 0 && len(compareResult.MissingCommits) == 0 {
				statuses[i].CommitCount = inferredCommitCount(compareResult)
			}
			statuses[i].MissingFromPrevious = len(compareResult.MissingCommits)
			statuses[i].State = deriveEnvironmentState(statuses, i)
		}

		results = append(results, RepositoryEnvironmentStatus{
			Repository:           inspection.Repository,
			Commits:              inspection.Commits,
			Branches:             inspection.Branches,
			RiskSignals:          inspection.RiskSignals,
			DeclaredDependencies: inspection.DeclaredDependencies,
			Statuses:             statuses,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Repository.Root < results[j].Repository.Root
	})

	return results, nil
}

func (s *Service) loadCommitFiles(ctx context.Context, adapter scm.Adapter, repoRoot string, commits []scm.Commit) (map[string][]string, error) {
	fileProvider, ok := adapter.(commitFileProvider)
	if !ok {
		return nil, nil
	}

	hashes := make([]string, 0, len(commits))
	for _, commit := range commits {
		hashes = append(hashes, commit.Hash)
	}

	return fileProvider.CommitFiles(ctx, repoRoot, hashes)
}

func refExists(ctx context.Context, adapter scm.Adapter, repoRoot, ref string) (bool, error) {
	checker, ok := adapter.(refChecker)
	if !ok {
		return true, nil
	}

	return checker.RefExists(ctx, repoRoot, ref)
}

func collectBranches(commits []scm.Commit) []string {
	seen := map[string]struct{}{}
	branches := make([]string, 0, len(commits))

	for _, commit := range commits {
		for _, branch := range commit.Branches {
			if _, ok := seen[branch]; ok {
				continue
			}
			seen[branch] = struct{}{}
			branches = append(branches, branch)
		}
	}

	sort.Strings(branches)
	return branches
}

func inferRiskSignals(filesByCommit map[string][]string) []RiskSignal {
	if len(filesByCommit) == 0 {
		return nil
	}

	type builder struct {
		level    string
		summary  string
		examples []string
		seen     map[string]struct{}
	}

	builders := map[string]*builder{}

	for _, files := range filesByCommit {
		for _, file := range files {
			code, level, summary := classifyFileRisk(file)
			if code == "" {
				continue
			}

			entry, ok := builders[code]
			if !ok {
				entry = &builder{
					level:    level,
					summary:  summary,
					examples: make([]string, 0, 3),
					seen:     map[string]struct{}{},
				}
				builders[code] = entry
			}

			if len(entry.examples) >= 3 {
				continue
			}
			if _, exists := entry.seen[file]; exists {
				continue
			}
			entry.seen[file] = struct{}{}
			entry.examples = append(entry.examples, file)
		}
	}

	signals := make([]RiskSignal, 0, len(builders))
	for code, entry := range builders {
		signals = append(signals, RiskSignal{
			Code:     code,
			Level:    entry.level,
			Summary:  entry.summary,
			Examples: entry.examples,
		})
	}

	sort.Slice(signals, func(i, j int) bool {
		return signals[i].Code < signals[j].Code
	})

	return signals
}

func classifyFileRisk(file string) (code string, level string, summary string) {
	lowerFile := strings.ToLower(file)

	switch {
	case strings.HasSuffix(lowerFile, ".mpr") || strings.HasSuffix(lowerFile, ".mpr.lock"):
		return "mendix-model", "manual-review", "Mendix model changes need manual review"
	case strings.HasSuffix(lowerFile, ".sql") || strings.Contains(lowerFile, "/db/") || strings.Contains(lowerFile, "migrations/") || strings.Contains(lowerFile, "schema/"):
		return "db-change", "manual-review", "Database-related changes should be reviewed before promotion"
	case strings.HasSuffix(lowerFile, ".env") ||
		strings.HasSuffix(lowerFile, ".yaml") ||
		strings.HasSuffix(lowerFile, ".yml") ||
		strings.HasSuffix(lowerFile, ".toml") ||
		strings.HasSuffix(lowerFile, ".ini") ||
		strings.HasSuffix(lowerFile, ".properties") ||
		strings.Contains(lowerFile, "config/") ||
		strings.Contains(lowerFile, "settings/") ||
		strings.Contains(lowerFile, "values."):
		return "config-change", "warning", "Configuration changes may need deployment or rollout review"
	default:
		return "", "", ""
	}
}

func deriveEnvironmentState(statuses []EnvironmentResult, index int) EnvironmentState {
	current := statuses[index]
	if current.State == EnvStateBranchMissing {
		return EnvStateBranchMissing
	}

	if index == 0 {
		if current.CommitCount == 0 {
			return EnvStateNotPresent
		}
		return EnvStatePresent
	}

	previous := statuses[index-1]
	if previous.State == EnvStateBranchMissing {
		if current.CommitCount == 0 {
			return EnvStateNotPresent
		}
		return EnvStatePresent
	}

	if current.MissingFromPrevious > 0 {
		return EnvStateBehind
	}
	if current.CommitCount == 0 {
		return EnvStateNotPresent
	}
	if previous.CommitCount == 0 {
		return EnvStatePresent
	}
	return EnvStateAligned
}

func inferredCommitCount(compareResult scm.CompareResult) int {
	if len(compareResult.TargetCommits) > 0 {
		return len(compareResult.TargetCommits)
	}
	return len(compareResult.SourceCommits)
}
