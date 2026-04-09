package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gig/internal/scm"
)

func (a *Adapter) ConflictState(ctx context.Context, repoRoot string) (scm.ConflictOperationState, bool, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return scm.ConflictOperationState{}, false, fmt.Errorf("repository root is required")
	}

	headHashOutput, err := a.runGit(ctx, repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return scm.ConflictOperationState{}, false, err
	}
	headHash := strings.TrimSpace(headHashOutput)

	currentBranch, _ := a.CurrentBranch(ctx, repoRoot)

	if path, ok, err := a.gitPathExists(ctx, repoRoot, "REBASE_HEAD"); err != nil {
		return scm.ConflictOperationState{}, false, err
	} else if ok {
		replayHash, err := readTrimmedFile(path)
		if err != nil {
			return scm.ConflictOperationState{}, false, err
		}

		headName, _ := a.readGitPathTrimmed(ctx, repoRoot, filepath.Join("rebase-merge", "head-name"))
		if headName == "" {
			headName, _ = a.readGitPathTrimmed(ctx, repoRoot, filepath.Join("rebase-apply", "head-name"))
		}
		sequenceBranch := shortRefName(headName)

		ontoHash, _ := a.readGitPathTrimmed(ctx, repoRoot, filepath.Join("rebase-merge", "onto"))
		if ontoHash == "" {
			ontoHash, _ = a.readGitPathTrimmed(ctx, repoRoot, filepath.Join("rebase-apply", "onto"))
		}

		currentSide, err := a.describeConflictSide(ctx, repoRoot, headHash, currentBranch, "Base line")
		if err != nil {
			return scm.ConflictOperationState{}, false, err
		}
		incomingSide, err := a.describeConflictSide(ctx, repoRoot, replayHash, sequenceBranch, "Replayed pick")
		if err != nil {
			return scm.ConflictOperationState{}, false, err
		}

		targetBranch := ""
		if ontoHash != "" {
			targetBranch, _ = a.primaryRefForCommit(ctx, repoRoot, ontoHash)
			if targetBranch == "" {
				targetBranch = ontoHash
			}
		}

		return scm.ConflictOperationState{
			Type:                scm.ConflictOperationRebase,
			RepoRoot:            repoRoot,
			CurrentBranch:       currentBranch,
			SequenceBranch:      sequenceBranch,
			TargetBranch:        targetBranch,
			ContinuationCommand: "git rebase --continue",
			CurrentSide:         currentSide,
			IncomingSide:        incomingSide,
		}, true, nil
	}

	if path, ok, err := a.gitPathExists(ctx, repoRoot, "CHERRY_PICK_HEAD"); err != nil {
		return scm.ConflictOperationState{}, false, err
	} else if ok {
		commitHash, err := readTrimmedFile(path)
		if err != nil {
			return scm.ConflictOperationState{}, false, err
		}

		currentSide, err := a.describeConflictSide(ctx, repoRoot, headHash, currentBranch, "Current")
		if err != nil {
			return scm.ConflictOperationState{}, false, err
		}
		incomingSide, err := a.describeConflictSide(ctx, repoRoot, commitHash, "", "Incoming")
		if err != nil {
			return scm.ConflictOperationState{}, false, err
		}

		return scm.ConflictOperationState{
			Type:                scm.ConflictOperationCherryPick,
			RepoRoot:            repoRoot,
			CurrentBranch:       currentBranch,
			ContinuationCommand: "git cherry-pick --continue",
			CurrentSide:         currentSide,
			IncomingSide:        incomingSide,
		}, true, nil
	}

	if path, ok, err := a.gitPathExists(ctx, repoRoot, "MERGE_HEAD"); err != nil {
		return scm.ConflictOperationState{}, false, err
	} else if ok {
		mergeHash, err := readTrimmedFile(path)
		if err != nil {
			return scm.ConflictOperationState{}, false, err
		}

		currentSide, err := a.describeConflictSide(ctx, repoRoot, headHash, currentBranch, "Current")
		if err != nil {
			return scm.ConflictOperationState{}, false, err
		}
		incomingBranch, _ := a.primaryRefForCommit(ctx, repoRoot, mergeHash)
		incomingSide, err := a.describeConflictSide(ctx, repoRoot, mergeHash, incomingBranch, "Incoming")
		if err != nil {
			return scm.ConflictOperationState{}, false, err
		}

		return scm.ConflictOperationState{
			Type:                scm.ConflictOperationMerge,
			RepoRoot:            repoRoot,
			CurrentBranch:       currentBranch,
			TargetBranch:        currentBranch,
			ContinuationCommand: "git merge --continue",
			CurrentSide:         currentSide,
			IncomingSide:        incomingSide,
		}, true, nil
	}

	return scm.ConflictOperationState{}, false, nil
}

