package plan

import (
	"context"
	"fmt"
	"sort"
	"strings"

	depsvc "gig/internal/dependency"
	inspectsvc "gig/internal/inspect"
	"gig/internal/repo"
	"gig/internal/scm"
	"gig/internal/ticket"
)

type adapterProvider interface {
	For(repoType scm.Type) (scm.Adapter, bool)
}

type refChecker interface {
	RefExists(ctx context.Context, repoRoot, ref string) (bool, error)
}

type Verdict string

const (
	VerdictSafe    Verdict = "safe"
	VerdictWarning Verdict = "warning"
	VerdictBlocked Verdict = "blocked"
)

type Action struct {
	Code    string `json:"code"`
	Summary string `json:"summary"`
}

type Summary struct {
	ScannedRepositories   int `json:"scannedRepositories"`
	TouchedRepositories   int `json:"touchedRepositories"`
	ReadyRepositories     int `json:"readyRepositories"`
	WarningRepositories   int `json:"warningRepositories"`
	BlockedRepositories   int `json:"blockedRepositories"`
	TotalCommitsToPromote int `json:"totalCommitsToPromote"`
	TotalManualSteps      int `json:"totalManualSteps"`
}

type RepositoryPlan struct {
	Repository            scm.Repository                 `json:"repository"`
	Compare               scm.CompareResult              `json:"compare"`
	Branches              []string                       `json:"branches,omitempty"`
	RiskSignals           []inspectsvc.RiskSignal        `json:"riskSignals,omitempty"`
	DeclaredDependencies  []depsvc.DeclaredDependency    `json:"declaredDependencies,omitempty"`
	ProviderEvidence      *scm.ProviderEvidence          `json:"providerEvidence,omitempty"`
	DependencyResolutions []depsvc.Resolution            `json:"dependencyResolutions,omitempty"`
	EnvironmentStatuses   []inspectsvc.EnvironmentResult `json:"environmentStatuses,omitempty"`
	ManualSteps           []Action                       `json:"manualSteps,omitempty"`
	Actions               []Action                       `json:"actions,omitempty"`
	Verdict               Verdict                        `json:"verdict"`
	Notes                 []string                       `json:"notes,omitempty"`
}

type PromotionPlan struct {
	TicketID     string                   `json:"ticketId"`
	FromBranch   string                   `json:"fromBranch"`
	ToBranch     string                   `json:"toBranch"`
	Environments []inspectsvc.Environment `json:"environments,omitempty"`
	Summary      Summary                  `json:"summary"`
	Verdict      Verdict                  `json:"verdict"`
	Notes        []string                 `json:"notes,omitempty"`
	Repositories []RepositoryPlan         `json:"repositories,omitempty"`
}

type RepositoryVerification struct {
	Repository            scm.Repository          `json:"repository"`
	Verdict               Verdict                 `json:"verdict"`
	Checks                []string                `json:"checks"`
	RiskSignals           []inspectsvc.RiskSignal `json:"riskSignals,omitempty"`
	ProviderEvidence      *scm.ProviderEvidence   `json:"providerEvidence,omitempty"`
	DependencyResolutions []depsvc.Resolution     `json:"dependencyResolutions,omitempty"`
	ManualSteps           []Action                `json:"manualSteps,omitempty"`
}

type Verification struct {
	TicketID     string                   `json:"ticketId"`
	FromBranch   string                   `json:"fromBranch"`
	ToBranch     string                   `json:"toBranch"`
	Environments []inspectsvc.Environment `json:"environments,omitempty"`
	Summary      Summary                  `json:"summary"`
	Verdict      Verdict                  `json:"verdict"`
	Reasons      []string                 `json:"reasons"`
	Repositories []RepositoryVerification `json:"repositories,omitempty"`
}

type Service struct {
	discoverer repo.Discoverer
	adapters   adapterProvider
	parser     ticket.Parser
	inspector  *inspectsvc.Service
}

func NewService(discoverer repo.Discoverer, adapters adapterProvider, parser ticket.Parser) *Service {
	return &Service{
		discoverer: discoverer,
		adapters:   adapters,
		parser:     parser,
		inspector:  inspectsvc.NewService(discoverer, adapters, parser),
	}
}

