package github

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

type repositoryPayload struct {
	DefaultBranch string `json:"default_branch"`
}

type branchPayload struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
}

type commitListItem struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
	} `json:"commit"`
}

type commitPayload struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
	} `json:"commit"`
	Files []struct {
		Filename string `json:"filename"`
	} `json:"files"`
}

type pullRequestPayload struct {
	Number   int    `json:"number"`
	Title    string `json:"title"`
	State    string `json:"state"`
	HTMLURL  string `json:"html_url"`
	MergedAt string `json:"merged_at"`
	Head     struct {
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

type deploymentPayload struct {
	ID          int    `json:"id"`
	SHA         string `json:"sha"`
	Ref         string `json:"ref"`
	Environment string `json:"environment"`
	StatusesURL string `json:"statuses_url"`
}

type deploymentStatusPayload struct {
	State       string `json:"state"`
	Environment string `json:"environment"`
	LogURL      string `json:"log_url"`
}

func NewAdapter(parser ticket.Parser) *Adapter {
	return &Adapter{
		parser:          parser,
		commitPageLimit: defaultCommitPageLimit,
	}
}

func (a *Adapter) Type() scm.Type {
	return scm.TypeGitHub
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
	repository, err := a.repository(ctx, repoRoot)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(repository.DefaultBranch), nil
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
			mergedBranches := append(existing.commit.Branches, commit.Branches...)
			existing.commit.Branches = dedupeStrings(mergedBranches)
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
	owner, name, err := parseOwnerRepo(repoRoot)
	if err != nil {
		return false, err
	}

	var branch branchPayload
	endpoint := fmt.Sprintf("repos/%s/%s/branches/%s", owner, name, url.PathEscape(strings.TrimSpace(ref)))
	if err := a.api(ctx, endpoint, &branch); err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return strings.TrimSpace(branch.Name) != "", nil
}

func (a *Adapter) ProtectedBranches(ctx context.Context, repoRoot string) ([]string, error) {
	owner, name, err := parseOwnerRepo(repoRoot)
	if err != nil {
		return nil, err
	}

	branches := make([]string, 0)
	for page := 1; page <= 5; page++ {
		var payload []branchPayload
		endpoint := fmt.Sprintf("repos/%s/%s/branches?protected=true&per_page=100&page=%d", owner, name, page)
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
			if strings.TrimSpace(branch.Name) == "" {
				continue
			}
			branches = append(branches, branch.Name)
		}
		if len(payload) < 100 {
			break
		}
	}

	if len(branches) == 0 {
		repository, err := a.repository(ctx, repoRoot)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(repository.DefaultBranch) != "" {
			branches = append(branches, strings.TrimSpace(repository.DefaultBranch))
		}
	}

	return dedupeStrings(branches), nil
}

func (a *Adapter) CommitFiles(ctx context.Context, repoRoot string, hashes []string) (map[string][]string, error) {
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

		commit, err := a.commit(ctx, repoRoot, hash)
		if err != nil {
			return nil, err
		}

		files := make([]string, 0, len(commit.Files))
		fileSeen := map[string]struct{}{}
		for _, file := range commit.Files {
			filename := strings.TrimSpace(file.Filename)
			if filename == "" {
				continue
			}
			if _, ok := fileSeen[filename]; ok {
				continue
			}
			fileSeen[filename] = struct{}{}
			files = append(files, filename)
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
		messagesByCommit[hash] = strings.TrimRight(commit.Commit.Message, "\n")
	}

	return messagesByCommit, nil
}

func (a *Adapter) ProviderEvidence(ctx context.Context, repoRoot string, query scm.EvidenceQuery) (scm.ProviderEvidence, error) {
	owner, name, err := parseOwnerRepo(repoRoot)
	if err != nil {
		return scm.ProviderEvidence{}, err
	}

	pullRequestsByID := map[string]scm.PullRequestEvidence{}
	deploymentsByID := map[string]scm.DeploymentEvidence{}
	for _, commit := range query.Commits {
		hash := strings.TrimSpace(commit.Hash)
		if hash == "" {
			continue
		}

		pullRequests, err := a.pullRequestsForCommit(ctx, owner, name, hash)
		if err != nil {
			return scm.ProviderEvidence{}, err
		}
		for _, item := range pullRequests {
			id := fmt.Sprintf("#%d", item.Number)
			state := strings.TrimSpace(item.State)
			if strings.TrimSpace(item.MergedAt) != "" {
				state = "merged"
			}
			pullRequestsByID[id] = scm.PullRequestEvidence{
				ID:           id,
				Title:        strings.TrimSpace(item.Title),
				State:        state,
				SourceBranch: strings.TrimSpace(item.Head.Ref),
				TargetBranch: strings.TrimSpace(item.Base.Ref),
				URL:          strings.TrimSpace(item.HTMLURL),
				CommitHash:   hash,
			}
		}

		deployments, err := a.deploymentsForCommit(ctx, owner, name, hash)
		if err != nil {
			return scm.ProviderEvidence{}, err
		}
		for _, item := range deployments {
			status, err := a.deploymentStatus(ctx, owner, name, item.ID)
			if err != nil {
				return scm.ProviderEvidence{}, err
			}

			id := fmt.Sprintf("%d", item.ID)
			deploymentsByID[id] = scm.DeploymentEvidence{
				ID:          id,
				Environment: firstNonEmpty(status.Environment, item.Environment),
				State:       strings.TrimSpace(status.State),
				Ref:         strings.TrimSpace(item.Ref),
				URL:         strings.TrimSpace(status.LogURL),
				CommitHash:  firstNonEmpty(strings.TrimSpace(item.SHA), hash),
			}
		}
	}

	return scm.ProviderEvidence{
		PullRequests: mapPullRequestEvidence(pullRequestsByID),
		Deployments:  mapDeploymentEvidence(deploymentsByID),
	}, nil
}

func (a *Adapter) searchBranchCommits(ctx context.Context, repoRoot, branch, ticketID string) ([]scm.Commit, error) {
	owner, name, err := parseOwnerRepo(repoRoot)
	if err != nil {
		return nil, err
	}

	commits := make([]scm.Commit, 0)
	for page := 1; page <= a.pageLimit(); page++ {
		var payload []commitListItem
		endpoint := fmt.Sprintf("repos/%s/%s/commits?sha=%s&per_page=100&page=%d", owner, name, url.QueryEscape(branch), page)
		if err := a.api(ctx, endpoint, &payload); err != nil {
			return nil, err
		}
		if len(payload) == 0 {
			break
		}

		for _, item := range payload {
			message := strings.TrimSpace(item.Commit.Message)
			if !a.parser.Matches(ticketID, message) {
				continue
			}
			commits = append(commits, scm.Commit{
				Hash:     strings.TrimSpace(item.SHA),
				Subject:  commitSubject(message),
				Branches: []string{branch},
			})
		}

		if len(payload) < 100 {
			break
		}
	}

	return commits, nil
}

func (a *Adapter) repository(ctx context.Context, repoRoot string) (repositoryPayload, error) {
	owner, name, err := parseOwnerRepo(repoRoot)
	if err != nil {
		return repositoryPayload{}, err
	}

	var repository repositoryPayload
	if err := a.api(ctx, fmt.Sprintf("repos/%s/%s", owner, name), &repository); err != nil {
		return repositoryPayload{}, err
	}
	return repository, nil
}

func (a *Adapter) commit(ctx context.Context, repoRoot, hash string) (commitPayload, error) {
	owner, name, err := parseOwnerRepo(repoRoot)
	if err != nil {
		return commitPayload{}, err
	}

	var commit commitPayload
	if err := a.api(ctx, fmt.Sprintf("repos/%s/%s/commits/%s", owner, name, url.PathEscape(hash)), &commit); err != nil {
		return commitPayload{}, err
	}
	return commit, nil
}

func (a *Adapter) pullRequestsForCommit(ctx context.Context, owner, name, hash string) ([]pullRequestPayload, error) {
	pullRequests := make([]pullRequestPayload, 0)
	for page := 1; page <= 5; page++ {
		var payload []pullRequestPayload
		endpoint := fmt.Sprintf("repos/%s/%s/commits/%s/pulls?per_page=100&page=%d", owner, name, url.PathEscape(hash), page)
		if err := a.api(ctx, endpoint, &payload); err != nil {
			if isNotFound(err) {
				break
			}
			return nil, err
		}
		if len(payload) == 0 {
			break
		}
		pullRequests = append(pullRequests, payload...)
		if len(payload) < 100 {
			break
		}
	}
	return pullRequests, nil
}

func (a *Adapter) deploymentsForCommit(ctx context.Context, owner, name, hash string) ([]deploymentPayload, error) {
	deployments := make([]deploymentPayload, 0)
	for page := 1; page <= 5; page++ {
		var payload []deploymentPayload
		endpoint := fmt.Sprintf("repos/%s/%s/deployments?sha=%s&per_page=100&page=%d", owner, name, url.QueryEscape(hash), page)
		if err := a.api(ctx, endpoint, &payload); err != nil {
			if isNotFound(err) {
				break
			}
			return nil, err
		}
		if len(payload) == 0 {
			break
		}
		deployments = append(deployments, payload...)
		if len(payload) < 100 {
			break
		}
	}
	return deployments, nil
}

func (a *Adapter) deploymentStatus(ctx context.Context, owner, name string, deploymentID int) (deploymentStatusPayload, error) {
	var payload []deploymentStatusPayload
	endpoint := fmt.Sprintf("repos/%s/%s/deployments/%d/statuses?per_page=100&page=1", owner, name, deploymentID)
	if err := a.api(ctx, endpoint, &payload); err != nil {
		if isNotFound(err) {
			return deploymentStatusPayload{}, nil
		}
		return deploymentStatusPayload{}, err
	}
	if len(payload) == 0 {
		return deploymentStatusPayload{}, nil
	}
	return payload[0], nil
}

func (a *Adapter) api(ctx context.Context, endpoint string, destination any) error {
	output, err := a.runGH(ctx, "api", "-H", "Accept: application/vnd.github+json", endpoint)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(output), destination); err != nil {
		return fmt.Errorf("parse github api response for %s: %w", endpoint, err)
	}
	return nil
}

