package assistant

import (
	"context"
	"fmt"
	"sort"
	"strings"

	assisttools "gig/internal/assistant/tools"
	"gig/internal/config"
	conflictsvc "gig/internal/conflict"
	inspectsvc "gig/internal/inspect"
	manifestsvc "gig/internal/manifest"
	plansvc "gig/internal/plan"
	releaseplansvc "gig/internal/releaseplan"
	"gig/internal/scm"
	snapshotsvc "gig/internal/snapshot"
)

type Analyzer interface {
	AnalyzeAudit(ctx context.Context, bundle AuditBundle, options AnalyzeOptions) (AnalysisResponse, error)
	AnalyzeRelease(ctx context.Context, bundle ReleaseBundle, options AnalyzeOptions) (AnalysisResponse, error)
	AnalyzeResolve(ctx context.Context, bundle ResolveBundle, options AnalyzeOptions) (AnalysisResponse, error)
	AnalyzeFollowUp(ctx context.Context, prompt string, options PromptOptions) (AnalysisResponse, error)
}

type analyzerFactory func(config ClientConfig) Analyzer

type Service struct {
	inspector *inspectsvc.Service
	planner   *plansvc.Service
	manifest  *manifestsvc.Service
	conflicts *conflictsvc.Service
	analyzer  analyzerFactory
}

type AuditRequest struct {
	WorkspacePath string
	ScopeLabel    string
	CommandTarget string
	ConfigPath    string
	TicketID      string
	FromBranch    string
	ToBranch      string
	Environments  []inspectsvc.Environment
	Repositories  []scm.Repository
	LoadedConfig  config.Loaded
}

type ExecuteOptions struct {
	BaseURL      string
	GatewayURL   string
	LangGraphURL string
	Mode         RunMode
	Audience     Audience
}

type BundleHints struct {
	CommandTarget string `json:"commandTarget,omitempty"`
	ConfigPath    string `json:"configPath,omitempty"`
}

type Audience string

const (
	AudienceQA             Audience = "qa"
	AudienceClient         Audience = "client"
	AudienceReleaseManager Audience = "release-manager"
)

type AuditBundle struct {
	ScopeLabel          string                            `json:"scopeLabel"`
	Workspace           string                            `json:"workspace"`
	TicketID            string                            `json:"ticketId"`
	FromBranch          string                            `json:"fromBranch"`
	ToBranch            string                            `json:"toBranch"`
	Environments        []inspectsvc.Environment          `json:"environments,omitempty"`
	ScannedRepositories int                               `json:"scannedRepositories"`
	Inspection          []inspectsvc.RepositoryInspection `json:"inspection,omitempty"`
	PromotionPlan       plansvc.PromotionPlan             `json:"promotionPlan"`
	Verification        plansvc.Verification              `json:"verification"`
	Packet              manifestsvc.ReleasePacket         `json:"packet"`
	ManifestHighlights  []string                          `json:"manifestHighlights,omitempty"`
	Hints               BundleHints                       `json:"hints,omitempty"`
}

type AuditResult struct {
	Bundle   AuditBundle `json:"bundle"`
	ThreadID string      `json:"threadId"`
	Mode     RunMode     `json:"mode"`
	Audience Audience    `json:"audience"`
	Response string      `json:"response"`
}

type ReleaseRequest struct {
	WorkspacePath string
	ScopeLabel    string
	CommandTarget string
	ConfigPath    string
	ReleaseID     string
	TicketIDs     []string
	FromBranch    string
	ToBranch      string
	Environments  []inspectsvc.Environment
	Repositories  []scm.Repository
	LoadedConfig  config.Loaded
}

type ResolveRequest struct {
	WorkspacePath string
	ScopeLabel    string
	CommandTarget string
	ConfigPath    string
	TicketID      string
}

type ReleaseBundle struct {
	ScopeLabel         string                       `json:"scopeLabel"`
	Workspace          string                       `json:"workspace"`
	ReleaseID          string                       `json:"releaseId"`
	SnapshotDir        string                       `json:"snapshotDir"`
	FromBranch         string                       `json:"fromBranch,omitempty"`
	ToBranch           string                       `json:"toBranch,omitempty"`
	Environments       []inspectsvc.Environment     `json:"environments,omitempty"`
	Snapshots          []snapshotsvc.TicketSnapshot `json:"snapshots,omitempty"`
	ReleasePlan        releaseplansvc.ReleasePlan   `json:"releasePlan"`
	Packets            []manifestsvc.ReleasePacket  `json:"packets,omitempty"`
	EvidenceSummary    ReleaseEvidenceSummary       `json:"evidenceSummary,omitempty"`
	RepositoryEvidence []ReleaseRepositoryEvidence  `json:"repositoryEvidence,omitempty"`
	TicketOverlap      []ReleaseTicketOverlap       `json:"ticketOverlap,omitempty"`
	Hotspots           []ReleaseHotspot             `json:"hotspots,omitempty"`
	ExecutiveSummary   []string                     `json:"executiveSummary,omitempty"`
	OperatorSummary    []string                     `json:"operatorSummary,omitempty"`
	Hints              BundleHints                  `json:"hints,omitempty"`
}

type ReleaseEvidenceSummary struct {
	RepositoriesWithEvidence     int `json:"repositoriesWithEvidence,omitempty"`
	PullRequests                 int `json:"pullRequests,omitempty"`
	Deployments                  int `json:"deployments,omitempty"`
	Checks                       int `json:"checks,omitempty"`
	FailingChecks                int `json:"failingChecks,omitempty"`
	LinkedIssues                 int `json:"linkedIssues,omitempty"`
	Releases                     int `json:"releases,omitempty"`
	OverlappingTickets           int `json:"overlappingTickets,omitempty"`
	NewTicketsSinceLatestRelease int `json:"newTicketsSinceLatestRelease,omitempty"`
	Hotspots                     int `json:"hotspots,omitempty"`
}

type ReleaseRepositoryEvidence struct {
	Repository                   scm.Repository            `json:"repository"`
	TicketIDs                    []string                  `json:"ticketIds,omitempty"`
	PullRequests                 []scm.PullRequestEvidence `json:"pullRequests,omitempty"`
	Deployments                  []scm.DeploymentEvidence  `json:"deployments,omitempty"`
	Checks                       []scm.CheckEvidence       `json:"checks,omitempty"`
	LinkedIssues                 []scm.IssueEvidence       `json:"linkedIssues,omitempty"`
	LatestRelease                *scm.ReleaseEvidence      `json:"latestRelease,omitempty"`
	OverlapTickets               []string                  `json:"overlapTickets,omitempty"`
	NewTicketsSinceLatestRelease []string                  `json:"newTicketsSinceLatestRelease,omitempty"`
}

