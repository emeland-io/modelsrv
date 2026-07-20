package endpointprobe

import (
	"go.emeland.io/modelsrv/pkg/eventfilter"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

// NewRescanFilter returns a passthrough event filter that triggers debounced
// certprobe rescans when ApiInstance resources are created, updated, or deleted.
// Annotation changes on an ApiInstance arrive as ApiInstance Update events.
func NewRescanFilter(s *Scheduler) eventfilter.Filter {
	return eventfilter.Filter{
		DisplayName: "certprobe-rescan",
		Description: "Triggers debounced certprobe rescans on ApiInstance changes",
		Fn:          rescanFilterFunc(s),
	}
}

func rescanFilterFunc(s *Scheduler) eventfilter.FilterFunc {
	return func(_ model.Model, ev events.Event) []events.Event {
		if ev.ResourceType == events.APIInstanceResource && ev.Operation != events.UnknownOperation {
			s.Notify()
		}
		return []events.Event{ev}
	}
}
