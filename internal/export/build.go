package exporter

import (
	"fmt"
	"sort"
	"strings"

	"gig/internal/buildinfo"
	depsvc "gig/internal/dependency"
	inspectsvc "gig/internal/inspect"
	manifestsvc "gig/internal/manifest"
	plansvc "gig/internal/plan"
	"gig/internal/scm"
)

var (
	decisionHeaders = []string{"Decision", "Reason", "Severity", "Owner", "Next action", "Evidence reference"}
	riskHeaders     = []string{"Severity", "Category", "Finding", "Impact", "Evidence", "Recommended action", "Owner", "Status"}
	missingHeaders  = []string{"Ticket", "Commit", "Short commit", "Subject", "Author", "Date", "Present in source", "Present in target", "Risk", "Recommended action"}
	commitHeaders   = []string{"Ticket", "Commit", "Short commit", "Subject", "Author", "Author email, if available", "Date", "Branch", "Files changed, if available", "Source URL, if remote"}
	manualHeaders   = []string{"Step", "Reason", "Source evidence", "Required before release", "Owner", "Status", "Notes"}
	evidenceHeaders = []string{"ID", "Type", "Description", "Source", "Reference", "Captured at"}
)

func BuildVerificationExport(plans []plansvc.PromotionPlan, verifications []plansvc.Verification, options Options) ReleaseExport {
	options = normalizeOptions(options)
	sheets := []Sheet{
		buildVerificationSummarySheet(plans, verifications, options),
		{
			Name:    "Decision",
			CSVName: "decision.csv",
			Headers: decisionHeaders,
			Rows:    buildVerificationDecisionRows(plans, verifications, options),
		},
		{
			Name:    "Risks",
			CSVName: "risks.csv",
			Headers: riskHeaders,
			Rows:    buildPlanRiskRows(plans, nil),
		},
		{
			Name:    "Missing Changes",
			CSVName: "missing-changes.csv",
			Headers: missingHeaders,
			Rows:    buildPlanMissingRows(plans),
		},
		{
			Name:    "Commits",
			CSVName: "commits.csv",
			Headers: commitHeaders,
			Rows:    buildPlanCommitRows(plans),
		},
		{
			Name:    "Manual Steps",
			CSVName: "manual-steps.csv",
			Headers: manualHeaders,
			Rows:    buildPlanManualStepRows(plans, nil),
		},
		{
			Name:    "Evidence",
			CSVName: "evidence.csv",
			Headers: evidenceHeaders,
			Rows:    buildPlanEvidenceRows(plans, options),
		},
		buildMetadataSheet(options),
	}
	return ReleaseExport{
		Sheets:    sheets,
		SingleCSV: buildVerificationSingleCSV(plans, verifications),
	}
}

func BuildReleasePacketExport(packets []manifestsvc.ReleasePacket, options Options) ReleaseExport {
	options = normalizeOptions(options)
	sheets := []Sheet{
		buildCoverSheet(packets, options),
		{
			Name:    "Release Decision",
			CSVName: "release-decision.csv",
			Headers: decisionHeaders,
			Rows:    buildPacketDecisionRows(packets, options),
		},
		{
			Name:    "Scope",
			CSVName: "scope.csv",
			Headers: []string{"Type", "Value", "Notes"},
			Rows:    buildPacketScopeRows(packets, options),
		},
		{
			Name:    "Risks",
			CSVName: "risks.csv",
			Headers: riskHeaders,
			Rows:    buildPacketRiskRows(packets),
		},
		{
			Name:    "Missing Changes",
			CSVName: "missing-changes.csv",
			Headers: missingHeaders,
			Rows:    buildPacketMissingRows(packets),
		},
		{
			Name:    "Commits",
			CSVName: "commits.csv",
			Headers: commitHeaders,
			Rows:    buildPacketCommitRows(packets),
		},
		{
			Name:    "Manual Steps",
			CSVName: "manual-steps.csv",
			Headers: manualHeaders,
			Rows:    buildPacketManualStepRows(packets),
		},
		{
			Name:    "Verification",
			CSVName: "verification.csv",
			Headers: []string{"Check", "Result", "Details", "Evidence", "Next action"},
			Rows:    buildPacketVerificationRows(packets, options),
		},
		{
			Name:    "Approvals",
			CSVName: "approvals.csv",
			Headers: []string{"Role", "Name", "Decision", "Date", "Notes"},
			Rows: [][]string{
				{"Release manager", "", "", "", ""},
				{"Engineering lead", "", "", "", ""},
				{"QA", "", "", "", ""},
				{"Product owner", "", "", "", ""},
				{"Compliance", "", "", "", ""},
			},
		},
		{
			Name:    "Evidence",
			CSVName: "evidence.csv",
			Headers: evidenceHeaders,
			Rows:    buildPacketEvidenceRows(packets, options),
		},
		buildMetadataSheet(options),
	}
	return ReleaseExport{Sheets: sheets}
}

