package doctor

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"gig/internal/config"
	"gig/internal/repo"
	"gig/internal/scm"
)

type adapterProvider interface {
	For(repoType scm.Type) (scm.Adapter, bool)
}

type refChecker interface {
	RefExists(ctx context.Context, repoRoot, ref string) (bool, error)
}

type Verdict string

const (
	VerdictOK      Verdict = "ok"
	VerdictWarning Verdict = "warning"
	VerdictBlocked Verdict = "blocked"
)

type Finding struct {
	Severity Verdict `json:"severity"`
	Code     string  `json:"code"`
	Summary  string  `json:"summary"`
}

type EnvironmentCheck struct {
	Environment config.Environment `json:"environment"`
	Exists      bool               `json:"exists"`
	Summary     string             `json:"summary"`
}

type RepositoryResult struct {
	Repository        scm.Repository     `json:"repository"`
	ConfigEntry       *config.Repository `json:"configEntry,omitempty"`
	Verdict           Verdict            `json:"verdict"`
	Findings          []Finding          `json:"findings,omitempty"`
	EnvironmentChecks []EnvironmentCheck `json:"environmentChecks,omitempty"`
}

type Summary struct {
	ScannedRepositories    int `json:"scannedRepositories"`
	ConfiguredRepositories int `json:"configuredRepositories"`
	CoveredRepositories    int `json:"coveredRepositories"`
	MissingCatalogEntries  int `json:"missingCatalogEntries"`
	MissingEnvironmentRefs int `json:"missingEnvironmentRefs"`
	WarningCount           int `json:"warningCount"`
}

type Report struct {
	Workspace    string             `json:"workspace"`
	ConfigPath   string             `json:"configPath,omitempty"`
	UsingBuiltIn bool               `json:"usingBuiltIn"`
	Verdict      Verdict            `json:"verdict"`
	Summary      Summary            `json:"summary"`
	Findings     []Finding          `json:"findings,omitempty"`
	Repositories []RepositoryResult `json:"repositories,omitempty"`
}

type Service struct {
	discoverer repo.Discoverer
	adapters   adapterProvider
}

func NewService(discoverer repo.Discoverer, adapters adapterProvider) *Service {
	return &Service{
		discoverer: discoverer,
		adapters:   adapters,
	}
}

func (s *Service) Run(ctx context.Context, workspacePath string, loaded config.Loaded) (Report, error) {
	repositories, err := s.discoverer.Discover(ctx, workspacePath)
	if err != nil {
		return Report{}, err
	}

	report := Report{
		Workspace:    workspacePath,
		ConfigPath:   loaded.Path,
		UsingBuiltIn: !loaded.Found,
		Summary: Summary{
			ScannedRepositories:    len(repositories),
			ConfiguredRepositories: len(loaded.Config.Repositories),
		},
	}

	if len(repositories) == 0 {
		report.Verdict = VerdictBlocked
		report.Findings = append(report.Findings, Finding{
			Severity: VerdictBlocked,
			Code:     "no-repositories",
			Summary:  "No repositories were found under the selected workspace path.",
		})
		return report, nil
	}

	if !loaded.Found {
		report.Findings = append(report.Findings, Finding{
			Severity: VerdictWarning,
			Code:     "config-not-found",
			Summary:  "No gig config file was found. Built-in defaults are being used.",
		})
	}
	if !loaded.ExplicitEnvironments {
		report.Findings = append(report.Findings, Finding{
			Severity: VerdictWarning,
			Code:     "default-environments",
			Summary:  "Environment mapping is still using defaults. Add team-specific env branches to the config file.",
		})
	}
	if len(loaded.Config.Repositories) == 0 {
		report.Findings = append(report.Findings, Finding{
			Severity: VerdictWarning,
			Code:     "empty-repository-catalog",
			Summary:  "No repository catalog entries were defined. Add service, owner, and kind information for each repo.",
		})
	}

	results := make([]RepositoryResult, 0, len(repositories))
	for _, repository := range repositories {
		result, err := s.inspectRepository(ctx, workspacePath, repository, loaded.Config)
		if err != nil {
			return Report{}, err
		}
		results = append(results, result)

		if result.ConfigEntry != nil {
			report.Summary.CoveredRepositories++
		} else {
			report.Summary.MissingCatalogEntries++
		}
		for _, finding := range result.Findings {
			if finding.Severity == VerdictWarning {
				report.Summary.WarningCount++
			}
		}
		for _, check := range result.EnvironmentChecks {
			if !check.Exists {
				report.Summary.MissingEnvironmentRefs++
				report.Summary.WarningCount++
			}
		}
	}

	for _, repository := range loaded.Config.Repositories {
		if repository.Path == "" {
			continue
		}
		found := false
		for _, result := range results {
			relativePath := normalizeRelativePath(workspacePath, result.Repository.Root)
			if relativePath == repository.Path {
				found = true
				break
			}
		}
		if !found {
			report.Findings = append(report.Findings, Finding{
				Severity: VerdictWarning,
				Code:     "configured-repo-missing",
				Summary:  fmt.Sprintf("Configured repository entry %s was not found under the selected workspace.", repository.Path),
			})
			report.Summary.WarningCount++
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Repository.Root < results[j].Repository.Root
	})
	for i := range results {
		sort.Slice(results[i].Findings, func(left, right int) bool {
			return results[i].Findings[left].Code < results[i].Findings[right].Code
		})
	}
	sort.Slice(report.Findings, func(i, j int) bool {
		return report.Findings[i].Code < report.Findings[j].Code
	})

	report.Repositories = results
	report.Verdict = deriveVerdict(report)
	return report, nil
}

