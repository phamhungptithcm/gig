package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gig/internal/scm"
	"gig/internal/ticket"
)

type Adapter struct {
	parser ticket.Parser
}

func NewAdapter(parser ticket.Parser) *Adapter {
	return &Adapter{parser: parser}
}

func (a *Adapter) Type() scm.Type {
	return scm.TypeGit
}

func (a *Adapter) DetectRoot(path string) (string, bool, error) {
	start, err := normalizePath(path)
	if err != nil {
		return "", false, err
	}

	for {
		ok, err := a.IsRepository(start)
		if err != nil {
			return "", false, err
		}
		if ok {
			return start, true, nil
		}

		parent := filepath.Dir(start)
		if parent == start {
			return "", false, nil
		}
		start = parent
	}
}

func (a *Adapter) IsRepository(path string) (bool, error) {
	_, err := os.Stat(filepath.Join(path, ".git"))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, err
}

func (a *Adapter) CurrentBranch(ctx context.Context, repoRoot string) (string, error) {
	output, err := a.runGit(ctx, repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}

	branch := strings.TrimSpace(output)
	if branch != "HEAD" {
		return branch, nil
	}

	hash, err := a.runGit(ctx, repoRoot, "rev-parse", "--short", "HEAD")
	if err != nil {
		return "HEAD", nil
	}

	return "detached@" + strings.TrimSpace(hash), nil
}

func (a *Adapter) SearchCommits(ctx context.Context, repoRoot string, query scm.SearchQuery) ([]scm.Commit, error) {
	if err := a.parser.Validate(query.TicketID); err != nil {
		return nil, err
	}

	return a.logCommits(ctx, repoRoot, query.Branch, query.TicketID)
}

func (a *Adapter) CompareBranches(ctx context.Context, repoRoot string, query scm.CompareQuery) (scm.CompareResult, error) {
	if err := a.parser.Validate(query.TicketID); err != nil {
		return scm.CompareResult{}, err
	}
	if strings.TrimSpace(query.FromBranch) == "" || strings.TrimSpace(query.ToBranch) == "" {
		return scm.CompareResult{}, fmt.Errorf("both --from and --to branches are required")
	}

	if err := a.ensureRef(ctx, repoRoot, query.FromBranch); err != nil {
		return scm.CompareResult{}, err
	}
	if err := a.ensureRef(ctx, repoRoot, query.ToBranch); err != nil {
		return scm.CompareResult{}, err
	}

	sourceCommits, err := a.logCommits(ctx, repoRoot, query.FromBranch, query.TicketID)
	if err != nil {
		return scm.CompareResult{}, err
	}

	targetCommits, err := a.logCommits(ctx, repoRoot, query.ToBranch, query.TicketID)
	if err != nil {
		return scm.CompareResult{}, err
	}

	missingCommits, err := a.missingCommits(ctx, repoRoot, query.FromBranch, query.ToBranch, sourceCommits)
	if err != nil {
		return scm.CompareResult{}, err
	}

	return scm.CompareResult{
		FromBranch:     query.FromBranch,
		ToBranch:       query.ToBranch,
		SourceCommits:  sourceCommits,
		TargetCommits:  targetCommits,
		MissingCommits: missingCommits,
	}, nil
}

func (a *Adapter) PrepareCherryPick(context.Context, string, scm.CherryPickPlan) error {
	return scm.ErrUnsupported
}

func (a *Adapter) RefExists(ctx context.Context, repoRoot, ref string) (bool, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return false, fmt.Errorf("git executable not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "rev-parse", "--verify", ref)
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (a *Adapter) CommitFiles(ctx context.Context, repoRoot string, hashes []string) (map[string][]string, error) {
	filesByCommit := make(map[string][]string, len(hashes))
	seen := make(map[string]struct{}, len(hashes))

	for _, hash := range hashes {
		if _, ok := seen[hash]; ok {
			continue
		}
		seen[hash] = struct{}{}

		output, err := a.runGit(ctx, repoRoot, "show", "--pretty=format:", "--name-only", hash)
		if err != nil {
			return nil, err
		}

		fileSeen := map[string]struct{}{}
		files := make([]string, 0)
		for _, line := range strings.Split(output, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if _, ok := fileSeen[line]; ok {
				continue
			}
			fileSeen[line] = struct{}{}
			files = append(files, line)
		}

		sort.Strings(files)
		filesByCommit[hash] = files
	}

	return filesByCommit, nil
}

func (a *Adapter) ensureRef(ctx context.Context, repoRoot, ref string) error {
	_, err := a.runGit(ctx, repoRoot, "rev-parse", "--verify", ref)
	if err != nil {
		return fmt.Errorf("unable to resolve branch %q in %s: %w", ref, repoRoot, err)
	}

	return nil
}

func (a *Adapter) logCommits(ctx context.Context, repoRoot, branch, ticketID string) ([]scm.Commit, error) {
	args := []string{"log"}
	if branch == "" {
		args = append(args, "--all")
	} else {
		args = append(args, branch)
	}

	args = append(
		args,
		"--regexp-ignore-case",
		"--extended-regexp",
		"--grep", ticket.RegexPattern(ticketID),
		"--pretty=format:%H%x1f%s%x1f%b%x1e",
	)

	output, err := a.runGit(ctx, repoRoot, args...)
	if err != nil {
		return nil, err
	}

	records := strings.Split(output, "\x1e")
	commits := make([]scm.Commit, 0, len(records))

	for _, record := range records {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}

		fields := strings.SplitN(record, "\x1f", 3)
		if len(fields) < 3 {
			continue
		}

		hash := strings.TrimSpace(fields[0])
		subject := strings.TrimSpace(fields[1])
		body := strings.TrimSpace(fields[2])

		if !a.parser.Matches(ticketID, subject+"\n"+body) {
			continue
		}

		commit := scm.Commit{
			Hash:    hash,
			Subject: subject,
		}

		if branch != "" {
			commit.Branches = []string{branch}
		} else {
			branches, err := a.branchesContaining(ctx, repoRoot, hash)
			if err != nil {
				return nil, err
			}
			commit.Branches = branches
		}

		commits = append(commits, commit)
	}

	return commits, nil
}

func (a *Adapter) missingCommits(ctx context.Context, repoRoot, fromBranch, toBranch string, sourceCommits []scm.Commit) ([]scm.Commit, error) {
	if len(sourceCommits) == 0 {
		return nil, nil
	}

	output, err := a.runGit(ctx, repoRoot, "cherry", toBranch, fromBranch)
	if err != nil {
		return nil, err
	}

	missingByHash := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}

		if fields[0] == "+" {
			missingByHash[fields[1]] = true
		}
	}

	missing := make([]scm.Commit, 0, len(sourceCommits))
	for _, commit := range sourceCommits {
		if missingByHash[commit.Hash] {
			missing = append(missing, commit)
		}
	}

	return missing, nil
}

func (a *Adapter) branchesContaining(ctx context.Context, repoRoot, hash string) ([]string, error) {
	output, err := a.runGit(
		ctx,
		repoRoot,
		"for-each-ref",
		"--contains", hash,
		"--format=%(refname:short)",
		"refs/heads",
		"refs/remotes",
	)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	branches := make([]string, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		branches = append(branches, line)
	}

	sort.Strings(branches)
	return branches, nil
}

func (a *Adapter) runGit(ctx context.Context, repoRoot string, args ...string) (string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", fmt.Errorf("git executable not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repoRoot}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), message)
	}

	return string(output), nil
}

func normalizePath(path string) (string, error) {
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return absPath, nil
	}

	return filepath.Dir(absPath), nil
}
