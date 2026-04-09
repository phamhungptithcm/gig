package releaseplan

import (
	"fmt"
	"sort"
	"strings"
	"time"

	depsvc "gig/internal/dependency"
	inspectsvc "gig/internal/inspect"
	plansvc "gig/internal/plan"
	"gig/internal/scm"
	snapshotsvc "gig/internal/snapshot"
)

type Summary struct {
	Snapshots             int `json:"snapshots"`
	Tickets               int `json:"tickets"`
	SafeTickets           int `json:"safeTickets"`
	WarningTickets        int `json:"warningTickets"`
	BlockedTickets        int `json:"blockedTickets"`
	TouchedRepositories   int `json:"touchedRepositories"`
	TotalCommitsToPromote int `json:"totalCommitsToPromote"`
	TotalManualSteps      int `json:"totalManualSteps"`
}

type TicketPlan struct {
	TicketID            string          `json:"ticketId"`
	CapturedAt          time.Time       `json:"capturedAt"`
	Verdict             plansvc.Verdict `json:"verdict"`
	TouchedRepositories int             `json:"touchedRepositories"`
	CommitsToPromote    int             `json:"commitsToPromote"`
	ManualSteps         int             `json:"manualSteps"`
}

type RepositoryPlan struct {
	Repository            scm.Repository          `json:"repository"`
	TicketIDs             []string                `json:"ticketIds,omitempty"`
	Verdict               plansvc.Verdict         `json:"verdict"`
	CommitsToInclude      int                     `json:"commitsToInclude"`
	RiskSignals           []inspectsvc.RiskSignal `json:"riskSignals,omitempty"`
	DependencyResolutions []depsvc.Resolution     `json:"dependencyResolutions,omitempty"`
	ManualSteps           []plansvc.Action        `json:"manualSteps,omitempty"`
	Actions               []plansvc.Action        `json:"actions,omitempty"`
	Notes                 []string                `json:"notes,omitempty"`
}

type ReleasePlan struct {
	ReleaseID    string                   `json:"releaseId"`
	Workspace    string                   `json:"workspace"`
	SnapshotDir  string                   `json:"snapshotDir"`
	FromBranch   string                   `json:"fromBranch,omitempty"`
	ToBranch     string                   `json:"toBranch,omitempty"`
	Environments []inspectsvc.Environment `json:"environments,omitempty"`
	Summary      Summary                  `json:"summary"`
	Verdict      plansvc.Verdict          `json:"verdict"`
	Notes        []string                 `json:"notes,omitempty"`
	Tickets      []TicketPlan             `json:"tickets,omitempty"`
	Repositories []RepositoryPlan         `json:"repositories,omitempty"`
}

