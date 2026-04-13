package sourcecontrol

import (
	"context"
	"fmt"
	"io"

	"gig/internal/scm"
	azuredevopsscm "gig/internal/scm/azuredevops"
	bitbucketscm "gig/internal/scm/bitbucket"
	githubscm "gig/internal/scm/github"
	gitlabscm "gig/internal/scm/gitlab"
)

type RepositoryDiscoveryOptions struct {
	Organization string
	Project      string
}

func DiscoverableProviders() []scm.Type {
	return []scm.Type{
		scm.TypeGitHub,
		scm.TypeGitLab,
		scm.TypeBitbucket,
		scm.TypeAzureDevOps,
	}
}

func SupportsRepositoryDiscovery(provider scm.Type) bool {
	switch provider {
	case scm.TypeGitHub, scm.TypeGitLab, scm.TypeBitbucket, scm.TypeAzureDevOps:
		return true
	default:
		return false
	}
}

func DiscoverRepositories(ctx context.Context, provider scm.Type, options RepositoryDiscoveryOptions, stdin io.Reader, stdout, stderr io.Writer) ([]scm.Repository, error) {
	switch provider {
	case scm.TypeGitHub:
		session := githubscm.NewSession(stdin, stdout, stderr)
		if err := session.EnsureAuthenticated(ctx); err != nil {
			return nil, err
		}
		return session.ListRepositories(ctx, options.Organization)
	case scm.TypeGitLab:
		session := gitlabscm.NewSession(stdin, stdout, stderr)
		if err := session.EnsureAuthenticated(ctx); err != nil {
			return nil, err
		}
		return session.ListRepositories(ctx, options.Organization)
	case scm.TypeBitbucket:
		session := bitbucketscm.NewSession(stdin, stdout, stderr)
		if err := session.EnsureAuthenticated(ctx); err != nil {
			return nil, err
		}
		return session.ListRepositories(ctx, options.Organization)
	case scm.TypeAzureDevOps:
		session := azuredevopsscm.NewSession(stdin, stdout, stderr)
		if err := session.EnsureAuthenticated(ctx); err != nil {
			return nil, err
		}
		return session.ListRepositories(ctx, options.Organization, options.Project)
	default:
		return nil, fmt.Errorf("%s repository discovery is not supported", ProviderLabel(provider))
	}
}