func normalizeOptions(options Options) Options {
	if strings.TrimSpace(options.ScopeLabel) == "" {
		options.ScopeLabel = strings.TrimSpace(options.WorkingDirectory)
	}
	if strings.TrimSpace(options.WorkingDirectory) == "" {
		options.WorkingDirectory = strings.TrimSpace(options.ScopeLabel)
	}
	if strings.TrimSpace(options.Provider) == "" {
		options.Provider = "local"
	}
	if strings.TrimSpace(options.Mode) == "" {
		if options.Provider == "local" || options.Provider == "git" || options.Provider == "svn" {
			options.Mode = "local"
		} else {
			options.Mode = "remote"
		}
	}
	return options
}

func buildVerificationSummarySheet(plans []plansvc.PromotionPlan, verifications []plansvc.Verification, options Options) Sheet {
	summary := summarizePlans(plans)
	if len(plans) == 0 {
		summary = summarizeVerifications(verifications)
	}
	fields := []Field{
		{"Ticket", strings.Join(ticketIDsFromPlans(plans, verifications), ", ")},
		{"Repo scope", options.ScopeLabel},
		{"Mode", options.Mode},
		{"Provider", options.Provider},
		{"Source", firstPlanSource(plans, verifications)},
		{"Target", firstPlanTarget(plans, verifications)},
		{"Commits found", fmt.Sprintf("%d", countSourceCommits(plans, verifications))},
		{"Missing commits", fmt.Sprintf("%d", summary.TotalCommitsToPromote)},
		{"Risks found", fmt.Sprintf("%d", countPlanRisks(plans))},
		{"Manual steps found", fmt.Sprintf("%d", summary.TotalManualSteps)},
		{"Verification status", string(overallVerdict(plans, verifications))},
		{"Generated at", options.generatedAtString()},
	}
	return fieldSheet("Summary", "summary.csv", fields)
}

func buildCoverSheet(packets []manifestsvc.ReleasePacket, options Options) Sheet {
	fields := []Field{
		{"Ticket", strings.Join(ticketIDsFromPackets(packets), ", ")},
		{"Repository", repositoryScopeFromPackets(packets, options)},
		{"Source branch", firstPacketSource(packets)},
		{"Target branch", firstPacketTarget(packets)},
		{"Generated at", options.generatedAtString()},
		{"Generated by", options.GeneratedBy},
		{"Command", options.Command},
		{"Overall status", string(packetOverallVerdict(packets))},
		{"Recommended action", recommendedAction(packetDecision(packets, options))},
	}
	return fieldSheet("Cover", "summary.csv", fields)
}

func buildVerificationDecisionRows(plans []plansvc.PromotionPlan, verifications []plansvc.Verification, options Options) [][]string {
	decision := verificationDecision(plans, verifications, options)
	return [][]string{{
		string(decision),
		strings.Join(verificationReasons(plans, verifications, options), "; "),
		decisionSeverity(decision),
		"",
		recommendedAction(decision),
		"verification",
	}}
}

func buildPacketDecisionRows(packets []manifestsvc.ReleasePacket, options Options) [][]string {
	decision := packetDecision(packets, options)
	return [][]string{{
		string(decision),
		strings.Join(packetReasons(packets, options), "; "),
		decisionSeverity(decision),
		packetOwner(packets),
		recommendedAction(decision),
		"packet",
	}}
}

func buildPacketScopeRows(packets []manifestsvc.ReleasePacket, options Options) [][]string {
	rows := [][]string{
		{"Ticket", strings.Join(ticketIDsFromPackets(packets), ", "), "Release packet ticket scope."},
		{"Repository", repositoryScopeFromPackets(packets, options), "Remote targets are shown as provider scopes when available."},
		{"Provider", options.Provider, "Source-control provider or local SCM."},
		{"Source branch", firstPacketSource(packets), "Promotion source."},
		{"Target branch", firstPacketTarget(packets), "Release target."},
		{"Included commits", fmt.Sprintf("%d", countPacketIncludedCommits(packets)), "Ticket commits expected to move with this packet."},
		{"Excluded commits, if known", "", "Not reported by the current evidence model."},
		{"Config source", firstNonEmpty(options.ConfigPath, firstPacketConfigPath(packets), "none"), "Config is optional and used only as an enhancement."},
		{"Auth source, redacted", firstNonEmpty(options.AuthSource, "redacted"), "Credentials and tokens are never exported."},
	}
	return rows
}

