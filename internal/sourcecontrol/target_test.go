package sourcecontrol_test

import (
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