func (s *Service) BuildPromotionPlan(ctx context.Context, path, ticketID, fromBranch, toBranch string, environments []inspectsvc.Environment) (PromotionPlan, error) {
	if err := s.parser.Validate(ticketID); err != nil {
		return PromotionPlan{}, err
	}
	if strings.TrimSpace(fromBranch) == "" || strings.TrimSpace(toBranch) == "" {
		return PromotionPlan{}, fmt.Errorf("both --from and --to branches are required")
	}

	repositories, err := s.discoverer.Discover(ctx, path)
	if err != nil {
		return PromotionPlan{}, err
	}

	return s.BuildPromotionPlanInRepositories(ctx, repositories, ticketID, fromBranch, toBranch, environments)
}

func (s *Service) BuildPromotionPlanInRepositories(ctx context.Context, repositories []scm.Repository, ticketID, fromBranch, toBranch string, environments []inspectsvc.Environment) (PromotionPlan, error) {
	if err := s.parser.Validate(ticketID); err != nil {
		return PromotionPlan{}, err
	}
	if strings.TrimSpace(fromBranch) == "" || strings.TrimSpace(toBranch) == "" {
		return PromotionPlan{}, fmt.Errorf("both --from and --to branches are required")
	}

	repositoryStatuses, err := s.inspector.EnvironmentStatusInRepositories(ctx, repositories, ticketID, environments)
	if err != nil {
		return PromotionPlan{}, err
	}
	allDeclaredDependencies := collectDeclaredDependencies(repositoryStatuses)
	dependencyResolver := depsvc.NewResolver(s.adapters, s.parser)
	resolvedDependencies, err := dependencyResolver.ResolveInRepositories(ctx, repositories, allDeclaredDependencies, fromBranch, toBranch)
	if err != nil {
		return PromotionPlan{}, err
	}
	resolvedDependenciesByTicket := make(map[string]depsvc.Resolution, len(resolvedDependencies))
	for _, resolution := range resolvedDependencies {
		resolvedDependenciesByTicket[resolution.DependsOn] = resolution
	}

	plans := make([]RepositoryPlan, 0, len(repositoryStatuses))
	summary := Summary{
		ScannedRepositories: len(repositories),
		TouchedRepositories: len(repositoryStatuses),
	}

	for _, repositoryStatus := range repositoryStatuses {
		adapter, ok := s.adapters.For(repositoryStatus.Repository.Type)
		if !ok {
			continue
		}

		fromExists, err := refExists(ctx, adapter, repositoryStatus.Repository.Root, fromBranch)
		if err != nil {
			return PromotionPlan{}, err
		}
		toExists, err := refExists(ctx, adapter, repositoryStatus.Repository.Root, toBranch)
		if err != nil {
			return PromotionPlan{}, err
		}

		notes := make([]string, 0, 4)
		actions := make([]Action, 0, 4)
		dependencyResolutions := resolveRepositoryDependencies(repositoryStatus.DeclaredDependencies, resolvedDependenciesByTicket)
		riskSignals := append([]inspectsvc.RiskSignal{}, repositoryStatus.RiskSignals...)
		riskSignals = append(riskSignals, dependencyRiskSignals(dependencyResolutions, fromBranch, toBranch)...)
		sortRiskSignals(riskSignals)
		manualSteps := manualStepsForRiskSignals(riskSignals)
		compare := scm.CompareResult{
			FromBranch: fromBranch,
			ToBranch:   toBranch,
		}

		if !fromExists {
			notes = append(notes, fmt.Sprintf("Source branch %s was not found.", fromBranch))
			actions = append(actions, Action{
				Code:    "resolve-source-branch",
				Summary: fmt.Sprintf("Create or remap the source branch %s before promotion planning.", fromBranch),
			})
		}
		if !toExists {
			notes = append(notes, fmt.Sprintf("Target branch %s was not found.", toBranch))
			actions = append(actions, Action{
				Code:    "resolve-target-branch",
				Summary: fmt.Sprintf("Create or remap the target branch %s before promotion planning.", toBranch),
			})
		}

		if fromExists && toExists {
			compare, err = adapter.CompareBranches(ctx, repositoryStatus.Repository.Root, scm.CompareQuery{
				TicketID:   ticketID,
				FromBranch: fromBranch,
				ToBranch:   toBranch,
			})
			if err != nil {
				return PromotionPlan{}, err
			}
		}

		fromStatus := lookupEnvironmentStatus(repositoryStatus.Statuses, fromBranch)
		if fromStatus != nil && fromStatus.State == inspectsvc.EnvStateBehind {
			notes = append(notes, fmt.Sprintf("Source branch %s is behind an earlier environment by %d commit(s).", fromBranch, fromStatus.MissingFromPrevious))
			actions = append(actions, Action{
				Code:    "sync-source-environment",
				Summary: fmt.Sprintf("Bring %s up to date before promoting to %s.", fromBranch, toBranch),
			})
		}

		if fromExists && len(compare.SourceCommits) == 0 {
			notes = append(notes, fmt.Sprintf("Source branch %s does not currently contain ticket commits.", fromBranch))
			actions = append(actions, Action{
				Code:    "verify-source-scope",
				Summary: fmt.Sprintf("Confirm whether %s is the correct approved source for this ticket.", fromBranch),
			})
		}

		if len(compare.MissingCommits) > 0 {
			actions = append(actions, Action{
				Code:    "include-missing-commits",
				Summary: fmt.Sprintf("Include %d missing ticket commit(s) from %s into %s.", len(compare.MissingCommits), fromBranch, toBranch),
			})
		} else if len(compare.SourceCommits) > 0 {
			notes = append(notes, fmt.Sprintf("Target branch %s already contains the ticket commits from %s.", toBranch, fromBranch))
			actions = append(actions, Action{
				Code:    "already-aligned",
				Summary: fmt.Sprintf("No ticket commits are missing in %s.", toBranch),
			})
		}

		notes = append(notes, dependencyNotes(dependencyResolutions, fromBranch, toBranch)...)
		actions = append(actions, dependencyActions(dependencyResolutions, ticketID, toBranch)...)
		verdict := deriveRepositoryVerdict(fromExists, toExists, compare, fromStatus, riskSignals, manualSteps)
		summary.TotalCommitsToPromote += len(compare.MissingCommits)
		summary.TotalManualSteps += len(manualSteps)

		switch verdict {
		case VerdictSafe:
			summary.ReadyRepositories++
		case VerdictWarning:
			summary.WarningRepositories++
		case VerdictBlocked:
			summary.BlockedRepositories++
		}

		plans = append(plans, RepositoryPlan{
			Repository:            repositoryStatus.Repository,
			Compare:               compare,
			Branches:              repositoryStatus.Branches,
			RiskSignals:           riskSignals,
			DeclaredDependencies:  repositoryStatus.DeclaredDependencies,
			ProviderEvidence:      scm.NormalizeProviderEvidence(repositoryStatus.ProviderEvidence),
			DependencyResolutions: dependencyResolutions,
			EnvironmentStatuses:   repositoryStatus.Statuses,
			ManualSteps:           manualSteps,
			Actions:               actions,
			Verdict:               verdict,
			Notes:                 notes,
		})
	}

	sort.Slice(plans, func(i, j int) bool {
		return plans[i].Repository.Root < plans[j].Repository.Root
	})

	promotionPlan := PromotionPlan{
		TicketID:     ticketID,
		FromBranch:   fromBranch,
		ToBranch:     toBranch,
		Environments: environments,
		Summary:      summary,
		Verdict:      derivePlanVerdict(summary),
		Notes:        append(buildPlanNotes(summary), buildDependencyPlanNotes(plans, toBranch)...),
		Repositories: plans,
	}

	return promotionPlan, nil
}

