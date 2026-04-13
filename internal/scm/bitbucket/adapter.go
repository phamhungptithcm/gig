package bitbucket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"gig/internal/scm"
	"gig/internal/ticket"
)

type credentialSource func(context.Context) (credentials, error)

type Adapter struct {
	parser      ticket.Parser
	client      *http.Client
	baseURL     string
	pageSize    int
	credentials credentialSource
}

type repositoryPayload struct {
	MainBranch struct {
		Name string `json:"name"`
	} `json:"mainbranch"`
}

type branchPayload struct {
	Name string `json:"name"`
}

type refsPayload struct {
	Values []branchPayload `json:"values"`
	Next   string          `json:"next"`
}

type branchRestrictionPayload struct {
	Pattern         string `json:"pattern"`
	BranchMatchKind string `json:"branch_match_kind"`
	BranchType      string `json:"branch_type"`
}

type branchRestrictionsPayload struct {
	Values []branchRestrictionPayload `json:"values"`
	Next   string                     `json:"next"`
}

type effectiveBranchingModelPayload struct {
	Development struct {
		Branch branchPayload `json:"branch"`
	} `json:"development"`
	Production struct {
		Branch branchPayload `json:"branch"`
	} `json:"production"`
}

type commitListItem struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Summary struct {
		Raw string `json:"raw"`
	} `json:"summary"`
}

type commitsPayload struct {
	Values []commitListItem `json:"values"`
	Next   string           `json:"next"`
}

type commitPayload struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Summary struct {
		Raw string `json:"raw"`
	} `json:"summary"`
}

type diffstatPayload struct {
	Values []diffstatItem `json:"values"`
	Next   string         `json:"next"`
}

type diffstatItem struct {
	New diffstatPath `json:"new"`
	Old diffstatPath `json:"old"`
}

type diffstatPath struct {
	Path string `json:"path"`
}

