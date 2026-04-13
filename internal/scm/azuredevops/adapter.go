package azuredevops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"gig/internal/scm"
	"gig/internal/ticket"
)

const apiVersion = "7.1"

type Adapter struct {
	parser   ticket.Parser
	run      commandRunner
	client   *http.Client
	baseURL  string
	pageSize int
}

type repositoryPayload struct {
	DefaultBranch string `json:"defaultBranch"`
}

type refsPayload struct {
	Value []refPayload `json:"value"`
	Count int          `json:"count"`
}

type refPayload struct {
	Name string `json:"name"`
}

type commitsPayload struct {
	Value []commitPayload `json:"value"`
	Count int             `json:"count"`
}

type commitPayload struct {
	CommitID string `json:"commitId"`
	Comment  string `json:"comment"`
}

type changesPayload struct {
	Changes []changePayload `json:"changes"`
}

type changePayload struct {
	Item struct {
		Path string `json:"path"`
	} `json:"item"`
}

type pullRequestsPayload struct {
	Value []pullRequestPayload `json:"value"`
	Count int                  `json:"count"`
}

type pullRequestPayload struct {
	PullRequestID int    `json:"pullRequestId"`
	Title         string `json:"title"`
	Status        string `json:"status"`
	SourceRefName string `json:"sourceRefName"`
	TargetRefName string `json:"targetRefName"`
	ArtifactID    string `json:"artifactId"`
	URL           string `json:"url"`
}

type deploymentsPayload struct {
	Value []map[string]any `json:"value"`
	Count int              `json:"count"`
}

func NewAdapter(parser ticket.Parser) *Adapter {
	return &Adapter{
		parser:   parser,
		client:   http.DefaultClient,
		baseURL:  defaultBaseURL(),
		pageSize: 100,
	}
}

func (a *Adapter) Type() scm.Type {
	return scm.TypeAzureDevOps
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

	return strings.TrimPrefix(strings.TrimSpace(repository.DefaultBranch), "refs/heads/"), nil
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
	descriptor, err := parseDescriptor(repoRoot)
	if err != nil {
		return false, err
	}

	endpoint := fmt.Sprintf("%s/%s/%s/_apis/git/repositories/%s/refs?filter=heads/%s&api-version=%s",
		strings.TrimRight(a.baseURL, "/"),
		url.PathEscape(descriptor.Organization),
		url.PathEscape(descriptor.Project),
		url.PathEscape(descriptor.Repository),
		url.QueryEscape(strings.TrimSpace(ref)),
		apiVersion,
	)

	var payload refsPayload
	if err := a.api(ctx, endpoint, &payload); err != nil {
		return false, err
	}

	for _, item := range payload.Value {
		if normalizeRefName(item.Name) == strings.TrimSpace(ref) {
			return true, nil
		}
	}

	return false, nil
}

func (a *Adapter) ProtectedBranches(ctx context.Context, repoRoot string) ([]string, error) {
	descriptor, err := parseDescriptor(repoRoot)
	if err != nil {
		return nil, err
	}

	repository, err := a.repository(ctx, repoRoot)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/%s/%s/_apis/git/repositories/%s/refs?filter=heads/&api-version=%s",
		strings.TrimRight(a.baseURL, "/"),
		url.PathEscape(descriptor.Organization),
		url.PathEscape(descriptor.Project),
		url.PathEscape(descriptor.Repository),
		apiVersion,
	)

	var payload refsPayload
	if err := a.api(ctx, endpoint, &payload); err != nil {
		return nil, err
	}

	branches := make([]string, 0, len(payload.Value))
	for _, item := range payload.Value {
		branch := normalizeRefName(item.Name)
		if branch == "" {
			continue
		}
		branches = append(branches, branch)
	}

	return scm.SelectRemoteAuditBranches(branches, normalizeRefName(repository.DefaultBranch)), nil
}

