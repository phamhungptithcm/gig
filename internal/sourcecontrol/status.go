package sourcecontrol

import (
	"context"
	"io"
	"strings"

	"gig/internal/scm"
	azuredevopsscm "gig/internal/scm/azuredevops"
	bitbucketscm "gig/internal/scm/bitbucket"
	githubscm "gig/internal/scm/github"
	gitlabscm "gig/internal/scm/gitlab"
)

type ProviderStatus struct {
	Provider scm.Type `json:"provider"`
	Ready    bool     `json:"ready"`
	Detail   string   `json:"detail,omitempty"`
}

func CheckProviderStatus(ctx context.Context, provider scm.Type, stdin io.Reader) ProviderStatus {
	var err error
	switch provider {
	case scm.TypeGitHub:
		err = githubscm.NewSession(stdin, io.Discard, io.Discard).Status(ctx)
	case scm.TypeGitLab:
		err = gitlabscm.NewSession(stdin, io.Discard, io.Discard).Status(ctx)
	case scm.TypeBitbucket:
		err = bitbucketscm.NewSession(stdin, io.Discard, io.Discard).Status(ctx)
	case scm.TypeAzureDevOps:
		err = azuredevopsscm.NewSession(stdin, io.Discard, io.Discard).Status(ctx)
	default:
		return ProviderStatus{Provider: provider, Detail: "status unavailable"}
	}

	if err == nil {
		return ProviderStatus{
			Provider: provider,
			Ready:    true,
			Detail:   "ready",
		}
	}

	detail := "login required"
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "executable not found"):
		detail = "cli missing"
	case strings.Contains(message, "credential"), strings.Contains(message, "token"), strings.Contains(message, "password"):
		detail = "credentials needed"
	}

	return ProviderStatus{
		Provider: provider,
		Ready:    false,
		Detail:   detail,
	}
}