type pullRequestPayload struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	State string `json:"state"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	Source struct {
		Branch branchPayload `json:"branch"`
	} `json:"source"`
	Destination struct {
		Branch branchPayload `json:"branch"`
	} `json:"destination"`
}

type pullRequestsPayload struct {
	Values []pullRequestPayload `json:"values"`
	Next   string               `json:"next"`
}

type deploymentsPayload struct {
	Values []map[string]any `json:"values"`
	Next   string           `json:"next"`
}

func NewAdapter(parser ticket.Parser) *Adapter {
	return &Adapter{
		parser:   parser,
		client:   http.DefaultClient,
		baseURL:  resolveAPIBaseURL(),
		pageSize: 100,
	}
}

func (a *Adapter) Type() scm.Type {
	return scm.TypeBitbucket
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
	return strings.TrimSpace(repository.MainBranch.Name), nil
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
	workspace, repo, err := parseWorkspaceRepo(repoRoot)
	if err != nil {
		return false, err
	}

	endpoint := fmt.Sprintf("/repositories/%s/%s/refs/branches/%s",
		url.PathEscape(workspace),
		url.PathEscape(repo),
		url.PathEscape(strings.TrimSpace(ref)),
	)

	var branch branchPayload
	if err := a.api(ctx, endpoint, &branch); err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return strings.TrimSpace(branch.Name) != "", nil
}

func (a *Adapter) ProtectedBranches(ctx context.Context, repoRoot string) ([]string, error) {
	workspace, repo, err := parseWorkspaceRepo(repoRoot)
	if err != nil {
		return nil, err
	}

	repository, err := a.repository(ctx, repoRoot)
	if err != nil {
		return nil, err
	}
	mainBranch := strings.TrimSpace(repository.MainBranch.Name)

	selected := make([]string, 0, 4)
	seen := map[string]struct{}{}
	add := func(branch string) {
		branch = strings.TrimSpace(branch)
		if branch == "" {
			return
		}
		if _, ok := seen[branch]; ok {
			return
		}
		seen[branch] = struct{}{}
		selected = append(selected, branch)
	}

	for page := 1; page <= 5; page++ {
		endpoint := fmt.Sprintf("/repositories/%s/%s/branch-restrictions?pagelen=%d&page=%d",
			url.PathEscape(workspace),
			url.PathEscape(repo),
			a.pageSize,
			page,
		)

		var payload branchRestrictionsPayload
		if err := a.api(ctx, endpoint, &payload); err != nil {
			if isNotFound(err) {
				break
			}
			return nil, err
		}
		if len(payload.Values) == 0 {
			break
		}

		needsModel := false
		for _, item := range payload.Values {
			if strings.TrimSpace(item.BranchMatchKind) == "branching_model" {
				needsModel = true
				break
			}
		}

		var (
			model    effectiveBranchingModelPayload
			modelErr error
		)
		if needsModel {
			model, modelErr = a.effectiveBranchingModel(ctx, repoRoot)
		}

		for _, item := range payload.Values {
			switch strings.TrimSpace(item.BranchMatchKind) {
			case "branching_model":
				if modelErr != nil {
					continue
				}
				switch strings.TrimSpace(item.BranchType) {
				case "development":
					add(model.Development.Branch.Name)
				case "production":
					add(model.Production.Branch.Name)
				}
			default:
				pattern := strings.TrimSpace(item.Pattern)
				if pattern == "" || strings.ContainsAny(pattern, "*?[]{}") {
					continue
				}
				add(pattern)
			}
		}

		if payload.Next == "" {
			break
		}
	}

	if len(selected) != 0 {
		return scm.SelectRemoteAuditBranches(append(selected, mainBranch), mainBranch), nil
	}

	allBranches := make([]string, 0)
	for page := 1; page <= 5; page++ {
		endpoint := fmt.Sprintf("/repositories/%s/%s/refs/branches?pagelen=%d&page=%d",
			url.PathEscape(workspace),
			url.PathEscape(repo),
			a.pageSize,
			page,
		)

		var payload refsPayload
		if err := a.api(ctx, endpoint, &payload); err != nil {
			return nil, err
		}
		if len(payload.Values) == 0 {
			break
		}
		for _, branch := range payload.Values {
			allBranches = append(allBranches, branch.Name)
		}
		if payload.Next == "" {
			break
		}
	}

	allBranches = append(allBranches, mainBranch)
	return scm.SelectRemoteAuditBranches(allBranches, mainBranch), nil
}

func (a *Adapter) CommitFiles(ctx context.Context, repoRoot string, hashes []string) (map[string][]string, error) {
	workspace, repo, err := parseWorkspaceRepo(repoRoot)
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
			endpoint := fmt.Sprintf("/repositories/%s/%s/diffstat/%s?pagelen=%d&page=%d",
				url.PathEscape(workspace),
				url.PathEscape(repo),
				url.PathEscape(hash),
				a.pageSize,
				page,
			)

			var payload diffstatPayload
			if err := a.api(ctx, endpoint, &payload); err != nil {
				return nil, err
			}
			if len(payload.Values) == 0 {
				break
			}
			for _, item := range payload.Values {
				for _, candidate := range []string{item.New.Path, item.Old.Path} {
					filename := strings.TrimSpace(candidate)
					if filename == "" {
						continue
					}
					if _, ok := fileSeen[filename]; ok {
						continue
					}
					fileSeen[filename] = struct{}{}
					files = append(files, filename)
				}
			}
			if payload.Next == "" {
				break
			}
		}
		sort.Strings(files)
		filesByCommit[hash] = files
	}

	return filesByCommit, nil
}

func (a *Adapter) CommitMessages(ctx context.Context, repoRoot string, hashes []string) (map[string]string, error) {
	workspace, repo, err := parseWorkspaceRepo(repoRoot)
	if err != nil {
		return nil, err
	}

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

		endpoint := fmt.Sprintf("/repositories/%s/%s/commit/%s",
			url.PathEscape(workspace),
			url.PathEscape(repo),
			url.PathEscape(hash),
		)

		var payload commitPayload
		if err := a.api(ctx, endpoint, &payload); err != nil {
			return nil, err
		}
		message := strings.TrimRight(strings.TrimSpace(payload.Message), "\n")
		if message == "" {
			message = strings.TrimRight(strings.TrimSpace(payload.Summary.Raw), "\n")
		}
		messagesByCommit[hash] = message
	}

	return messagesByCommit, nil
}

func (a *Adapter) ProviderEvidence(ctx context.Context, repoRoot string, query scm.EvidenceQuery) (scm.ProviderEvidence, error) {
	workspace, repo, err := parseWorkspaceRepo(repoRoot)
	if err != nil {
		return scm.ProviderEvidence{}, err
	}

	pullRequestsByID := map[string]scm.PullRequestEvidence{}
	for _, commit := range query.Commits {
		hash := strings.TrimSpace(commit.Hash)
		if hash == "" {
			continue
		}

		pullRequests, err := a.pullRequestsForCommit(ctx, workspace, repo, hash)
		if err != nil {
			return scm.ProviderEvidence{}, err
		}
		for _, item := range pullRequests {
			id := fmt.Sprintf("#%d", item.ID)
			pullRequestsByID[id] = scm.PullRequestEvidence{
				ID:           id,
				Title:        strings.TrimSpace(item.Title),
				State:        normalizeBitbucketState(item.State),
				SourceBranch: strings.TrimSpace(item.Source.Branch.Name),
				TargetBranch: strings.TrimSpace(item.Destination.Branch.Name),
				URL:          strings.TrimSpace(item.Links.HTML.Href),
				CommitHash:   hash,
			}
		}
	}

	commitHashes := map[string]struct{}{}
	for _, commit := range query.Commits {
		hash := strings.TrimSpace(commit.Hash)
		if hash == "" {
			continue
		}
		commitHashes[hash] = struct{}{}
	}

	deployments, err := a.deploymentsForCommits(ctx, workspace, repo, commitHashes)
	if err != nil {
		return scm.ProviderEvidence{}, err
	}

	return scm.ProviderEvidence{
		PullRequests: mapPullRequestEvidence(pullRequestsByID),
		Deployments:  deployments,
	}, nil
}

func (a *Adapter) searchBranchCommits(ctx context.Context, repoRoot, branch, ticketID string) ([]scm.Commit, error) {
	workspace, repo, err := parseWorkspaceRepo(repoRoot)
	if err != nil {
		return nil, err
	}

	commits := make([]scm.Commit, 0)
	seen := map[string]struct{}{}
	for page := 1; page <= 5; page++ {
		endpoint := fmt.Sprintf("/repositories/%s/%s/commits/%s?pagelen=%d&page=%d",
			url.PathEscape(workspace),
			url.PathEscape(repo),
			url.PathEscape(strings.TrimSpace(branch)),
			a.pageSize,
			page,
		)

		var payload commitsPayload
		if err := a.api(ctx, endpoint, &payload); err != nil {
			return nil, err
		}
		if len(payload.Values) == 0 {
			break
		}

		for _, item := range payload.Values {
			message := commitMessage(item)
			if !a.parser.Matches(ticketID, message) {
				continue
			}
			hash := strings.TrimSpace(item.Hash)
			if hash == "" {
				continue
			}
			if _, ok := seen[hash]; ok {
				continue
			}
			seen[hash] = struct{}{}
			commits = append(commits, scm.Commit{
				Hash:     hash,
				Subject:  commitSubject(message),
				Branches: []string{branch},
			})
		}

		if payload.Next == "" {
			break
		}
	}

	return commits, nil
}

func (a *Adapter) repository(ctx context.Context, repoRoot string) (repositoryPayload, error) {
	workspace, repo, err := parseWorkspaceRepo(repoRoot)
	if err != nil {
		return repositoryPayload{}, err
	}

	endpoint := fmt.Sprintf("/repositories/%s/%s",
		url.PathEscape(workspace),
		url.PathEscape(repo),
	)

	var payload repositoryPayload
	if err := a.api(ctx, endpoint, &payload); err != nil {
		return repositoryPayload{}, err
	}
	return payload, nil
}

func (a *Adapter) pullRequestsForCommit(ctx context.Context, workspace, repo, hash string) ([]pullRequestPayload, error) {
	results := make([]pullRequestPayload, 0)
	for page := 1; page <= 5; page++ {
		endpoint := fmt.Sprintf("/repositories/%s/%s/commit/%s/pullrequests?pagelen=%d&page=%d",
			url.PathEscape(workspace),
			url.PathEscape(repo),
			url.PathEscape(hash),
			a.pageSize,
			page,
		)

		var payload pullRequestsPayload
		if err := a.api(ctx, endpoint, &payload); err != nil {
			if isNotFound(err) {
				break
			}
			return nil, err
		}
		if len(payload.Values) == 0 {
			break
		}
		results = append(results, payload.Values...)
		if payload.Next == "" {
			break
		}
	}

	return results, nil
}

func (a *Adapter) deploymentsForCommits(ctx context.Context, workspace, repo string, hashes map[string]struct{}) ([]scm.DeploymentEvidence, error) {
	evidenceByID := map[string]scm.DeploymentEvidence{}
	for page := 1; page <= 5; page++ {
		endpoint := fmt.Sprintf("/repositories/%s/%s/deployments?pagelen=%d&page=%d",
			url.PathEscape(workspace),
			url.PathEscape(repo),
			a.pageSize,
			page,
		)

		var payload deploymentsPayload
		if err := a.api(ctx, endpoint, &payload); err != nil {
			if isNotFound(err) {
				break
			}
			return nil, err
		}
		if len(payload.Values) == 0 {
			break
		}

		for _, item := range payload.Values {
			hash := strings.TrimSpace(nestedString(item, "release", "commit", "hash"))
			if _, ok := hashes[hash]; !ok {
				continue
			}
			id := strings.TrimSpace(firstNonEmpty(
				nestedString(item, "uuid"),
				nestedString(item, "deployment_uuid"),
			))
			if id == "" {
				continue
			}
			evidenceByID[id] = scm.DeploymentEvidence{
				ID:          id,
				Environment: firstNonEmpty(nestedString(item, "environment", "name"), nestedString(item, "environment", "type")),
				State:       firstNonEmpty(nestedString(item, "state", "name"), nestedString(item, "state", "type")),
				Ref:         firstNonEmpty(nestedString(item, "release", "ref_name"), nestedString(item, "release", "name")),
				URL:         firstNonEmpty(nestedString(item, "links", "html", "href"), nestedString(item, "links", "self", "href")),
				CommitHash:  hash,
			}
		}

		if payload.Next == "" {
			break
		}
	}

	return mapDeploymentEvidence(evidenceByID), nil
}

func (a *Adapter) effectiveBranchingModel(ctx context.Context, repoRoot string) (effectiveBranchingModelPayload, error) {
	workspace, repo, err := parseWorkspaceRepo(repoRoot)
	if err != nil {
		return effectiveBranchingModelPayload{}, err
	}

	endpoint := fmt.Sprintf("/repositories/%s/%s/effective-branching-model",
		url.PathEscape(workspace),
		url.PathEscape(repo),
	)

	var payload effectiveBranchingModelPayload
	if err := a.api(ctx, endpoint, &payload); err != nil {
		return effectiveBranchingModelPayload{}, err
	}
	return payload, nil
}

func (a *Adapter) api(ctx context.Context, endpoint string, destination any) error {
	creds, err := a.resolveCredentials(ctx)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(a.baseURL, "/")+endpoint, nil)
	if err != nil {
		return err
	}
	request.SetBasicAuth(creds.Email, creds.APIToken)
	request.Header.Set("Accept", "application/json")

	client := a.client
	if client == nil {
		client = http.DefaultClient
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if response.StatusCode == http.StatusNotFound {
		return notFoundError{message: string(body)}
	}
	if response.StatusCode >= 400 {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = response.Status
		}
		return fmt.Errorf("bitbucket api %s failed: %s", endpoint, message)
	}

	if err := json.Unmarshal(body, destination); err != nil {
		return fmt.Errorf("parse bitbucket api response for %s: %w", endpoint, err)
	}
	return nil
}

func (a *Adapter) resolveCredentials(ctx context.Context) (credentials, error) {
	if a.credentials != nil {
		return a.credentials(ctx)
	}

	return NewSession(nil, nil, nil).Credentials(ctx)
}

type notFoundError struct {
	message string
}

func (e notFoundError) Error() string {
	if strings.TrimSpace(e.message) == "" {
		return "resource not found"
	}
	return strings.TrimSpace(e.message)
}

func isNotFound(err error) bool {
	var target notFoundError
	return errors.As(err, &target)
}

func parseRepositoryRoot(repoRoot string) (string, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if !strings.HasPrefix(strings.ToLower(repoRoot), "bitbucket:") {
		return "", errors.New("not a bitbucket repository target")
	}
	_, _, err := parseWorkspaceRepo(repoRoot)
	if err != nil {
		return "", err
	}
	return repoRoot, nil
}

func parseWorkspaceRepo(repoRoot string) (string, string, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	repoRoot = strings.TrimPrefix(repoRoot, "bitbucket:")
	repoRoot = strings.TrimPrefix(repoRoot, "BITBUCKET:")
	parts := strings.Split(repoRoot, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("bitbucket repository target must be in workspace/repo form")
	}
	workspace := strings.TrimSpace(parts[0])
	repo := strings.TrimSpace(parts[1])
	if workspace == "" || repo == "" {
		return "", "", fmt.Errorf("bitbucket repository target must be in workspace/repo form")
	}
	return workspace, repo, nil
}

func commitMessage(item commitListItem) string {
	message := strings.TrimSpace(item.Message)
	if message != "" {
		return message
	}
	return strings.TrimSpace(item.Summary.Raw)
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

func normalizeBitbucketState(state string) string {
	state = strings.TrimSpace(strings.ToLower(state))
	switch state {
	case "fulfilled":
		return "merged"
	case "declined":
		return "closed"
	default:
		return state
	}
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

func nestedString(value map[string]any, path ...string) string {
	current := any(value)
	for _, segment := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current, ok = object[segment]
		if !ok {
			return ""
		}
	}

	switch typed := current.(type) {
	case string:
		return typed
	default:
		return ""
	}
}