type ReleaseHotspot struct {
	Repository scm.Repository `json:"repository"`
	TicketIDs  []string       `json:"ticketIds,omitempty"`
	Severity   string         `json:"severity"`
	Reasons    []string       `json:"reasons,omitempty"`
}

type ReleaseTicketOverlap struct {
	Repository     scm.Repository `json:"repository"`
	TicketIDs      []string       `json:"ticketIds,omitempty"`
	PullRequestIDs []string       `json:"pullRequestIds,omitempty"`
	IssueIDs       []string       `json:"issueIds,omitempty"`
}

type ReleaseResult struct {
	Bundle   ReleaseBundle `json:"bundle"`
	ThreadID string        `json:"threadId"`
	Mode     RunMode       `json:"mode"`
	Audience Audience      `json:"audience"`
	Response string        `json:"response"`
}

type ResolveAction struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

type ResolveBlock struct {
	Index       int              `json:"index"`
	StartLine   int              `json:"startLine"`
	EndLine     int              `json:"endLine"`
	Current     string           `json:"current"`
	Base        string           `json:"base,omitempty"`
	Incoming    string           `json:"incoming"`
	CurrentRef  scm.ConflictSide `json:"currentRef"`
	IncomingRef scm.ConflictSide `json:"incomingRef"`
}

type ResolveActiveConflict struct {
	File          conflictsvc.FileStatus     `json:"file"`
	Block         ResolveBlock               `json:"block"`
	Risk          conflictsvc.RiskAssessment `json:"risk"`
	ScopeWarnings []string                   `json:"scopeWarnings,omitempty"`
}

type ResolveBundle struct {
	ScopeLabel       string                 `json:"scopeLabel"`
	Workspace        string                 `json:"workspace"`
	ScopedTicketID   string                 `json:"scopedTicketId,omitempty"`
	Status           conflictsvc.Status     `json:"status"`
	ActiveConflict   *ResolveActiveConflict `json:"activeConflict,omitempty"`
	SupportedActions []ResolveAction        `json:"supportedActions,omitempty"`
	Hints            BundleHints            `json:"hints,omitempty"`
}

type ResolveResult struct {
	Bundle   ResolveBundle `json:"bundle"`
	ThreadID string        `json:"threadId"`
	Mode     RunMode       `json:"mode"`
	Audience Audience      `json:"audience"`
	Response string        `json:"response"`
}

func NewService(inspector *inspectsvc.Service, planner *plansvc.Service, manifest *manifestsvc.Service, conflicts *conflictsvc.Service) *Service {
	return &Service{
		inspector: inspector,
		planner:   planner,
		manifest:  manifest,
		conflicts: conflicts,
		analyzer: func(config ClientConfig) Analyzer {
			return NewDeerFlowClient(config)
		},
	}
}

func NewServiceWithAnalyzerFactory(inspector *inspectsvc.Service, planner *plansvc.Service, manifest *manifestsvc.Service, conflicts *conflictsvc.Service, factory analyzerFactory) *Service {
	service := NewService(inspector, planner, manifest, conflicts)
	if factory != nil {
		service.analyzer = factory
	}
	return service
}

func ParseAudience(raw string) (Audience, error) {
	switch Audience(strings.ToLower(strings.TrimSpace(raw))) {
	case AudienceQA:
		return AudienceQA, nil
	case AudienceClient:
		return AudienceClient, nil
	case AudienceReleaseManager, "":
		return AudienceReleaseManager, nil
	default:
		return "", fmt.Errorf("unsupported audience %q", raw)
	}
}

func defaultAudience(audience Audience) Audience {
	if audience == "" {
		return AudienceReleaseManager
	}
	return audience
}

func (s *Service) Audit(ctx context.Context, request AuditRequest, options ExecuteOptions) (AuditResult, error) {
	bundle, err := s.BuildAuditBundle(ctx, request)
	if err != nil {
		return AuditResult{}, err
	}

	client := s.analyzer(ClientConfig{
		BaseURL:      options.BaseURL,
		GatewayURL:   options.GatewayURL,
		LangGraphURL: options.LangGraphURL,
	})
	response, err := client.AnalyzeAudit(ctx, bundle, AnalyzeOptions{
		Mode:     options.Mode,
		Audience: defaultAudience(options.Audience),
		Bridge:   s.buildAuditToolBridge(request),
	})
	if err != nil {
		return AuditResult{}, err
	}

	return AuditResult{
		Bundle:   bundle,
		ThreadID: response.ThreadID,
		Mode:     defaultRunMode(options.Mode),
		Audience: defaultAudience(options.Audience),
		Response: response.Response,
	}, nil
}

func (s *Service) Release(ctx context.Context, request ReleaseRequest, options ExecuteOptions) (ReleaseResult, error) {
	bundle, err := s.BuildReleaseBundle(ctx, request)
	if err != nil {
		return ReleaseResult{}, err
	}

	client := s.analyzer(ClientConfig{
		BaseURL:      options.BaseURL,
		GatewayURL:   options.GatewayURL,
		LangGraphURL: options.LangGraphURL,
	})
	response, err := client.AnalyzeRelease(ctx, bundle, AnalyzeOptions{
		Mode:     options.Mode,
		Audience: defaultAudience(options.Audience),
		Bridge:   s.buildReleaseToolBridge(request),
	})
	if err != nil {
		return ReleaseResult{}, err
	}

	return ReleaseResult{
		Bundle:   bundle,
		ThreadID: response.ThreadID,
		Mode:     defaultRunMode(options.Mode),
		Audience: defaultAudience(options.Audience),
		Response: response.Response,
	}, nil
}

func (s *Service) Resolve(ctx context.Context, request ResolveRequest, options ExecuteOptions) (ResolveResult, error) {
	bundle, err := s.BuildResolveBundle(ctx, request)
	if err != nil {
		return ResolveResult{}, err
	}

	client := s.analyzer(ClientConfig{
		BaseURL:      options.BaseURL,
		GatewayURL:   options.GatewayURL,
		LangGraphURL: options.LangGraphURL,
	})
	response, err := client.AnalyzeResolve(ctx, bundle, AnalyzeOptions{
		Mode:     options.Mode,
		Audience: defaultAudience(options.Audience),
		Bridge:   s.buildResolveToolBridge(request),
	})
	if err != nil {
		return ResolveResult{}, err
	}

	return ResolveResult{
		Bundle:   bundle,
		ThreadID: response.ThreadID,
		Mode:     defaultRunMode(options.Mode),
		Audience: defaultAudience(options.Audience),
		Response: response.Response,
	}, nil
}

