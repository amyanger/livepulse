package alert

import (
	"fmt"

	"github.com/gen2brain/beeep"
)

type DesktopDispatcher struct{}

func NewDesktopDispatcher() *DesktopDispatcher {
	return &DesktopDispatcher{}
}

func (d *DesktopDispatcher) Dispatch(t Transition) {
	title := fmt.Sprintf("livepulse: %s", t.MonitorID)
	var body string
	switch t.To {
	case StatusDown:
		body = fmt.Sprintf("Monitor %s is DOWN", t.MonitorID)
	case StatusRecovered:
		body = fmt.Sprintf("Monitor %s has RECOVERED", t.MonitorID)
	default:
		body = fmt.Sprintf("Monitor %s status: %s", t.MonitorID, t.To)
	}
	_ = beeep.Notify(title, body, "")
}
