package sourcecontrol

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"gig/internal/scm"
	"gig/internal/toolcheck"
)

type ProviderStatus struct {
	Provider scm.Type `json:"provider"`
	Ready    bool     `json:"ready"`
	Detail   string   `json:"detail,omitempty"`
}

func CheckProviderStatus(ctx context.Context, provider scm.Type, stdin io.Reader) ProviderStatus {
	statusCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var err error
	switch provider {
	case scm.TypeGitHub, scm.TypeGitLab, scm.TypeBitbucket, scm.TypeAzureDevOps, scm.TypeRemoteSVN:
		err = ProviderStatusError(statusCtx, provider, nil, stdin)
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

	detail := toolcheck.Detail(err)
	message := strings.ToLower(err.Error())
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		detail = "status timed out"
	case strings.Contains(message, "credential"), strings.Contains(message, "token"), strings.Contains(message, "password"):
		detail = "credentials needed"
	}

	return ProviderStatus{
		Provider: provider,
		Ready:    false,
		Detail:   detail,
	}
}