func buildPlanRiskRows(plans []plansvc.PromotionPlan, owners map[string]string) [][]string {
	rows := make([][]string, 0)
	for _, plan := range plans {
		for _, repository := range plan.Repositories {
			owner := ownerForRepository(owners, repository.Repository.Root)
			rows = append(rows, riskRowsForSignals(repository.RiskSignals, repository.Repository.Name, owner)...)
			rows = append(rows, riskRowsForDependencies(repository.DependencyResolutions, repository.Repository.Name, owner)...)
		}
	}
	return rows
}

func buildPacketRiskRows(packets []manifestsvc.ReleasePacket) [][]string {
	rows := make([][]string, 0)
	for _, packet := range packets {
		for _, repository := range packet.Repositories {
			owner := packetRepositoryOwner(repository)
			rows = append(rows, riskRowsForSignals(repository.RiskSignals, repository.Repository.Name, owner)...)
			rows = append(rows, riskRowsForDependencies(repository.DependencyResolutions, repository.Repository.Name, owner)...)
		}
	}
	return rows
}

func riskRowsForSignals(signals []inspectsvc.RiskSignal, repository, owner string) [][]string {
	rows := make([][]string, 0, len(signals))
	for _, signal := range signals {
		severity := severityForRiskLevel(signal.Level)
		finding := firstNonEmpty(signal.Summary, signal.Code)
		rows = append(rows, []string{
			severity,
			firstNonEmpty(signal.Code, "risk"),
			finding,
			riskImpact(signal.Level),
			strings.Join(signal.Examples, ", "),
			recommendedRiskAction(signal.Code),
			owner,
			"Open",
		})
		_ = repository
	}
	return rows
}

func riskRowsForDependencies(resolutions []depsvc.Resolution, repository, owner string) [][]string {
	rows := make([][]string, 0, len(resolutions))
	for _, resolution := range resolutions {
		switch resolution.Status {
		case depsvc.StatusMissingTarget:
			rows = append(rows, []string{
				"High",
				"missing-dependency",
				fmt.Sprintf("Dependency ticket %s is missing from the target branch.", resolution.DependsOn),
				"Release scope may be incomplete.",
				strings.Join(resolution.EvidenceCommits, ", "),
				"Include or explicitly accept the dependency before release.",
				owner,
				"Open",
			})
		case depsvc.StatusUnresolved:
			rows = append(rows, []string{
				"Medium",
				"unresolved-dependency",
				fmt.Sprintf("Dependency ticket %s could not be confirmed.", resolution.DependsOn),
				"Release scope needs manual confirmation.",
				strings.Join(resolution.EvidenceCommits, ", "),
				"Confirm whether the dependency must travel with this ticket.",
				owner,
				"Open",
			})
		}
		_ = repository
	}
	return rows
}

func buildPlanMissingRows(plans []plansvc.PromotionPlan) [][]string {
	rows := make([][]string, 0)
	for _, plan := range plans {
		for _, repository := range plan.Repositories {
			for _, commit := range repository.Compare.MissingCommits {
				rows = append(rows, missingCommitRow(plan.TicketID, commit, "Yes", "No"))
			}
		}
	}
	return rows
}

func buildPacketMissingRows(packets []manifestsvc.ReleasePacket) [][]string {
	rows := make([][]string, 0)
	for _, packet := range packets {
		for _, repository := range packet.Repositories {
			for _, commit := range repository.CommitsToInclude {
				rows = append(rows, missingCommitRow(packet.TicketID, commit, "Yes", "No"))
			}
		}
	}
	return rows
}

func missingCommitRow(ticketID string, commit scm.Commit, presentSource, presentTarget string) []string {
	return []string{
		ticketID,
		commit.Hash,
		commit.ShortHash(),
		commit.Subject,
		"",
		"",
		presentSource,
		presentTarget,
		"Missing from target",
		"Promote, cherry-pick, or explicitly exclude this commit before release.",
	}
}

func buildPlanCommitRows(plans []plansvc.PromotionPlan) [][]string {
	rows := make([][]string, 0)
	for _, plan := range plans {
		for _, repository := range plan.Repositories {
			for _, commit := range repository.Compare.SourceCommits {
				rows = append(rows, commitRow(plan.TicketID, commit, plan.FromBranch, ""))
			}
			for _, commit := range repository.Compare.TargetCommits {
				rows = append(rows, commitRow(plan.TicketID, commit, plan.ToBranch, ""))
			}
		}
	}
	return rows
}