func (s *Service) FollowUpAudit(ctx context.Context, request AuditRequest, question, threadID string, options ExecuteOptions) (AuditResult, error) {
	bundle, err := s.BuildAuditBundle(ctx, request)
	if err != nil {
		return AuditResult{}, err
	}
	prompt, err := buildAuditFollowUpPrompt(bundle, defaultAudience(options.Audience), question)
	if err != nil {
		return AuditResult{}, err
	}
	response, err := s.analyzeFollowUpPrompt(ctx, prompt, threadID, options, s.buildAuditToolBridge(request))
	if err != nil {
		return AuditResult{}, err
	}
	return AuditResult{
		Bundle:   bundle,
		ThreadID: response.ThreadID,
		Mode:     defaultRunMode(options.Mode),
		Audience: defaultAudience(options.Audience),
		Response: response.Response,
	}, nil
}

func (s *Service) FollowUpRelease(ctx context.Context, request ReleaseRequest, question, threadID string, options ExecuteOptions) (ReleaseResult, error) {
	bundle, err := s.BuildReleaseBundle(ctx, request)
	if err != nil {
		return ReleaseResult{}, err
	}
	prompt, err := buildReleaseFollowUpPrompt(bundle, defaultAudience(options.Audience), question)
	if err != nil {
		return ReleaseResult{}, err
	}
	response, err := s.analyzeFollowUpPrompt(ctx, prompt, threadID, options, s.buildReleaseToolBridge(request))
	if err != nil {
		return ReleaseResult{}, err
	}
	return ReleaseResult{
		Bundle:   bundle,
		ThreadID: response.ThreadID,
		Mode:     defaultRunMode(options.Mode),
		Audience: defaultAudience(options.Audience),
		Response: response.Response,
	}, nil
}

func (s *Service) FollowUpResolve(ctx context.Context, request ResolveRequest, question, threadID string, options ExecuteOptions) (ResolveResult, error) {
	bundle, err := s.BuildResolveBundle(ctx, request)
	if err != nil {
		return ResolveResult{}, err
	}
	prompt, err := buildResolveFollowUpPrompt(bundle, defaultAudience(options.Audience), question)
	if err != nil {
		return ResolveResult{}, err
	}
	response, err := s.analyzeFollowUpPrompt(ctx, prompt, threadID, options, s.buildResolveToolBridge(request))
	if err != nil {
		return ResolveResult{}, err
	}
	return ResolveResult{
		Bundle:   bundle,
		ThreadID: response.ThreadID,
		Mode:     defaultRunMode(options.Mode),
		Audience: defaultAudience(options.Audience),
		Response: response.Response,
	}, nil
}

func (s *Service) BuildAuditBundle(ctx context.Context, request AuditRequest) (AuditBundle, error) {
	if len(request.Repositories) == 0 {
		return AuditBundle{}, fmt.Errorf("at least one repository is required")
	}

	inspections, err := s.inspector.InspectInRepositories(ctx, request.Repositories, request.TicketID)
	if err != nil {
		return AuditBundle{}, err
	}

	promotionPlan, err := s.planner.BuildPromotionPlanInRepositories(
		ctx,
		request.Repositories,
		request.TicketID,
		request.FromBranch,
		request.ToBranch,
		request.Environments,
	)
	if err != nil {
		return AuditBundle{}, err
	}

	releasePacket := manifestsvc.BuildReleasePacket(request.WorkspacePath, request.LoadedConfig, promotionPlan)

	return AuditBundle{
		ScopeLabel:          request.ScopeLabel,
		Workspace:           request.WorkspacePath,
		TicketID:            request.TicketID,
		FromBranch:          request.FromBranch,
		ToBranch:            request.ToBranch,
		Environments:        append([]inspectsvc.Environment(nil), request.Environments...),
		ScannedRepositories: len(request.Repositories),
		Inspection:          inspections,
		PromotionPlan:       promotionPlan,
		Verification:        plansvc.BuildVerification(promotionPlan),
		Packet:              releasePacket,
		ManifestHighlights:  append([]string(nil), releasePacket.Highlights...),
		Hints: BundleHints{
			CommandTarget: request.CommandTarget,
			ConfigPath:    request.ConfigPath,
		},
	}, nil
}

func (s *Service) BuildResolveBundle(ctx context.Context, request ResolveRequest) (ResolveBundle, error) {
	if s.conflicts == nil {
		return ResolveBundle{}, fmt.Errorf("conflict service is not configured")
	}

	session, active, err := s.conflicts.LoadActiveConflict(ctx, request.WorkspacePath, "", 0, request.TicketID)
	if err != nil {
		return ResolveBundle{}, err
	}

	scopeLabel := strings.TrimSpace(request.ScopeLabel)
	if scopeLabel == "" {
		scopeLabel = strings.TrimSpace(session.Status.Repository.Root)
	}

	workspace := strings.TrimSpace(session.Status.Repository.Root)
	if workspace == "" {
		workspace = strings.TrimSpace(request.WorkspacePath)
	}

	var activeConflict *ResolveActiveConflict
	if active != nil {
		activeConflict = &ResolveActiveConflict{
			File: active.File,
			Block: ResolveBlock{
				Index:       active.Block.Index,
				StartLine:   active.Block.StartLine,
				EndLine:     active.Block.EndLine,
				Current:     trimConflictContent(active.Block.Current),
				Base:        trimConflictContent(active.Block.Base),
				Incoming:    trimConflictContent(active.Block.Incoming),
				CurrentRef:  active.Block.CurrentRef,
				IncomingRef: active.Block.IncomingRef,
			},
			Risk:          active.Risk,
			ScopeWarnings: append([]string(nil), active.ScopeWarnings...),
		}
	}

	return ResolveBundle{
		ScopeLabel:       scopeLabel,
		Workspace:        workspace,
		ScopedTicketID:   session.Status.ScopeTicketID,
		Status:           session.Status,
		ActiveConflict:   activeConflict,
		SupportedActions: defaultResolveActions(),
		Hints: BundleHints{
			CommandTarget: request.CommandTarget,
			ConfigPath:    request.ConfigPath,
		},
	}, nil
}

