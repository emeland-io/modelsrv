package endpointprobe

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/common"
)

// ModelClient adapts an in-memory [model.Model] to [ApiInstanceClient]
// so the probe scheduler can run inside modelsrv without HTTP.
type ModelClient struct {
	Model model.Model
}

// NewModelClient returns an ApiInstanceClient backed by m.
func NewModelClient(m model.Model) *ModelClient {
	return &ModelClient{Model: m}
}

// GetApiInstances implements [ApiInstanceClient].
func (c *ModelClient) GetApiInstances() ([]common.InstanceListItem, error) {
	instances, err := c.Model.GetApiInstances()
	if err != nil {
		return nil, err
	}
	items := make([]common.InstanceListItem, 0, len(instances))
	for _, ai := range instances {
		items = append(items, common.InstanceListItem{
			Id:   ai.GetInstanceId(),
			Name: ai.GetDisplayName(),
		})
	}
	return items, nil
}

// GetApiInstanceById implements [ApiInstanceClient].
func (c *ModelClient) GetApiInstanceById(id uuid.UUID) (api.ApiInstance, error) {
	ai := c.Model.GetApiInstanceById(id)
	if ai == nil {
		return nil, common.ErrApiInstanceNotFound
	}
	return ai, nil
}
