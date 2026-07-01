package oapi

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/finding"
)

func findingToView(m model.Model, baseURL string, f finding.Finding) FindingView {
	if f == nil {
		return FindingView{}
	}
	id := f.GetFindingId()
	out := FindingView{
		FindingId:   uuidToOpenAPI(id),
		DisplayName: f.GetDisplayName(),
		Reference:   fmt.Sprintf("%s/landscape/findings/%s", baseURL, id.String()),
		Annotations: AnnotationsToDto(f.GetAnnotations()),
	}
	if desc := f.GetDescription(); desc != "" {
		out.Description = &desc
	}
	if typeID := f.GetFindingTypeId(); typeID != uuid.Nil {
		typeView := FindingTypeView{
			FindingTypeId: uuidToOpenAPI(typeID),
		}
		if ft := m.GetFindingTypeById(typeID); ft != nil {
			typeView.DisplayName = ft.GetDisplayName()
			if desc := ft.GetDescription(); desc != "" {
				typeView.Description = &desc
			}
		}
		out.FindingType = typeView
	}
	refs := f.GetResources()
	out.Resources = make([]ResourceView, 0, len(refs))
	for _, ref := range refs {
		if ref == nil {
			continue
		}
		out.Resources = append(out.Resources, resourceViewFromRef(m, ref))
	}
	return out
}

func resourceViewFromRef(m model.Model, ref *common.ResourceRef) ResourceView {
	displayName := model.ResourceDisplayName(m, ref)
	out := ResourceView{
		Id:           uuidToOpenAPI(ref.ResourceId),
		ResourceType: ref.ResourceType.String(),
	}
	if displayName != "" {
		out.DisplayName = &displayName
	}
	return out
}
