package scm

import "strings"

func MissingCommitsByEvidence(sourceCommits, targetCommits []Commit) []Commit {
	targetHashes := make(map[string]struct{}, len(targetCommits))
	targetSubjects := make(map[string]struct{}, len(targetCommits))
	for _, commit := range targetCommits {
		if hash := strings.TrimSpace(commit.Hash); hash != "" {
			targetHashes[hash] = struct{}{}
		}
		if subject := normalizeCommitSubject(commit.Subject); subject != "" {
			targetSubjects[subject] = struct{}{}
		}
	}

	missing := make([]Commit, 0, len(sourceCommits))
	for _, commit := range sourceCommits {
		if hash := strings.TrimSpace(commit.Hash); hash != "" {
			if _, ok := targetHashes[hash]; ok {
				continue
			}
		}
		if subject := normalizeCommitSubject(commit.Subject); subject != "" {
			if _, ok := targetSubjects[subject]; ok {
				continue
			}
		}
		missing = append(missing, commit)
	}
	return missing
}

func normalizeCommitSubject(subject string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(subject))), " ")
}
