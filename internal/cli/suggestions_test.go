package cli

import (
	"testing"

	"gig/internal/output"
	"gig/internal/scm"
	"gig/internal/sourcecontrol"
	"gig/internal/workarea"
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
	if suggestions[0].Label != "login" {
		t.Fatalf("suggestions[0].Label = %q, want login", suggestions[0].Label)
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
	if !containsSuggestionLabel(suggestions, "clarify") || !containsSuggestionLabel(suggestions, "remember") {
		t.Fatalf("suggestions = %#v, want workflow labels", suggestions)
	}
}

func TestBuildSmartSuggestionsGuidesFirstRunHabit(t *testing.T) {
	t.Parallel()

	suggestions := buildSmartSuggestions(suggestionContext{
		Command:  "frontdoor",
		TicketID: "ABC-123",
	})

	if len(suggestions) == 0 || suggestions[0].Label != "connect" || suggestions[0].Command != "gig login github" {
		t.Fatalf("suggestions = %#v, want connect/login first", suggestions)
	}
	if !containsSuggestionCommand(suggestions, "repo payments") ||
		!containsSuggestionCommand(suggestions, "repo") ||
		!containsSuggestionCommand(suggestions, "gh owner/name") {
		t.Fatalf("suggestions = %#v, want prompt-first repo starters", suggestions)
	}
	if !containsSuggestionLabel(suggestions, "habit") || !containsSuggestionLabel(suggestions, "inside") {
		t.Fatalf("suggestions = %#v, want habit and prompt-shortcut notes", suggestions)
	}
}

func TestBuildSmartSuggestionsKeepsCurrentProjectShort(t *testing.T) {
	t.Parallel()

	current := workarea.Definition{
		Name:       "payments",
		RepoTarget: "github:acme/payments",
		FromBranch: "staging",
		ToBranch:   "main",
	}
	suggestions := buildSmartSuggestions(suggestionContext{
		Command:   "frontdoor",
		TicketID:  "ABC-123",
		Current:   &current,
		HasAssist: true,
	})

	if !containsSuggestionCommand(suggestions, "gig ABC-123") ||
		!containsSuggestionCommand(suggestions, "gig verify ABC-123") ||
		!containsSuggestionCommand(suggestions, "gig packet ABC-123") {
		t.Fatalf("suggestions = %#v, want core short commands", suggestions)
	}
	if !containsSuggestionCommand(suggestions, "gig ask \"what is still blocked?\"") {
		t.Fatalf("suggestions = %#v, want assist continuation", suggestions)
	}
	if !containsSuggestionLabel(suggestions, "inside") || !containsSuggestionLabel(suggestions, "context") {
		t.Fatalf("suggestions = %#v, want prompt and saved-project context", suggestions)
	}
}

func TestFrontDoorSessionSuggestionsOfferSaveAfterRepeatedRepo(t *testing.T) {
	t.Parallel()

	session := frontDoorSessionState{
		LastTicketID:   "ABC-123",
		LastRepoTarget: "github:acme/payments",
		LastAction:     frontDoorActionVerify,
		LastVerdict:    "warning",
		RepoUses:       map[string]int{"github:acme/payments": 2},
	}
	suggestions := frontDoorSessionSuggestions(&session, nil, false)

	if !containsSuggestionCommand(suggestions, "save payments") {
		t.Fatalf("suggestions = %#v, want natural save project offer", suggestions)
	}
	if !containsSuggestionCommand(suggestions, "plan") || !containsSuggestionCommand(suggestions, "explain") {
		t.Fatalf("suggestions = %#v, want warning follow-up commands", suggestions)
	}
}

func TestFrontDoorLoginInputFromOutput(t *testing.T) {
	t.Parallel()

	got := frontDoorLoginInputFromOutput("auth failed\nrun gig login github before retrying\n")
	if got != "login github" {
		t.Fatalf("frontDoorLoginInputFromOutput() = %q, want login github", got)
	}
}

func TestExtractFrontDoorVerdict(t *testing.T) {
	t.Parallel()

	if got := extractFrontDoorVerdict("Verdict       WARNING\n"); got != "warning" {
		t.Fatalf("extractFrontDoorVerdict() = %q, want warning", got)
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

func containsSuggestionLabel(suggestions []output.FrontDoorSuggestion, label string) bool {
	for _, suggestion := range suggestions {
		if suggestion.Label == label {
			return true
		}
	}
	return false
}