func Build(releaseID, workspacePath, snapshotDir string, snapshots []snapshotsvc.TicketSnapshot) (ReleasePlan, error) {
	if len(snapshots) == 0 {
		return ReleasePlan{}, fmt.Errorf("at least one snapshot is required")
	}

	first := snapshots[0]
	releasePlan := ReleasePlan{
		ReleaseID:    releaseID,
		Workspace:    workspacePath,
		SnapshotDir:  snapshotDir,
		FromBranch:   first.FromBranch,
		ToBranch:     first.ToBranch,
		Environments: cloneEnvironments(first.Environments),
	}

	summary := Summary{
		Snapshots: len(snapshots),
		Tickets:   len(snapshots),
	}

	notes := make([]string, 0, 8)
	tickets := make([]TicketPlan, 0, len(snapshots))
	repositoryMap := make(map[string]*RepositoryPlan, 8)

	for _, snapshot := range snapshots {
		if mismatch := releaseMismatchNotes(snapshot, releasePlan.Workspace, releasePlan.FromBranch, releasePlan.ToBranch, releasePlan.Environments); len(mismatch) > 0 {
			notes = append(notes, mismatch...)
		}

		tickets = append(tickets, TicketPlan{
			TicketID:            snapshot.TicketID,
			CapturedAt:          snapshot.CapturedAt,
			Verdict:             snapshot.Plan.Verdict,
			TouchedRepositories: snapshot.Plan.Summary.TouchedRepositories,
			CommitsToPromote:    snapshot.Plan.Summary.TotalCommitsToPromote,
			ManualSteps:         snapshot.Plan.Summary.TotalManualSteps,
		})

		summary.TotalCommitsToPromote += snapshot.Plan.Summary.TotalCommitsToPromote
		summary.TotalManualSteps += snapshot.Plan.Summary.TotalManualSteps
		switch snapshot.Plan.Verdict {
		case plansvc.VerdictSafe:
			summary.SafeTickets++
		case plansvc.VerdictWarning:
			summary.WarningTickets++
		case plansvc.VerdictBlocked:
			summary.BlockedTickets++
		}

		for _, repositoryPlan := range snapshot.Plan.Repositories {
			key := string(repositoryPlan.Repository.Type) + "|" + repositoryPlan.Repository.Root
			aggregate, ok := repositoryMap[key]
			if !ok {
				copied := RepositoryPlan{
					Repository:            repositoryPlan.Repository,
					Verdict:               repositoryPlan.Verdict,
					CommitsToInclude:      len(repositoryPlan.Compare.MissingCommits),
					RiskSignals:           cloneRiskSignals(repositoryPlan.RiskSignals),
					DependencyResolutions: cloneDependencyResolutions(repositoryPlan.DependencyResolutions),
					ManualSteps:           prefixActions(snapshot.TicketID, repositoryPlan.ManualSteps),
					Actions:               prefixActions(snapshot.TicketID, repositoryPlan.Actions),
					Notes:                 prefixNotes(snapshot.TicketID, repositoryPlan.Notes),
					TicketIDs:             []string{snapshot.TicketID},
				}
				repositoryMap[key] = &copied
				continue
			}

			if !containsString(aggregate.TicketIDs, snapshot.TicketID) {
				aggregate.TicketIDs = append(aggregate.TicketIDs, snapshot.TicketID)
			}
			aggregate.Verdict = maxVerdict(aggregate.Verdict, repositoryPlan.Verdict)
			aggregate.CommitsToInclude += len(repositoryPlan.Compare.MissingCommits)
			aggregate.RiskSignals = mergeRiskSignals(aggregate.RiskSignals, repositoryPlan.RiskSignals)
			aggregate.DependencyResolutions = mergeDependencyResolutions(aggregate.DependencyResolutions, repositoryPlan.DependencyResolutions)
			aggregate.ManualSteps = mergeActions(aggregate.ManualSteps, prefixActions(snapshot.TicketID, repositoryPlan.ManualSteps))
			aggregate.Actions = mergeActions(aggregate.Actions, prefixActions(snapshot.TicketID, repositoryPlan.Actions))
			aggregate.Notes = mergeStrings(aggregate.Notes, prefixNotes(snapshot.TicketID, repositoryPlan.Notes))
		}
	}

	repositories := make([]RepositoryPlan, 0, len(repositoryMap))
	for _, repositoryPlan := range repositoryMap {
		sort.Strings(repositoryPlan.TicketIDs)
		repositories = append(repositories, *repositoryPlan)
	}
	sort.Slice(repositories, func(i, j int) bool {
		return repositories[i].Repository.Root < repositories[j].Repository.Root
	})
	sort.Slice(tickets, func(i, j int) bool {
		return tickets[i].TicketID < tickets[j].TicketID
	})

	summary.TouchedRepositories = len(repositories)
	notes = append(notes, buildSummaryNotes(summary, releasePlan.ToBranch)...)
	notes = dedupeStrings(notes)

	releasePlan.Summary = summary
	releasePlan.Verdict = deriveReleaseVerdict(summary, notes)
	releasePlan.Notes = notes
	releasePlan.Tickets = tickets
	releasePlan.Repositories = repositories

	return releasePlan, nil
}

func releaseMismatchNotes(snapshot snapshotsvc.TicketSnapshot, workspacePath, fromBranch, toBranch string, environments []inspectsvc.Environment) []string {
	notes := make([]string, 0, 3)
	if snapshot.Workspace != workspacePath {
		notes = append(notes, fmt.Sprintf("Snapshot %s was captured from %s instead of %s.", snapshot.TicketID, snapshot.Workspace, workspacePath))
	}
	if snapshot.FromBranch != fromBranch || snapshot.ToBranch != toBranch {
		notes = append(notes, fmt.Sprintf("Snapshot %s uses %s -> %s instead of %s -> %s.", snapshot.TicketID, snapshot.FromBranch, snapshot.ToBranch, fromBranch, toBranch))
	}
	if !sameEnvironments(snapshot.Environments, environments) {
		notes = append(notes, fmt.Sprintf("Snapshot %s uses a different environment mapping.", snapshot.TicketID))
	}
	return notes
}

func buildSummaryNotes(summary Summary, toBranch string) []string {
	notes := make([]string, 0, 4)
	if summary.BlockedTickets > 0 {
		notes = append(notes, fmt.Sprintf("%d ticket snapshot(s) are currently blocked.", summary.BlockedTickets))
	}
	if summary.WarningTickets > 0 {
		notes = append(notes, fmt.Sprintf("%d ticket snapshot(s) still need manual review.", summary.WarningTickets))
	}
	if summary.TotalCommitsToPromote > 0 {
		notes = append(notes, fmt.Sprintf("%d total ticket commit(s) are still planned for %s.", summary.TotalCommitsToPromote, toBranch))
	}
	if summary.TotalManualSteps > 0 {
		notes = append(notes, fmt.Sprintf("%d manual review step(s) are still open across this release.", summary.TotalManualSteps))
	}
	return notes
}

