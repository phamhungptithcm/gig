package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"sort"
	"strings"

	"gig/internal/scm"
	"gig/internal/ticket"
)

const defaultCommitPageLimit = 20

type commandRunner func(ctx context.Context, args ...string) (string, error)

type Adapter struct {
	parser          ticket.Parser
	run             commandRunner
	commitPageLimit int
}

type projectPayload struct {
	DefaultBranch string `json:"default_branch"`
}

type branchPayload struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
}

type commitListItem struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	Title   string `json:"title"`
}

type commitPayload struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	Title   string `json:"title"`
}

type commitDiffPayload struct {
	NewPath string `json:"new_path"`
	OldPath string `json:"old_path"`
}

func NewAdapter(parser ticket.Parser) *Adapter {
	return &Adapter{
		parser:          parser,
		commitPageLimit: defaultCommitPageLimit,
	}
}

func (a *Adapter) Type() scm.Type {
	return scm.TypeGitLab
}

func (a *Adapter) DetectRoot(path string) (string, bool, error) {
	repository, err := parseRepositoryRoot(path)
	if err != nil {
		return "", false, nil
	}
	return repository, true, nil
}

func (a *Adapter) IsRepository(path string) (bool, error) {
	_, err := parseRepositoryRoot(path)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (a *Adapter) CurrentBranch(ctx context.Context, repoRoot string) (string, error) {
	project, err := a.project(ctx, repoRoot)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(project.DefaultBranch), nil
}

func (a *Adapter) SearchCommits(ctx context.Context, repoRoot string, query scm.SearchQuery) ([]scm.Commit, error) {
	if err := a.parser.Validate(query.TicketID); err != nil {
		return nil, err
	}

	if branch := strings.TrimSpace(query.Branch); branch != "" {
		return a.searchBranchCommits(ctx, repoRoot, branch, query.TicketID)
	}

	branches, err := a.ProtectedBranches(ctx, repoRoot)
	if err != nil {
		return nil, err
	}

	type indexedCommit struct {
		order  int
		commit scm.Commit
	}

	commitsByHash := map[string]indexedCommit{}
	order := 0
	for _, branch := range branches {
		branchCommits, err := a.searchBranchCommits(ctx, repoRoot, branch, query.TicketID)
		if err != nil {
			return nil, err
		}
		for _, commit := range branchCommits {
			existing, ok := commitsByHash[commit.Hash]
			if !ok {
				commitsByHash[commit.Hash] = indexedCommit{order: order, commit: commit}
				order++
				continue
			}
			existing.commit.Branches = dedupeStrings(append(existing.commit.Branches, commit.Branches...))
			commitsByHash[commit.Hash] = existing
		}
	}

	commits := make([]indexedCommit, 0, len(commitsByHash))
	for _, commit := range commitsByHash {
		sort.Strings(commit.commit.Branches)
		commits = append(commits, commit)
	}

	sort.SliceStable(commits, func(i, j int) bool {
		return commits[i].order < commits[j].order
	})

	results := make([]scm.Commit, 0, len(commits))
	for _, commit := range commits {
		results = append(results, commit.commit)
	}

	return results, nil
}

func (a *Adapter) CompareBranches(ctx context.Context, repoRoot string, query scm.CompareQuery) (scm.CompareResult, error) {
	if err := a.parser.Validate(query.TicketID); err != nil {
		return scm.CompareResult{}, err
	}
	if strings.TrimSpace(query.FromBranch) == "" || strings.TrimSpace(query.ToBranch) == "" {
		return scm.CompareResult{}, fmt.Errorf("both --from and --to branches are required")
	}

	fromExists, err := a.RefExists(ctx, repoRoot, query.FromBranch)
	if err != nil {
		return scm.CompareResult{}, err
	}
	if !fromExists {
		return scm.CompareResult{}, fmt.Errorf("unable to resolve branch %q in %s", query.FromBranch, repoRoot)
	}

	toExists, err := a.RefExists(ctx, repoRoot, query.ToBranch)
	if err != nil {
		return scm.CompareResult{}, err
	}
	if !toExists {
		return scm.CompareResult{}, fmt.Errorf("unable to resolve branch %q in %s", query.ToBranch, repoRoot)
	}

	sourceCommits, err := a.searchBranchCommits(ctx, repoRoot, query.FromBranch, query.TicketID)
	if err != nil {
		return scm.CompareResult{}, err
	}
	targetCommits, err := a.searchBranchCommits(ctx, repoRoot, query.ToBranch, query.TicketID)
	if err != nil {
		return scm.CompareResult{}, err
	}

	targetByHash := make(map[string]struct{}, len(targetCommits))
	for _, commit := range targetCommits {
		targetByHash[commit.Hash] = struct{}{}
	}

	missingCommits := make([]scm.Commit, 0, len(sourceCommits))
	for _, commit := range sourceCommits {
		if _, ok := targetByHash[commit.Hash]; ok {
			continue
		}
		missingCommits = append(missingCommits, commit)
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
	projectPath, err := parseProjectPath(repoRoot)
	if err != nil {
		return false, err
	}

	var branch branchPayload
	endpoint := fmt.Sprintf("projects/%s/repository/branches/%s", url.PathEscape(projectPath), url.PathEscape(strings.TrimSpace(ref)))
	if err := a.api(ctx, endpoint, &branch); err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return strings.TrimSpace(branch.Name) != "", nil
}

func (a *Adapter) ProtectedBranches(ctx context.Context, repoRoot string) ([]string, error) {
	projectPath, err := parseProjectPath(repoRoot)
	if err != nil {
		return nil, err
	}

	branches := make([]string, 0)
	for page := 1; page <= 5; page++ {
		var payload []branchPayload
		endpoint := fmt.Sprintf("projects/%s/repository/branches?per_page=100&page=%d", url.PathEscape(projectPath), page)
		if err := a.api(ctx, endpoint, &payload); err != nil {
			if isNotFound(err) {
				break
			}
			return nil, err
		}
		if len(payload) == 0 {
			break
		}
		for _, branch := range payload {
			if !branch.Protected || strings.TrimSpace(branch.Name) == "" {
				continue
			}
			branches = append(branches, branch.Name)
		}
		if len(payload) < 100 {
			break
		}
	}

	if len(branches) == 0 {
		project, err := a.project(ctx, repoRoot)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(project.DefaultBranch) != "" {
			branches = append(branches, strings.TrimSpace(project.DefaultBranch))
		}
	}

	return dedupeStrings(branches), nil
}

func (a *Adapter) CommitFiles(ctx context.Context, repoRoot string, hashes []string) (map[string][]string, error) {
	projectPath, err := parseProjectPath(repoRoot)
	if err != nil {
		return nil, err
	}

	filesByCommit := make(map[string][]string, len(hashes))
	seen := map[string]struct{}{}

	for _, hash := range hashes {
		hash = strings.TrimSpace(hash)
		if hash == "" {
			continue
		}
		if _, ok := seen[hash]; ok {
			continue
		}
		seen[hash] = struct{}{}

		files := make([]string, 0)
		fileSeen := map[string]struct{}{}
		for page := 1; page <= 5; page++ {
			var payload []commitDiffPayload
			endpoint := fmt.Sprintf("projects/%s/repository/commits/%s/diff?per_page=100&page=%d", url.PathEscape(projectPath), url.PathEscape(hash), page)
			if err := a.api(ctx, endpoint, &payload); err != nil {
				return nil, err
			}
			if len(payload) == 0 {
				break
			}
			for _, file := range payload {
				filename := strings.TrimSpace(file.NewPath)
				if filename == "" {
					filename = strings.TrimSpace(file.OldPath)
				}
				if filename == "" {
					continue
				}
				if _, ok := fileSeen[filename]; ok {
					continue
				}
				fileSeen[filename] = struct{}{}
				files = append(files, filename)
			}
			if len(payload) < 100 {
				break
			}
		}
		sort.Strings(files)
		filesByCommit[hash] = files
	}

	return filesByCommit, nil
}

func (a *Adapter) CommitMessages(ctx context.Context, repoRoot string, hashes []string) (map[string]string, error) {
	messagesByCommit := make(map[string]string, len(hashes))
	seen := map[string]struct{}{}

	for _, hash := range hashes {
		hash = strings.TrimSpace(hash)
		if hash == "" {
			continue
		}
		if _, ok := seen[hash]; ok {
			continue
		}
		seen[hash] = struct{}{}

		commit, err := a.commit(ctx, repoRoot, hash)
		if err != nil {
			return nil, err
		}
		messagesByCommit[hash] = strings.TrimRight(commit.Message, "\n")
	}

	return messagesByCommit, nil
}

func (a *Adapter) searchBranchCommits(ctx context.Context, repoRoot, branch, ticketID string) ([]scm.Commit, error) {
	projectPath, err := parseProjectPath(repoRoot)
	if err != nil {
		return nil, err
	}

	commits := make([]scm.Commit, 0)
	for page := 1; page <= a.pageLimit(); page++ {
		var payload []commitListItem
		endpoint := fmt.Sprintf("projects/%s/repository/commits?ref_name=%s&per_page=100&page=%d", url.PathEscape(projectPath), url.QueryEscape(branch), page)
		if err := a.api(ctx, endpoint, &payload); err != nil {
			return nil, err
		}
		if len(payload) == 0 {
			break
		}

		for _, item := range payload {
			message := strings.TrimSpace(item.Message)
			if !a.parser.Matches(ticketID, message) {
				continue
			}
			commits = append(commits, scm.Commit{
				Hash:     strings.TrimSpace(item.ID),
				Subject:  commitSubject(message, item.Title),
				Branches: []string{branch},
			})
		}

		if len(payload) < 100 {
			break
		}
	}

	return commits, nil
}

func (a *Adapter) project(ctx context.Context, repoRoot string) (projectPayload, error) {
	projectPath, err := parseProjectPath(repoRoot)
	if err != nil {
		return projectPayload{}, err
	}

	var project projectPayload
	if err := a.api(ctx, fmt.Sprintf("projects/%s", url.PathEscape(projectPath)), &project); err != nil {
		return projectPayload{}, err
	}
	return project, nil
}

func (a *Adapter) commit(ctx context.Context, repoRoot, hash string) (commitPayload, error) {
	projectPath, err := parseProjectPath(repoRoot)
	if err != nil {
		return commitPayload{}, err
	}

	var commit commitPayload
	if err := a.api(ctx, fmt.Sprintf("projects/%s/repository/commits/%s", url.PathEscape(projectPath), url.PathEscape(hash)), &commit); err != nil {
		return commitPayload{}, err
	}
	return commit, nil
}

func (a *Adapter) api(ctx context.Context, endpoint string, destination any) error {
	output, err := a.runGLab(ctx, "api", endpoint)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(output), destination); err != nil {
		return fmt.Errorf("parse gitlab api response for %s: %w", endpoint, err)
	}
	return nil
}

func (a *Adapter) runGLab(ctx context.Context, args ...string) (string, error) {
	if a.run != nil {
		return a.run(ctx, args...)
	}
	if _, err := exec.LookPath("glab"); err != nil {
		return "", fmt.Errorf("glab executable not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "glab", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("glab %s failed: %s", strings.Join(args, " "), message)
	}

	return string(output), nil
}

func (a *Adapter) pageLimit() int {
	if a.commitPageLimit <= 0 {
		return defaultCommitPageLimit
	}
	return a.commitPageLimit
}

func parseRepositoryRoot(repoRoot string) (string, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if !strings.HasPrefix(strings.ToLower(repoRoot), "gitlab:") {
		return "", errors.New("not a gitlab repository target")
	}
	_, err := parseProjectPath(repoRoot)
	if err != nil {
		return "", err
	}
	return repoRoot, nil
}

func parseProjectPath(repoRoot string) (string, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	repoRoot = strings.TrimPrefix(repoRoot, "gitlab:")
	repoRoot = strings.TrimPrefix(repoRoot, "GITLAB:")
	repoRoot = strings.TrimPrefix(repoRoot, "/")
	repoRoot = strings.TrimSuffix(repoRoot, ".git")
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return "", fmt.Errorf("gitlab repository target must be in group/project form")
	}

	parts := strings.Split(repoRoot, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("gitlab repository target must be in group/project form")
	}

	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return "", fmt.Errorf("gitlab repository target must be in group/project form")
		}
	}

	return strings.Join(parts, "/"), nil
}

func commitSubject(message, fallback string) string {
	message = strings.TrimSpace(message)
	if message != "" {
		line, _, _ := strings.Cut(message, "\n")
		if strings.TrimSpace(line) != "" {
			return strings.TrimSpace(line)
		}
	}
	return strings.TrimSpace(fallback)
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	deduped := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		deduped = append(deduped, value)
	}
	return deduped
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "http 404") || strings.Contains(message, "404 not found")
}