func (s *Service) VerifyPromotion(ctx context.Context, path, ticketID, fromBranch, toBranch string, environments []inspectsvc.Environment) (Verification, error) {
	promotionPlan, err := s.BuildPromotionPlan(ctx, path, ticketID, fromBranch, toBranch, environments)
	if err != nil {
		return Verification{}, err
	}

	return BuildVerification(promotionPlan), nil
}

func BuildVerification(promotionPlan PromotionPlan) Verification {
	repositories := make([]RepositoryVerification, 0, len(promotionPlan.Repositories))
	for _, repositoryPlan := range promotionPlan.Repositories {
		checks := make([]string, 0, 6)
		if len(repositoryPlan.Compare.SourceCommits) > 0 {
			checks = append(checks, fmt.Sprintf("%s contains %d ticket commit(s).", promotionPlan.FromBranch, len(repositoryPlan.Compare.SourceCommits)))
		}
		if len(repositoryPlan.Compare.SourceCommits) == 0 {
			checks = append(checks, fmt.Sprintf("%s does not contain ticket commits yet.", promotionPlan.FromBranch))
		}
		if len(repositoryPlan.Compare.MissingCommits) > 0 {
			checks = append(checks, fmt.Sprintf("%s is missing %d ticket commit(s).", promotionPlan.ToBranch, len(repositoryPlan.Compare.MissingCommits)))
		} else if len(repositoryPlan.Compare.SourceCommits) > 0 {
			checks = append(checks, fmt.Sprintf("%s already contains the ticket commits from %s.", promotionPlan.ToBranch, promotionPlan.FromBranch))
		}
		for _, note := range repositoryPlan.Notes {
			checks = append(checks, note)
		}
		if len(repositoryPlan.RiskSignals) > 0 {
			checks = append(checks, fmt.Sprintf("%d risk signal(s) require review.", len(repositoryPlan.RiskSignals)))
		}

		repositories = append(repositories, RepositoryVerification{
			Repository:            repositoryPlan.Repository,
			Verdict:               repositoryPlan.Verdict,
			Checks:                checks,
			RiskSignals:           repositoryPlan.RiskSignals,
			ProviderEvidence:      scm.NormalizeProviderEvidence(repositoryPlan.ProviderEvidence),
			DependencyResolutions: repositoryPlan.DependencyResolutions,
			ManualSteps:           repositoryPlan.ManualSteps,
		})
	}

	return Verification{
		TicketID:     promotionPlan.TicketID,
		FromBranch:   promotionPlan.FromBranch,
		ToBranch:     promotionPlan.ToBranch,
		Environments: promotionPlan.Environments,
		Summary:      promotionPlan.Summary,
		Verdict:      promotionPlan.Verdict,
		Reasons:      buildVerificationReasons(promotionPlan),
		Repositories: repositories,
	}
}

