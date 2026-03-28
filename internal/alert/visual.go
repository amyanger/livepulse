package alert

import "fmt"

type VisualDispatcher struct{}

func NewVisualDispatcher() *VisualDispatcher {
	return &VisualDispatcher{}
}

func (d *VisualDispatcher) Dispatch(t Transition) {
	fmt.Print("\a")
}
