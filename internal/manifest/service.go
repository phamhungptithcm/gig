package manifest

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"gig/internal/config"
	depsvc "gig/internal/dependency"
	inspectsvc "gig/internal/inspect"
	plansvc "gig/internal/plan"
	"gig/internal/scm"
)

type AudienceSection struct {
	Title     string   `json:"title"`
	Summary   []string `json:"summary,omitempty"`
	Checklist []string `json:"checklist,omitempty"`
}

type RepositoryPacket struct {
	Repository            scm.Repository                 `json:"repository"`
	ConfigEntry           *config.Repository             `json:"configEntry,omitempty"`
	Verdict               plansvc.Verdict                `json:"verdict"`
	EnvironmentStatuses   []inspectsvc.EnvironmentResult `json:"environmentStatuses,omitempty"`
	RiskSignals           []inspectsvc.RiskSignal        `json:"riskSignals,omitempty"`
	ProviderEvidence      *scm.ProviderEvidence          `json:"providerEvidence,omitempty"`
	DependencyResolutions []depsvc.Resolution            `json:"dependencyResolutions,omitempty"`
	ManualSteps           []plansvc.Action               `json:"manualSteps,omitempty"`
	Actions               []plansvc.Action               `json:"actions,omitempty"`
	Notes                 []string                       `json:"notes,omitempty"`
	CommitsToInclude      []scm.Commit                   `json:"commitsToInclude,omitempty"`
}

type ReleasePacket struct {
	Workspace      string             `json:"workspace"`
	ConfigPath     string             `json:"configPath,omitempty"`
	TicketID       string             `json:"ticketId"`
	FromBranch     string             `json:"fromBranch"`
	ToBranch       string             `json:"toBranch"`
	Verdict        plansvc.Verdict    `json:"verdict"`
	Summary        plansvc.Summary    `json:"summary"`
	Highlights     []string           `json:"highlights,omitempty"`
	QA             AudienceSection    `json:"qa"`
	Client         AudienceSection    `json:"client"`
	ReleaseManager AudienceSection    `json:"releaseManager"`
	Repositories   []RepositoryPacket `json:"repositories,omitempty"`
}

type Service struct {
	planner *plansvc.Service
}

func NewService(planner *plansvc.Service) *Service {
	return &Service{planner: planner}
}

func (s *Service) Generate(ctx context.Context, workspacePath string, loaded config.Loaded, ticketID, fromBranch, toBranch string, environments []inspectsvc.Environment) (ReleasePacket, error) {
	promotionPlan, err := s.planner.BuildPromotionPlan(ctx, workspacePath, ticketID, fromBranch, toBranch, environments)
	if err != nil {
		return ReleasePacket{}, err
	}

	return BuildReleasePacket(workspacePath, loaded, promotionPlan), nil
}

func BuildReleasePacket(workspacePath string, loaded config.Loaded, promotionPlan plansvc.PromotionPlan) ReleasePacket {
	repositories := make([]RepositoryPacket, 0, len(promotionPlan.Repositories))
	for _, repositoryPlan := range promotionPlan.Repositories {
		var configEntry *config.Repository
		if entry, ok := loaded.Config.FindRepository(workspacePath, repositoryPlan.Repository.Root, repositoryPlan.Repository.Name); ok {
			entryCopy := entry
			configEntry = &entryCopy
		}

		repositories = append(repositories, RepositoryPacket{
			Repository:            repositoryPlan.Repository,
			ConfigEntry:           configEntry,
			Verdict:               repositoryPlan.Verdict,
			EnvironmentStatuses:   repositoryPlan.EnvironmentStatuses,
			RiskSignals:           repositoryPlan.RiskSignals,
			ProviderEvidence:      scm.NormalizeProviderEvidence(repositoryPlan.ProviderEvidence),
			DependencyResolutions: repositoryPlan.DependencyResolutions,
			ManualSteps:           repositoryPlan.ManualSteps,
			Actions:               repositoryPlan.Actions,
			Notes:                 append(configNotes(configEntry), repositoryPlan.Notes...),
			CommitsToInclude:      repositoryPlan.Compare.MissingCommits,
		})
	}

	sort.Slice(repositories, func(i, j int) bool {
		return repositories[i].Repository.Root < repositories[j].Repository.Root
	})

	return ReleasePacket{
		Workspace:      workspacePath,
		ConfigPath:     loaded.Path,
		TicketID:       promotionPlan.TicketID,
		FromBranch:     promotionPlan.FromBranch,
		ToBranch:       promotionPlan.ToBranch,
		Verdict:        promotionPlan.Verdict,
		Summary:        promotionPlan.Summary,
		Highlights:     buildHighlights(promotionPlan, repositories),
		QA:             buildQASection(promotionPlan, repositories),
		Client:         buildClientSection(promotionPlan, repositories),
		ReleaseManager: buildReleaseManagerSection(promotionPlan, repositories),
		Repositories:   repositories,
	}
}

