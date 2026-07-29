package endpointprobe

import (
	"context"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.uber.org/zap"
)

// ApiInstanceClient lists and fetches ApiInstances from modelsrv.
type ApiInstanceClient interface {
	GetApiInstances() ([]common.InstanceListItem, error)
	GetApiInstanceById(id uuid.UUID) (api.ApiInstance, error)
}

// CollectTargets lists ApiInstances, derives probe targets via TargetFromApiInstance,
// and dedupes by DedupeKey (host:port, first wins). Skips instances without endpoint
// host annotations and logs warnings for fetch/annotation errors.
func CollectTargets(ctx context.Context, client ApiInstanceClient, log *zap.SugaredLogger) ([]ProbeTarget, error) {
	if log == nil {
		log = zap.NewNop().Sugar()
	}

	items, err := client.GetApiInstances()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	targets := make([]ProbeTarget, 0, len(items))

	for _, item := range items {
		if ctx.Err() != nil {
			break
		}

		ai, err := client.GetApiInstanceById(item.Id)
		if err != nil {
			log.Warnw("failed to fetch api instance",
				"apiInstanceId", item.Id,
				"error", err,
			)
			continue
		}

		target, ok, err := TargetFromApiInstance(ai)
		if err != nil {
			log.Warnw("invalid endpoint annotations",
				"apiInstanceId", item.Id,
				"error", err,
			)
			continue
		}
		if !ok {
			continue
		}

		if _, dup := seen[target.DedupeKey]; dup {
			continue
		}
		seen[target.DedupeKey] = struct{}{}
		targets = append(targets, target)
	}

	return targets, nil
}
