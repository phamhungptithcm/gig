package git

import (
	"context"
	"strings"
)

func (a *Adapter) CommitMessages(ctx context.Context, repoRoot string, hashes []string) (map[string]string, error) {
	messages := make(map[string]string, len(hashes))
	seen := make(map[string]struct{}, len(hashes))

	for _, hash := range hashes {
		hash = strings.TrimSpace(hash)
		if hash == "" {
			continue
		}
		if _, ok := seen[hash]; ok {
			continue
		}
		seen[hash] = struct{}{}

		output, err := a.runGit(ctx, repoRoot, "show", "-s", "--format=%B", hash)
		if err != nil {
			return nil, err
		}

		messages[hash] = strings.TrimRight(output, "\n")
	}

	return messages, nil
}