func buildPacketCommitRows(packets []manifestsvc.ReleasePacket) [][]string {
	rows := make([][]string, 0)
	for _, packet := range packets {
		for _, repository := range packet.Repositories {
			for _, commit := range repository.CommitsToInclude {
				rows = append(rows, commitRow(packet.TicketID, commit, packet.FromBranch, ""))
			}
		}
	}
	return rows
}

func commitRow(ticketID string, commit scm.Commit, branch, sourceURL string) []string {
	return []string{ticketID, commit.Hash, commit.ShortHash(), commit.Subject, "", "", "", branch, "", sourceURL}
}

func buildPlanManualStepRows(plans []plansvc.PromotionPlan, owners map[string]string) [][]string {
	rows := make([][]string, 0)
	for _, plan := range plans {
		for _, repository := range plan.Repositories {
			owner := ownerForRepository(owners, repository.Repository.Root)
			for _, step := range repository.ManualSteps {
				rows = append(rows, []string{step.Summary, step.Code, repository.Repository.Name, "Yes", owner, "Open", ""})
			}
		}
	}
	return rows
}

func buildPacketManualStepRows(packets []manifestsvc.ReleasePacket) [][]string {
	rows := make([][]string, 0)
	for _, packet := range packets {
		for _, repository := range packet.Repositories {
			owner := packetRepositoryOwner(repository)
			for _, step := range repository.ManualSteps {
				rows = append(rows, []string{step.Summary, step.Code, repository.Repository.Name, "Yes", owner, "Open", ""})
			}
			for _, action := range repository.Actions {
				if action.Code == "already-aligned" {
					continue
				}
				rows = append(rows, []string{action.Summary, action.Code, repository.Repository.Name, "Yes", owner, "Open", ""})
			}
		}
	}
	return rows
}

func buildVerificationSingleCSV(plans []plansvc.PromotionPlan, verifications []plansvc.Verification) Sheet {
	headers := []string{"Ticket", "Repository", "Check", "Result", "Details", "Evidence", "Next action"}
	rows := make([][]string, 0)
	for _, verification := range verifications {
		if len(verification.Repositories) == 0 {
			rows = append(rows, []string{verification.TicketID, "", "Ticket commits discovered", resultForVerdict(verification.Verdict), strings.Join(verification.Reasons, "; "), "", nextActionForVerdict(verification.Verdict)})
			continue
		}
		for _, repository := range verification.Repositories {
			for _, check := range repository.Checks {
				rows = append(rows, []string{verification.TicketID, repository.Repository.Name, "Verification check", resultForVerdict(repository.Verdict), check, repository.Repository.Root, nextActionForVerdict(repository.Verdict)})
			}
		}
	}
	if len(rows) == 0 {
		for _, plan := range plans {
			rows = append(rows, []string{plan.TicketID, "", "Ticket commits discovered", resultForVerdict(plan.Verdict), strings.Join(plan.Notes, "; "), "", nextActionForVerdict(plan.Verdict)})
		}
	}
	return Sheet{Name: "Verification", CSVName: "verify.csv", Headers: headers, Rows: rows}
}

func buildPacketVerificationRows(packets []manifestsvc.ReleasePacket, options Options) [][]string {
	rows := make([][]string, 0, len(packets)*6)
	for _, packet := range packets {
		commits := countPacketIncludedCommits([]manifestsvc.ReleasePacket{packet})
		risks := countPacketRisks([]manifestsvc.ReleasePacket{packet})
		manualSteps := packet.Summary.TotalManualSteps
		rows = append(rows,
			[]string{"Ticket commits discovered", resultForBool(packet.Summary.TouchedRepositories > 0), fmt.Sprintf("%d touched repositories", packet.Summary.TouchedRepositories), packet.TicketID, nextActionForVerdict(packet.Verdict)},
			[]string{"Source/target comparison", resultForBool(commits == 0), fmt.Sprintf("%d missing commits from %s to %s", commits, packet.FromBranch, packet.ToBranch), "Missing Changes", nextActionForVerdict(packet.Verdict)},
			[]string{"Missing changes scan", resultForBool(commits == 0), fmt.Sprintf("%d missing commits", commits), "Missing Changes", nextActionForVerdict(packet.Verdict)},
			[]string{"Risk inference", resultForBool(risks == 0), fmt.Sprintf("%d risk findings", risks), "Risks", recommendedRiskReviewAction(risks)},
			[]string{"Manual step inference", resultForBool(manualSteps == 0), fmt.Sprintf("%d manual steps", manualSteps), "Manual Steps", recommendedManualStepAction(manualSteps)},
			[]string{"Provider auth", providerAuthResult(options), providerAuthDetails(options), "Metadata", ""},
			[]string{"Branch topology inference", resultForBool(packet.FromBranch != "" && packet.ToBranch != ""), packet.FromBranch + " -> " + packet.ToBranch, "Scope", ""},
		)
	}
	return rows
}