func (s *Service) BuildReleaseBundle(ctx context.Context, request ReleaseRequest) (ReleaseBundle, error) {
	releaseID, err := snapshotsvc.NormalizeReleaseID(request.ReleaseID)
	if err != nil {
		return ReleaseBundle{}, err
	}

	if len(request.TicketIDs) != 0 {
		return s.buildLiveReleaseBundle(ctx, request, releaseID)
	}

	snapshots, snapshotDir, err := snapshotsvc.LoadReleaseSnapshots(request.WorkspacePath, releaseID)
	if err != nil {
		return ReleaseBundle{}, err
	}

	releasePlan, err := releaseplansvc.Build(releaseID, request.WorkspacePath, snapshotDir, snapshots)
	if err != nil {
		return ReleaseBundle{}, err
	}

	packets := make([]manifestsvc.ReleasePacket, 0, len(snapshots))
	for _, snapshot := range snapshots {
		packets = append(packets, manifestsvc.BuildReleasePacket(request.WorkspacePath, request.LoadedConfig, snapshot.Plan))
	}

	repositoryEvidence := collectReleaseRepositoryEvidence(releasePlan)
	ticketOverlap := collectReleaseTicketOverlap(releasePlan)
	hotspots := collectReleaseHotspots(releasePlan)
	evidenceSummary := summarizeReleaseEvidence(repositoryEvidence, hotspots)
	executiveSummary := buildReleaseExecutiveSummary(releasePlan, evidenceSummary)
	operatorSummary := buildReleaseOperatorSummary(repositoryEvidence, ticketOverlap, hotspots)

	return ReleaseBundle{
		ScopeLabel:         request.ScopeLabel,
		Workspace:          request.WorkspacePath,
		ReleaseID:          releaseID,
		SnapshotDir:        snapshotDir,
		FromBranch:         releasePlan.FromBranch,
		ToBranch:           releasePlan.ToBranch,
		Environments:       cloneReleaseEnvironments(releasePlan.Environments),
		Snapshots:          cloneSnapshots(snapshots),
		ReleasePlan:        releasePlan,
		Packets:            packets,
		EvidenceSummary:    evidenceSummary,
		RepositoryEvidence: repositoryEvidence,
		TicketOverlap:      ticketOverlap,
		Hotspots:           hotspots,
		ExecutiveSummary:   executiveSummary,
		OperatorSummary:    operatorSummary,
		Hints: BundleHints{
			CommandTarget: request.CommandTarget,
			ConfigPath:    request.ConfigPath,
		},
	}, nil
}

func (s *Service) buildLiveReleaseBundle(ctx context.Context, request ReleaseRequest, releaseID string) (ReleaseBundle, error) {
	if len(request.Repositories) == 0 {
		return ReleaseBundle{}, fmt.Errorf("at least one repository is required")
	}

	scopeLabel := strings.TrimSpace(request.ScopeLabel)
	if scopeLabel == "" {
		scopeLabel = strings.TrimSpace(request.WorkspacePath)
	}

	snapshotService := snapshotsvc.NewService(s.inspector, s.planner)
	snapshots := make([]snapshotsvc.TicketSnapshot, 0, len(request.TicketIDs))
	seen := make(map[string]struct{}, len(request.TicketIDs))

	for _, ticketID := range request.TicketIDs {
		ticketID = strings.TrimSpace(ticketID)
		if ticketID == "" {
			continue
		}
		if _, ok := seen[ticketID]; ok {
			continue
		}
		seen[ticketID] = struct{}{}

		snapshot, err := snapshotService.CaptureInRepositoriesWithOptions(
			ctx,
			scopeLabel,
			request.LoadedConfig,
			request.Repositories,
			ticketID,
			request.FromBranch,
			request.ToBranch,
			request.Environments,
			snapshotsvc.CaptureOptions{ReleaseID: releaseID},
		)
		if err != nil {
			return ReleaseBundle{}, err
		}
		snapshots = append(snapshots, snapshot)
	}

	if len(snapshots) == 0 {
		return ReleaseBundle{}, fmt.Errorf("at least one ticket is required")
	}

	releasePlan, err := releaseplansvc.Build(releaseID, scopeLabel, "", snapshots)
	if err != nil {
		return ReleaseBundle{}, err
	}

	packets := make([]manifestsvc.ReleasePacket, 0, len(snapshots))
	for _, snapshot := range snapshots {
		packets = append(packets, manifestsvc.BuildReleasePacket(request.WorkspacePath, request.LoadedConfig, snapshot.Plan))
	}

	repositoryEvidence := collectReleaseRepositoryEvidence(releasePlan)
	ticketOverlap := collectReleaseTicketOverlap(releasePlan)
	hotspots := collectReleaseHotspots(releasePlan)
	evidenceSummary := summarizeReleaseEvidence(repositoryEvidence, hotspots)
	executiveSummary := buildReleaseExecutiveSummary(releasePlan, evidenceSummary)
	operatorSummary := buildReleaseOperatorSummary(repositoryEvidence, ticketOverlap, hotspots)

	return ReleaseBundle{
		ScopeLabel:         scopeLabel,
		Workspace:          scopeLabel,
		ReleaseID:          releaseID,
		SnapshotDir:        "",
		FromBranch:         releasePlan.FromBranch,
		ToBranch:           releasePlan.ToBranch,
		Environments:       cloneReleaseEnvironments(releasePlan.Environments),
		Snapshots:          cloneSnapshots(snapshots),
		ReleasePlan:        releasePlan,
		Packets:            packets,
		EvidenceSummary:    evidenceSummary,
		RepositoryEvidence: repositoryEvidence,
		TicketOverlap:      ticketOverlap,
		Hotspots:           hotspots,
		ExecutiveSummary:   executiveSummary,
		OperatorSummary:    operatorSummary,
		Hints: BundleHints{
			CommandTarget: request.CommandTarget,
			ConfigPath:    request.ConfigPath,
		},
	}, nil
}

func cloneReleaseEnvironments(environments []inspectsvc.Environment) []inspectsvc.Environment {
	if len(environments) == 0 {
		return nil
	}
	cloned := make([]inspectsvc.Environment, len(environments))
	copy(cloned, environments)
	return cloned
}

func cloneSnapshots(snapshots []snapshotsvc.TicketSnapshot) []snapshotsvc.TicketSnapshot {
	if len(snapshots) == 0 {
		return nil
	}
	cloned := make([]snapshotsvc.TicketSnapshot, len(snapshots))
	copy(cloned, snapshots)
	return cloned
}

