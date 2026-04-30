package output

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"gig/internal/workarea"
)

type FrontDoorDetectedRepository struct {
	Name    string `json:"name,omitempty"`
	Root    string `json:"root,omitempty"`
	Type    string `json:"type,omitempty"`
	Branch  string `json:"branch,omitempty"`
	Command string `json:"command,omitempty"`
}

type FrontDoorState struct {
	Current                 *workarea.Definition         `json:"current,omitempty"`
	Workareas               []workarea.Definition        `json:"workareas,omitempty"`
	Detected                *FrontDoorDetectedRepository `json:"detected,omitempty"`
	Version                 string                       `json:"version,omitempty"`
	HeroStatus              string                       `json:"heroStatus,omitempty"`
	StatusRows              []KeyValue                   `json:"statusRows,omitempty"`
	ProviderCoverage        []KeyValue                   `json:"providerCoverage,omitempty"`
	ResumeTitle             string                       `json:"resumeTitle,omitempty"`
	ResumeSummary           string                       `json:"resumeSummary,omitempty"`
	ResumeScope             string                       `json:"resumeScope,omitempty"`
	ResumeQuestion          string                       `json:"resumeQuestion,omitempty"`
	ResumeSuggestedQuestion string                       `json:"resumeSuggestedQuestion,omitempty"`
	Prompt                  string                       `json:"prompt,omitempty"`
	Examples                []string                     `json:"examples,omitempty"`
	Suggestions             []FrontDoorSuggestion        `json:"suggestions,omitempty"`
}

type FrontDoorSuggestion struct {
	Command string `json:"command,omitempty"`
	Note    string `json:"note,omitempty"`
}

func RenderFrontDoor(w io.Writer, state FrontDoorState) error {
	ui := NewConsole(w)

	if err := renderFrontDoorHero(w, ui, state); err != nil {
		return err
	}
	if err := ui.Blank(); err != nil {
		return err
	}
	if state.ResumeSummary != "" {
		if err := ui.Section(blankAsDefault(state.ResumeTitle, "Resume AI context")); err != nil {
			return err
		}
		if err := ui.Bullets(state.ResumeSummary); err != nil {
			return err
		}
		if strings.TrimSpace(state.ResumeScope) != "" {
			if err := ui.Note("Scope: " + strings.TrimSpace(state.ResumeScope)); err != nil {
				return err
			}
		}
		if strings.TrimSpace(state.ResumeQuestion) != "" {
			if err := ui.Note("Last question: " + strings.TrimSpace(state.ResumeQuestion)); err != nil {
				return err
			}
		}
		if err := ui.Commands(
			fmt.Sprintf("gig ask %q", blankAsDefault(state.ResumeSuggestedQuestion, "what is still blocked?")),
			"gig resume",
		); err != nil {
			return err
		}
		if err := ui.Blank(); err != nil {
			return err
		}
	}

	if len(state.StatusRows) > 0 {
		if err := ui.Section("Startup status"); err != nil {
			return err
		}
		if err := ui.Rows(state.StatusRows...); err != nil {
			return err
		}
		if err := ui.Blank(); err != nil {
			return err
		}
	}

	if len(state.ProviderCoverage) > 0 {
		if err := ui.Section("Provider coverage"); err != nil {
			return err
		}
		if err := ui.Rows(state.ProviderCoverage...); err != nil {
			return err
		}
		if err := ui.Blank(); err != nil {
			return err
		}
	}

	if err := renderFrontDoorSuggestions(w, ui, state); err != nil {
		return err
	}

	if state.Current == nil {
		if len(state.Workareas) > 0 {
			if err := ui.Blank(); err != nil {
				return err
			}
			if err := ui.Section("Saved projects"); err != nil {
				return err
			}
			for _, definition := range state.Workareas {
				if err := ui.Bullets(definition.Name); err != nil {
					return err
				}
				if target := formatWorkareaTarget(definition); target != "" {
					if _, err := fmt.Fprintf(w, "    %s  %s\n", ui.Muted("target"), target); err != nil {
						return err
					}
				}
			}
			if err := ui.Commands("gig project use"); err != nil {
				return err
			}
		}
	}

	if err := ui.Blank(); err != nil {
		return err
	}
	if err := ui.Section("Try one line"); err != nil {
		return err
	}
	return renderPromptBox(w, ui, blankAsDefault(state.Prompt, "ask gig > ABC-123"), state.Examples)
}