func buildPlanEvidenceRows(plans []plansvc.PromotionPlan, options Options) [][]string {
	rows := make([][]string, 0)
	for _, plan := range plans {
		for _, repository := range plan.Repositories {
			rows = appendEvidenceRows(rows, repository.Repository.Name, repository.ProviderEvidence, options)
		}
	}
	return rows
}

func buildPacketEvidenceRows(packets []manifestsvc.ReleasePacket, options Options) [][]string {
	rows := make([][]string, 0)
	for _, packet := range packets {
		for _, repository := range packet.Repositories {
			rows = appendEvidenceRows(rows, repository.Repository.Name, repository.ProviderEvidence, options)
		}
	}
	return rows
}

func appendEvidenceRows(rows [][]string, repository string, evidence *scm.ProviderEvidence, options Options) [][]string {
	if evidence == nil || evidence.IsZero() {
		return rows
	}
	for _, item := range sortedPullRequests(evidence.PullRequests) {
		rows = append(rows, evidenceRow(len(rows)+1, "Pull request", pullRequestDescription(item), repository, firstNonEmpty(item.URL, item.CommitHash), options))
		for _, issue := range sortedIssues(item.LinkedIssues) {
			rows = append(rows, evidenceRow(len(rows)+1, "Linked issue", issueDescription(issue), repository, firstNonEmpty(issue.URL, issue.ID), options))
		}
	}
	for _, item := range sortedDeployments(evidence.Deployments) {
		rows = append(rows, evidenceRow(len(rows)+1, "Deployment", deploymentDescription(item), repository, firstNonEmpty(item.URL, item.CommitHash), options))
	}
	for _, item := range sortedChecks(evidence.Checks) {
		rows = append(rows, evidenceRow(len(rows)+1, "Check", checkDescription(item), repository, firstNonEmpty(item.URL, item.CommitHash), options))
	}
	for _, item := range sortedIssues(evidence.Issues) {
		rows = append(rows, evidenceRow(len(rows)+1, "Issue", issueDescription(item), repository, firstNonEmpty(item.URL, item.ID), options))
	}
	for _, item := range sortedReleases(evidence.Releases) {
		rows = append(rows, evidenceRow(len(rows)+1, "Release", releaseDescription(item), repository, firstNonEmpty(item.URL, item.Tag, item.ID), options))
	}
	return rows
}

func evidenceRow(id int, evidenceType, description, source, reference string, options Options) []string {
	return []string{fmt.Sprintf("E%d", id), evidenceType, description, source, reference, options.generatedAtString()}
}

func buildMetadataSheet(options Options) Sheet {
	fields := []Field{
		{"gig version", buildinfo.Summary()},
		{"command", options.Command},
		{"working directory or repo target", options.WorkingDirectory},
		{"provider", options.Provider},
		{"generated at", options.generatedAtString()},
		{"timezone", options.timezone()},
		{"config file used", options.ConfigPath},
		{"git version", options.ToolVersions["git"]},
		{"svn version", options.ToolVersions["svn"]},
		{"gh version", options.ToolVersions["gh"]},
		{"glab version", options.ToolVersions["glab"]},
		{"az version", options.ToolVersions["az"]},
		{"JSON schema version", options.JSONSchemaVersion},
	}
	return fieldSheet("Metadata", "metadata.csv", fields)
}

type releaseDecision string

const (
	decisionReady       releaseDecision = "ready"
	decisionNeedsReview releaseDecision = "needs_review"
	decisionBlocked     releaseDecision = "blocked"
	decisionUnknown     releaseDecision = "unknown"
)

func verificationDecision(plans []plansvc.PromotionPlan, verifications []plansvc.Verification, options Options) releaseDecision {
	if options.DataIncomplete {
		return decisionUnknown
	}
	switch overallVerdict(plans, verifications) {
	case plansvc.VerdictSafe:
		return decisionReady
	case plansvc.VerdictWarning:
		return decisionNeedsReview
	default:
		return decisionBlocked
	}
}

func packetDecision(packets []manifestsvc.ReleasePacket, options Options) releaseDecision {
	if options.DataIncomplete {
		return decisionUnknown
	}
	switch packetOverallVerdict(packets) {
	case plansvc.VerdictSafe:
		return decisionReady
	case plansvc.VerdictWarning:
		return decisionNeedsReview
	default:
		return decisionBlocked
	}
}

