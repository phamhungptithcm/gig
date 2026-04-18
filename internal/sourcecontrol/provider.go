package sourcecontrol

import (
	"context"
	"fmt"
	"io"
	"strings"

	"gig/internal/diagnostics"
	"gig/internal/scm"
	azuredevopsscm "gig/internal/scm/azuredevops"
	bitbucketscm "gig/internal/scm/bitbucket"
	githubscm "gig/internal/scm/github"
	gitlabscm "gig/internal/scm/gitlab"
	svnscm "gig/internal/scm/svn"
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
		return scm.TypeRemoteSVN, nil
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
	case scm.TypeRemoteSVN:
		return "SVN"
	case scm.TypeGit:
		return "Git"
	default:
		return string(provider)
	}
}

func SupportsRemoteAudit(provider scm.Type) bool {
	switch provider {
	case scm.TypeGitHub, scm.TypeGitLab, scm.TypeBitbucket, scm.TypeAzureDevOps, scm.TypeRemoteSVN:
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
	required := map[scm.Type][]string{}
	for _, repository := range repositories {
		if !repository.Type.IsRemote() {
			continue
		}
		required[repository.Type] = append(required[repository.Type], repository.Root)
	}

	for provider, roots := range required {
		diagnostics.Emit(ctx, "info", "provider.access", "ensuring provider access", diagnostics.Meta{
			Repo:    strings.Join(roots, ","),
			SCM:     string(provider),
			Details: map[string]any{"repositories": append([]string(nil), roots...)},
		}, nil)
		switch provider {
		case scm.TypeRemoteSVN:
			if err := svnscm.NewSession(stdin, stdout, stderr).EnsureAuthenticated(ctx, roots...); err != nil {
				diagnostics.Emit(ctx, "error", "provider.access", "provider access failed", diagnostics.Meta{
					Repo: strings.Join(roots, ","),
					SCM:  string(provider),
				}, err)
				return err
			}
		default:
			if err := Login(ctx, provider, stdin, stdout, stderr); err != nil {
				diagnostics.Emit(ctx, "error", "provider.access", "provider access failed", diagnostics.Meta{
					Repo: strings.Join(roots, ","),
					SCM:  string(provider),
				}, err)
				return err
			}
		}
		diagnostics.Emit(ctx, "info", "provider.access", "provider access ready", diagnostics.Meta{
			Repo: strings.Join(roots, ","),
			SCM:  string(provider),
		}, nil)
	}

	return nil
}

func Login(ctx context.Context, provider scm.Type, stdin io.Reader, stdout, stderr io.Writer) error {
	diagnostics.Emit(ctx, "info", "provider.login", "provider login check started", diagnostics.Meta{
		SCM: string(provider),
	}, nil)

	var err error
	switch provider {
	case scm.TypeGitHub:
		err = githubscm.NewSession(stdin, stdout, stderr).EnsureAuthenticated(ctx)
	case scm.TypeGitLab:
		err = gitlabscm.NewSession(stdin, stdout, stderr).EnsureAuthenticated(ctx)
	case scm.TypeBitbucket:
		err = bitbucketscm.NewSession(stdin, stdout, stderr).EnsureAuthenticated(ctx)
	case scm.TypeAzureDevOps:
		err = azuredevopsscm.NewSession(stdin, stdout, stderr).EnsureAuthenticated(ctx)
	case scm.TypeSVN, scm.TypeRemoteSVN:
		err = svnscm.NewSession(stdin, stdout, stderr).Login(ctx)
	default:
		err = fmt.Errorf("provider %q is not supported", provider)
	}

	if err != nil {
		diagnostics.Emit(ctx, "error", "provider.login", "provider login check failed", diagnostics.Meta{
			SCM: string(provider),
		}, err)
		return err
	}
	diagnostics.Emit(ctx, "info", "provider.login", "provider login check passed", diagnostics.Meta{
		SCM: string(provider),
	}, nil)
	return nil
}