func (a *Adapter) runGH(ctx context.Context, args ...string) (string, error) {
	if a.run != nil {
		return a.run(ctx, args...)
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("gh executable not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("gh %s failed: %s", strings.Join(args, " "), message)
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
	if !strings.HasPrefix(strings.ToLower(repoRoot), "github:") {
		return "", errors.New("not a github repository target")
	}
	_, _, err := parseOwnerRepo(repoRoot)
	if err != nil {
		return "", err
	}
	return repoRoot, nil
}

func parseOwnerRepo(repoRoot string) (string, string, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	repoRoot = strings.TrimPrefix(repoRoot, "github:")
	repoRoot = strings.TrimPrefix(repoRoot, "GITHUB:")
	parts := strings.Split(repoRoot, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("github repository target must be in owner/name form")
	}

	owner := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(strings.TrimSuffix(parts[1], ".git"))
	if owner == "" || name == "" {
		return "", "", fmt.Errorf("github repository target must be in owner/name form")
	}
	return owner, name, nil
}

func commitSubject(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	line, _, _ := strings.Cut(message, "\n")
	return strings.TrimSpace(line)
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

func mapPullRequestEvidence(values map[string]scm.PullRequestEvidence) []scm.PullRequestEvidence {
	evidence := make([]scm.PullRequestEvidence, 0, len(values))
	for _, item := range values {
		evidence = append(evidence, item)
	}
	sort.SliceStable(evidence, func(i, j int) bool {
		return evidence[i].ID < evidence[j].ID
	})
	return evidence
}

func mapDeploymentEvidence(values map[string]scm.DeploymentEvidence) []scm.DeploymentEvidence {
	evidence := make([]scm.DeploymentEvidence, 0, len(values))
	for _, item := range values {
		evidence = append(evidence, item)
	}
	sort.SliceStable(evidence, func(i, j int) bool {
		return evidence[i].ID < evidence[j].ID
	})
	return evidence
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "http 404") || strings.Contains(message, "404 not found")
}
