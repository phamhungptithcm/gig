package cli

import (
	"fmt"
	"strings"

	"gig/internal/output"
	"gig/internal/scm"
	"gig/internal/sourcecontrol"
	"gig/internal/workarea"
)

type suggestionContext struct {
	Command         string
	TicketID        string
	RepoTarget      string
	Path            string
	Provider        scm.Type
	AuthStatus      *sourcecontrol.ProviderStatus
	Topology        *sourcecontrol.TopologyInference
	FromBranch      string
	ToBranch        string
	EnvironmentSpec string
	ConfigPath      string
	ConfigDetected  bool
	Current         *workarea.Definition
	Detected        *output.FrontDoorDetectedRepository
	HasAssist       bool
	NeedsBranches   bool
}

func buildSmartSuggestions(ctx suggestionContext) []output.FrontDoorSuggestion {
	ticketID := strings.TrimSpace(ctx.TicketID)
	if ticketID == "" {
		ticketID = "ABC-123"
	}
	command := strings.TrimSpace(ctx.Command)
	if command == "" {
		command = "inspect"
	}

	suggestions := make([]output.FrontDoorSuggestion, 0, 6)
	addCommand := func(command string) {
		command = strings.TrimSpace(command)
		if command == "" || hasSuggestion(suggestions, command) {
			return
		}
		suggestions = append(suggestions, output.FrontDoorSuggestion{Command: command})
	}
	addNote := func(note string) {
		note = strings.TrimSpace(note)
		if note == "" {
			return
		}
		suggestions = append(suggestions, output.FrontDoorSuggestion{Note: note})
	}

	if loginCommand := authLoginSuggestion(ctx); loginCommand != "" {
		addCommand(loginCommand)
		if ctx.AuthStatus != nil && ctx.AuthStatus.Detail != "" {
			switch strings.TrimSpace(ctx.AuthStatus.Detail) {
			case "cli missing":
				addNote("provider CLI missing; install it before login")
			default:
				addNote("provider " + ctx.AuthStatus.Detail + "; login before running live remote commands")
			}
		}
	}

	if ctx.NeedsBranches || topologyNeedsExplicitBranches(ctx.Topology) {
		scopeFlag := suggestionScopeFlag(ctx)
		addCommand(fmt.Sprintf("gig %s %s%s --from <source> --to <target>", commandName(command), ticketID, scopeFlag))
		if strings.TrimSpace(ctx.RepoTarget) != "" {
			addCommand(fmt.Sprintf("gig project add%s --from <source> --to <target> --use", scopeFlag))
		}
		addNote("add --envs only when the detected branch order is ambiguous")
		return suggestions
	}

	switch {
	case ctx.Current != nil:
		addCommand("gig " + ticketID)
		addCommand("gig verify " + ticketID)
		addCommand("gig packet " + ticketID)
		if ctx.HasAssist {
			addCommand("gig ask \"what is still blocked?\"")
		}
		if promotion := suggestionWorkareaPromotion(*ctx.Current); promotion != "" {
			addNote("promotion " + promotion)
		}
	case ctx.Detected != nil:
		if suggestionDetectedIsRemote(*ctx.Detected) {
			addCommand("gig " + ticketID)
			addCommand("gig verify " + ticketID)
			addCommand("gig packet " + ticketID)
			addNote("remote checkout detected from origin; use --repo only when running outside this folder")
		} else {
			addCommand("gig " + ticketID + " --path .")
			if fromBranch, toBranch, ok := suggestionDetectedPromotionBranches(*ctx.Detected); ok {
				addCommand(fmt.Sprintf("gig verify %s --path . --from %s --to %s", ticketID, fromBranch, toBranch))
				addCommand(fmt.Sprintf("gig packet %s --path . --from %s --to %s", ticketID, fromBranch, toBranch))
			} else {
				addCommand("gig project add local --path . --from <source> --to <target> --use")
			}
			addNote("local " + strings.ToLower(suggestionBlankAsDefault(ctx.Detected.Type, "repository")) + " detected; add a project when this becomes daily context")
		}
	default:
		if len(suggestions) == 0 {
			addCommand("gig login github")
		}
		if repo := strings.TrimSpace(ctx.RepoTarget); repo != "" {
			addCommand("gig " + ticketID + " --repo " + repo)
		} else {
			addCommand("gig " + ticketID + " --repo github:owner/name")
		}
		addCommand("gig project add --provider github --use")
		addNote("paste a repo target when you already know it; use --path . for local Git or SVN")
	}

	if strings.TrimSpace(ctx.ConfigPath) != "" {
		addNote("using config overrides from " + strings.TrimSpace(ctx.ConfigPath))
	} else if ctx.ConfigDetected {
		addNote("config detected; keep --from and --to only when inference is wrong")
	}

	return suggestions
}

