package orchestrator

import (
	"fmt"
	"strings"
)

type Action struct {
	Prompt  string
	Command string
}

type SlashCommandOptions struct {
	CreateStory        string
	DevStory           string
	CodeReview         string
	CreateStoryOptions string
	DevStoryOptions    string
	CodeReviewOptions  string
}

const defaultCodeReviewOptions = "yolo and fix findings if any, or don't if not. If none are found git commit & push, only if none are found."

func DefaultSlashCommandOptions() SlashCommandOptions {
	return SlashCommandOptions{
		CreateStory:        "/bmad-create-story",
		DevStory:           "/bmad-dev-story",
		CodeReview:         "/bmad-code-review",
		CreateStoryOptions: "",
		DevStoryOptions:    "",
		CodeReviewOptions:  defaultCodeReviewOptions,
	}
}

func normalizeSlashCommandOptions(opts SlashCommandOptions) SlashCommandOptions {
	defaults := DefaultSlashCommandOptions()

	normalized := SlashCommandOptions{
		CreateStory:        strings.TrimSpace(opts.CreateStory),
		DevStory:           strings.TrimSpace(opts.DevStory),
		CodeReview:         strings.TrimSpace(opts.CodeReview),
		CreateStoryOptions: strings.TrimSpace(opts.CreateStoryOptions),
		DevStoryOptions:    strings.TrimSpace(opts.DevStoryOptions),
		CodeReviewOptions:  strings.TrimSpace(opts.CodeReviewOptions),
	}

	if normalized.CreateStory == "" {
		normalized.CreateStory = defaults.CreateStory
	}
	if normalized.DevStory == "" {
		normalized.DevStory = defaults.DevStory
	}
	if normalized.CodeReview == "" {
		normalized.CodeReview = defaults.CodeReview
	}
	if normalized.CodeReviewOptions == "" {
		normalized.CodeReviewOptions = defaults.CodeReviewOptions
	}

	return normalized
}

func PlanPrimaryActions(status, storyNumber string, slashCommands SlashCommandOptions) ([]Action, error) {
	slashCommands = normalizeSlashCommandOptions(slashCommands)

	switch normalizeStatus(status) {
	case "backlog":
		return []Action{
			newAction(buildSlashPrompt(slashCommands.CreateStory, storyNumber, slashCommands.CreateStoryOptions)),
			newAction(buildSlashPrompt(slashCommands.DevStory, storyNumber, slashCommands.DevStoryOptions)),
		}, nil
	case "ready-for-dev", "in-progress":
		return []Action{
			newAction(buildSlashPrompt(slashCommands.DevStory, storyNumber, slashCommands.DevStoryOptions)),
		}, nil
	case "review", "done":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported story status %q", status)
	}
}

func ReviewAction(storyNumber string, slashCommands SlashCommandOptions) Action {
	slashCommands = normalizeSlashCommandOptions(slashCommands)
	return newAction(buildSlashPrompt(slashCommands.CodeReview, storyNumber, slashCommands.CodeReviewOptions))
}

func ShouldContinueReview(status string, published bool) bool {
	return normalizeStatus(status) != "done" || !published
}

func buildSlashPrompt(command, storyNumber, options string) string {
	prompt := strings.TrimSpace(fmt.Sprintf("%s %s", strings.TrimSpace(command), strings.TrimSpace(storyNumber)))
	options = strings.TrimSpace(options)
	if options == "" {
		return prompt
	}
	return fmt.Sprintf("%s %s", prompt, options)
}

func newAction(prompt string) Action {
	return Action{
		Prompt:  prompt,
		Command: fmt.Sprintf(`copilot --yolo --no-ask-user -s -p %q`, prompt),
	}
}
