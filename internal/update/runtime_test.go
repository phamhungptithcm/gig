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

func TestNormalizeNPMVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		input     string
		expects   string
		expectErr bool
	}{
		{name: "latest stays latest", input: "latest", expects: "latest"},
		{name: "strip padded month and day", input: "v2026.04.09", expects: "2026.4.9"},
		{name: "accept unprefixed version", input: "2026.4.9", expects: "2026.4.9"},
		{name: "reject invalid format", input: "2026.4", expectErr: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeNPMVersion(tc.input)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("NormalizeNPMVersion(%q) error = nil, want error", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeNPMVersion(%q) error = %v", tc.input, err)
			}
			if got != tc.expects {
				t.Fatalf("NormalizeNPMVersion(%q) = %q, want %q", tc.input, got, tc.expects)
			}
		})
	}
}

func TestDetectInstallMode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		path    string
		env     map[string]string
		expects InstallMode
	}{
		{
			name:    "prefer explicit npm env",
			path:    "/usr/local/bin/gig",
			env:     map[string]string{"GIG_INSTALL_MODE": "npm"},
			expects: ModeNPM,
		},
		{
			name:    "detect npm node_modules path",
			path:    "/usr/local/lib/node_modules/@phamhungptithcm/gig/npm/vendor/gig",
			expects: ModeNPM,
		},
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

			lookupEnv := func(key string) (string, bool) {
				value, ok := tc.env[key]
				return value, ok
			}

			if got := DetectInstallMode(tc.path, lookupEnv); got != tc.expects {
				t.Fatalf("DetectInstallMode(%q) = %q, want %q", tc.path, got, tc.expects)
			}
		})
	}
}

func TestResolveNPMPackageName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		env     map[string]string
		expects string
	}{
		{
			name:    "default package name",
			expects: DefaultNPMPackageName,
		},
		{
			name:    "prefer env override",
			env:     map[string]string{"GIG_NPM_PACKAGE_NAME": "@acme/gig"},
			expects: "@acme/gig",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lookupEnv := func(key string) (string, bool) {
				value, ok := tc.env[key]
				return value, ok
			}

			if got := ResolveNPMPackageName(lookupEnv); got != tc.expects {
				t.Fatalf("ResolveNPMPackageName() = %q, want %q", got, tc.expects)
			}
		})
	}
}