func hasSuggestion(suggestions []output.FrontDoorSuggestion, command string) bool {
	for _, suggestion := range suggestions {
		if suggestion.Command == command {
			return true
		}
	}
	return false
}

func authLoginSuggestion(ctx suggestionContext) string {
	if ctx.AuthStatus == nil || ctx.AuthStatus.Ready {
		return ""
	}
	if strings.EqualFold(strings.TrimSpace(ctx.AuthStatus.Detail), "not checked") {
		return ""
	}
	provider := ctx.Provider
	if provider == "" {
		provider = ctx.AuthStatus.Provider
	}
	name := providerCommandName(provider)
	if name == "" {
		return ""
	}
	return "gig login " + name
}

func topologyNeedsExplicitBranches(inference *sourcecontrol.TopologyInference) bool {
	return inference != nil && inference.Confidence != "" && inference.Confidence != sourcecontrol.TopologyConfidenceHigh
}

func suggestionScopeFlag(ctx suggestionContext) string {
	if repo := strings.TrimSpace(ctx.RepoTarget); repo != "" {
		return " --repo " + repo
	}
	if ctx.Current != nil {
		return ""
	}
	if path := strings.TrimSpace(ctx.Path); path != "" && path != "." {
		return " --path " + path
	}
	return " --path ."
}

func commandName(command string) string {
	switch strings.ToLower(strings.TrimSpace(command)) {
	case "", "inspect":
		return "inspect"
	case "manifest", "manifest generate":
		return "packet"
	default:
		return strings.ToLower(strings.TrimSpace(command))
	}
}

func providerCommandName(provider scm.Type) string {
	switch provider {
	case scm.TypeGitHub:
		return "github"
	case scm.TypeGitLab:
		return "gitlab"
	case scm.TypeBitbucket:
		return "bitbucket"
	case scm.TypeAzureDevOps:
		return "azure-devops"
	case scm.TypeSVN, scm.TypeRemoteSVN:
		return "svn"
	default:
		return strings.TrimSpace(string(provider))
	}
}

func suggestionDetectedIsRemote(repository output.FrontDoorDetectedRepository) bool {
	root := strings.ToLower(strings.TrimSpace(repository.Root))
	return strings.HasPrefix(root, "github:") ||
		strings.HasPrefix(root, "gitlab:") ||
		strings.HasPrefix(root, "bitbucket:") ||
		strings.HasPrefix(root, "azure-devops:") ||
		strings.HasPrefix(root, "svn:")
}

func suggestionDetectedPromotionBranches(repository output.FrontDoorDetectedRepository) (string, string, bool) {
	fromBranch := strings.TrimSpace(repository.Branch)
	if fromBranch == "" || suggestionIsProductionBranchName(fromBranch) {
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

func suggestionIsProductionBranchName(branch string) bool {
	switch strings.ToLower(strings.TrimSpace(branch)) {
	case "main", "master", "prod", "production", "trunk":
		return true
	default:
		return false
	}
}

func suggestionWorkareaPromotion(definition workarea.Definition) string {
	if definition.FromBranch == "" && definition.ToBranch == "" {
		return ""
	}
	return fmt.Sprintf("%s -> %s", suggestionBlankAsAuto(definition.FromBranch), suggestionBlankAsAuto(definition.ToBranch))
}

func suggestionBlankAsAuto(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "auto"
	}
	return value
}

func suggestionBlankAsDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