func (s *Service) analyzeFollowUpPrompt(ctx context.Context, prompt, threadID string, options ExecuteOptions, bridge *assisttools.Bridge) (AnalysisResponse, error) {
	client := s.analyzer(ClientConfig{
		BaseURL:      options.BaseURL,
		GatewayURL:   options.GatewayURL,
		LangGraphURL: options.LangGraphURL,
	})
	return client.AnalyzeFollowUp(ctx, prompt, PromptOptions{
		ThreadID: strings.TrimSpace(threadID),
		Mode:     options.Mode,
		Bridge:   bridge,
	})
}

func (s *Service) buildAuditToolBridge(request AuditRequest) *assisttools.Bridge {
	return assisttools.NewBridge(assisttools.Runtime{
		Inspect: func(ctx context.Context, toolRequest assisttools.InspectRequest) (any, error) {
			bundle, err := s.BuildAuditBundle(ctx, request)
			if err != nil {
				return nil, err
			}
			return auditInspectPayload(bundle, toolRequest), nil
		},
		Verify: func(ctx context.Context, toolRequest assisttools.VerifyRequest) (any, error) {
			bundle, err := s.BuildAuditBundle(ctx, request)
			if err != nil {
				return nil, err
			}
			return auditVerifyPayload(bundle, toolRequest), nil
		},
		Manifest: func(ctx context.Context, toolRequest assisttools.ManifestRequest) (any, error) {
			bundle, err := s.BuildAuditBundle(ctx, request)
			if err != nil {
				return nil, err
			}
			return auditManifestPayload(bundle, toolRequest), nil
		},
	})
}

func (s *Service) buildReleaseToolBridge(request ReleaseRequest) *assisttools.Bridge {
	return assisttools.NewBridge(assisttools.Runtime{
		Inspect: func(ctx context.Context, toolRequest assisttools.InspectRequest) (any, error) {
			bundle, err := s.BuildReleaseBundle(ctx, request)
			if err != nil {
				return nil, err
			}
			return releaseInspectPayload(bundle, toolRequest), nil
		},
		Verify: func(ctx context.Context, toolRequest assisttools.VerifyRequest) (any, error) {
			bundle, err := s.BuildReleaseBundle(ctx, request)
			if err != nil {
				return nil, err
			}
			return releaseVerifyPayload(bundle, toolRequest), nil
		},
		Manifest: func(ctx context.Context, toolRequest assisttools.ManifestRequest) (any, error) {
			bundle, err := s.BuildReleaseBundle(ctx, request)
			if err != nil {
				return nil, err
			}
			return releaseManifestPayload(bundle, toolRequest), nil
		},
	})
}

func (s *Service) buildResolveToolBridge(request ResolveRequest) *assisttools.Bridge {
	return assisttools.NewBridge(assisttools.Runtime{
		Inspect: func(ctx context.Context, toolRequest assisttools.InspectRequest) (any, error) {
			bundle, err := s.BuildResolveBundle(ctx, request)
			if err != nil {
				return nil, err
			}
			return resolveInspectPayload(bundle, toolRequest), nil
		},
	})
}

func auditInspectPayload(bundle AuditBundle, request assisttools.InspectRequest) any {
	switch normalizeToolFocus(request.Focus) {
	case "repositories":
		return map[string]any{
			"scopeLabel": bundle.ScopeLabel,
			"ticketId":   bundle.TicketID,
			"inspection": filterAuditInspection(bundle.Inspection, request.Repository),
		}
	case "risks":
		return map[string]any{
			"ticketId":     bundle.TicketID,
			"fromBranch":   bundle.FromBranch,
			"toBranch":     bundle.ToBranch,
			"repositories": filterPromotionRepositories(bundle.PromotionPlan.Repositories, request.Repository),
		}
	default:
		return map[string]any{
			"scopeLabel":         bundle.ScopeLabel,
			"ticketId":           bundle.TicketID,
			"fromBranch":         bundle.FromBranch,
			"toBranch":           bundle.ToBranch,
			"inspection":         filterAuditInspection(bundle.Inspection, request.Repository),
			"promotionPlan":      bundle.PromotionPlan,
			"manifestHighlights": append([]string(nil), bundle.ManifestHighlights...),
		}
	}
}

func auditVerifyPayload(bundle AuditBundle, request assisttools.VerifyRequest) any {
	switch normalizeToolFocus(request.Focus) {
	case "repositories":
		return map[string]any{
			"ticketId":     bundle.TicketID,
			"verification": bundle.Verification.Repositories,
		}
	case "reasons":
		return map[string]any{
			"ticketId": bundle.TicketID,
			"verdict":  bundle.Verification.Verdict,
			"reasons":  append([]string(nil), bundle.Verification.Reasons...),
		}
	default:
		return map[string]any{
			"ticketId":      bundle.TicketID,
			"fromBranch":    bundle.FromBranch,
			"toBranch":      bundle.ToBranch,
			"summary":       bundle.Verification.Summary,
			"verdict":       bundle.Verification.Verdict,
			"reasons":       append([]string(nil), bundle.Verification.Reasons...),
			"repositories":  bundle.Verification.Repositories,
			"manualSteps":   bundle.PromotionPlan.Summary.TotalManualSteps,
			"scannedScopes": bundle.ScannedRepositories,
		}
	}
}

func auditManifestPayload(bundle AuditBundle, request assisttools.ManifestRequest) any {
	switch normalizeAudience(request.Audience) {
	case string(AudienceQA):
		return map[string]any{
			"ticketId": bundle.TicketID,
			"verdict":  bundle.Packet.Verdict,
			"section":  bundle.Packet.QA,
		}
	case string(AudienceClient):
		return map[string]any{
			"ticketId": bundle.TicketID,
			"verdict":  bundle.Packet.Verdict,
			"section":  bundle.Packet.Client,
		}
	case string(AudienceReleaseManager):
		return map[string]any{
			"ticketId": bundle.TicketID,
			"verdict":  bundle.Packet.Verdict,
			"section":  bundle.Packet.ReleaseManager,
		}
	default:
		return bundle.Packet
	}
}