func (s *Service) inspectRepository(ctx context.Context, workspacePath string, repository scm.Repository, cfg config.Config) (RepositoryResult, error) {
	result := RepositoryResult{
		Repository: repository,
		Verdict:    VerdictOK,
	}

	if entry, ok := cfg.FindRepository(workspacePath, repository.Root, repository.Name); ok {
		entryCopy := entry
		result.ConfigEntry = &entryCopy
		if strings.TrimSpace(entry.Service) == "" {
			result.Findings = append(result.Findings, Finding{
				Severity: VerdictWarning,
				Code:     "missing-service",
				Summary:  "Config entry is missing a service name.",
			})
		}
		if strings.TrimSpace(entry.Owner) == "" {
			result.Findings = append(result.Findings, Finding{
				Severity: VerdictWarning,
				Code:     "missing-owner",
				Summary:  "Config entry is missing an owner.",
			})
		}
		if strings.TrimSpace(entry.Kind) == "" {
			result.Findings = append(result.Findings, Finding{
				Severity: VerdictWarning,
				Code:     "missing-kind",
				Summary:  "Config entry is missing a repository kind such as app, db, mendix, or infra.",
			})
		}
	} else {
		result.Findings = append(result.Findings, Finding{
			Severity: VerdictWarning,
			Code:     "missing-catalog-entry",
			Summary:  "No config entry was found for this repository.",
		})
	}

	adapter, ok := s.adapters.For(repository.Type)
	if ok {
		for _, environment := range cfg.Environments {
			exists, err := refExists(ctx, adapter, repository.Root, environment.Branch)
			if err != nil {
				return RepositoryResult{}, err
			}

			check := EnvironmentCheck{
				Environment: environment,
				Exists:      exists,
			}
			if exists {
				check.Summary = fmt.Sprintf("%s branch %s is available.", environment.Name, environment.Branch)
			} else {
				check.Summary = fmt.Sprintf("%s branch %s was not found.", environment.Name, environment.Branch)
			}
			result.EnvironmentChecks = append(result.EnvironmentChecks, check)
		}
	}

	if len(result.Findings) > 0 {
		result.Verdict = VerdictWarning
	}
	for _, check := range result.EnvironmentChecks {
		if !check.Exists {
			result.Verdict = VerdictWarning
			break
		}
	}

	return result, nil
}

func deriveVerdict(report Report) Verdict {
	if report.Summary.ScannedRepositories == 0 {
		return VerdictBlocked
	}
	if len(report.Findings) > 0 || report.Summary.MissingCatalogEntries > 0 || report.Summary.MissingEnvironmentRefs > 0 {
		return VerdictWarning
	}
	return VerdictOK
}

func refExists(ctx context.Context, adapter scm.Adapter, repoRoot, ref string) (bool, error) {
	checker, ok := adapter.(refChecker)
	if !ok {
		return true, nil
	}

	return checker.RefExists(ctx, repoRoot, ref)
}

func normalizeRelativePath(workspacePath, repoRoot string) string {
	relativePath, err := filepathRel(workspacePath, repoRoot)
	if err != nil {
		return repoRoot
	}
	return relativePath
}

var filepathRel = func(basePath, targetPath string) (string, error) {
	relativePath, err := filepath.Rel(basePath, targetPath)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Clean(relativePath)), nil
}
