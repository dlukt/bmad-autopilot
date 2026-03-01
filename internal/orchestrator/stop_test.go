package orchestrator

import "testing"

func TestStopControllerPoweroffToggle(t *testing.T) {
	controller := NewStopController()

	if controller.PoweroffArmed() {
		t.Fatal("expected poweroff to start disarmed")
	}

	if armed := controller.TogglePoweroff(); !armed {
		t.Fatal("expected first toggle to arm poweroff")
	}
	if !controller.PoweroffArmed() {
		t.Fatal("expected poweroff to be armed")
	}

	if armed := controller.TogglePoweroff(); armed {
		t.Fatal("expected second toggle to disarm poweroff")
	}
	if controller.PoweroffArmed() {
		t.Fatal("expected poweroff to be disarmed")
	}
}

func TestStopControllerRequestCancelStop(t *testing.T) {
	controller := NewStopController()

	if controller.ShouldStop() {
		t.Fatal("expected stop to start disabled")
	}

	controller.RequestStop()
	if !controller.ShouldStop() {
		t.Fatal("expected stop to be requested")
	}

	controller.CancelStop()
	if controller.ShouldStop() {
		t.Fatal("expected stop to be canceled")
	}
}