func deriveReleaseVerdict(summary Summary, notes []string) plansvc.Verdict {
	if summary.Tickets == 0 {
		return plansvc.VerdictBlocked
	}
	for _, note := range notes {
		if strings.Contains(note, "instead of") || strings.Contains(note, "different environment mapping") {
			return plansvc.VerdictBlocked
		}
	}
	switch {
	case summary.BlockedTickets > 0:
		return plansvc.VerdictBlocked
	case summary.WarningTickets > 0:
		return plansvc.VerdictWarning
	default:
		return plansvc.VerdictSafe
	}
}

func maxVerdict(left, right plansvc.Verdict) plansvc.Verdict {
	if left == plansvc.VerdictBlocked || right == plansvc.VerdictBlocked {
		return plansvc.VerdictBlocked
	}
	if left == plansvc.VerdictWarning || right == plansvc.VerdictWarning {
		return plansvc.VerdictWarning
	}
	return plansvc.VerdictSafe
}

func cloneEnvironments(environments []inspectsvc.Environment) []inspectsvc.Environment {
	if len(environments) == 0 {
		return nil
	}
	cloned := make([]inspectsvc.Environment, len(environments))
	copy(cloned, environments)
	return cloned
}

func cloneRiskSignals(signals []inspectsvc.RiskSignal) []inspectsvc.RiskSignal {
	if len(signals) == 0 {
		return nil
	}
	cloned := make([]inspectsvc.RiskSignal, len(signals))
	copy(cloned, signals)
	return cloned
}

func cloneDependencyResolutions(resolutions []depsvc.Resolution) []depsvc.Resolution {
	if len(resolutions) == 0 {
		return nil
	}
	cloned := make([]depsvc.Resolution, len(resolutions))
	copy(cloned, resolutions)
	return cloned
}

func prefixActions(ticketID string, actions []plansvc.Action) []plansvc.Action {
	if len(actions) == 0 {
		return nil
	}
	prefixed := make([]plansvc.Action, 0, len(actions))
	for _, action := range actions {
		prefixed = append(prefixed, plansvc.Action{
			Code:    ticketID + ":" + action.Code,
			Summary: fmt.Sprintf("%s: %s", ticketID, action.Summary),
		})
	}
	return prefixed
}

func prefixNotes(ticketID string, notes []string) []string {
	if len(notes) == 0 {
		return nil
	}
	prefixed := make([]string, 0, len(notes))
	for _, note := range notes {
		prefixed = append(prefixed, fmt.Sprintf("%s: %s", ticketID, note))
	}
	return prefixed
}

func mergeRiskSignals(existing, incoming []inspectsvc.RiskSignal) []inspectsvc.RiskSignal {
	for _, signal := range incoming {
		key := signal.Code + "|" + signal.Level + "|" + signal.Summary
		found := false
		for _, current := range existing {
			if key == current.Code+"|"+current.Level+"|"+current.Summary {
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, signal)
		}
	}
	sort.Slice(existing, func(i, j int) bool {
		return existing[i].Code < existing[j].Code
	})
	return existing
}

func mergeDependencyResolutions(existing, incoming []depsvc.Resolution) []depsvc.Resolution {
	for _, resolution := range incoming {
		key := resolution.TicketID + "|" + resolution.DependsOn + "|" + string(resolution.Status)
		found := false
		for _, current := range existing {
			if key == current.TicketID+"|"+current.DependsOn+"|"+string(current.Status) {
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, resolution)
		}
	}
	sort.Slice(existing, func(i, j int) bool {
		if existing[i].TicketID == existing[j].TicketID {
			return existing[i].DependsOn < existing[j].DependsOn
		}
		return existing[i].TicketID < existing[j].TicketID
	})
	return existing
}

func mergeActions(existing, incoming []plansvc.Action) []plansvc.Action {
	for _, action := range incoming {
		found := false
		for _, current := range existing {
			if action.Code == current.Code && action.Summary == current.Summary {
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, action)
		}
	}
	sort.Slice(existing, func(i, j int) bool {
		return existing[i].Code < existing[j].Code
	})
	return existing
}

func mergeStrings(existing, incoming []string) []string {
	existing = append(existing, incoming...)
	return dedupeStrings(existing)
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	unique := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	sort.Strings(unique)
	return unique
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sameEnvironments(left, right []inspectsvc.Environment) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
