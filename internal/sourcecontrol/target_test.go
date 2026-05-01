package sourcecontrol_test

import (
	"strings"
	"testing"

	"gig/internal/scm"
	"gig/internal/sourcecontrol"
)

func TestParseRepositoryTargetGitHubShortcut(t *testing.T) {
	t.Parallel()

	repository, err := sourcecontrol.ParseRepositoryTarget("github:acme/payments")
	if err != nil {
		t.Fatalf("ParseRepositoryTarget() error = %v", err)
	}

	if repository.Type != scm.TypeGitHub {
		t.Fatalf("Type = %s, want %s", repository.Type, scm.TypeGitHub)
	}
	if repository.Root != "github:acme/payments" {
		t.Fatalf("Root = %q, want github:acme/payments", repository.Root)
	}
	if repository.Name != "payments" {
		t.Fatalf("Name = %q, want payments", repository.Name)
	}
}

func TestParseRepositoryTargetProviderAliasShortcuts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		target string
		want   string
	}{
		{target: "gh:acme/payments", want: "github:acme/payments"},
		{target: "gl:acme/platform/payments", want: "gitlab:acme/platform/payments"},
		{target: "bb:acme/payments", want: "bitbucket:acme/payments"},
		{target: "ado:acme/Payments/release-audit", want: "azure-devops:acme/Payments/release-audit"},
	}

	for _, test := range tests {
		t.Run(test.target, func(t *testing.T) {
			repository, err := sourcecontrol.ParseRepositoryTarget(test.target)
			if err != nil {
				t.Fatalf("ParseRepositoryTarget() error = %v", err)
			}
			if repository.Root != test.want {
				t.Fatalf("Root = %q, want %q", repository.Root, test.want)
			}
		})
	}
}

func TestParseRepositoryTargetGitHubURL(t *testing.T) {
	t.Parallel()

	repository, err := sourcecontrol.ParseRepositoryTarget("https://github.com/acme/payments.git")
	if err != nil {
		t.Fatalf("ParseRepositoryTarget() error = %v", err)
	}

	if repository.Root != "github:acme/payments" {
		t.Fatalf("Root = %q, want github:acme/payments", repository.Root)
	}
}

func TestParseRepositoryTargetGitHubSSHURL(t *testing.T) {
	t.Parallel()

	repository, err := sourcecontrol.ParseRepositoryTarget("git@github.com:acme/payments.git")
	if err != nil {
		t.Fatalf("ParseRepositoryTarget() error = %v", err)
	}

	if repository.Root != "github:acme/payments" {
		t.Fatalf("Root = %q, want github:acme/payments", repository.Root)
	}
}

func TestParseRepositoryTargetRejectsUnsupportedTarget(t *testing.T) {
	t.Parallel()

	if _, err := sourcecontrol.ParseRepositoryTarget("perforce:acme/payments"); err == nil {
		t.Fatal("ParseRepositoryTarget() error = nil, want unsupported target error")
	}
}

func TestParseRepositoryTargetGitLabShortcut(t *testing.T) {
	t.Parallel()

	repository, err := sourcecontrol.ParseRepositoryTarget("gitlab:acme/platform/payments")
	if err != nil {
		t.Fatalf("ParseRepositoryTarget() error = %v", err)
	}

	if repository.Type != scm.TypeGitLab {
		t.Fatalf("Type = %s, want %s", repository.Type, scm.TypeGitLab)
	}
	if repository.Root != "gitlab:acme/platform/payments" {
		t.Fatalf("Root = %q, want gitlab:acme/platform/payments", repository.Root)
	}
	if repository.Name != "payments" {
		t.Fatalf("Name = %q, want payments", repository.Name)
	}
}

func TestParseRepositoryTargetGitLabSSHURL(t *testing.T) {
	t.Parallel()

	repository, err := sourcecontrol.ParseRepositoryTarget("ssh://git@gitlab.com/acme/platform/payments.git")
	if err != nil {
		t.Fatalf("ParseRepositoryTarget() error = %v", err)
	}

	if repository.Root != "gitlab:acme/platform/payments" {
		t.Fatalf("Root = %q, want gitlab:acme/platform/payments", repository.Root)
	}
}

func TestParseRepositoryTargetBitbucketURL(t *testing.T) {
	t.Parallel()

	repository, err := sourcecontrol.ParseRepositoryTarget("https://bitbucket.org/acme/payments.git")
	if err != nil {
		t.Fatalf("ParseRepositoryTarget() error = %v", err)
	}

	if repository.Type != scm.TypeBitbucket {
		t.Fatalf("Type = %s, want %s", repository.Type, scm.TypeBitbucket)
	}
	if repository.Root != "bitbucket:acme/payments" {
		t.Fatalf("Root = %q, want bitbucket:acme/payments", repository.Root)
	}
}