func buildHighlights(promotionPlan plansvc.PromotionPlan, repositories []RepositoryPacket) []string {
	highlights := []string{
		fmt.Sprintf("This ticket currently touches %d %s in the selected workspace.", promotionPlan.Summary.TouchedRepositories, pluralize(promotionPlan.Summary.TouchedRepositories, "repository", "repositories")),
		fmt.Sprintf("%d commit(s) are still planned from %s to %s.", promotionPlan.Summary.TotalCommitsToPromote, promotionPlan.FromBranch, promotionPlan.ToBranch),
	}

	switch promotionPlan.Verdict {
	case plansvc.VerdictBlocked:
		highlights = append(highlights, "Promotion is blocked until the repository issues below are resolved.")
	case plansvc.VerdictWarning:
		highlights = append(highlights, "Promotion needs manual review before it should be treated as release-ready.")
	case plansvc.VerdictSafe:
		highlights = append(highlights, "No blocking repository issues were found for the selected move.")
	}

	if promotionPlan.Summary.TotalManualSteps > 0 {
		highlights = append(highlights, fmt.Sprintf("%d manual review step(s) were inferred from risky file changes.", promotionPlan.Summary.TotalManualSteps))
	}
	missingDependencies, unresolvedDependencies := dependencyCounts(repositories)
	if missingDependencies > 0 {
		highlights = append(highlights, fmt.Sprintf("%d dependency ticket(s) are still missing from %s.", missingDependencies, promotionPlan.ToBranch))
	}
	if unresolvedDependencies > 0 {
		highlights = append(highlights, fmt.Sprintf("%d declared dependency ticket(s) could not be confirmed in the scanned workspace.", unresolvedDependencies))
	}

	return highlights
}

func buildQASection(promotionPlan plansvc.PromotionPlan, repositories []RepositoryPacket) AudienceSection {
	checklist := []string{
		fmt.Sprintf("Verify the final ticket behavior on %s after the promotion is complete.", promotionPlan.ToBranch),
		fmt.Sprintf("Confirm every follow-up fix from %s is visible in %s.", promotionPlan.FromBranch, promotionPlan.ToBranch),
	}

	for _, repository := range repositories {
		if repository.Verdict == plansvc.VerdictBlocked {
			checklist = append(checklist, fmt.Sprintf("Re-check %s because release evidence is still blocked or incomplete.", displayRepository(repository)))
		}
		if len(repository.RiskSignals) > 0 {
			checklist = append(checklist, fmt.Sprintf("Do focused regression checks for %s because risky file changes were detected.", displayRepository(repository)))
		}
	}

	return AudienceSection{
		Title: "QA Checklist",
		Summary: []string{
			"Use this section when QA needs a short release packet instead of reading raw git history.",
		},
		Checklist: dedupeStrings(checklist),
	}
}

func buildClientSection(promotionPlan plansvc.PromotionPlan, repositories []RepositoryPacket) AudienceSection {
	checklist := []string{
		"Use the repository summary below to confirm the final scope that is moving forward.",
		"Confirm the latest client feedback and follow-up fixes are included in the target branch.",
	}

	if promotionPlan.Summary.TotalCommitsToPromote > 0 {
		checklist = append(checklist, "Review the commit list for each affected repo if you want to confirm the exact follow-up fixes included in this release packet.")
	}

	if hasRiskSignals(repositories) {
		checklist = append(checklist, "Call out any DB, config, or Mendix-related work because those changes usually need extra communication during release review.")
	}

	return AudienceSection{
		Title: "Client Review Notes",
		Summary: []string{
			fmt.Sprintf("This packet summarizes what is expected to move from %s to %s for ticket %s.", promotionPlan.FromBranch, promotionPlan.ToBranch, promotionPlan.TicketID),
		},
		Checklist: dedupeStrings(checklist),
	}
}

func buildReleaseManagerSection(promotionPlan plansvc.PromotionPlan, repositories []RepositoryPacket) AudienceSection {
	checklist := []string{
		fmt.Sprintf("Use %s as the approved source and %s as the target for this ticket.", promotionPlan.FromBranch, promotionPlan.ToBranch),
		"Record release evidence after promotion so future checks can compare against a clean baseline.",
	}

	for _, repository := range repositories {
		for _, action := range repository.Actions {
			checklist = append(checklist, action.Summary)
		}
		for _, step := range repository.ManualSteps {
			checklist = append(checklist, step.Summary)
		}
		if repository.ConfigEntry != nil && strings.TrimSpace(repository.ConfigEntry.Owner) != "" {
			checklist = append(checklist, fmt.Sprintf("Coordinate with %s for %s if manual confirmation is needed.", repository.ConfigEntry.Owner, displayRepository(repository)))
		}
	}

	return AudienceSection{
		Title: "Release Manager Checklist",
		Summary: []string{
			"Use this section to coordinate the actual promotion, communication, and rollback readiness.",
		},
		Checklist: dedupeStrings(checklist),
	}
}

func configNotes(entry *config.Repository) []string {
	if entry == nil || len(entry.Notes) == 0 {
		return nil
	}
	notes := make([]string, 0, len(entry.Notes))
	for _, note := range entry.Notes {
		note = strings.TrimSpace(note)
		if note == "" {
			continue
		}
		notes = append(notes, note)
	}
	return notes
}

func displayRepository(repository RepositoryPacket) string {
	if repository.ConfigEntry != nil && strings.TrimSpace(repository.ConfigEntry.Service) != "" {
		return repository.ConfigEntry.Service
	}
	return repository.Repository.Name
}

func hasRiskSignals(repositories []RepositoryPacket) bool {
	for _, repository := range repositories {
		if len(repository.RiskSignals) > 0 {
			return true
		}
	}
	return false
}

func dependencyCounts(repositories []RepositoryPacket) (int, int) {
	statuses := map[string]depsvc.Status{}
	for _, repository := range repositories {
		for _, resolution := range repository.DependencyResolutions {
			statuses[resolution.DependsOn] = resolution.Status
		}
	}

	missingDependencies := 0
	unresolvedDependencies := 0
	for _, status := range statuses {
		switch status {
		case depsvc.StatusMissingTarget:
			missingDependencies++
		case depsvc.StatusUnresolved:
			unresolvedDependencies++
		}
	}

	return missingDependencies, unresolvedDependencies
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}
