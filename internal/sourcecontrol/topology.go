package sourcecontrol

import (
	"fmt"
	"sort"
	"strings"

	inspectsvc "gig/internal/inspect"
)

type TopologyConfidence string

const (
	TopologyConfidenceHigh   TopologyConfidence = "high"
	TopologyConfidenceMedium TopologyConfidence = "medium"
	TopologyConfidenceLow    TopologyConfidence = "low"
)

type TopologyInference struct {
	Source            string                   `json:"source,omitempty"`
	Confidence        TopologyConfidence       `json:"confidence,omitempty"`
	Summary           string                   `json:"summary,omitempty"`
	Reasons           []string                 `json:"reasons,omitempty"`
	ProtectedBranches []string                 `json:"protectedBranches,omitempty"`
	Environments      []inspectsvc.Environment `json:"environments,omitempty"`
	FromBranch        string                   `json:"fromBranch,omitempty"`
	ToBranch          string                   `json:"toBranch,omitempty"`
}

type branchRole string

const (
	branchRoleUnknown branchRole = "unknown"
	branchRoleDev     branchRole = "dev"
	branchRoleQA      branchRole = "qa"
	branchRoleUAT     branchRole = "uat"
	branchRoleStaging branchRole = "staging"
	branchRoleRelease branchRole = "release"
	branchRoleProd    branchRole = "prod"
)

type branchCandidate struct {
	Branch              string
	Name                string
	Order               int
	Role                branchRole
	Recognized          bool
	Production          bool
	CanonicalProduction bool
}

func InferProtectedBranchTopology(branches []string) TopologyInference {
	candidates, normalized := collectTopologyCandidates(branches)
	inference := TopologyInference{
		Source:            "protected-branches",
		Confidence:        TopologyConfidenceLow,
		ProtectedBranches: normalized,
		Environments:      buildTopologyEnvironments(candidates),
	}

	if len(candidates) == 0 {
		inference.Summary = "No protected environment branches were detected."
		return inference
	}

	unknown := make([]string, 0, len(candidates))
	productionIndexes := make([]int, 0, 2)
	for index, candidate := range candidates {
		if !candidate.Recognized {
			unknown = append(unknown, candidate.Branch)
		}
		if candidate.Production {
			productionIndexes = append(productionIndexes, index)
		}
	}

	switch {
	case len(productionIndexes) == 0:
		inference.Summary = "Protected branches are not clear enough for safe promotion inference."
		inference.Reasons = append(inference.Reasons, "No production-like protected branch was detected.")
		return inference
	case len(productionIndexes) > 1:
		inference.Summary = "Protected branches are not clear enough for safe promotion inference."
		inference.Reasons = append(inference.Reasons, "Multiple production-like protected branches were detected.")
		return inference
	}

	productionIndex := productionIndexes[0]
	if len(unknown) > 0 {
		inference.Summary = "Protected branches are not clear enough for safe promotion inference."
		inference.Reasons = append(inference.Reasons, fmt.Sprintf("Unrecognized protected branches: %s.", strings.Join(unknown, ", ")))
		if productionIndex == 0 {
			inference.Reasons = append(inference.Reasons, "No clear source branch appears before the production branch.")
		}
		return inference
	}

	if len(candidates) == 1 || productionIndex == 0 {
		inference.Confidence = TopologyConfidenceMedium
		inference.Summary = "Protected branches are partially informative but not enough to infer a full promotion path."
		inference.Reasons = append(inference.Reasons, "Only one protected branch is available before production, so gig cannot infer a promotion source.")
		return inference
	}

	previousOrder := candidates[productionIndex-1].Order
	previousOrderCount := 0
	for index, candidate := range candidates {
		if index >= productionIndex {
			break
		}
		if candidate.Order == previousOrder {
			previousOrderCount++
		}
	}
	if previousOrderCount > 1 {
		inference.Confidence = TopologyConfidenceMedium
		inference.Summary = "Protected branches show likely environments, but gig is not sure which branch should promote next."
		inference.Reasons = append(inference.Reasons, "Multiple protected branches look like the last pre-production stage.")
		return inference
	}

	inference.Confidence = TopologyConfidenceHigh
	inference.FromBranch = candidates[productionIndex-1].Branch
	inference.ToBranch = candidates[productionIndex].Branch
	inference.Summary = fmt.Sprintf("Provider protected branches form a clear promotion path: %s -> %s.", inference.FromBranch, inference.ToBranch)
	return inference
}

func InferEnvironments(branches []string) []inspectsvc.Environment {
	return InferProtectedBranchTopology(branches).Environments
}

func InferPromotionBranches(environments []inspectsvc.Environment, fromBranch, toBranch string) (string, string, error) {
	if len(environments) == 0 {
		return "", "", fmt.Errorf("unable to infer branch topology from source control")
	}

	fromBranch = strings.TrimSpace(fromBranch)
	toBranch = strings.TrimSpace(toBranch)

	if toBranch == "" {
		toBranch = environments[len(environments)-1].Branch
	}

	if fromBranch == "" {
		toIndex := environmentIndexByBranch(environments, toBranch)
		if toIndex <= 0 {
			return "", "", fmt.Errorf("unable to infer source branch for target %s", toBranch)
		}
		fromBranch = environments[toIndex-1].Branch
	}

	if environmentIndexByBranch(environments, fromBranch) == -1 {
		return "", "", fmt.Errorf("source branch %s is not part of the detected protected branch topology", fromBranch)
	}
	if environmentIndexByBranch(environments, toBranch) == -1 {
		return "", "", fmt.Errorf("target branch %s is not part of the detected protected branch topology", toBranch)
	}

	return fromBranch, toBranch, nil
}