func renderFrontDoorSuggestions(w io.Writer, ui Console, state FrontDoorState) error {
	if err := ui.Section("Suggested next"); err != nil {
		return err
	}

	if len(state.Suggestions) > 0 {
		for _, suggestion := range state.Suggestions {
			if strings.TrimSpace(suggestion.Command) != "" {
				if err := ui.Commands(suggestion.Command); err != nil {
					return err
				}
			}
			if strings.TrimSpace(suggestion.Note) != "" {
				if err := ui.Note(suggestion.Note); err != nil {
					return err
				}
			}
		}
		if _, err := fmt.Fprintf(w, "  %s %s\n", ui.Muted("more"), ui.Command("gig --help")); err != nil {
			return err
		}
		return nil
	}

	switch {
	case state.Current != nil:
		commands := []string{
			"gig ABC-123",
			"gig verify ABC-123",
			"gig packet ABC-123",
		}
		if state.ResumeSummary != "" {
			commands = append(commands, "gig ask \"what is still blocked?\"")
		}
		if err := ui.Commands(commands...); err != nil {
			return err
		}
		if promotion := formatFrontDoorPromotion(*state.Current); promotion != "" {
			if err := ui.Note("promotion " + promotion); err != nil {
				return err
			}
		}
	case state.Detected != nil:
		if frontDoorDetectedIsRemote(*state.Detected) {
			commands := []string{
				"gig ABC-123",
				"gig verify ABC-123",
				"gig packet ABC-123",
			}
			if err := ui.Commands(commands...); err != nil {
				return err
			}
			if err := ui.Note("remote checkout detected from origin; use --repo only when running outside this folder"); err != nil {
				return err
			}
		} else {
			commands := []string{"gig ABC-123 --path ."}
			if fromBranch, toBranch, ok := detectedPromotionBranches(*state.Detected); ok {
				commands = append(commands,
					fmt.Sprintf("gig verify ABC-123 --path . --from %s --to %s", fromBranch, toBranch),
					fmt.Sprintf("gig packet ABC-123 --path . --from %s --to %s", fromBranch, toBranch),
				)
			} else {
				commands = append(commands, "gig project add local --path . --from <source> --to <target> --use")
			}
			if err := ui.Commands(commands...); err != nil {
				return err
			}
			if err := ui.Note("local " + strings.ToLower(blankAsDefault(state.Detected.Type, "repository")) + " detected; add a project when this becomes daily context"); err != nil {
				return err
			}
		}
	default:
		if err := ui.Commands(
			"gig login",
			"gig ABC-123 --repo github:owner/name",
			"gig project add --provider github --use",
		); err != nil {
			return err
		}
		if err := ui.Note("paste a repo target when you already know it; use --path . for local Git or SVN"); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "  %s %s\n", ui.Muted("more"), ui.Command("gig --help")); err != nil {
		return err
	}
	return nil
}

func detectedPromotionBranches(repository FrontDoorDetectedRepository) (string, string, bool) {
	fromBranch := strings.TrimSpace(repository.Branch)
	if fromBranch == "" || isProductionBranchName(fromBranch) {
		return "", "", false
	}

	toBranch := "main"
	if strings.Contains(strings.ToLower(repository.Type), "svn") {
		toBranch = "trunk"
	}
	if strings.EqualFold(fromBranch, toBranch) {
		return "", "", false
	}
	return fromBranch, toBranch, true
}

func isProductionBranchName(branch string) bool {
	switch strings.ToLower(strings.TrimSpace(branch)) {
	case "main", "master", "prod", "production", "trunk":
		return true
	default:
		return false
	}
}

func renderFrontDoorHero(w io.Writer, ui Console, state FrontDoorState) error {
	lines := []string{
		fmt.Sprintf(">_ gig  (%s)", blankAsDefault(state.Version, "dev")),
		"ticket-aware release audit",
		"docs   https://phamhungptithcm.github.io/gig",
		"",
	}

	if state.Current != nil {
		lines = append(lines,
			fmt.Sprintf("scope   project %s", state.Current.Name),
			fmt.Sprintf("target  %s", formatWorkareaTarget(*state.Current)),
			fmt.Sprintf("branch  %s", blankAsDefault(state.Current.FromBranch, "auto")),
			fmt.Sprintf("release %s", formatFrontDoorHeroTarget(*state.Current)),
			"next    gig ABC-123",
		)
	} else if state.Detected != nil {
		lines = append(lines,
			fmt.Sprintf("source  %s", blankAsDefault(state.Detected.Type, "Local repository")),
			fmt.Sprintf("%-7s %s", frontDoorDetectedRootLabel(*state.Detected), frontDoorDetectedRootValue(*state.Detected)),
			"branch",
			fmt.Sprintf("  %s", blankAsDefault(state.Detected.Branch, "unknown")),
			fmt.Sprintf("next    %s", blankAsDefault(state.Detected.Command, "gig ABC-123 --path .")),
		)
	} else {
		lines = append(lines,
			"scope   choose a repository",
			"source  GitHub | GitLab | Bitbucket | Azure DevOps | SVN",
			"next    gig login",
		)
	}
	lines = append(lines, fmt.Sprintf("status  %s", blankAsDefault(state.HeroStatus, "no project selected yet")))

	return writeFrontDoorBox(w, ui, lines)
}

