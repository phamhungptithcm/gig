package cli

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"gig/internal/scm"
	"gig/internal/sourcecontrol"
)

func (a *App) resolveLoginProvider(ctx context.Context, reader *bufio.Reader, providerValue string) (scm.Type, bool, error) {
	if strings.TrimSpace(providerValue) != "" {
		provider, err := sourcecontrol.ParseProvider(providerValue)
		return provider, false, err
	}

	if provider, ok := a.inferLoginProviderFromCurrentCheckout(ctx); ok {
		return provider, true, nil
	}

	providers := []scm.Type{
		scm.TypeGitHub,
		scm.TypeGitLab,
		scm.TypeBitbucket,
		scm.TypeAzureDevOps,
		scm.TypeRemoteSVN,
	}
	items := make([]pickerItem, 0, len(providers))
	for _, provider := range providers {
		items = append(items, pickerItem{
			Value:    loginProviderValue(provider),
			Title:    sourcecontrol.ProviderLabel(provider),
			Subtitle: loginProviderSubtitle(provider),
			Keywords: loginProviderKeywords(provider),
		})
	}

	selected, err := a.runPicker(reader, "Pick a provider:", items)
	if err != nil {
		return "", false, err
	}
	provider, err := sourcecontrol.ParseProvider(selected.Value)
	return provider, false, err
}

func (a *App) inferLoginProviderFromCurrentCheckout(ctx context.Context) (scm.Type, bool) {
	repository, ok, err := a.scanner.Current(ctx, ".")
	if err != nil || !ok {
		return "", false
	}

	switch repository.Type {
	case scm.TypeSVN:
		return scm.TypeRemoteSVN, true
	case scm.TypeGit:
		remoteRepository, ok := inferRepositoryFromGitRemote(ctx, repository.Root)
		if !ok {
			return "", false
		}
		return remoteRepository.Type, true
	default:
		if repository.Type.IsRemote() {
			return repository.Type, true
		}
	}
	return "", false
}

func inferProviderFromGitRemote(ctx context.Context, repoRoot string) (scm.Type, bool) {
	repository, ok := inferRepositoryFromGitRemote(ctx, repoRoot)
	if !ok {
		return "", false
	}
	return repository.Type, true
}

func inferRepositoryFromGitRemote(ctx context.Context, repoRoot string) (scm.Repository, bool) {
	output, err := exec.CommandContext(ctx, "git", "-C", repoRoot, "config", "--get", "remote.origin.url").Output()
	if err != nil {
		return scm.Repository{}, false
	}
	return inferRepositoryFromGitRemoteURL(string(output))
}

func inferProviderFromGitRemoteURL(remoteURL string) (scm.Type, bool) {
	repository, ok := inferRepositoryFromGitRemoteURL(remoteURL)
	if !ok {
		return "", false
	}
	return repository.Type, true
}

func inferRepositoryFromGitRemoteURL(remoteURL string) (scm.Repository, bool) {
	normalized := strings.ToLower(strings.TrimSpace(remoteURL))
	normalized = strings.TrimSuffix(normalized, ".git")
	if normalized == "" {
		return scm.Repository{}, false
	}

	if repository, err := sourcecontrol.ParseRepositoryTarget(remoteURL); err == nil && repository.Type.IsRemote() {
		return repository, true
	}
	return scm.Repository{}, false
}

func (a *App) inferRemoteRepositoryFromCurrentCheckout(ctx context.Context) (scm.Repository, scm.Repository, bool) {
	localRepository, ok, err := a.scanner.Current(ctx, ".")
	if err != nil || !ok {
		return scm.Repository{}, scm.Repository{}, false
	}

	switch localRepository.Type {
	case scm.TypeGit:
		remoteRepository, ok := inferRepositoryFromGitRemote(ctx, localRepository.Root)
		if !ok {
			return scm.Repository{}, scm.Repository{}, false
		}
		remoteRepository.CurrentBranch = localRepository.CurrentBranch
		return remoteRepository, localRepository, true
	default:
		return scm.Repository{}, scm.Repository{}, false
	}
}

func formatDetectedLoginProvider(provider scm.Type) string {
	return fmt.Sprintf("Detected %s from current checkout.", sourcecontrol.ProviderLabel(provider))
}

func loginProviderValue(provider scm.Type) string {
	if provider == scm.TypeRemoteSVN {
		return "svn"
	}
	return string(provider)
}

func loginProviderSubtitle(provider scm.Type) string {
	switch provider {
	case scm.TypeGitHub:
		return "Recommended first-run path for remote-first release audits."
	case scm.TypeGitLab:
		return "Use GitLab repositories and merge-request evidence without cloning first."
	case scm.TypeBitbucket:
		return "Use Bitbucket repositories with interactive API-token login."
	case scm.TypeAzureDevOps:
		return "Use Azure DevOps repositories, pull requests, and work-item evidence."
	case scm.TypeRemoteSVN:
		return "Use svn: repository targets or a local checkout for SVN and Mendix releases."
	default:
		return ""
	}
}

func loginProviderKeywords(provider scm.Type) []string {
	keywords := []string{string(provider), strings.ToLower(sourcecontrol.ProviderLabel(provider))}
	switch provider {
	case scm.TypeGitHub:
		keywords = append(keywords, "gh")
	case scm.TypeGitLab:
		keywords = append(keywords, "glab")
	case scm.TypeAzureDevOps:
		keywords = append(keywords, "ado", "azdo")
	case scm.TypeRemoteSVN:
		keywords = append(keywords, "svn", "subversion", string(scm.TypeSVN))
	}
	return keywords
}
