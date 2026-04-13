package sourcecontrol

import (
	"context"
	"fmt"
	"io"
	"strings"

	"gig/internal/scm"
	githubscm "gig/internal/scm/github"
	gitlabscm "gig/internal/scm/gitlab"
)

type adapterLookup interface {
	For(repoType scm.Type) (scm.Adapter, bool)
}

func ParseProvider(raw string) (scm.Type, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "github", "gh":
		return scm.TypeGitHub, nil
	case "gitlab", "glab":
		return scm.TypeGitLab, nil
	case "bitbucket":
		return scm.TypeBitbucket, nil
	case "azure-devops", "azuredevops", "ado", "azdo":
		return scm.TypeAzureDevOps, nil
	case "svn", "subversion":
		return scm.TypeSVN, nil
	default:
		return "", fmt.Errorf("provider %q is not recognized", raw)
	}
}

func ProviderLabel(provider scm.Type) string {
	switch provider {
	case scm.TypeGitHub:
		return "GitHub"
	case scm.TypeGitLab:
		return "GitLab"
	case scm.TypeBitbucket:
		return "Bitbucket"
	case scm.TypeAzureDevOps:
		return "Azure DevOps"
	case scm.TypeSVN:
		return "SVN"
	case scm.TypeGit:
		return "Git"
	default:
		return string(provider)
	}
}

func SupportsRemoteAudit(provider scm.Type) bool {
	switch provider {
	case scm.TypeGitHub, scm.TypeGitLab:
		return true
	default:
		return false
	}
}

func ValidateRemoteAuditSupport(repositories []scm.Repository, adapters adapterLookup) error {
	for _, repository := range repositories {
		if !repository.Type.IsRemote() {
			continue
		}
		if !SupportsRemoteAudit(repository.Type) {
			return fmt.Errorf("%s repository targets are recognized, but remote audit is not implemented yet", ProviderLabel(repository.Type))
		}
		if _, ok := adapters.For(repository.Type); !ok {
			return fmt.Errorf("%s repository targets are configured, but no adapter is registered", ProviderLabel(repository.Type))
		}
	}

	return nil
}

func EnsureAccess(ctx context.Context, repositories []scm.Repository, stdin io.Reader, stdout, stderr io.Writer) error {
	required := map[scm.Type]struct{}{}
	for _, repository := range repositories {
		if !repository.Type.IsRemote() {
			continue
		}
		required[repository.Type] = struct{}{}
	}

	for provider := range required {
		if err := Login(ctx, provider, stdin, stdout, stderr); err != nil {
			return err
		}
	}

	return nil
}

func Login(ctx context.Context, provider scm.Type, stdin io.Reader, stdout, stderr io.Writer) error {
	switch provider {
	case scm.TypeGitHub:
		return githubscm.NewSession(stdin, stdout, stderr).EnsureAuthenticated(ctx)
	case scm.TypeGitLab:
		return gitlabscm.NewSession(stdin, stdout, stderr).EnsureAuthenticated(ctx)
	case scm.TypeBitbucket, scm.TypeAzureDevOps, scm.TypeSVN:
		return fmt.Errorf("%s login is not implemented yet", ProviderLabel(provider))
	default:
		return fmt.Errorf("provider %q is not supported", provider)
	}
}
