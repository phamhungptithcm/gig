package scm

import (
	"sort"
	"strings"
)

type branchOrderCandidate struct {
	branch string
	order  int
}

func SelectRemoteAuditBranches(branches []string, defaultBranch string) []string {
	defaultBranch = normalizeBranchName(defaultBranch)
	candidates := make([]branchOrderCandidate, 0, len(branches)+1)
	seen := map[string]struct{}{}

	add := func(branch string, allowTransient bool) {
		branch = normalizeBranchName(branch)
		if branch == "" {
			return
		}
		if !allowTransient && isTransientAuditBranch(branch) {
			return
		}
		if _, ok := seen[branch]; ok {
			return
		}
		seen[branch] = struct{}{}
		candidates = append(candidates, branchOrderCandidate{
			branch: branch,
			order:  branchAuditPriority(branch),
		})
	}

	for _, branch := range branches {
		normalized := normalizeBranchName(branch)
		add(normalized, normalized == defaultBranch)
	}

	if defaultBranch != "" {
		add(defaultBranch, true)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].order == candidates[j].order {
			return candidates[i].branch < candidates[j].branch
		}
		return candidates[i].order < candidates[j].order
	})

	selected := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		selected = append(selected, candidate.branch)
	}

	return selected
}

func normalizeBranchName(branch string) string {
	branch = strings.TrimSpace(branch)
	branch = strings.TrimPrefix(branch, "refs/heads/")
	return branch
}

func branchAuditPriority(branch string) int {
	lower := strings.ToLower(strings.TrimSpace(branch))

	switch {
	case lower == "dev" || lower == "develop" || lower == "development" || lower == "integration":
		return 10
	case lower == "test" || lower == "qa":
		return 20
	case lower == "uat":
		return 30
	case lower == "staging" || lower == "stage" || lower == "preprod":
		return 40
	case strings.HasPrefix(lower, "release/"):
		return 45
	case lower == "main" || lower == "master" || lower == "prod" || lower == "production":
		return 50
	default:
		return 35
	}
}

func isTransientAuditBranch(branch string) bool {
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