func deriveRepositoryVerdict(fromExists, toExists bool, compare scm.CompareResult, fromStatus *inspectsvc.EnvironmentResult, riskSignals []inspectsvc.RiskSignal, manualSteps []Action) Verdict {
	if !fromExists || !toExists {
		return VerdictBlocked
	}
	if fromStatus != nil && fromStatus.State == inspectsvc.EnvStateBehind {
		return VerdictBlocked
	}
	if len(compare.SourceCommits) == 0 {
		return VerdictBlocked
	}
	if hasBlockedRiskSignals(riskSignals) {
		return VerdictBlocked
	}
	if len(manualSteps) > 0 || hasWarningRiskSignals(riskSignals) {
		return VerdictWarning
	}
	return VerdictSafe
}

func derivePlanVerdict(summary Summary) Verdict {
	switch {
	case summary.TouchedRepositories == 0:
		return VerdictBlocked
	case summary.BlockedRepositories > 0:
		return VerdictBlocked
	case summary.WarningRepositories > 0:
		return VerdictWarning
	default:
		return VerdictSafe
	}
}

func buildPlanNotes(summary Summary) []string {
	notes := make([]string, 0, 3)
	if summary.TouchedRepositories == 0 {
		notes = append(notes, "Ticket was not found in detected repositories.")
		return notes
	}
	if summary.BlockedRepositories > 0 {
		notes = append(notes, "Resolve blocked repositories before promoting this ticket.")
	}
	if summary.WarningRepositories > 0 {
		notes = append(notes, "Manual review is required for one or more repositories.")
	}
	if summary.TotalCommitsToPromote == 0 {
		notes = append(notes, "No ticket commits are missing between the requested source and target branches.")
	}
	return notes
}