func decisionSeverity(decision releaseDecision) string {
	switch decision {
	case decisionBlocked:
		return "High"
	case decisionNeedsReview:
		return "Medium"
	case decisionUnknown:
		return "Info"
	default:
		return "Info"
	}
}

func recommendedAction(decision releaseDecision) string {
	switch decision {
	case decisionReady:
		return "Proceed with release approval using the attached evidence."
	case decisionNeedsReview:
		return "Review risks, manual steps, and incomplete evidence before release."
	case decisionUnknown:
		return "Resolve setup, auth, provider, or config access and rerun gig."
	default:
		return "Resolve blocking checks and missing release changes before release."
	}
}

func verificationReasons(plans []plansvc.PromotionPlan, verifications []plansvc.Verification, options Options) []string {
	if options.DataIncomplete && strings.TrimSpace(options.IncompleteReason) != "" {
		return []string{options.IncompleteReason}
	}
	reasons := make([]string, 0)
	for _, verification := range verifications {
		reasons = append(reasons, verification.Reasons...)
	}
	if len(reasons) > 0 {
		return dedupe(reasons)
	}
	for _, plan := range plans {
		reasons = append(reasons, plan.Notes...)
	}
	return dedupe(reasons)
}

func packetReasons(packets []manifestsvc.ReleasePacket, options Options) []string {
	if options.DataIncomplete && strings.TrimSpace(options.IncompleteReason) != "" {
		return []string{options.IncompleteReason}
	}
	reasons := make([]string, 0)
	for _, packet := range packets {
		reasons = append(reasons, packet.Highlights...)
	}
	return dedupe(reasons)
}

func overallVerdict(plans []plansvc.PromotionPlan, verifications []plansvc.Verification) plansvc.Verdict {
	summary := summarizePlans(plans)
	if len(plans) == 0 {
		summary = summarizeVerifications(verifications)
	}
	switch {
	case summary.BlockedRepositories > 0 || summary.TouchedRepositories == 0:
		return plansvc.VerdictBlocked
	case summary.WarningRepositories > 0:
		return plansvc.VerdictWarning
	default:
		return plansvc.VerdictSafe
	}
}

func packetOverallVerdict(packets []manifestsvc.ReleasePacket) plansvc.Verdict {
	anyWarning := false
	if len(packets) == 0 {
		return plansvc.VerdictBlocked
	}
	for _, packet := range packets {
		switch packet.Verdict {
		case plansvc.VerdictBlocked:
			return plansvc.VerdictBlocked
		case plansvc.VerdictWarning:
			anyWarning = true
		}
	}
	if anyWarning {
		return plansvc.VerdictWarning
	}
	return plansvc.VerdictSafe
}

func summarizePlans(plans []plansvc.PromotionPlan) plansvc.Summary {
	var summary plansvc.Summary
	for _, plan := range plans {
		summary.ScannedRepositories += plan.Summary.ScannedRepositories
		summary.TouchedRepositories += plan.Summary.TouchedRepositories
		summary.ReadyRepositories += plan.Summary.ReadyRepositories
		summary.WarningRepositories += plan.Summary.WarningRepositories
		summary.BlockedRepositories += plan.Summary.BlockedRepositories
		summary.TotalCommitsToPromote += plan.Summary.TotalCommitsToPromote
		summary.TotalManualSteps += plan.Summary.TotalManualSteps
	}
	return summary
}

func summarizeVerifications(verifications []plansvc.Verification) plansvc.Summary {
	var summary plansvc.Summary
	for _, verification := range verifications {
		summary.ScannedRepositories += verification.Summary.ScannedRepositories
		summary.TouchedRepositories += verification.Summary.TouchedRepositories
		summary.ReadyRepositories += verification.Summary.ReadyRepositories
		summary.WarningRepositories += verification.Summary.WarningRepositories
		summary.BlockedRepositories += verification.Summary.BlockedRepositories
		summary.TotalCommitsToPromote += verification.Summary.TotalCommitsToPromote
		summary.TotalManualSteps += verification.Summary.TotalManualSteps
	}
	return summary
}

func countSourceCommits(plans []plansvc.PromotionPlan, verifications []plansvc.Verification) int {
	if len(plans) == 0 {
		count := 0
		for _, verification := range verifications {
			for _, repository := range verification.Repositories {
				if len(repository.Checks) > 0 {
					count++
				}
			}
		}
		return count
	}
	count := 0
	for _, plan := range plans {
		for _, repository := range plan.Repositories {
			count += len(repository.Compare.SourceCommits)
		}
	}
	return count
}