func environmentIndexByBranch(environments []inspectsvc.Environment, branch string) int {
	for index, environment := range environments {
		if environment.Branch == branch {
			return index
		}
	}
	return -1
}

func collectTopologyCandidates(branches []string) ([]branchCandidate, []string) {
	candidates := make([]branchCandidate, 0, len(branches))
	normalized := make([]string, 0, len(branches))
	seen := map[string]struct{}{}

	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		if branch == "" || isTransientBranch(branch) {
			continue
		}
		if _, ok := seen[branch]; ok {
			continue
		}
		seen[branch] = struct{}{}

		candidate := classifyBranch(branch)
		candidates = append(candidates, candidate)
		normalized = append(normalized, candidate.Branch)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Order == candidates[j].Order {
			return candidates[i].Branch < candidates[j].Branch
		}
		return candidates[i].Order < candidates[j].Order
	})

	if hasCanonicalProductionBranch(candidates) {
		filtered := candidates[:0]
		for _, candidate := range candidates {
			if candidate.Production && !candidate.CanonicalProduction {
				continue
			}
			filtered = append(filtered, candidate)
		}
		candidates = filtered
	}

	normalized = normalized[:0]
	for _, candidate := range candidates {
		normalized = append(normalized, candidate.Branch)
	}

	return candidates, normalized
}

func buildTopologyEnvironments(candidates []branchCandidate) []inspectsvc.Environment {
	environments := make([]inspectsvc.Environment, 0, len(candidates))
	nameSeen := map[string]int{}
	for _, candidate := range candidates {
		name := candidate.Name
		if count := nameSeen[name]; count > 0 {
			name = fmt.Sprintf("%s%d", name, count+1)
		}
		nameSeen[candidate.Name]++
		environments = append(environments, inspectsvc.Environment{
			Name:   name,
			Branch: candidate.Branch,
		})
	}
	return environments
}

func classifyBranch(branch string) branchCandidate {
	lower := strings.ToLower(strings.TrimSpace(branch))
	candidate := branchCandidate{
		Branch: branch,
		Name:   sanitizeEnvironmentName(branch),
		Order:  35,
		Role:   branchRoleUnknown,
	}

	switch {
	case lower == "dev" || lower == "develop" || lower == "development" || lower == "integration":
		candidate.Name = "dev"
		candidate.Order = 10
		candidate.Role = branchRoleDev
		candidate.Recognized = true
	case lower == "test" || lower == "qa":
		candidate.Name = lower
		candidate.Order = 20
		candidate.Role = branchRoleQA
		candidate.Recognized = true
	case lower == "uat":
		candidate.Name = "uat"
		candidate.Order = 30
		candidate.Role = branchRoleUAT
		candidate.Recognized = true
	case lower == "staging" || lower == "stage" || lower == "preprod" || lower == "pre-prod":
		candidate.Name = "staging"
		candidate.Order = 40
		candidate.Role = branchRoleStaging
		candidate.Recognized = true
	case isReleaseCandidateBranch(lower):
		candidate.Name = "release"
		candidate.Order = 45
		candidate.Role = branchRoleRelease
		candidate.Recognized = true
	case lower == "main" || lower == "master" || lower == "prod" || lower == "production" || lower == "trunk":
		candidate.Name = "prod"
		candidate.Order = 50
		candidate.Role = branchRoleProd
		candidate.Recognized = true
		candidate.Production = true
		candidate.CanonicalProduction = lower != "trunk"
	}

	return candidate
}

func isReleaseCandidateBranch(lower string) bool {
	return strings.HasPrefix(lower, "release/") ||
		strings.HasPrefix(lower, "release-") ||
		strings.HasPrefix(lower, "release_") ||
		lower == "rc" ||
		strings.HasPrefix(lower, "rc/") ||
		strings.HasPrefix(lower, "rc-") ||
		strings.HasPrefix(lower, "rc_")
}

func sanitizeEnvironmentName(branch string) string {
	name := strings.ToLower(strings.TrimSpace(branch))
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "_", "-")
	if name == "" {
		return "env"
	}
	return name
}

func isTransientBranch(branch string) bool {
	lower := strings.ToLower(strings.TrimSpace(branch))
	transientPrefixes := []string{
		"feature/",
		"feat/",
		"fix/",
		"bugfix/",
		"chore/",
		"task/",
		"spike/",
		"wip/",
		"user/",
		"users/",
		"personal/",
		"hotfix/",
	}
	for _, prefix := range transientPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func hasCanonicalProductionBranch(candidates []branchCandidate) bool {
	for _, candidate := range candidates {
		if candidate.CanonicalProduction {
			return true
		}
	}
	return false
}
