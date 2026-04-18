package cli

import "testing"

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
			name: "repo shortcut",
			line: "repo github:acme/payments ABC-123",
			want: frontDoorCommand{Action: frontDoorActionInspect, TicketID: "ABC-123", RepoTarget: "github:acme/payments"},
		},
		{
			name: "login defaults to github",
			line: "login",
			want: frontDoorCommand{Action: frontDoorActionLogin, Provider: "github"},
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
			if got != test.want {
				t.Fatalf("parseFrontDoorCommand() = %#v, want %#v", got, test.want)
			}
		})
	}
}