func frontDoorDetectedIsRemote(repository FrontDoorDetectedRepository) bool {
	root := strings.ToLower(strings.TrimSpace(repository.Root))
	return strings.HasPrefix(root, "github:") ||
		strings.HasPrefix(root, "gitlab:") ||
		strings.HasPrefix(root, "bitbucket:") ||
		strings.HasPrefix(root, "azure-devops:") ||
		strings.HasPrefix(root, "svn:")
}

func frontDoorDetectedRootLabel(repository FrontDoorDetectedRepository) string {
	if frontDoorDetectedIsRemote(repository) {
		return "target"
	}
	return "path"
}

func frontDoorDetectedRootValue(repository FrontDoorDetectedRepository) string {
	if frontDoorDetectedIsRemote(repository) {
		return blankAsDefault(repository.Root, "unknown")
	}
	return formatFrontDoorPath(repository.Root)
}

func writeFrontDoorBox(w io.Writer, ui Console, lines []string) error {
	maxWidth := 0
	if ui.Width() > 0 {
		maxWidth = ui.Width() - 4
		if maxWidth < 24 {
			maxWidth = 24
		}
	}

	width := 0
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, " ")
		if maxWidth > 0 {
			line = ui.Truncate(line, maxWidth)
		}
		normalized = append(normalized, line)
		if len(line) > width {
			width = len(line)
		}
	}

	border := "+" + strings.Repeat("-", width+2) + "+"
	if _, err := fmt.Fprintln(w, ui.Emphasis(border)); err != nil {
		return err
	}
	for index, line := range normalized {
		padded := line + strings.Repeat(" ", width-len(line))
		value := padded
		switch {
		case index == 1:
			value = ui.Command(padded)
		case line == "branch":
			value = ui.Muted(padded)
		case strings.HasPrefix(line, "  "):
			value = ui.Emphasis(padded)
		case strings.HasPrefix(line, "next"):
			value = ui.Command(padded)
		case strings.HasPrefix(line, "status"):
			value = ui.Muted(padded)
		}
		if _, err := fmt.Fprintf(w, "%s %s %s\n", ui.Emphasis("|"), value, ui.Emphasis("|")); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w, ui.Emphasis(border))
	return err
}

func formatFrontDoorHeroTarget(definition workarea.Definition) string {
	if definition.ToBranch != "" {
		return definition.ToBranch
	}
	if definition.EnvironmentSpec != "" {
		return "from saved topology"
	}
	return "auto"
}

func formatFrontDoorPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "."
	}
	if base := filepath.Base(path); base != "." && base != string(filepath.Separator) {
		return base
	}
	return path
}

func renderPromptBox(w io.Writer, ui Console, prompt string, examples []string) error {
	maxWidth := 0
	if ui.Width() > 0 {
		maxWidth = ui.Width() - 4
		if maxWidth < 28 {
			maxWidth = 28
		}
	}

	width := len(prompt)
	for _, example := range examples {
		if len(example) > width {
			width = len(example)
		}
	}
	if width < 28 {
		width = 28
	}
	if maxWidth > 0 && width > maxWidth {
		width = maxWidth
	}

	border := "+" + strings.Repeat("-", width+2) + "+"
	if _, err := fmt.Fprintln(w, ui.Emphasis(border)); err != nil {
		return err
	}
	prompt = ui.Truncate(prompt, width)
	paddedPrompt := prompt + strings.Repeat(" ", width-len(prompt))
	if _, err := fmt.Fprintf(w, "%s %s %s\n", ui.Emphasis("|"), ui.Command(paddedPrompt), ui.Emphasis("|")); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ui.Emphasis(border)); err != nil {
		return err
	}
	exampleRows := make([]KeyValue, 0, len(examples))
	for _, example := range examples {
		if strings.TrimSpace(example) == "" {
			continue
		}
		exampleRows = append(exampleRows, KeyValue{Label: "try", Value: example})
	}
	if len(exampleRows) > 0 {
		if err := ui.NestedRows(exampleRows...); err != nil {
			return err
		}
	}
	return nil
}

func formatFrontDoorPromotion(definition workarea.Definition) string {
	if definition.FromBranch == "" && definition.ToBranch == "" {
		return ""
	}
	return fmt.Sprintf("%s -> %s", blankAsAuto(definition.FromBranch), blankAsAuto(definition.ToBranch))
}

func blankAsDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