func countPlanRisks(plans []plansvc.PromotionPlan) int {
	count := 0
	for _, plan := range plans {
		for _, repository := range plan.Repositories {
			count += len(repository.RiskSignals)
			for _, resolution := range repository.DependencyResolutions {
				if resolution.Status != depsvc.StatusSatisfied {
					count++
				}
			}
		}
	}
	return count
}

func countPacketRisks(packets []manifestsvc.ReleasePacket) int {
	count := 0
	for _, packet := range packets {
		for _, repository := range packet.Repositories {
			count += len(repository.RiskSignals)
			for _, resolution := range repository.DependencyResolutions {
				if resolution.Status != depsvc.StatusSatisfied {
					count++
				}
			}
		}
	}
	return count
}

func countPacketIncludedCommits(packets []manifestsvc.ReleasePacket) int {
	count := 0
	for _, packet := range packets {
		for _, repository := range packet.Repositories {
			count += len(repository.CommitsToInclude)
		}
	}
	return count
}

func ticketIDsFromPlans(plans []plansvc.PromotionPlan, verifications []plansvc.Verification) []string {
	ids := make([]string, 0, len(plans)+len(verifications))
	for _, plan := range plans {
		ids = append(ids, plan.TicketID)
	}
	for _, verification := range verifications {
		ids = append(ids, verification.TicketID)
	}
	return dedupe(ids)
}

func ticketIDsFromPackets(packets []manifestsvc.ReleasePacket) []string {
	ids := make([]string, 0, len(packets))
	for _, packet := range packets {
		ids = append(ids, packet.TicketID)
	}
	return dedupe(ids)
}

func firstPlanSource(plans []plansvc.PromotionPlan, verifications []plansvc.Verification) string {
	if len(plans) > 0 {
		return plans[0].FromBranch
	}
	if len(verifications) > 0 {
		return verifications[0].FromBranch
	}
	return ""
}

func firstPlanTarget(plans []plansvc.PromotionPlan, verifications []plansvc.Verification) string {
	if len(plans) > 0 {
		return plans[0].ToBranch
	}
	if len(verifications) > 0 {
		return verifications[0].ToBranch
	}
	return ""
}

func firstPacketSource(packets []manifestsvc.ReleasePacket) string {
	if len(packets) == 0 {
		return ""
	}
	return packets[0].FromBranch
}

func firstPacketTarget(packets []manifestsvc.ReleasePacket) string {
	if len(packets) == 0 {
		return ""
	}
	return packets[0].ToBranch
}

func firstPacketConfigPath(packets []manifestsvc.ReleasePacket) string {
	for _, packet := range packets {
		if strings.TrimSpace(packet.ConfigPath) != "" {
			return packet.ConfigPath
		}
	}
	return ""
}

func repositoryScopeFromPackets(packets []manifestsvc.ReleasePacket, options Options) string {
	if strings.TrimSpace(options.ScopeLabel) != "" {
		return options.ScopeLabel
	}
	if len(packets) == 0 {
		return ""
	}
	return packets[0].Workspace
}

func packetOwner(packets []manifestsvc.ReleasePacket) string {
	owners := make([]string, 0)
	for _, packet := range packets {
		for _, repository := range packet.Repositories {
			if owner := packetRepositoryOwner(repository); owner != "" {
				owners = append(owners, owner)
			}
		}
	}
	return strings.Join(dedupe(owners), ", ")
}

func packetRepositoryOwner(repository manifestsvc.RepositoryPacket) string {
	if repository.ConfigEntry == nil {
		return ""
	}
	return strings.TrimSpace(repository.ConfigEntry.Owner)
}

func ownerForRepository(owners map[string]string, root string) string {
	if owners == nil {
		return ""
	}
	return strings.TrimSpace(owners[root])
}

func severityForRiskLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "critical":
		return "Critical"
	case "blocked", "blocker", "high":
		return "High"
	case "warning", "manual-review", "medium":
		return "Medium"
	case "low":
		return "Low"
	default:
		return "Info"
	}
}

func riskImpact(level string) string {
	switch severityForRiskLevel(level) {
	case "Critical", "High":
		return "Release may be blocked or incomplete."
	case "Medium":
		return "Release needs manual review before approval."
	default:
		return "Informational release evidence."
	}
}

func recommendedRiskAction(code string) string {
	switch strings.TrimSpace(code) {
	case "db-change":
		return "Review migration order, rollback, and deployment timing."
	case "config-change":
		return "Confirm environment config, secrets, and rollout timing."
	case "mendix-model":
		return "Validate package deployment steps and compatibility."
	default:
		return "Review and resolve or formally accept this finding."
	}
}

