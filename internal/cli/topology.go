package cli

import (
	"context"
	"fmt"
	"strings"

	"gig/internal/diagnostics"
	inspectsvc "gig/internal/inspect"
	"gig/internal/scm"
	"gig/internal/sourcecontrol"
)

type operationTopology struct {
	Environments []inspectsvc.Environment
	Inference    *sourcecontrol.TopologyInference
}

func resolveOperationContext(ctx context.Context, runtime commandRuntime, repositories []scm.Repository, envSpec, fromBranch, toBranch string) ([]inspectsvc.Environment, string, string, error) {
	topology, err := resolveOperationTopology(ctx, runtime, repositories, envSpec)
	if err != nil {
		return nil, "", "", err
	}

	fromBranch = strings.TrimSpace(fromBranch)
	toBranch = strings.TrimSpace(toBranch)
	if fromBranch != "" && toBranch != "" {
		return topology.Environments, fromBranch, toBranch, nil
	}

	if !containsRemoteRepositories(repositories) {
		return nil, "", "", fmt.Errorf("both --from and --to branches are required")
	}

	if topology.Inference != nil && topology.Inference.Confidence != sourcecontrol.TopologyConfidenceHigh {
		err := topologyInferenceError(repositories, *topology.Inference)
		diagnostics.Emit(ctx, "warning", "topology.resolve", "promotion path requires explicit branches", topologyDiagnosticMeta(repositories, "", "", topology.Inference), err)
		return nil, "", "", err
	}

	inferredFrom, inferredTo, err := sourcecontrol.InferPromotionBranches(topology.Environments, fromBranch, toBranch)
	if err != nil {
		diagnostics.Emit(ctx, "error", "topology.resolve", "promotion path inference failed", topologyDiagnosticMeta(repositories, fromBranch, toBranch, topology.Inference), err)
		return nil, "", "", err
	}

	diagnostics.Emit(ctx, "info", "topology.resolve", "promotion path resolved", topologyDiagnosticMeta(repositories, inferredFrom, inferredTo, topology.Inference), nil)
	return topology.Environments, inferredFrom, inferredTo, nil
}

func resolveOperationEnvironments(ctx context.Context, runtime commandRuntime, repositories []scm.Repository, spec string) ([]inspectsvc.Environment, error) {
	topology, err := resolveOperationTopology(ctx, runtime, repositories, spec)
	if err != nil {
		return nil, err
	}
	return topology.Environments, nil
}

func resolveOperationTopology(ctx context.Context, runtime commandRuntime, repositories []scm.Repository, spec string) (operationTopology, error) {
	if strings.TrimSpace(spec) != "" {
		environments, err := parseEnvironmentSpec(spec)
		if err != nil {
			return operationTopology{}, err
		}
		diagnostics.Emit(ctx, "info", "topology.environments", "using explicit environment mapping", topologyDiagnosticMeta(repositories, "", "", nil), nil)
		return operationTopology{Environments: environments}, nil
	}

	if runtime.loaded.ExplicitEnvironments || !containsRemoteRepositories(repositories) {
		environments, err := resolveEnvironments("", runtime.loaded)
		if err != nil {
			return operationTopology{}, err
		}
		diagnostics.Emit(ctx, "info", "topology.environments", "using configured environment mapping", topologyDiagnosticMeta(repositories, "", "", nil), nil)
		return operationTopology{Environments: environments}, nil
	}

	protectedBranches, err := protectedBranchesForRepositories(ctx, runtime, repositories)
	if err != nil {
		diagnostics.Emit(ctx, "error", "topology.protected-branches", "loading protected branches failed", topologyDiagnosticMeta(repositories, "", "", nil), err)
		return operationTopology{}, err
	}

	inference := sourcecontrol.InferProtectedBranchTopology(protectedBranches)
	diagnostics.Emit(ctx, "info", "topology.protected-branches", "evaluated protected branch topology", topologyDiagnosticMeta(repositories, "", "", &inference), nil)
	if len(inference.Environments) == 0 || inference.Confidence != sourcecontrol.TopologyConfidenceHigh {
		err := topologyInferenceError(repositories, inference)
		diagnostics.Emit(ctx, "warning", "topology.protected-branches", "topology requires explicit branch defaults", topologyDiagnosticMeta(repositories, "", "", &inference), err)
		return operationTopology{}, err
	}

	return operationTopology{
		Environments: inference.Environments,
		Inference:    &inference,
	}, nil
}

func protectedBranchesForRepositories(ctx context.Context, runtime commandRuntime, repositories []scm.Repository) ([]string, error) {
	branches := make([]string, 0)
	seen := map[string]struct{}{}

	for _, repository := range repositories {
		adapter, ok := runtime.adapters.For(repository.Type)
		if !ok {
			continue
		}
		provider, ok := adapter.(scm.ProtectedBranchProvider)
		if !ok {
			continue
		}
		protectedBranches, err := provider.ProtectedBranches(ctx, repository.Root)
		if err != nil {
			return nil, err
		}
		for _, branch := range protectedBranches {
			if _, ok := seen[branch]; ok {
				continue
			}
			seen[branch] = struct{}{}
			branches = append(branches, branch)
		}
	}

	return branches, nil
}

func containsRemoteRepositories(repositories []scm.Repository) bool {
	for _, repository := range repositories {
		if repository.Type.IsRemote() {
			return true
		}
	}
	return false
}

func topologyInferenceError(repositories []scm.Repository, inference sourcecontrol.TopologyInference) error {
	scope := topologyScopeLabel(repositories)
	parts := make([]string, 0, 3)
	if summary := strings.TrimSpace(inference.Summary); summary != "" {
		parts = append(parts, summary)
	}
	if len(inference.ProtectedBranches) > 0 {
		parts = append(parts, "Protected branches: "+strings.Join(inference.ProtectedBranches, ", ")+".")
	}
	parts = append(parts, "Pass --envs and explicit --from/--to, or save workarea defaults.")
	return fmt.Errorf("gig is not sure about the protected branch topology for %s. %s", scope, strings.Join(parts, " "))
}

func topologyScopeLabel(repositories []scm.Repository) string {
	if len(repositories) == 1 {
		return repositories[0].Root
	}
	return "the selected remote repositories"
}

func topologyDiagnosticMeta(repositories []scm.Repository, fromBranch, toBranch string, inference *sourcecontrol.TopologyInference) diagnostics.Meta {
	meta := diagnostics.Meta{
		Command:    "",
		FromBranch: strings.TrimSpace(fromBranch),
		ToBranch:   strings.TrimSpace(toBranch),
	}
	if len(repositories) == 1 {
		meta.Repo = repositories[0].Root
		meta.SCM = string(repositories[0].Type)
	}
	if inference != nil {
		meta.Details = map[string]any{
			"confidence":        inference.Confidence,
			"protectedBranches": append([]string(nil), inference.ProtectedBranches...),
			"summary":           inference.Summary,
		}
	}
	return meta
}
