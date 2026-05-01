package cli

import (
	"reflect"
	"testing"
)

func TestParseFrontDoorCommand(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		hasCurrent bool
		hasSaved   bool
		hasAssist  bool
		want       frontDoorCommand
	}{
		{
			name: "empty opens picker",
			line: "",
			want: frontDoorCommand{Action: frontDoorActionPicker},
		},
		{
			name: "ticket alone inspects",
			line: "ABC-123",
			want: frontDoorCommand{Action: frontDoorActionInspect, TicketID: "ABC-123"},
		},
		{
			name: "verify with repo target",
			line: "verify ABC-123 github:acme/payments",
			want: frontDoorCommand{Action: frontDoorActionVerify, TicketID: "ABC-123", RepoTarget: "github:acme/payments"},
		},
		{
			name: "manifest generate form",
			line: "manifest generate ABC-123",
			want: frontDoorCommand{Action: frontDoorActionManifest, TicketID: "ABC-123"},
		},
		{
			name: "packet shortcut",
			line: "packet ABC-123",
			want: frontDoorCommand{Action: frontDoorActionManifest, TicketID: "ABC-123"},
		},
		{
			name: "inspect alias",
			line: "i ABC-123",
			want: frontDoorCommand{Action: frontDoorActionInspect, TicketID: "ABC-123"},
		},
		{
			name: "verify alias reuses ticket later",
			line: "v",
			want: frontDoorCommand{Action: frontDoorActionVerify},
		},
		{
			name: "packet alias reuses ticket later",
			line: "p",
			want: frontDoorCommand{Action: frontDoorActionManifest},
		},
		{
			name: "help shortcut opens compact help",
			line: "?",
			want: frontDoorCommand{Action: frontDoorActionHelp},
		},
		{
			name: "repo opens repository resolver",
			line: "repo",
			want: frontDoorCommand{Action: frontDoorActionRepo},
		},
		{
			name: "repo short name searches repository scope",
			line: "repo payments",
			want: frontDoorCommand{Action: frontDoorActionRepo, RepoQuery: "payments"},
		},
		{
			name: "repo target only remembers scope",
			line: "repo github:acme/payments",
			want: frontDoorCommand{Action: frontDoorActionRepo, RepoTarget: "github:acme/payments"},
		},
		{
			name: "github provider alias remembers target",
			line: "gh acme/payments",
			want: frontDoorCommand{Action: frontDoorActionRepo, RepoTarget: "gh:acme/payments"},
		},
		{
			name: "provider alias with ticket inspects",
			line: "gl acme/platform/payments ABC-123 --format json",
			want: frontDoorCommand{Action: frontDoorActionInspect, TicketID: "ABC-123", RepoTarget: "gl:acme/platform/payments", ExtraArgs: []string{"--format", "json"}},
		},
		{
			name: "save natural name",
			line: "save payments",
			want: frontDoorCommand{Action: frontDoorActionSave, Message: "payments"},
		},
		{
			name: "use natural project",
			line: "use payments",
			want: frontDoorCommand{Action: frontDoorActionProject, Args: []string{"use", "payments"}},
		},
		{
			name: "last shortcut reruns previous command",
			line: "last",
			want: frontDoorCommand{Action: frontDoorActionLast},
		},
		{
			name: "long prompt command parses common flags and keeps extra flags",
			line: "gig verify ABC-123 --repo github:acme/payments --from staging --to main --format json",
			want: frontDoorCommand{Action: frontDoorActionVerify, TicketID: "ABC-123", RepoTarget: "github:acme/payments", FromBranch: "staging", ToBranch: "main", ExtraArgs: []string{"--format", "json"}},
		},
		{
			name: "long ticket file command does not treat flag as ticket",
			line: "gig verify --ticket-file tickets.txt --repo github:acme/payments --format json",
			want: frontDoorCommand{Action: frontDoorActionVerify, RepoTarget: "github:acme/payments", ExtraArgs: []string{"--ticket-file", "tickets.txt", "--format", "json"}},
		},
		{
			name: "repo shortcut",
			line: "repo github:acme/payments ABC-123",
			want: frontDoorCommand{Action: frontDoorActionInspect, TicketID: "ABC-123", RepoTarget: "github:acme/payments"},
		},
		{
			name: "login without provider keeps provider empty",
			line: "login",
			want: frontDoorCommand{Action: frontDoorActionLogin},
		},
		{
			name:      "ask explicit question",
			line:      "ask what is still blocked?",
			hasAssist: true,
			want:      frontDoorCommand{Action: frontDoorActionAsk, Message: "what is still blocked?"},
		},
		{
			name:      "natural language becomes ask when assist is ready",
			line:      "what changed since yesterday?",
			hasAssist: true,
			want:      frontDoorCommand{Action: frontDoorActionAsk, Message: "what changed since yesterday?"},
		},
		{
			name:      "resume shortcut opens saved assist session",
			line:      "resume",
			hasAssist: true,
			want:      frontDoorCommand{Action: frontDoorActionResume},
		},
		{
			name: "exit closes session",
			line: "exit",
			want: frontDoorCommand{Action: frontDoorActionExit},
		},
		{
			name: "quit closes session",
			line: "quit",
			want: frontDoorCommand{Action: frontDoorActionExit},
		},
		{
			name: "numeric fallback without current project",
			line: "2",
			want: frontDoorCommand{Action: frontDoorActionEnterTarget},
		},
		{
			name:       "numeric fallback with current project",
			line:       "2",
			hasCurrent: true,
			want:       frontDoorCommand{Action: frontDoorActionVerify},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseFrontDoorCommand(test.line, test.hasCurrent, test.hasSaved, test.hasAssist)
			if err != nil {
				t.Fatalf("parseFrontDoorCommand() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("parseFrontDoorCommand() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestFrontDoorCommandNeedsTicketPrompt(t *testing.T) {
	tests := []struct {
		name    string
		command frontDoorCommand
		want    bool
	}{
		{
			name:    "short verify without remembered ticket prompts",
			command: frontDoorCommand{Action: frontDoorActionVerify},
			want:    true,
		},
		{
			name:    "positional ticket does not prompt",
			command: frontDoorCommand{Action: frontDoorActionVerify, TicketID: "ABC-123"},
		},
		{
			name:    "ticket file does not prompt",
			command: frontDoorCommand{Action: frontDoorActionVerify, ExtraArgs: []string{"--ticket-file", "tickets.txt"}},
		},
		{
			name:    "release packet does not prompt",
			command: frontDoorCommand{Action: frontDoorActionManifest, ExtraArgs: []string{"--release=rel-2026-04-09"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := frontDoorCommandNeedsTicketPrompt(test.command); got != test.want {
				t.Fatalf("frontDoorCommandNeedsTicketPrompt() = %v, want %v", got, test.want)
			}
		})
	}
}
