package orchestrator

import "sync/atomic"

type StopChecker interface {
	ShouldStop() bool
}

type StopController struct {
	requested     atomic.Bool
	poweroffArmed atomic.Bool
}

func NewStopController() *StopController {
	return &StopController{}
}

func (c *StopController) RequestStop() {
	if c == nil {
		return
	}
	c.requested.Store(true)
}

func (c *StopController) CancelStop() {
	if c == nil {
		return
	}
	c.requested.Store(false)
}

func (c *StopController) ShouldStop() bool {
	if c == nil {
		return false
	}
	return c.requested.Load()
}

func (c *StopController) TogglePoweroff() bool {
	if c == nil {
		return false
	}

	next := !c.poweroffArmed.Load()
	c.poweroffArmed.Store(next)
	return next
}

func (c *StopController) PoweroffArmed() bool {
	if c == nil {
		return false
	}
	return c.poweroffArmed.Load()
}

type neverStopChecker struct{}

func (neverStopChecker) ShouldStop() bool {
	return false
}

func withDefaultStopChecker(checker StopChecker) StopChecker {
	if checker == nil {
		return neverStopChecker{}
	}
	return checker
}
