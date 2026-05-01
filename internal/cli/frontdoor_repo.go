package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"gig/internal/scm"
	"gig/internal/sourcecontrol"
	"gig/internal/workarea"
)

type frontDoorRepositoryCandidate struct {
	Repository scm.Repository
	Source     string
	Recent     bool
	Current    bool
}

func (a *App) resolveFrontDoorPromptRepository(ctx context.Context, reader *bufio.Reader, store *workarea.Store, query, target string) (scm.Repository, error) {
	target = strings.TrimSpace(target)
	if target != "" {
		repository, err := sourcecontrol.ParseRepositoryTarget(target)
		if err != nil {
			return scm.Repository{}, err
		}
		return repository, nil
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return a.discoverWorkareaRepositoryWithReader(ctx, reader, "", "", "")
	}

	if provider, err := sourcecontrol.ParseProvider(query); err == nil && sourcecontrol.SupportsRepositoryDiscovery(provider) {
		return a.discoverWorkareaRepositoryWithReader(ctx, reader, string(provider), "", "")
	}

	if repository, ok, err := parseFrontDoorSingleRepository(query); err != nil || ok {
		return repository, err
	}

	candidates, err := frontDoorStoredRepositoryCandidates(store)
	if err != nil {
		return scm.Repository{}, err
	}
	matches := filterFrontDoorRepositoryCandidates(candidates, query)
	notes := []string{}
	if len(matches) == 0 {
		var discovered []frontDoorRepositoryCandidate
		discovered, notes, err = a.discoverFrontDoorMatchingRepositories(ctx, reader, query)
		if err != nil {
			return scm.Repository{}, err
		}
		matches = discovered
	}
	if len(matches) == 0 {
		message := fmt.Sprintf("no repository matching %q was found; try `repo`, `gh owner/name`, paste a URL, or `login github`", query)
		if len(notes) > 0 {
			message += ". " + strings.Join(notes, " ")
		}
		return scm.Repository{}, fmt.Errorf("%s", message)
	}
	return a.selectFrontDoorRepositoryCandidate(reader, query, matches)
}

func parseFrontDoorSingleRepository(raw string) (scm.Repository, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return scm.Repository{}, false, nil
	}
	repository, err := sourcecontrol.ParseRepositoryTarget(raw)
	if err != nil {
		return scm.Repository{}, false, nil
	}
	if strings.TrimSpace(repository.Root) == "" {
		return scm.Repository{}, false, nil
	}
	return repository, true, nil
}

func canonicalFrontDoorRepoTarget(raw string) string {
	if repository, ok, err := parseFrontDoorSingleRepository(raw); err == nil && ok {
		return repository.Root
	}
	return strings.TrimSpace(raw)
}

func frontDoorStoredRepositoryCandidates(store *workarea.Store) ([]frontDoorRepositoryCandidate, error) {
	workareas, current, err := store.List()
	if err != nil {
		return nil, err
	}

	candidates := make([]frontDoorRepositoryCandidate, 0, len(workareas))
	seen := map[string]struct{}{}
	add := func(repository scm.Repository, source string, recent, current bool) {
		root := strings.TrimSpace(repository.Root)
		if root == "" {
			return
		}
		key := strings.ToLower(root)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, frontDoorRepositoryCandidate{
			Repository: repository,
			Source:     source,
			Recent:     recent,
			Current:    current,
		})
	}

	for _, definition := range workareas {
		repositories, err := sourcecontrol.ParseRepositoryTargets(definition.RepoTarget)
		if err != nil {
			continue
		}
		for _, repository := range repositories {
			add(repository, "project "+definition.Name, false, strings.EqualFold(definition.Name, current))
		}
	}

	recentRepositories, err := store.RecentRepositories("", 0)
	if err != nil {
		return nil, err
	}
	for _, recent := range recentRepositories {
		repository, err := sourcecontrol.ParseRepositoryTarget(recent.Root)
		if err != nil {
			continue
		}
		if strings.TrimSpace(repository.Name) == "" {
			repository.Name = strings.TrimSpace(recent.Name)
		}
		add(repository, "recent", true, false)
	}

	return candidates, nil
}

func (a *App) discoverFrontDoorMatchingRepositories(ctx context.Context, reader *bufio.Reader, query string) ([]frontDoorRepositoryCandidate, []string, error) {
	candidates := []frontDoorRepositoryCandidate{}
	notes := []string{}
	seen := map[string]struct{}{}
	for _, provider := range sourcecontrol.DiscoverableProviders() {
		label := sourcecontrol.ProviderLabel(provider)
		status := sourcecontrol.CheckProviderStatus(ctx, provider, reader)
		if !status.Ready {
			detail := strings.TrimSpace(status.Detail)
			if detail == "" {
				detail = "not ready"
			}
			notes = append(notes, fmt.Sprintf("%s skipped: %s.", label, detail))
			continue
		}
		options := sourcecontrol.RepositoryDiscoveryOptions{}
		if provider == scm.TypeAzureDevOps {
			organization, err := a.promptForOptionalLine(reader, "Azure DevOps organization (Enter to skip)")
			if err != nil {
				return nil, notes, err
			}
			if strings.TrimSpace(organization) == "" {
				notes = append(notes, "Azure DevOps skipped: organization needed; try ado org/project/repo.")
				continue
			}
			options.Organization = organization
		}
		repositories, err := sourcecontrol.DiscoverRepositories(ctx, provider, options, reader, a.stdout, a.stderr)
		if err != nil {
			notes = append(notes, fmt.Sprintf("%s skipped: %s.", label, err.Error()))
			continue
		}
		matched := 0
		for _, repository := range repositories {
			root := strings.TrimSpace(repository.Root)
			if root == "" || !frontDoorRepositoryMatches(repository, query) {
				continue
			}
			matched++
			key := strings.ToLower(root)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			candidates = append(candidates, frontDoorRepositoryCandidate{
				Repository: repository,
				Source:     sourcecontrol.ProviderLabel(provider),
			})
		}
		if matched == 0 {
			notes = append(notes, fmt.Sprintf("%s searched: no matching repos.", label))
		}
	}
	return candidates, notes, nil
}

