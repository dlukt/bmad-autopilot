package orchestrator

import "testing"

func TestPlanPrimaryActionsBacklog(t *testing.T) {
	actions, err := PlanPrimaryActions("backlog", "1-2", DefaultSlashCommandOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
	if actions[0].Command != `copilot --yolo --no-ask-user -s -p "/bmad-create-story 1-2"` {
		t.Fatalf("unexpected first command: %q", actions[0].Command)
	}
	if actions[1].Command != `copilot --yolo --no-ask-user -s -p "/bmad-dev-story 1-2"` {
		t.Fatalf("unexpected second command: %q", actions[1].Command)
	}
}

func TestPlanPrimaryActionsReadyForDev(t *testing.T) {
	actions, err := PlanPrimaryActions("ready-for-dev", "3-4", DefaultSlashCommandOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Command != `copilot --yolo --no-ask-user -s -p "/bmad-dev-story 3-4"` {
		t.Fatalf("unexpected command: %q", actions[0].Command)
	}
}

func TestPlanPrimaryActionsWithCustomOptions(t *testing.T) {
	options := DefaultSlashCommandOptions()
	options.CreateStoryOptions = "--template backend"
	options.DevStoryOptions = "--agent senior --yolo"

	actions, err := PlanPrimaryActions("backlog", "2-7", options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
	if actions[0].Prompt != "/bmad-create-story 2-7 --template backend" {
		t.Fatalf("unexpected create-story prompt: %q", actions[0].Prompt)
	}
	if actions[1].Prompt != "/bmad-dev-story 2-7 --agent senior --yolo" {
		t.Fatalf("unexpected dev-story prompt: %q", actions[1].Prompt)
	}
}

func TestReviewActionDefaultsAndCustomOptions(t *testing.T) {
	defaultAction := ReviewAction("5-3", DefaultSlashCommandOptions())
	if defaultAction.Prompt != "/bmad-code-review 5-3 "+defaultCodeReviewOptions {
		t.Fatalf("unexpected default review prompt: %q", defaultAction.Prompt)
	}

	customOptions := DefaultSlashCommandOptions()
	customOptions.CodeReviewOptions = "--strict --fix"
	customAction := ReviewAction("5-3", customOptions)
	if customAction.Prompt != "/bmad-code-review 5-3 --strict --fix" {
		t.Fatalf("unexpected custom review prompt: %q", customAction.Prompt)
	}
}

func TestShouldContinueReview(t *testing.T) {
	if !ShouldContinueReview("review", false) {
		t.Fatal("expected review status to continue")
	}
	if !ShouldContinueReview("done", false) {
		t.Fatal("expected done without push evidence to continue")
	}
	if ShouldContinueReview("done", true) {
		t.Fatal("expected done with push evidence to stop")
	}
}
