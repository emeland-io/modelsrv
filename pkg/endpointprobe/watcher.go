package endpointprobe

import (
	"go.emeland.io/modelsrv/pkg/eventfilter"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/api"
)

// NewConfigSyncFilter returns a passthrough event filter that keeps an OTel
// collector config in sync with ApiInstance endpoint annotations.
func NewConfigSyncFilter(w *ConfigWriter) eventfilter.Filter {
	return eventfilter.Filter{
		DisplayName: "http_check-config-sync",
		Description: "Maintains the OTel http_check collector config from ApiInstance endpoint annotations",
		Fn:          configSyncFilterFunc(w),
	}
}

func configSyncFilterFunc(w *ConfigWriter) eventfilter.FilterFunc {
	return func(_ model.Model, ev events.Event) []events.Event {
		if ev.ResourceType != events.APIInstanceResource {
			return []events.Event{ev}
		}

		switch ev.Operation {
		case events.DeleteOperation:
			w.Remove(ev.ResourceId)
		case events.CreateOperation, events.UpdateOperation:
			if len(ev.Objects) == 0 {
				return []events.Event{ev}
			}
			ai, ok := ev.Objects[0].(api.ApiInstance)
			if !ok {
				return []events.Event{ev}
			}
			target, hasTarget, err := TargetFromApiInstance(ai)
			if err != nil {
				w.logger.Warnw("invalid endpoint annotations; removing target",
					"apiInstanceId", ai.GetInstanceId(),
					"error", err,
				)
				w.Remove(ai.GetInstanceId())
				return []events.Event{ev}
			}
			if !hasTarget {
				w.Remove(ai.GetInstanceId())
				return []events.Event{ev}
			}
			w.Upsert(target)
		}

		return []events.Event{ev}
	}
}
