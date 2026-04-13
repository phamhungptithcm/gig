package assistant

import (
	"context"
	"fmt"
	"strings"

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
	ScopeLabel   string                       `json:"scopeLabel"`
	Workspace    string                       `json:"workspace"`
	ReleaseID    string                       `json:"releaseId"`
	SnapshotDir  string                       `json:"snapshotDir"`
	FromBranch   string                       `json:"fromBranch,omitempty"`
	ToBranch     string                       `json:"toBranch,omitempty"`
	Environments []inspectsvc.Environment     `json:"environments,omitempty"`
	Snapshots    []snapshotsvc.TicketSnapshot `json:"snapshots,omitempty"`
	ReleasePlan  releaseplansvc.ReleasePlan   `json:"releasePlan"`
	Packets      []manifestsvc.ReleasePacket  `json:"packets,omitempty"`
	Hints        BundleHints                  `json:"hints,omitempty"`
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

	return ReleaseBundle{
		ScopeLabel:   request.ScopeLabel,
		Workspace:    request.WorkspacePath,
		ReleaseID:    releaseID,
		SnapshotDir:  snapshotDir,
		FromBranch:   releasePlan.FromBranch,
		ToBranch:     releasePlan.ToBranch,
		Environments: cloneReleaseEnvironments(releasePlan.Environments),
		Snapshots:    cloneSnapshots(snapshots),
		ReleasePlan:  releasePlan,
		Packets:      packets,
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

	return ReleaseBundle{
		ScopeLabel:   scopeLabel,
		Workspace:    scopeLabel,
		ReleaseID:    releaseID,
		SnapshotDir:  "",
		FromBranch:   releasePlan.FromBranch,
		ToBranch:     releasePlan.ToBranch,
		Environments: cloneReleaseEnvironments(releasePlan.Environments),
		Snapshots:    cloneSnapshots(snapshots),
		ReleasePlan:  releasePlan,
		Packets:      packets,
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
