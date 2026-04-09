package update

import "testing"

func TestNormalizeVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		expects string
	}{
		{name: "default latest", input: "", expects: "latest"},
		{name: "keep latest", input: "latest", expects: "latest"},
		{name: "prefix version", input: "0.1.5", expects: "v0.1.5"},
		{name: "keep lower v", input: "v0.1.5", expects: "v0.1.5"},
		{name: "normalize upper v", input: "V0.1.5", expects: "v0.1.5"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := NormalizeVersion(tc.input); got != tc.expects {
				t.Fatalf("NormalizeVersion(%q) = %q, want %q", tc.input, got, tc.expects)
			}
		})
	}
}

func TestDetectInstallMode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		path    string
		expects InstallMode
	}{
		{
			name:    "detect homebrew cellar path",
			path:    "/opt/homebrew/Cellar/gig-cli/0.1.5/bin/gig",
			expects: ModeHomebrew,
		},
		{
			name:    "detect scoop path",
			path:    `C:\Users\demo\scoop\apps\gig-cli\current\gig.exe`,
			expects: ModeScoop,
		},
		{
			name:    "fallback to direct install",
			path:    "/Users/demo/.local/bin/gig",
			expects: ModeDirect,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := DetectInstallMode(tc.path); got != tc.expects {
				t.Fatalf("DetectInstallMode(%q) = %q, want %q", tc.path, got, tc.expects)
			}
		})
	}
}