func buildVerificationReasons(promotionPlan PromotionPlan) []string {
	reasons := make([]string, 0, 4)
	summary := promotionPlan.Summary
	switch promotionPlan.Verdict {
	case VerdictBlocked:
		if summary.TouchedRepositories == 0 {
			reasons = append(reasons, "The ticket was not found in detected repositories.")
		}
		if summary.BlockedRepositories > 0 {
			reasons = append(reasons, "At least one repository is missing the correct source or target release evidence.")
		}
	case VerdictWarning:
		reasons = append(reasons, "Promotion is possible, but manual review is required for risky repositories.")
	case VerdictSafe:
		reasons = append(reasons, "Source branch contains the ticket scope and no repository is currently blocked.")
	}
	if summary.TotalCommitsToPromote == 0 && summary.TouchedRepositories > 0 {
		reasons = append(reasons, "Target branch already contains the ticket commits from the selected source.")
	} else if summary.TotalCommitsToPromote > 0 {
		reasons = append(reasons, fmt.Sprintf("%d ticket commit(s) are still planned for promotion.", summary.TotalCommitsToPromote))
	}
	if summary.TotalManualSteps > 0 {
		reasons = append(reasons, fmt.Sprintf("%d manual review step(s) were inferred from the changed files.", summary.TotalManualSteps))
	}
	missingDependencies, unresolvedDependencies := dependencyCountsFromPlans(promotionPlan.Repositories)
	if missingDependencies > 0 {
		reasons = append(reasons, fmt.Sprintf("%d dependency ticket(s) are still missing from %s.", missingDependencies, promotionPlan.ToBranch))
	}
	if unresolvedDependencies > 0 {
		reasons = append(reasons, fmt.Sprintf("%d declared dependency ticket(s) could not be confirmed in the scanned workspace.", unresolvedDependencies))
	}
	return reasons
}

func manualStepsForRiskSignals(riskSignals []inspectsvc.RiskSignal) []Action {
	steps := make([]Action, 0, len(riskSignals))
	for _, riskSignal := range riskSignals {
		switch riskSignal.Code {
		case "db-change":
			steps = append(steps, Action{
				Code:    "review-db-rollout",
				Summary: "Review DB migration ordering, rollback steps, and deployment timing before promotion.",
			})
		case "config-change":
			steps = append(steps, Action{
				Code:    "review-config-rollout",
				Summary: "Confirm environment config, secrets, and rollout timing before promotion.",
			})
		case "mendix-model":
			steps = append(steps, Action{
				Code:    "review-mendix-deploy",
				Summary: "Validate Mendix package deployment steps and compatibility before promotion.",
			})
		}
	}

	sort.Slice(steps, func(i, j int) bool {
		return steps[i].Code < steps[j].Code
	})

	return steps
}

func lookupEnvironmentStatus(statuses []inspectsvc.EnvironmentResult, branch string) *inspectsvc.EnvironmentResult {
	for i := range statuses {
		if statuses[i].Environment.Branch == branch {
			return &statuses[i]
		}
	}

	return nil
}

func collectDeclaredDependencies(results []inspectsvc.RepositoryEnvironmentStatus) []depsvc.DeclaredDependency {
	dependencies := make([]depsvc.DeclaredDependency, 0)
	for _, result := range results {
		dependencies = append(dependencies, result.DeclaredDependencies...)
	}
	return dependencies
}

func resolveRepositoryDependencies(declared []depsvc.DeclaredDependency, resolvedByTicket map[string]depsvc.Resolution) []depsvc.Resolution {
	resolutions := make([]depsvc.Resolution, 0, len(declared))
	seen := make(map[string]struct{}, len(declared))
	for _, dependency := range declared {
		resolution, ok := resolvedByTicket[dependency.DependsOn]
		if !ok {
			continue
		}
		if _, ok := seen[dependency.DependsOn]; ok {
			continue
		}
		seen[dependency.DependsOn] = struct{}{}

		copyResolution := resolution
		copyResolution.EvidenceCommits = evidenceCommitsForDependency(declared, dependency.DependsOn)
		resolutions = append(resolutions, copyResolution)
	}

	sort.Slice(resolutions, func(i, j int) bool {
		return resolutions[i].DependsOn < resolutions[j].DependsOn
	})
	return resolutions
}

func evidenceCommitsForDependency(declared []depsvc.DeclaredDependency, dependsOn string) []string {
	evidence := make([]string, 0, 2)
	seen := map[string]struct{}{}
	for _, dependency := range declared {
		if dependency.DependsOn != dependsOn {
			continue
		}
		if _, ok := seen[dependency.CommitHash]; ok {
			continue
		}
		seen[dependency.CommitHash] = struct{}{}
		evidence = append(evidence, dependency.CommitHash)
	}
	sort.Strings(evidence)
	return evidence
}