func releaseInspectPayload(bundle ReleaseBundle, request assisttools.InspectRequest) any {
	switch normalizeToolFocus(request.Focus) {
	case "repositories":
		return map[string]any{
			"releaseId":    bundle.ReleaseID,
			"scopeLabel":   bundle.ScopeLabel,
			"repositories": filterReleasePlanRepositories(bundle.ReleasePlan.Repositories, request.Repository),
		}
	case "tickets":
		return map[string]any{
			"releaseId": bundle.ReleaseID,
			"tickets":   bundle.ReleasePlan.Tickets,
		}
	case "evidence":
		return map[string]any{
			"releaseId":          bundle.ReleaseID,
			"evidenceSummary":    bundle.EvidenceSummary,
			"repositoryEvidence": filterReleaseEvidence(bundle.RepositoryEvidence, request.Repository),
			"ticketOverlap":      filterReleaseTicketOverlap(bundle.TicketOverlap, request.Repository),
			"hotspots":           filterReleaseHotspots(bundle.Hotspots, request.Repository),
			"executiveSummary":   append([]string(nil), bundle.ExecutiveSummary...),
			"operatorSummary":    append([]string(nil), bundle.OperatorSummary...),
		}
	default:
		return map[string]any{
			"releaseId":          bundle.ReleaseID,
			"scopeLabel":         bundle.ScopeLabel,
			"summary":            bundle.ReleasePlan.Summary,
			"evidenceSummary":    bundle.EvidenceSummary,
			"repositories":       filterReleasePlanRepositories(bundle.ReleasePlan.Repositories, request.Repository),
			"repositoryEvidence": filterReleaseEvidence(bundle.RepositoryEvidence, request.Repository),
			"ticketOverlap":      filterReleaseTicketOverlap(bundle.TicketOverlap, request.Repository),
			"hotspots":           filterReleaseHotspots(bundle.Hotspots, request.Repository),
			"executiveSummary":   append([]string(nil), bundle.ExecutiveSummary...),
			"operatorSummary":    append([]string(nil), bundle.OperatorSummary...),
		}
	}
}

func releaseVerifyPayload(bundle ReleaseBundle, request assisttools.VerifyRequest) any {
	switch normalizeToolFocus(request.Focus) {
	case "repositories":
		return map[string]any{
			"releaseId":    bundle.ReleaseID,
			"repositories": bundle.ReleasePlan.Repositories,
		}
	default:
		return map[string]any{
			"releaseId":        bundle.ReleaseID,
			"scopeLabel":       bundle.ScopeLabel,
			"summary":          bundle.ReleasePlan.Summary,
			"verdict":          bundle.ReleasePlan.Verdict,
			"notes":            append([]string(nil), bundle.ReleasePlan.Notes...),
			"tickets":          bundle.ReleasePlan.Tickets,
			"repositories":     bundle.ReleasePlan.Repositories,
			"executiveSummary": append([]string(nil), bundle.ExecutiveSummary...),
			"operatorSummary":  append([]string(nil), bundle.OperatorSummary...),
		}
	}
}

func releaseManifestPayload(bundle ReleaseBundle, request assisttools.ManifestRequest) any {
	audience := normalizeAudience(request.Audience)
	if audience == "" {
		return bundle.Packets
	}

	items := make([]map[string]any, 0, len(bundle.Packets))
	for _, packet := range bundle.Packets {
		item := map[string]any{
			"ticketId": packet.TicketID,
			"verdict":  packet.Verdict,
		}
		switch audience {
		case string(AudienceQA):
			item["section"] = packet.QA
		case string(AudienceClient):
			item["section"] = packet.Client
		case string(AudienceReleaseManager):
			item["section"] = packet.ReleaseManager
		default:
			item["packet"] = packet
		}
		items = append(items, item)
	}
	return map[string]any{
		"releaseId": bundle.ReleaseID,
		"audience":  audience,
		"packets":   items,
	}
}

func resolveInspectPayload(bundle ResolveBundle, _ assisttools.InspectRequest) any {
	return map[string]any{
		"scopeLabel":       bundle.ScopeLabel,
		"workspace":        bundle.Workspace,
		"scopedTicketId":   bundle.ScopedTicketID,
		"status":           bundle.Status,
		"activeConflict":   bundle.ActiveConflict,
		"supportedActions": bundle.SupportedActions,
	}
}

