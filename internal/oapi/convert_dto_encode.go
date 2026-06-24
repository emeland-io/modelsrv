package oapi

import (
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.emeland.io/modelsrv/pkg/model/annotations"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlprod "go.emeland.io/modelsrv/pkg/model/product"
)

func AnnotationsToDto(a annotations.Annotations) *[]Annotation {
	if a == nil {
		return nil
	}
	out := make([]Annotation, 0)
	for key := range a.GetKeys() {
		out = append(out, Annotation{Key: key, Value: a.GetValue(key)})
	}
	if len(out) == 0 {
		return nil
	}
	return &out
}

func resourceRefsToDto(refs []*common.ResourceRef) []ResourceRef {
	out := make([]ResourceRef, 0, len(refs))
	for _, r := range refs {
		if r == nil {
			continue
		}
		out = append(out, ResourceRef{
			ResourceId:   openapi_types.UUID(r.ResourceId),
			ResourceType: r.ResourceType.String(),
		})
	}
	return out
}

func versionToDto(v common.Version) *Version {
	if v.Version == "" && v.AvailableFrom == nil && v.DeprecatedFrom == nil && v.TerminatedFrom == nil {
		return nil
	}
	out := &Version{Version: v.Version}
	if v.AvailableFrom != nil {
		t := *v.AvailableFrom
		out.AvailableFrom = &t
	}
	if v.DeprecatedFrom != nil {
		t := *v.DeprecatedFrom
		out.DeprecatedFrom = &t
	}
	if v.TerminatedFrom != nil {
		t := *v.TerminatedFrom
		out.TerminatedFrom = &t
	}
	return out
}

func productionVersionsToDto(in []mdlprod.ProductionVersion) *[]ProductionVersion {
	if len(in) == 0 {
		return nil
	}
	out := make([]ProductionVersion, len(in))
	for i := range in {
		out[i].AvailableFrom = in[i].AvailableFrom
		out[i].DeprecatedFrom = in[i].DeprecatedFrom
		out[i].TerminatedFrom = in[i].TerminatedFrom
		if len(in[i].Artefacts) > 0 {
			arts := make([]openapi_types.UUID, len(in[i].Artefacts))
			for j, a := range in[i].Artefacts {
				arts[j] = openapi_types.UUID(a)
			}
			out[i].Artefacts = &arts
		}
	}
	return &out
}

func uuidToOpenAPI(id uuid.UUID) openapi_types.UUID {
	return openapi_types.UUID(id)
}

func uuidPtr(id uuid.UUID) *openapi_types.UUID {
	if id == uuid.Nil {
		return nil
	}
	u := openapi_types.UUID(id)
	return &u
}
