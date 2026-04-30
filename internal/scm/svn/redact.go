package svn

import "strings"

func redactCommandArgs(args []string) string {
	redacted := append([]string(nil), args...)
	for i, arg := range redacted {
		normalized := strings.ToLower(strings.TrimSpace(arg))
		switch {
		case normalized == "--password":
			if i+1 < len(redacted) {
				redacted[i+1] = "<redacted>"
			}
		case strings.HasPrefix(normalized, "--password="):
			redacted[i] = "--password=<redacted>"
		}
	}
	return strings.Join(redacted, " ")
}
