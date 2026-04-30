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
	fromBranch = strings.TrimSpace(fromBranch)
	toBranch = strings.TrimSpace(toBranch)
	topology, err := resolveOperationTopologyWithOptions(ctx, runtime, repositories, envSpec, operationTopologyOptions{
		AllowAmbiguous: fromBranch != "" && toBranch != "",
	})
	if err != nil {
		return nil, "", "", err
	}

	if fromBranch != "" && toBranch != "" {
		environments := topology.Environments
		if strings.TrimSpace(envSpec) == "" {
			environments = ensurePromotionEnvironments(environments, fromBranch, toBranch)
		}
		diagnostics.Emit(ctx, "info", "topology.resolve", "using explicit promotion path", topologyDiagnosticMeta(repositories, fromBranch, toBranch, topology.Inference), nil)
		return environments, fromBranch, toBranch, nil
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
	return resolveOperationTopologyWithOptions(ctx, runtime, repositories, spec, operationTopologyOptions{})
}

type operationTopologyOptions struct {
	AllowAmbiguous bool
}

func resolveOperationTopologyWithOptions(ctx context.Context, runtime commandRuntime, repositories []scm.Repository, spec string, options operationTopologyOptions) (operationTopology, error) {
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
	if !options.AllowAmbiguous && (len(inference.Environments) == 0 || inference.Confidence != sourcecontrol.TopologyConfidenceHigh) {
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

func ensurePromotionEnvironments(environments []inspectsvc.Environment, fromBranch, toBranch string) []inspectsvc.Environment {
	fromBranch = strings.TrimSpace(fromBranch)
	toBranch = strings.TrimSpace(toBranch)
	if fromBranch == "" && toBranch == "" {
		return environments
	}

	resolved := make([]inspectsvc.Environment, 0, len(environments)+2)
	seenBranches := map[string]struct{}{}
	for _, environment := range environments {
		name := strings.TrimSpace(environment.Name)
		branch := strings.TrimSpace(environment.Branch)
		if name == "" || branch == "" {
			continue
		}
		key := strings.ToLower(branch)
		if _, ok := seenBranches[key]; ok {
			continue
		}
		seenBranches[key] = struct{}{}
		resolved = append(resolved, inspectsvc.Environment{Name: name, Branch: branch})
	}

	hasBranch := func(branch string) bool {
		_, ok := seenBranches[strings.ToLower(strings.TrimSpace(branch))]
		return ok
	}
	addEnvironment := func(index int, branch string) {
		branch = strings.TrimSpace(branch)
		if branch == "" || hasBranch(branch) {
			return
		}
		environment := inspectsvc.Environment{Name: promotionEnvironmentName(branch), Branch: branch}
		if index < 0 || index >= len(resolved) {
			resolved = append(resolved, environment)
		} else {
			resolved = append(resolved[:index], append([]inspectsvc.Environment{environment}, resolved[index:]...)...)
		}
		seenBranches[strings.ToLower(branch)] = struct{}{}
	}

	if fromBranch != "" && !hasBranch(fromBranch) {
		insertAt := len(resolved)
		if toBranch != "" {
			if toIndex := environmentIndexByBranchInsensitive(resolved, toBranch); toIndex >= 0 {
				insertAt = toIndex
			}
		}
		addEnvironment(insertAt, fromBranch)
	}
	if toBranch != "" && !hasBranch(toBranch) {
		addEnvironment(len(resolved), toBranch)
	}

	return resolved
}

func environmentIndexByBranchInsensitive(environments []inspectsvc.Environment, branch string) int {
	branch = strings.ToLower(strings.TrimSpace(branch))
	for index, environment := range environments {
		if strings.ToLower(strings.TrimSpace(environment.Branch)) == branch {
			return index
		}
	}
	return -1
}

func promotionEnvironmentName(branch string) string {
	lower := strings.ToLower(strings.TrimSpace(branch))
	switch lower {
	case "dev", "develop", "development", "integration":
		return "dev"
	case "test", "qa":
		return lower
	case "uat":
		return "uat"
	case "staging", "stage", "preprod", "pre-prod":
		return "staging"
	case "main", "master", "prod", "production", "trunk":
		return "prod"
	}
	if strings.HasPrefix(lower, "release/") || strings.HasPrefix(lower, "release-") || strings.HasPrefix(lower, "release_") ||
		lower == "rc" || strings.HasPrefix(lower, "rc/") || strings.HasPrefix(lower, "rc-") || strings.HasPrefix(lower, "rc_") {
		return "release"
	}
	name := strings.ReplaceAll(lower, "/", "-")
	name = strings.ReplaceAll(name, "_", "-")
	if name == "" {
		return "env"
	}
	return name
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
	return &topologyResolutionError{
		Repositories: append([]scm.Repository(nil), repositories...),
		Inference:    inference,
	}
}

type topologyResolutionError struct {
	Repositories []scm.Repository
	Inference    sourcecontrol.TopologyInference
}

func (e *topologyResolutionError) Error() string {
	if e == nil {
		return ""
	}
	scope := topologyScopeLabel(e.Repositories)
	parts := make([]string, 0, 3)
	if summary := strings.TrimSpace(e.Inference.Summary); summary != "" {
		parts = append(parts, summary)
	}
	if len(e.Inference.ProtectedBranches) > 0 {
		parts = append(parts, "Protected branches: "+strings.Join(e.Inference.ProtectedBranches, ", ")+".")
	}
	parts = append(parts, "Run this command again with --from and --to, or use an interactive terminal so gig can ask for the promotion path.")
	return fmt.Sprintf("gig is not sure about the protected branch topology for %s. %s", scope, strings.Join(parts, " "))
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