func filterFrontDoorRepositoryCandidates(candidates []frontDoorRepositoryCandidate, query string) []frontDoorRepositoryCandidate {
	filtered := make([]frontDoorRepositoryCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if frontDoorRepositoryMatches(candidate.Repository, query) || strings.Contains(strings.ToLower(candidate.Source), strings.ToLower(strings.TrimSpace(query))) {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func frontDoorRepositoryMatches(repository scm.Repository, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	fields := []string{
		repository.Name,
		repository.Root,
		strings.TrimPrefix(repository.Root, string(repository.Type)+":"),
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(strings.TrimSpace(field)), query) {
			return true
		}
	}
	return false
}

func (a *App) selectFrontDoorRepositoryCandidate(reader *bufio.Reader, query string, candidates []frontDoorRepositoryCandidate) (scm.Repository, error) {
	if len(candidates) == 1 {
		return candidates[0].Repository, nil
	}

	items := make([]pickerItem, 0, len(candidates))
	for _, candidate := range candidates {
		subtitle := candidate.Repository.Root
		if strings.TrimSpace(candidate.Source) != "" {
			subtitle = subtitle + "  " + candidate.Source
		}
		items = append(items, pickerItem{
			Value:    candidate.Repository.Root,
			Title:    candidate.Repository.Name,
			Subtitle: subtitle,
			Keywords: []string{candidate.Repository.Name, candidate.Repository.Root, candidate.Source},
			Recent:   candidate.Recent,
			Current:  candidate.Current,
		})
	}

	selected, err := a.runPicker(reader, fmt.Sprintf("Found repositories matching %q:", strings.TrimSpace(query)), items)
	if err != nil {
		return scm.Repository{}, err
	}
	for _, candidate := range candidates {
		if strings.EqualFold(candidate.Repository.Root, selected.Value) {
			return candidate.Repository, nil
		}
	}
	return scm.Repository{}, fmt.Errorf("selected repository %q was not found", selected.Value)
}

func frontDoorProviderAliasCommand(alias string, args []string, pathValue, fromBranch, toBranch string) (frontDoorCommand, error) {
	alias = strings.ToLower(strings.TrimSpace(alias))
	if len(args) == 0 {
		return frontDoorCommand{Action: frontDoorActionRepo, RepoQuery: alias}, nil
	}

	target, err := frontDoorProviderAliasTarget(alias, args[0])
	if err != nil {
		return frontDoorCommand{}, err
	}
	ticketID, extraArgs := frontDoorTicketAndExtraArgs(args[1:])
	if ticketID != "" || frontDoorArgsProvideTicketScope(extraArgs) {
		return frontDoorCommand{
			Action:     frontDoorActionInspect,
			TicketID:   ticketID,
			RepoTarget: target,
			Path:       pathValue,
			FromBranch: fromBranch,
			ToBranch:   toBranch,
			ExtraArgs:  extraArgs,
		}, nil
	}
	return frontDoorCommand{Action: frontDoorActionRepo, RepoTarget: target}, nil
}

func frontDoorProviderAliasTarget(alias, identifier string) (string, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return "", fmt.Errorf("%s requires a repository name or URL", alias)
	}
	switch alias {
	case "gh":
		return "gh:" + identifier, nil
	case "gl":
		return "gl:" + identifier, nil
	case "bb":
		return "bb:" + identifier, nil
	case "ado", "azdo":
		return "ado:" + identifier, nil
	case "svn":
		return identifier, nil
	default:
		return "", fmt.Errorf("provider alias %q is not recognized", alias)
	}
}

func frontDoorRepoQueryTicketExtra(tokens []string) (string, string, []string) {
	if len(tokens) == 0 {
		return "", "", nil
	}
	query := strings.TrimSpace(tokens[0])
	if len(tokens) == 1 {
		return query, "", nil
	}
	ticketID, extraArgs := frontDoorTicketAndExtraArgs(tokens[1:])
	return query, ticketID, extraArgs
}

func inferFrontDoorSaveName(repoTarget string) string {
	repository, err := sourcecontrol.ParseRepositoryTarget(repoTarget)
	if err == nil && strings.TrimSpace(repository.Name) != "" {
		return strings.TrimSpace(repository.Name)
	}
	return inferWorkareaName(repoTarget, "")
}

func (a *App) promptForOptionalLine(reader *bufio.Reader, label string) (string, error) {
	if reader == nil {
		return "", nil
	}
	fmt.Fprintf(a.stdout, "%s: ", label)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
