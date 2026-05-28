package oapi

import (
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model/common"
)

// InstanceListFromDto maps wire list items into public [common.InstanceListItem] values.
func InstanceListFromDto(list *InstanceList) ([]common.InstanceListItem, error) {
	if list == nil {
		return nil, nil
	}
	out := make([]common.InstanceListItem, 0, len(*list))
	for i := range *list {
		it := (*list)[i]
		var id uuid.UUID
		var name string
		var reference string
		if it.InstanceId != nil {
			id = uuid.UUID(*it.InstanceId)
		}
		if it.DisplayName != nil {
			name = *it.DisplayName
		}
		if it.Reference != nil {
			reference = *it.Reference
		}
		out = append(out, common.InstanceListItem{Id: id, Name: name, Reference: reference})
	}
	return out, nil
}