func recommendedRiskReviewAction(risks int) string {
	if risks == 0 {
		return ""
	}
	return "Review and resolve or accept open risk findings."
}

func recommendedManualStepAction(steps int) string {
	if steps == 0 {
		return ""
	}
	return "Complete required manual steps before release."
}

func providerAuthResult(options Options) string {
	if options.Mode == "remote" {
		return "Passed"
	}
	if options.Provider == "local" || options.Provider == "git" || options.Provider == "svn" {
		return "Not applicable"
	}
	return "Passed"
}

func providerAuthDetails(options Options) string {
	if options.Mode == "remote" {
		return "Provider access succeeded before export generation."
	}
	if options.Provider == "local" || options.Provider == "git" || options.Provider == "svn" {
		return "Local source-control mode."
	}
	return "Provider access succeeded before export generation."
}

func resultForBool(ok bool) string {
	if ok {
		return "Passed"
	}
	return "Failed"
}

func resultForVerdict(verdict plansvc.Verdict) string {
	switch verdict {
	case plansvc.VerdictSafe:
		return "Passed"
	case plansvc.VerdictWarning:
		return "Needs review"
	default:
		return "Failed"
	}
}

func nextActionForVerdict(verdict plansvc.Verdict) string {
	switch verdict {
	case plansvc.VerdictSafe:
		return "Proceed with release approval."
	case plansvc.VerdictWarning:
		return "Review risks and manual steps."
	default:
		return "Resolve blocking checks before release."
	}
}

func pullRequestDescription(item scm.PullRequestEvidence) string {
	return strings.Join(nonEmpty(item.ID, item.Title, item.State, branchPair(item.SourceBranch, item.TargetBranch)), " | ")
}

func deploymentDescription(item scm.DeploymentEvidence) string {
	return strings.Join(nonEmpty(item.ID, item.Environment, item.State, item.Ref), " | ")
}

func checkDescription(item scm.CheckEvidence) string {
	return strings.Join(nonEmpty(item.Context, item.State), " | ")
}

func issueDescription(item scm.IssueEvidence) string {
	labels := ""
	if len(item.Labels) > 0 {
		labels = "labels " + strings.Join(item.Labels, ", ")
	}
	return strings.Join(nonEmpty(item.ID, item.Title, item.State, labels), " | ")
}

func releaseDescription(item scm.ReleaseEvidence) string {
	return strings.Join(nonEmpty(firstNonEmpty(item.Tag, item.ID), item.Name, item.State, item.Target, item.PublishedAt), " | ")
}

func branchPair(source, target string) string {
	source = strings.TrimSpace(source)
	target = strings.TrimSpace(target)
	if source != "" && target != "" {
		return source + " -> " + target
	}
	return firstNonEmpty(source, target)
}

func sortedPullRequests(values []scm.PullRequestEvidence) []scm.PullRequestEvidence {
	sorted := append([]scm.PullRequestEvidence(nil), values...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].ID == sorted[j].ID {
			return sorted[i].CommitHash < sorted[j].CommitHash
		}
		return sorted[i].ID < sorted[j].ID
	})
	return sorted
}

func sortedDeployments(values []scm.DeploymentEvidence) []scm.DeploymentEvidence {
	sorted := append([]scm.DeploymentEvidence(nil), values...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Environment == sorted[j].Environment {
			return sorted[i].ID < sorted[j].ID
		}
		return sorted[i].Environment < sorted[j].Environment
	})
	return sorted
}

func sortedChecks(values []scm.CheckEvidence) []scm.CheckEvidence {
	sorted := append([]scm.CheckEvidence(nil), values...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].CommitHash == sorted[j].CommitHash {
			return sorted[i].Context < sorted[j].Context
		}
		return sorted[i].CommitHash < sorted[j].CommitHash
	})
	return sorted
}

func sortedIssues(values []scm.IssueEvidence) []scm.IssueEvidence {
	sorted := append([]scm.IssueEvidence(nil), values...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})
	return sorted
}

func sortedReleases(values []scm.ReleaseEvidence) []scm.ReleaseEvidence {
	sorted := append([]scm.ReleaseEvidence(nil), values...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].PublishedAt == sorted[j].PublishedAt {
			return sorted[i].Tag < sorted[j].Tag
		}
		return sorted[i].PublishedAt > sorted[j].PublishedAt
	})
	return sorted
}

func dedupe(values []string) []string {
	seen := map[string]struct{}{}
	deduped := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		deduped = append(deduped, value)
	}
	return deduped
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func nonEmpty(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