func (a *Adapter) CommitFiles(ctx context.Context, repoRoot string, hashes []string) (map[string][]string, error) {
	descriptor, err := parseDescriptor(repoRoot)
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

		endpoint := fmt.Sprintf("%s/%s/%s/_apis/git/repositories/%s/commits/%s/changes?$top=%d&api-version=%s",
			strings.TrimRight(a.baseURL, "/"),
			url.PathEscape(descriptor.Organization),
			url.PathEscape(descriptor.Project),
			url.PathEscape(descriptor.Repository),
			url.PathEscape(hash),
			a.pageSize,
			apiVersion,
		)

		var payload changesPayload
		if err := a.api(ctx, endpoint, &payload); err != nil {
			return nil, err
		}

		files := make([]string, 0, len(payload.Changes))
		fileSeen := map[string]struct{}{}
		for _, change := range payload.Changes {
			filename := strings.TrimSpace(change.Item.Path)
			if filename == "" {
				continue
			}
			if _, ok := fileSeen[filename]; ok {
				continue
			}
			fileSeen[filename] = struct{}{}
			files = append(files, strings.TrimPrefix(filename, "/"))
		}
		sort.Strings(files)
		filesByCommit[hash] = files
	}

	return filesByCommit, nil
}

func (a *Adapter) CommitMessages(ctx context.Context, repoRoot string, hashes []string) (map[string]string, error) {
	descriptor, err := parseDescriptor(repoRoot)
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

		endpoint := fmt.Sprintf("%s/%s/%s/_apis/git/repositories/%s/commits/%s?api-version=%s",
			strings.TrimRight(a.baseURL, "/"),
			url.PathEscape(descriptor.Organization),
			url.PathEscape(descriptor.Project),
			url.PathEscape(descriptor.Repository),
			url.PathEscape(hash),
			apiVersion,
		)

		var commit commitPayload
		if err := a.api(ctx, endpoint, &commit); err != nil {
			return nil, err
		}
		messagesByCommit[hash] = strings.TrimRight(commit.Comment, "\n")
	}

	return messagesByCommit, nil
}