func TestParseRepositoryTargetAzureDevOpsSSHURL(t *testing.T) {
	t.Parallel()

	repository, err := sourcecontrol.ParseRepositoryTarget("git@ssh.dev.azure.com:v3/acme/Payments/release-audit")
	if err != nil {
		t.Fatalf("ParseRepositoryTarget() error = %v", err)
	}

	if repository.Root != "azure-devops:acme/Payments/release-audit" {
		t.Fatalf("Root = %q, want azure-devops:acme/Payments/release-audit", repository.Root)
	}
}

func TestParseRepositoryTargetAzureDevOpsURL(t *testing.T) {
	t.Parallel()

	repository, err := sourcecontrol.ParseRepositoryTarget("https://dev.azure.com/acme/Payments/_git/release-audit")
	if err != nil {
		t.Fatalf("ParseRepositoryTarget() error = %v", err)
	}

	if repository.Type != scm.TypeAzureDevOps {
		t.Fatalf("Type = %s, want %s", repository.Type, scm.TypeAzureDevOps)
	}
	if repository.Root != "azure-devops:acme/Payments/release-audit" {
		t.Fatalf("Root = %q, want azure-devops:acme/Payments/release-audit", repository.Root)
	}
}

func TestParseRepositoryTargetAzureDevOpsShortcut(t *testing.T) {
	t.Parallel()

	repository, err := sourcecontrol.ParseRepositoryTarget("azure-devops:acme/Payments/release-audit")
	if err != nil {
		t.Fatalf("ParseRepositoryTarget() error = %v", err)
	}

	if repository.Type != scm.TypeAzureDevOps {
		t.Fatalf("Type = %s, want %s", repository.Type, scm.TypeAzureDevOps)
	}
	if repository.Root != "azure-devops:acme/Payments/release-audit" {
		t.Fatalf("Root = %q, want azure-devops:acme/Payments/release-audit", repository.Root)
	}
}

func TestParseRepositoryTargetSVNPrefixedURL(t *testing.T) {
	t.Parallel()

	repository, err := sourcecontrol.ParseRepositoryTarget("svn:https://svn.example.com/repos/app/branches/dev/HorizonCRM")
	if err != nil {
		t.Fatalf("ParseRepositoryTarget() error = %v", err)
	}

	if repository.Type != scm.TypeRemoteSVN {
		t.Fatalf("Type = %s, want %s", repository.Type, scm.TypeRemoteSVN)
	}
	if repository.Root != "svn:https://svn.example.com/repos/app/branches/dev/HorizonCRM" {
		t.Fatalf("Root = %q, want svn:https://svn.example.com/repos/app/branches/dev/HorizonCRM", repository.Root)
	}
	if repository.Name != "HorizonCRM" {
		t.Fatalf("Name = %q, want HorizonCRM", repository.Name)
	}
}

func TestParseRepositoryTargetPlainSVNHTTPURL(t *testing.T) {
	t.Parallel()

	repository, err := sourcecontrol.ParseRepositoryTarget("https://svn.example.com/repos/app/branches/staging/ProductName")
	if err != nil {
		t.Fatalf("ParseRepositoryTarget() error = %v", err)
	}

	if repository.Type != scm.TypeRemoteSVN {
		t.Fatalf("Type = %s, want %s", repository.Type, scm.TypeRemoteSVN)
	}
	if repository.Root != "svn:https://svn.example.com/repos/app/branches/staging/ProductName" {
		t.Fatalf("Root = %q, want svn:https://svn.example.com/repos/app/branches/staging/ProductName", repository.Root)
	}
	if repository.Name != "ProductName" {
		t.Fatalf("Name = %q, want ProductName", repository.Name)
	}
}

func TestParseRepositoryTargetSVNSSHURL(t *testing.T) {
	t.Parallel()

	repository, err := sourcecontrol.ParseRepositoryTarget("svn+ssh://svn.example.com/repos/app/trunk")
	if err != nil {
		t.Fatalf("ParseRepositoryTarget() error = %v", err)
	}

	if repository.Type != scm.TypeRemoteSVN {
		t.Fatalf("Type = %s, want %s", repository.Type, scm.TypeRemoteSVN)
	}
	if repository.Root != "svn:svn+ssh://svn.example.com/repos/app/trunk" {
		t.Fatalf("Root = %q, want svn:svn+ssh://svn.example.com/repos/app/trunk", repository.Root)
	}
}

func TestParseRepositoryTargetRejectsSVNURLCredentials(t *testing.T) {
	t.Parallel()

	_, err := sourcecontrol.ParseRepositoryTarget("svn:https://demo:super-secret@svn.example.com/repos/app/trunk")
	if err == nil {
		t.Fatal("ParseRepositoryTarget() error = nil, want credentials rejected")
	}
	if strings.Contains(err.Error(), "super-secret") {
		t.Fatalf("error = %q, leaked password", err.Error())
	}
	if !strings.Contains(err.Error(), "gig login svn") {
		t.Fatalf("error = %q, want login guidance", err.Error())
	}
}
