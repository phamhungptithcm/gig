package sourcecontrol

import "gig/internal/scm"

type ProviderCapability struct {
	Provider                scm.Type `json:"provider"`
	Label                   string   `json:"label"`
	RepositoryDiscovery     bool     `json:"repositoryDiscovery"`
	ProtectedBranchTopology bool     `json:"protectedBranchTopology"`
	PullRequests            bool     `json:"pullRequests"`
	Deployments             bool     `json:"deployments"`
	Checks                  bool     `json:"checks"`
	LinkedIssues            bool     `json:"linkedIssues"`
	Releases                bool     `json:"releases"`
}

func ProviderCapabilities(provider scm.Type) ProviderCapability {
	capability := ProviderCapability{
		Provider: provider,
		Label:    ProviderLabel(provider),
	}

	switch provider {
	case scm.TypeGitHub:
		capability.RepositoryDiscovery = true
		capability.ProtectedBranchTopology = true
		capability.PullRequests = true
		capability.Deployments = true
		capability.Checks = true
		capability.LinkedIssues = true
		capability.Releases = true
	case scm.TypeGitLab:
		capability.RepositoryDiscovery = true
		capability.ProtectedBranchTopology = true
		capability.PullRequests = true
		capability.Deployments = true
		capability.Checks = true
		capability.LinkedIssues = true
		capability.Releases = true
	case scm.TypeBitbucket:
		capability.RepositoryDiscovery = true
		capability.ProtectedBranchTopology = true
		capability.PullRequests = true
		capability.Deployments = true
	case scm.TypeAzureDevOps:
		capability.RepositoryDiscovery = true
		capability.ProtectedBranchTopology = true
		capability.PullRequests = true
		capability.Deployments = true
		capability.Checks = true
		capability.LinkedIssues = true
	case scm.TypeRemoteSVN:
		capability.ProtectedBranchTopology = true
	}

	return capability
}

func OrderedProviderCapabilities() []ProviderCapability {
	return []ProviderCapability{
		ProviderCapabilities(scm.TypeGitHub),
		ProviderCapabilities(scm.TypeGitLab),
		ProviderCapabilities(scm.TypeBitbucket),
		ProviderCapabilities(scm.TypeAzureDevOps),
		ProviderCapabilities(scm.TypeRemoteSVN),
	}
}

func (c ProviderCapability) EvidenceTier() string {
	switch {
	case c.Checks || c.LinkedIssues || c.Releases:
		return "deep release evidence"
	case c.PullRequests || c.Deployments:
		return "basic release evidence"
	case c.ProtectedBranchTopology:
		return "audit topology only"
	default:
		return "limited remote support"
	}
}

func (c ProviderCapability) Summary() string {
	switch c.Provider {
	case scm.TypeGitHub:
		return "deep release evidence: PRs, deployments, checks, linked issues, releases"
	case scm.TypeGitLab:
		return "deep release evidence: merge requests, deployments, checks, linked issues, releases"
	case scm.TypeBitbucket:
		return "basic release evidence: pull requests, deployments, branching model"
	case scm.TypeAzureDevOps:
		return "deep release evidence: pull requests, deployments, checks, linked work items"
	case scm.TypeRemoteSVN:
		return "audit topology only: branch and trunk discovery"
	default:
		return c.EvidenceTier()
	}
}