func normalizeToolFocus(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeAudience(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func filterAuditInspection(items []inspectsvc.RepositoryInspection, repository string) []inspectsvc.RepositoryInspection {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return append([]inspectsvc.RepositoryInspection(nil), items...)
	}

	filtered := make([]inspectsvc.RepositoryInspection, 0, len(items))
	for _, item := range items {
		if repositoryMatches(item.Repository, repository) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterPromotionRepositories(items []plansvc.RepositoryPlan, repository string) []plansvc.RepositoryPlan {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return append([]plansvc.RepositoryPlan(nil), items...)
	}

	filtered := make([]plansvc.RepositoryPlan, 0, len(items))
	for _, item := range items {
		if repositoryMatches(item.Repository, repository) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterReleasePlanRepositories(items []releaseplansvc.RepositoryPlan, repository string) []releaseplansvc.RepositoryPlan {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return append([]releaseplansvc.RepositoryPlan(nil), items...)
	}

	filtered := make([]releaseplansvc.RepositoryPlan, 0, len(items))
	for _, item := range items {
		if repositoryMatches(item.Repository, repository) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterReleaseEvidence(items []ReleaseRepositoryEvidence, repository string) []ReleaseRepositoryEvidence {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return append([]ReleaseRepositoryEvidence(nil), items...)
	}

	filtered := make([]ReleaseRepositoryEvidence, 0, len(items))
	for _, item := range items {
		if repositoryMatches(item.Repository, repository) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterReleaseHotspots(items []ReleaseHotspot, repository string) []ReleaseHotspot {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return append([]ReleaseHotspot(nil), items...)
	}

	filtered := make([]ReleaseHotspot, 0, len(items))
	for _, item := range items {
		if repositoryMatches(item.Repository, repository) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterReleaseTicketOverlap(items []ReleaseTicketOverlap, repository string) []ReleaseTicketOverlap {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return append([]ReleaseTicketOverlap(nil), items...)
	}

	filtered := make([]ReleaseTicketOverlap, 0, len(items))
	for _, item := range items {
		if repositoryMatches(item.Repository, repository) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func repositoryMatches(repository scm.Repository, filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}

	return strings.EqualFold(repository.Name, filter) ||
		strings.EqualFold(repository.Root, filter) ||
		strings.Contains(strings.ToLower(repository.Root), strings.ToLower(filter))
}

func summarizeReleaseEvidence(items []ReleaseRepositoryEvidence, hotspots []ReleaseHotspot) ReleaseEvidenceSummary {
	var summary ReleaseEvidenceSummary
	issueIDs := map[string]struct{}{}
	releaseIDs := map[string]struct{}{}
	overlapTickets := map[string]struct{}{}
	newTickets := map[string]struct{}{}

	for _, item := range items {
		summary.RepositoriesWithEvidence++
		summary.PullRequests += len(item.PullRequests)
		summary.Deployments += len(item.Deployments)
		summary.Checks += len(item.Checks)
		for _, check := range item.Checks {
			if isFailingCheck(check.State) {
				summary.FailingChecks++
			}
		}
		for _, issue := range item.LinkedIssues {
			issueIDs[strings.TrimSpace(issue.ID)] = struct{}{}
		}
		if item.LatestRelease != nil {
			releaseIDs[strings.TrimSpace(item.LatestRelease.ID)] = struct{}{}
		}
		for _, ticketID := range item.OverlapTickets {
			overlapTickets[strings.TrimSpace(ticketID)] = struct{}{}
		}
		for _, ticketID := range item.NewTicketsSinceLatestRelease {
			newTickets[strings.TrimSpace(ticketID)] = struct{}{}
		}
	}
	summary.LinkedIssues = len(issueIDs)
	summary.Releases = len(releaseIDs)
	summary.OverlappingTickets = len(overlapTickets)
	summary.NewTicketsSinceLatestRelease = len(newTickets)
	summary.Hotspots = len(hotspots)
	return summary
}

func collectReleaseRepositoryEvidence(releasePlan releaseplansvc.ReleasePlan) []ReleaseRepositoryEvidence {
	items := make([]ReleaseRepositoryEvidence, 0, len(releasePlan.Repositories))
	for _, repositoryPlan := range releasePlan.Repositories {
		if repositoryPlan.ProviderEvidence == nil || repositoryPlan.ProviderEvidence.IsZero() {
			continue
		}
		latestRelease := latestReleaseEvidence(repositoryPlan.ProviderEvidence.Releases)
		overlapTickets, newTickets := releaseTicketDelta(repositoryPlan.TicketIDs, latestRelease)
		items = append(items, ReleaseRepositoryEvidence{
			Repository:                   repositoryPlan.Repository,
			TicketIDs:                    append([]string(nil), repositoryPlan.TicketIDs...),
			PullRequests:                 append([]scm.PullRequestEvidence(nil), repositoryPlan.ProviderEvidence.PullRequests...),
			Deployments:                  append([]scm.DeploymentEvidence(nil), repositoryPlan.ProviderEvidence.Deployments...),
			Checks:                       append([]scm.CheckEvidence(nil), repositoryPlan.ProviderEvidence.Checks...),
			LinkedIssues:                 append([]scm.IssueEvidence(nil), repositoryPlan.ProviderEvidence.Issues...),
			LatestRelease:                latestRelease,
			OverlapTickets:               overlapTickets,
			NewTicketsSinceLatestRelease: newTickets,
		})
	}
	return items
}

func collectReleaseTicketOverlap(releasePlan releaseplansvc.ReleasePlan) []ReleaseTicketOverlap {
	items := make([]ReleaseTicketOverlap, 0, len(releasePlan.Repositories))
	for _, repositoryPlan := range releasePlan.Repositories {
		if len(repositoryPlan.TicketIDs) < 2 {
			continue
		}
		item := ReleaseTicketOverlap{
			Repository: repositoryPlan.Repository,
			TicketIDs:  append([]string(nil), repositoryPlan.TicketIDs...),
		}
		if repositoryPlan.ProviderEvidence != nil {
			for _, pullRequest := range repositoryPlan.ProviderEvidence.PullRequests {
				item.PullRequestIDs = append(item.PullRequestIDs, strings.TrimSpace(pullRequest.ID))
			}
			for _, issue := range repositoryPlan.ProviderEvidence.Issues {
				item.IssueIDs = append(item.IssueIDs, strings.TrimSpace(issue.ID))
			}
			item.PullRequestIDs = dedupeSortedStrings(item.PullRequestIDs)
			item.IssueIDs = dedupeSortedStrings(item.IssueIDs)
		}
		items = append(items, item)
	}
	return items
}

func collectReleaseHotspots(releasePlan releaseplansvc.ReleasePlan) []ReleaseHotspot {
	hotspots := make([]ReleaseHotspot, 0, len(releasePlan.Repositories))
	for _, repositoryPlan := range releasePlan.Repositories {
		reasons := make([]string, 0, 4)
		severity := "info"
		if len(repositoryPlan.TicketIDs) > 1 {
			reasons = append(reasons, fmt.Sprintf("%d release tickets touch this repository.", len(repositoryPlan.TicketIDs)))
			severity = "warning"
		}
		if len(repositoryPlan.RiskSignals) > 0 {
			reasons = append(reasons, fmt.Sprintf("%d risk signal(s) still need review.", len(repositoryPlan.RiskSignals)))
			severity = "warning"
		}
		if len(repositoryPlan.ManualSteps) > 0 {
			reasons = append(reasons, fmt.Sprintf("%d manual step(s) are still open.", len(repositoryPlan.ManualSteps)))
			severity = "warning"
		}
		if repositoryPlan.ProviderEvidence != nil {
			failingChecks := 0
			for _, check := range repositoryPlan.ProviderEvidence.Checks {
				if isFailingCheck(check.State) {
					failingChecks++
				}
			}
			if failingChecks > 0 {
				reasons = append(reasons, fmt.Sprintf("%d check status(es) are not green.", failingChecks))
				severity = "blocked"
			}
		}
		if len(reasons) == 0 {
			continue
		}
		hotspots = append(hotspots, ReleaseHotspot{
			Repository: repositoryPlan.Repository,
			TicketIDs:  append([]string(nil), repositoryPlan.TicketIDs...),
			Severity:   severity,
			Reasons:    reasons,
		})
	}
	return hotspots
}

func buildReleaseExecutiveSummary(releasePlan releaseplansvc.ReleasePlan, summary ReleaseEvidenceSummary) []string {
	lines := []string{
		fmt.Sprintf(
			"Verdict %s across %d ticket(s): %d blocked, %d warning, %d safe.",
			strings.ToLower(strings.TrimSpace(string(releasePlan.Verdict))),
			releasePlan.Summary.Tickets,
			releasePlan.Summary.BlockedTickets,
			releasePlan.Summary.WarningTickets,
			releasePlan.Summary.SafeTickets,
		),
	}
	if summary.FailingChecks > 0 || summary.Deployments > 0 {
		lines = append(lines, fmt.Sprintf("%d check status(es) are not green and %d deployment signal(s) were captured.", summary.FailingChecks, summary.Deployments))
	}
	if summary.LinkedIssues > 0 {
		lines = append(lines, fmt.Sprintf("%d linked issue(s) add delivery context for this release.", summary.LinkedIssues))
	}
	if summary.OverlappingTickets > 0 || summary.NewTicketsSinceLatestRelease > 0 {
		lines = append(lines, fmt.Sprintf("%d ticket(s) overlap with the latest release reference and %d ticket(s) appear new since that release.", summary.OverlappingTickets, summary.NewTicketsSinceLatestRelease))
	}
	if summary.Hotspots > 0 {
		lines = append(lines, fmt.Sprintf("%d repository hotspot(s) still need operator attention.", summary.Hotspots))
	}
	return limitSummaryLines(lines, 4)
}

func buildReleaseOperatorSummary(items []ReleaseRepositoryEvidence, overlaps []ReleaseTicketOverlap, hotspots []ReleaseHotspot) []string {
	lines := make([]string, 0, len(items)+1)
	for _, item := range items {
		signals := make([]string, 0, 6)
		if failing := countFailingChecks(item.Checks); failing > 0 {
			signals = append(signals, fmt.Sprintf("%d non-green check(s)", failing))
		}
		if pending := countNonSuccessfulDeployments(item.Deployments); pending > 0 {
			signals = append(signals, fmt.Sprintf("%d deployment(s) not successful", pending))
		}
		if openIssues := countOpenIssues(item.LinkedIssues); openIssues > 0 {
			signals = append(signals, fmt.Sprintf("%d linked issue(s) still open", openIssues))
		}
		if item.LatestRelease != nil {
			signals = append(signals, "latest release "+latestReleaseLabel(*item.LatestRelease))
		}
		if len(item.OverlapTickets) > 0 {
			signals = append(signals, "already in latest release: "+strings.Join(item.OverlapTickets, ", "))
		}
		if len(item.NewTicketsSinceLatestRelease) > 0 {
			signals = append(signals, "new since latest release: "+strings.Join(item.NewTicketsSinceLatestRelease, ", "))
		}
		if len(signals) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s.", item.Repository.Root, strings.Join(signals, "; ")))
	}
	if len(overlaps) > 0 {
		lines = append(lines, fmt.Sprintf("%d repository overlap(s) carry multiple release tickets in the same scope.", len(overlaps)))
	}
	if len(lines) == 0 {
		for _, hotspot := range hotspots {
			if len(hotspot.Reasons) == 0 {
				continue
			}
			lines = append(lines, fmt.Sprintf("%s: %s.", hotspot.Repository.Root, strings.TrimSuffix(strings.TrimSpace(hotspot.Reasons[0]), ".")))
		}
	}
	return limitSummaryLines(lines, 4)
}

func isFailingCheck(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "", "success", "neutral", "skipped":
		return false
	default:
		return true
	}
}

func latestReleaseEvidence(items []scm.ReleaseEvidence) *scm.ReleaseEvidence {
	if len(items) == 0 {
		return nil
	}
	release := items[0]
	release.TicketIDs = append([]string(nil), release.TicketIDs...)
	return &release
}

func releaseTicketDelta(current []string, latestRelease *scm.ReleaseEvidence) ([]string, []string) {
	if latestRelease == nil || len(latestRelease.TicketIDs) == 0 {
		return nil, nil
	}
	overlap := intersectStrings(current, latestRelease.TicketIDs)
	newTickets := differenceStrings(current, latestRelease.TicketIDs)
	return overlap, newTickets
}

func intersectStrings(left, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(right))
	for _, item := range right {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		seen[item] = struct{}{}
	}
	items := make([]string, 0, len(left))
	for _, item := range left {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			items = append(items, item)
		}
	}
	return dedupeSortedStrings(items)
}

func differenceStrings(left, right []string) []string {
	if len(left) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(right))
	for _, item := range right {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		seen[item] = struct{}{}
	}
	items := make([]string, 0, len(left))
	for _, item := range left {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		items = append(items, item)
	}
	return dedupeSortedStrings(items)
}

func dedupeSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	sort.Strings(items)
	return items
}

func countFailingChecks(checks []scm.CheckEvidence) int {
	total := 0
	for _, check := range checks {
		if isFailingCheck(check.State) {
			total++
		}
	}
	return total
}

func countNonSuccessfulDeployments(items []scm.DeploymentEvidence) int {
	total := 0
	for _, item := range items {
		state := strings.ToLower(strings.TrimSpace(item.State))
		if state == "" || state == "success" || state == "active" {
			continue
		}
		total++
	}
	return total
}

func countOpenIssues(items []scm.IssueEvidence) int {
	total := 0
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.State), "closed") {
			continue
		}
		total++
	}
	return total
}