func (a *Adapter) ConflictFiles(ctx context.Context, repoRoot string) ([]scm.ConflictFile, error) {
	statusOutput, err := a.runGit(ctx, repoRoot, "status", "--porcelain=v2", "-z", "--untracked-files=no")
	if err != nil {
		return nil, err
	}
	conflictCodes := parseConflictCodes(statusOutput)

	stageOutput, err := a.runGit(ctx, repoRoot, "ls-files", "-u", "-z")
	if err != nil {
		return nil, err
	}

	byPath := make(map[string]*scm.ConflictFile)
	for _, entry := range strings.Split(stageOutput, "\x00") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		parts := strings.SplitN(entry, "\t", 2)
		if len(parts) != 2 {
			continue
		}

		meta := strings.Fields(parts[0])
		if len(meta) != 3 {
			continue
		}

		path := parts[1]
		file := byPath[path]
		if file == nil {
			file = &scm.ConflictFile{
				Path:         path,
				ConflictCode: conflictCodes[path],
			}
			byPath[path] = file
		}

		switch meta[2] {
		case "1":
			file.BaseMode = meta[0]
			file.BaseHash = meta[1]
		case "2":
			file.CurrentMode = meta[0]
			file.CurrentHash = meta[1]
		case "3":
			file.IncomingMode = meta[0]
			file.IncomingHash = meta[1]
		}
	}

	files := make([]scm.ConflictFile, 0, len(byPath))
	for _, file := range byPath {
		if file.ConflictCode == "" {
			file.ConflictCode = "UU"
		}
		files = append(files, *file)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

func (a *Adapter) ConflictBlob(ctx context.Context, repoRoot, objectHash string) ([]byte, error) {
	objectHash = strings.TrimSpace(objectHash)
	if objectHash == "" {
		return nil, nil
	}

	output, err := a.runGit(ctx, repoRoot, "cat-file", "-p", objectHash)
	if err != nil {
		return nil, err
	}

	return []byte(output), nil
}

func (a *Adapter) StageConflictFile(ctx context.Context, repoRoot, path string) error {
	_, err := a.runGit(ctx, repoRoot, "add", "--", path)
	return err
}

func (a *Adapter) describeConflictSide(ctx context.Context, repoRoot, hash, branch, label string) (scm.ConflictSide, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return scm.ConflictSide{Label: label, Branch: branch}, nil
	}

	message, err := a.CommitMessages(ctx, repoRoot, []string{hash})
	if err != nil {
		return scm.ConflictSide{}, err
	}

	subject := ""
	if raw, ok := message[hash]; ok {
		subject = firstNonEmptyLine(raw)
	}

	if branch == "" {
		branch, _ = a.primaryRefForCommit(ctx, repoRoot, hash)
	}

	return scm.ConflictSide{
		Label:      label,
		Branch:     branch,
		CommitHash: hash,
		Subject:    subject,
		TicketIDs:  a.parser.FindAll(strings.Join([]string{branch, subject}, "\n")),
	}, nil
}

func (a *Adapter) primaryRefForCommit(ctx context.Context, repoRoot, hash string) (string, error) {
	output, err := a.runGit(ctx, repoRoot, "for-each-ref", "--points-at", hash, "--format=%(refname:short)", "refs/heads", "refs/remotes")
	if err != nil {
		return "", err
	}

	candidates := uniqueSortedNonEmptyLines(output)
	if len(candidates) > 0 {
		return candidates[0], nil
	}

	output, err = a.runGit(ctx, repoRoot, "for-each-ref", "--contains", hash, "--format=%(refname:short)", "refs/heads", "refs/remotes")
	if err != nil {
		return "", err
	}

	candidates = uniqueSortedNonEmptyLines(output)
	if len(candidates) == 0 {
		return "", nil
	}
	return candidates[0], nil
}

func (a *Adapter) gitPathExists(ctx context.Context, repoRoot, path string) (string, bool, error) {
	resolvedOutput, err := a.runGit(ctx, repoRoot, "rev-parse", "--git-path", path)
	if err != nil {
		return "", false, err
	}
	resolved := strings.TrimSpace(resolvedOutput)
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(repoRoot, resolved)
	}

	if _, err := os.Stat(resolved); err == nil {
		return resolved, true, nil
	} else if os.IsNotExist(err) {
		return resolved, false, nil
	} else {
		return "", false, err
	}
}

func (a *Adapter) readGitPathTrimmed(ctx context.Context, repoRoot, path string) (string, error) {
	resolved, ok, err := a.gitPathExists(ctx, repoRoot, path)
	if err != nil || !ok {
		return "", err
	}

	return readTrimmedFile(resolved)
}

func parseConflictCodes(statusOutput string) map[string]string {
	entries := strings.Split(statusOutput, "\x00")
	result := make(map[string]string, len(entries))

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" || strings.HasPrefix(entry, "#") {
			continue
		}
		if !strings.HasPrefix(entry, "u ") {
			continue
		}

		fields := strings.SplitN(entry, " ", 11)
		if len(fields) != 11 {
			continue
		}

		result[fields[10]] = fields[1]
	}

	return result
}

func uniqueSortedNonEmptyLines(output string) []string {
	lines := strings.Split(output, "\n")
	seen := make(map[string]struct{}, len(lines))
	values := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		values = append(values, line)
	}
	sort.Strings(values)
	return values
}

func firstNonEmptyLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func readTrimmedFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func shortRefName(ref string) string {
	ref = strings.TrimSpace(ref)
	ref = strings.TrimPrefix(ref, "refs/heads/")
	ref = strings.TrimPrefix(ref, "refs/remotes/")
	return ref
}