func (a *Adapter) ProviderEvidence(ctx context.Context, repoRoot string, query scm.EvidenceQuery) (scm.ProviderEvidence, error) {
	descriptor, err := parseDescriptor(repoRoot)
	if err != nil {
		return scm.ProviderEvidence{}, err
	}

	pullRequestsByID := map[string]scm.PullRequestEvidence{}
	seenBranchPairs := map[string]struct{}{}
	for _, commit := range query.Commits {
		for _, branch := range commit.Branches {
			branch = strings.TrimSpace(branch)
			if branch == "" {
				continue
			}
			key := branch + "->"
			if _, ok := seenBranchPairs[key]; ok {
				continue
			}
			seenBranchPairs[key] = struct{}{}

			pullRequests, err := a.pullRequestsForBranch(ctx, descriptor, branch)
			if err != nil {
				return scm.ProviderEvidence{}, err
			}
			for _, item := range pullRequests {
				id := fmt.Sprintf("#%d", item.PullRequestID)
				pullRequestsByID[id] = scm.PullRequestEvidence{
					ID:           id,
					Title:        strings.TrimSpace(item.Title),
					State:        strings.TrimSpace(item.Status),
					SourceBranch: normalizeRefName(item.SourceRefName),
					TargetBranch: normalizeRefName(item.TargetRefName),
					URL:          firstNonEmpty(pullRequestWebURL(descriptor, item.PullRequestID), strings.TrimSpace(item.URL)),
					CommitHash:   firstCommitHashForBranch(query.Commits, branch),
				}
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

	deployments, err := a.deploymentsForCommits(ctx, descriptor, commitHashes)
	if err != nil {
		return scm.ProviderEvidence{}, err
	}

	return scm.ProviderEvidence{
		PullRequests: mapPullRequestEvidence(pullRequestsByID),
		Deployments:  deployments,
	}, nil
}

func (a *Adapter) searchBranchCommits(ctx context.Context, repoRoot, branch, ticketID string) ([]scm.Commit, error) {
	descriptor, err := parseDescriptor(repoRoot)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/%s/%s/_apis/git/repositories/%s/commits?searchCriteria.$top=%d&searchCriteria.itemVersion.versionType=branch&searchCriteria.itemVersion.version=%s&api-version=%s",
		strings.TrimRight(a.baseURL, "/"),
		url.PathEscape(descriptor.Organization),
		url.PathEscape(descriptor.Project),
		url.PathEscape(descriptor.Repository),
		a.pageSize,
		url.QueryEscape(branch),
		apiVersion,
	)

	var payload commitsPayload
	if err := a.api(ctx, endpoint, &payload); err != nil {
		return nil, err
	}

	commits := make([]scm.Commit, 0, len(payload.Value))
	for _, item := range payload.Value {
		message := strings.TrimSpace(item.Comment)
		if !a.parser.Matches(ticketID, message) {
			continue
		}
		commits = append(commits, scm.Commit{
			Hash:     strings.TrimSpace(item.CommitID),
			Subject:  commitSubject(message),
			Branches: []string{branch},
		})
	}

	return commits, nil
}

func (a *Adapter) repository(ctx context.Context, repoRoot string) (repositoryPayload, error) {
	descriptor, err := parseDescriptor(repoRoot)
	if err != nil {
		return repositoryPayload{}, err
	}

	endpoint := fmt.Sprintf("%s/%s/%s/_apis/git/repositories/%s?api-version=%s",
		strings.TrimRight(a.baseURL, "/"),
		url.PathEscape(descriptor.Organization),
		url.PathEscape(descriptor.Project),
		url.PathEscape(descriptor.Repository),
		apiVersion,
	)

	var repository repositoryPayload
	if err := a.api(ctx, endpoint, &repository); err != nil {
		return repositoryPayload{}, err
	}

	return repository, nil
}

func (a *Adapter) pullRequestsForBranch(ctx context.Context, descriptor descriptor, branch string) ([]pullRequestPayload, error) {
	endpoint := fmt.Sprintf("%s/%s/%s/_apis/git/repositories/%s/pullrequests?searchCriteria.sourceRefName=%s&searchCriteria.status=all&$top=100&api-version=%s",
		strings.TrimRight(a.baseURL, "/"),
		url.PathEscape(descriptor.Organization),
		url.PathEscape(descriptor.Project),
		url.PathEscape(descriptor.Repository),
		url.QueryEscape("refs/heads/"+strings.TrimSpace(branch)),
		apiVersion,
	)

	var payload pullRequestsPayload
	if err := a.api(ctx, endpoint, &payload); err != nil {
		return nil, err
	}
	return payload.Value, nil
}

func (a *Adapter) deploymentsForCommits(ctx context.Context, descriptor descriptor, hashes map[string]struct{}) ([]scm.DeploymentEvidence, error) {
	endpoint := fmt.Sprintf("%s/%s/%s/_apis/release/deployments?$top=100&api-version=%s",
		strings.TrimRight(releaseBaseURL(a.baseURL), "/"),
		url.PathEscape(descriptor.Organization),
		url.PathEscape(descriptor.Project),
		apiVersion,
	)

	var payload deploymentsPayload
	if err := a.api(ctx, endpoint, &payload); err != nil {
		return nil, err
	}

	evidenceByID := map[string]scm.DeploymentEvidence{}
	for _, item := range payload.Value {
		hash := strings.TrimSpace(firstNonEmpty(
			nestedString(item, "release", "artifacts", "0", "definitionReference", "version", "name"),
			nestedString(item, "release", "artifacts", "0", "definitionReference", "sourceVersion", "name"),
		))
		if _, ok := hashes[hash]; !ok {
			continue
		}

		id := firstNonEmpty(nestedString(item, "id"), nestedString(item, "deploymentId"))
		if id == "" {
			continue
		}
		evidenceByID[id] = scm.DeploymentEvidence{
			ID:          id,
			Environment: firstNonEmpty(nestedString(item, "releaseEnvironment", "name"), nestedString(item, "environment", "name")),
			State:       firstNonEmpty(nestedString(item, "deploymentStatus"), nestedString(item, "status")),
			Ref:         firstNonEmpty(nestedString(item, "release", "name"), nestedString(item, "release", "sourceBranch")),
			URL:         firstNonEmpty(nestedString(item, "_links", "web", "href"), nestedString(item, "webAccessUri")),
			CommitHash:  hash,
		}
	}

	return mapDeploymentEvidence(evidenceByID), nil
}

func (a *Adapter) api(ctx context.Context, endpoint string, destination any) error {
	token, err := a.accessToken(ctx)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
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

	if response.StatusCode >= 400 {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = response.Status
		}
		return fmt.Errorf("azure devops api %s failed: %s", endpoint, message)
	}

	if err := json.Unmarshal(body, destination); err != nil {
		return fmt.Errorf("parse azure devops api response for %s: %w", endpoint, err)
	}

	return nil
}

func (a *Adapter) accessToken(ctx context.Context) (string, error) {
	session := NewSession(nil, nil, nil)
	session.run = a.run
	return session.AccessToken(ctx)
}

func parseRepositoryRoot(repoRoot string) (string, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if !strings.HasPrefix(strings.ToLower(repoRoot), "azure-devops:") {
		return "", errors.New("not an azure devops repository target")
	}
	_, err := parseDescriptor(repoRoot)
	if err != nil {
		return "", err
	}
	return repoRoot, nil
}

type descriptor struct {
	Organization string
	Project      string
	Repository   string
}

func parseDescriptor(repoRoot string) (descriptor, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	repoRoot = strings.TrimPrefix(repoRoot, "azure-devops:")
	repoRoot = strings.TrimPrefix(repoRoot, "AZURE-DEVOPS:")
	parts := strings.Split(repoRoot, "/")
	if len(parts) != 3 {
		return descriptor{}, fmt.Errorf("azure devops repository target must be in org/project/repo form")
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return descriptor{}, fmt.Errorf("azure devops repository target must be in org/project/repo form")
		}
	}
	return descriptor{
		Organization: strings.TrimSpace(parts[0]),
		Project:      strings.TrimSpace(parts[1]),
		Repository:   strings.TrimSpace(parts[2]),
	}, nil
}

