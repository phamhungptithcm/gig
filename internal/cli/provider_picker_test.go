package cli

import (
	"testing"

	"gig/internal/scm"
)

func TestInferProviderFromGitRemoteURL(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		want      scm.Type
		wantRoot  string
		wantOK    bool
	}{
		{
			name:      "github ssh",
			remoteURL: "git@github.com:acme/payments.git",
			want:      scm.TypeGitHub,
			wantRoot:  "github:acme/payments",
			wantOK:    true,
		},
		{
			name:      "gitlab https",
			remoteURL: "https://gitlab.com/acme/platform/payments.git",
			want:      scm.TypeGitLab,
			wantRoot:  "gitlab:acme/platform/payments",
			wantOK:    true,
		},
		{
			name:      "bitbucket ssh",
			remoteURL: "git@bitbucket.org:acme/payments.git",
			want:      scm.TypeBitbucket,
			wantRoot:  "bitbucket:acme/payments",
			wantOK:    true,
		},
		{
			name:      "azure devops https",
			remoteURL: "https://dev.azure.com/acme/Payments/_git/release-audit",
			want:      scm.TypeAzureDevOps,
			wantRoot:  "azure-devops:acme/Payments/release-audit",
			wantOK:    true,
		},
		{
			name:      "unknown remote",
			remoteURL: "git@example.com:acme/payments.git",
			wantOK:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := inferProviderFromGitRemoteURL(test.remoteURL)
			if ok != test.wantOK {
				t.Fatalf("ok = %v, want %v", ok, test.wantOK)
			}
			if got != test.want {
				t.Fatalf("provider = %q, want %q", got, test.want)
			}
			repository, repoOK := inferRepositoryFromGitRemoteURL(test.remoteURL)
			if repoOK != test.wantOK {
				t.Fatalf("repository ok = %v, want %v", repoOK, test.wantOK)
			}
			if test.wantOK && repository.Root != test.wantRoot {
				t.Fatalf("repository root = %q, want %q", repository.Root, test.wantRoot)
			}
		})
	}
}
