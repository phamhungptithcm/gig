package sourcecontrol

import (
	"fmt"
	"sort"
	"strings"

	inspectsvc "gig/internal/inspect"
)

type branchCandidate struct {
	Branch string
	Name   string
	Order  int
}

func InferEnvironments(branches []string) []inspectsvc.Environment {
	candidates := make([]branchCandidate, 0, len(branches))
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

		name, order := classifyBranch(branch)
		candidates = append(candidates, branchCandidate{
			Branch: branch,
			Name:   name,
			Order:  order,
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Order == candidates[j].Order {
			return candidates[i].Branch < candidates[j].Branch
		}
		return candidates[i].Order < candidates[j].Order
	})

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

func classifyBranch(branch string) (string, int) {
	lower := strings.ToLower(strings.TrimSpace(branch))

	switch {
	case lower == "dev" || lower == "develop" || lower == "development" || lower == "integration":
		return "dev", 10
	case lower == "test" || lower == "qa":
		return lower, 20
	case lower == "uat":
		return "uat", 30
	case lower == "staging" || lower == "stage" || lower == "preprod":
		return "staging", 40
	case strings.HasPrefix(lower, "release/"):
		return "release", 45
	case lower == "main" || lower == "master" || lower == "prod" || lower == "production":
		return "prod", 50
	default:
		return sanitizeEnvironmentName(branch), 35
	}
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
	}
	for _, prefix := range transientPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}
