package cli

import (
	"testing"

	"gig/internal/output"
	"gig/internal/scm"
	"gig/internal/sourcecontrol"
)

func TestBuildSmartSuggestionsPrioritizesProviderLogin(t *testing.T) {
	t.Parallel()

	status := sourcecontrol.ProviderStatus{
		Provider: scm.TypeGitHub,
		Ready:    false,
		Detail:   "login required",
	}
	suggestions := buildSmartSuggestions(suggestionContext{
		Command:    "inspect",
		TicketID:   "ABC-123",
		RepoTarget: "github:acme/payments",
		Provider:   scm.TypeGitHub,
		AuthStatus: &status,
	})

	if len(suggestions) == 0 || suggestions[0].Command != "gig login github" {
		t.Fatalf("suggestions = %#v, want login first", suggestions)
	}
	if !containsSuggestionCommand(suggestions, "gig ABC-123 --repo github:acme/payments") {
		t.Fatalf("suggestions = %#v, want exact remote inspect retry", suggestions)
	}
}

func TestBuildSmartSuggestionsUsesTopologyContextForExplicitBranches(t *testing.T) {
	t.Parallel()

	inference := sourcecontrol.TopologyInference{
		Confidence:        sourcecontrol.TopologyConfidenceMedium,
		ProtectedBranches: []string{"main"},
	}
	suggestions := buildSmartSuggestions(suggestionContext{
		Command:    "verify",
		TicketID:   "ABC-123",
		RepoTarget: "github:acme/payments",
		Topology:   &inference,
	})

	if !containsSuggestionCommand(suggestions, "gig verify ABC-123 --repo github:acme/payments --from <source> --to <target>") {
		t.Fatalf("suggestions = %#v, want exact verify command with explicit branches", suggestions)
	}
	if !containsSuggestionCommand(suggestions, "gig project add --repo github:acme/payments --from <source> --to <target> --use") {
		t.Fatalf("suggestions = %#v, want project defaults command", suggestions)
	}
}

func containsSuggestionCommand(suggestions []output.FrontDoorSuggestion, command string) bool {
	for _, suggestion := range suggestions {
		if suggestion.Command == command {
			return true
		}
	}
	return false
}