func dependencyRiskSignals(resolutions []depsvc.Resolution, fromBranch, toBranch string) []inspectsvc.RiskSignal {
	signals := make([]inspectsvc.RiskSignal, 0, len(resolutions))
	for _, resolution := range resolutions {
		switch resolution.Status {
		case depsvc.StatusMissingTarget:
			signals = append(signals, inspectsvc.RiskSignal{
				Code:     "missing-dependency",
				Level:    "blocked",
				Summary:  fmt.Sprintf("Dependency ticket %s is present in %s but missing from %s", resolution.DependsOn, fromBranch, toBranch),
				Examples: []string{resolution.DependsOn},
			})
		case depsvc.StatusUnresolved:
			signals = append(signals, inspectsvc.RiskSignal{
				Code:     "unresolved-dependency",
				Level:    "warning",
				Summary:  fmt.Sprintf("Dependency ticket %s was declared but could not be confirmed in %s or %s", resolution.DependsOn, fromBranch, toBranch),
				Examples: []string{resolution.DependsOn},
			})
		}
	}
	return signals
}

func dependencyNotes(resolutions []depsvc.Resolution, fromBranch, toBranch string) []string {
	notes := make([]string, 0, len(resolutions))
	for _, resolution := range resolutions {
		switch resolution.Status {
		case depsvc.StatusMissingTarget:
			notes = append(notes, fmt.Sprintf("Declared dependency %s is present in %s but missing from %s.", resolution.DependsOn, fromBranch, toBranch))
		case depsvc.StatusUnresolved:
			notes = append(notes, fmt.Sprintf("Declared dependency %s could not be confirmed in %s or %s within the scanned workspace.", resolution.DependsOn, fromBranch, toBranch))
		}
	}
	return notes
}

func dependencyActions(resolutions []depsvc.Resolution, ticketID, toBranch string) []Action {
	actions := make([]Action, 0, len(resolutions))
	for _, resolution := range resolutions {
		switch resolution.Status {
		case depsvc.StatusMissingTarget:
			actions = append(actions, Action{
				Code:    "include-missing-dependency",
				Summary: fmt.Sprintf("Include dependency ticket %s in %s before promoting %s.", resolution.DependsOn, toBranch, ticketID),
			})
		case depsvc.StatusUnresolved:
			actions = append(actions, Action{
				Code:    "verify-dependency-scope",
				Summary: fmt.Sprintf("Confirm whether dependency ticket %s must travel with %s before promotion.", resolution.DependsOn, ticketID),
			})
		}
	}
	return actions
}

func buildDependencyPlanNotes(plans []RepositoryPlan, toBranch string) []string {
	missingDependencies, unresolvedDependencies := dependencyCountsFromPlans(plans)
	notes := make([]string, 0, 2)
	if missingDependencies > 0 {
		notes = append(notes, fmt.Sprintf("%d dependency ticket(s) are still missing from %s.", missingDependencies, toBranch))
	}
	if unresolvedDependencies > 0 {
		notes = append(notes, fmt.Sprintf("%d declared dependency ticket(s) could not be confirmed in the scanned workspace.", unresolvedDependencies))
	}
	return notes
}

func hasBlockedRiskSignals(riskSignals []inspectsvc.RiskSignal) bool {
	for _, riskSignal := range riskSignals {
		if riskSignal.Level == "blocked" {
			return true
		}
	}
	return false
}

func hasWarningRiskSignals(riskSignals []inspectsvc.RiskSignal) bool {
	for _, riskSignal := range riskSignals {
		switch riskSignal.Level {
		case "warning", "manual-review":
			return true
		}
	}
	return false
}

func sortRiskSignals(riskSignals []inspectsvc.RiskSignal) {
	sort.Slice(riskSignals, func(i, j int) bool {
		if riskSignals[i].Code == riskSignals[j].Code {
			return riskSignals[i].Summary < riskSignals[j].Summary
		}
		return riskSignals[i].Code < riskSignals[j].Code
	})
}

func dependencyCountsFromPlans(plans []RepositoryPlan) (int, int) {
	statuses := map[string]depsvc.Status{}
	for _, plan := range plans {
		for _, resolution := range plan.DependencyResolutions {
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

func refExists(ctx context.Context, adapter scm.Adapter, repoRoot, ref string) (bool, error) {
	checker, ok := adapter.(refChecker)
	if !ok {
		return true, nil
	}

	return checker.RefExists(ctx, repoRoot, ref)
}