func commitSubject(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	line, _, _ := strings.Cut(message, "\n")
	return strings.TrimSpace(line)
}

func normalizeRefName(ref string) string {
	ref = strings.TrimSpace(ref)
	ref = strings.TrimPrefix(ref, "refs/heads/")
	return ref
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

func firstCommitHashForBranch(commits []scm.Commit, branch string) string {
	for _, commit := range commits {
		for _, seenBranch := range commit.Branches {
			if strings.EqualFold(strings.TrimSpace(seenBranch), strings.TrimSpace(branch)) {
				return strings.TrimSpace(commit.Hash)
			}
		}
	}
	return ""
}

func nestedString(value map[string]any, path ...string) string {
	current := any(value)
	for _, segment := range path {
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[segment]
			if !ok {
				return ""
			}
			current = next
		case []any:
			if segment != "0" || len(typed) == 0 {
				return ""
			}
			current = typed[0]
		default:
			return ""
		}
	}

	switch typed := current.(type) {
	case string:
		return typed
	case float64:
		return fmt.Sprintf("%.0f", typed)
	default:
		return ""
	}
}

func releaseBaseURL(baseURL string) string {
	if value := strings.TrimSpace(os.Getenv("GIG_AZURE_DEVOPS_RELEASE_BASE_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return strings.TrimRight(baseURL, "/")
}

func pullRequestWebURL(descriptor descriptor, pullRequestID int) string {
	if strings.HasPrefix(strings.TrimSpace(os.Getenv("GIG_AZURE_DEVOPS_BASE_URL")), "http://") ||
		strings.HasPrefix(strings.TrimSpace(os.Getenv("GIG_AZURE_DEVOPS_BASE_URL")), "https://") {
		base := strings.TrimRight(os.Getenv("GIG_AZURE_DEVOPS_BASE_URL"), "/")
		return fmt.Sprintf("%s/%s/%s/_git/%s/pullrequest/%d",
			base,
			url.PathEscape(descriptor.Organization),
			url.PathEscape(descriptor.Project),
			url.PathEscape(descriptor.Repository),
			pullRequestID,
		)
	}

	return fmt.Sprintf("https://dev.azure.com/%s/%s/_git/%s/pullrequest/%d",
		url.PathEscape(descriptor.Organization),
		url.PathEscape(descriptor.Project),
		url.PathEscape(descriptor.Repository),
		pullRequestID,
	)
}

func defaultBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("GIG_AZURE_DEVOPS_BASE_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return "https://dev.azure.com"
}