func latestReleaseLabel(release scm.ReleaseEvidence) string {
	switch {
	case strings.TrimSpace(release.Tag) != "":
		return strings.TrimSpace(release.Tag)
	case strings.TrimSpace(release.Name) != "":
		return strings.TrimSpace(release.Name)
	default:
		return strings.TrimSpace(release.ID)
	}
}

func limitSummaryLines(lines []string, limit int) []string {
	if len(lines) == 0 {
		return nil
	}
	lines = dedupeStringsKeepOrder(lines)
	if limit <= 0 || len(lines) <= limit {
		return lines
	}
	remaining := len(lines) - limit
	trimmed := append([]string(nil), lines[:limit]...)
	trimmed = append(trimmed, fmt.Sprintf("%d additional release signal(s) are present in the bundle.", remaining))
	return trimmed
}

func dedupeStringsKeepOrder(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	return items
}

func defaultResolveActions() []ResolveAction {
	return []ResolveAction{
		{Key: "1", Description: "Accept the current side for the active conflict block."},
		{Key: "2", Description: "Accept the incoming side for the active conflict block."},
		{Key: "3", Description: "Keep both sides in current-then-incoming order."},
		{Key: "4", Description: "Keep both sides in incoming-then-current order."},
		{Key: "e", Description: "Open the conflicted file in $EDITOR for manual editing."},
		{Key: "s", Description: "Stage the file once no conflict markers remain."},
		{Key: "n", Description: "Move to the next conflict block."},
		{Key: "N", Description: "Move to the next conflicted file."},
	}
}

func trimConflictContent(content []byte) string {
	return strings.TrimRight(string(content), "\n")
}
