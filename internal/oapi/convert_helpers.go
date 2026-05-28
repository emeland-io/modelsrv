package oapi

import (
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/annotations"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/component"
	"go.emeland.io/modelsrv/pkg/model/iam"
	"go.emeland.io/modelsrv/pkg/model/system"
)

func refSystem(m model.Model, id uuid.UUID) *system.SystemRef {
	if id == uuid.Nil {
		return nil
	}
	if m != nil {
		if s := m.GetSystemById(id); s != nil {
			return &system.SystemRef{System: s, SystemId: id}
		}
	}
	return &system.SystemRef{SystemId: id}
}

func refAPI(m model.Model, id uuid.UUID) *mdlapi.ApiRef {
	if id == uuid.Nil {
		return nil
	}
	if m != nil {
		if a := m.GetApiById(id); a != nil {
			return &mdlapi.ApiRef{API: a, ApiID: id}
		}
	}
	return &mdlapi.ApiRef{ApiID: id}
}

func refSystemInstance(m model.Model, id uuid.UUID) *system.SystemInstanceRef {
	if id == uuid.Nil {
		return nil
	}
	if m != nil {
		if si := m.GetSystemInstanceById(id); si != nil {
			return &system.SystemInstanceRef{SystemInstance: si, InstanceId: id}
		}
	}
	return &system.SystemInstanceRef{InstanceId: id}
}

func refComponent(m model.Model, id uuid.UUID) *component.ComponentRef {
	if id == uuid.Nil {
		return nil
	}
	if m != nil {
		if c := m.GetComponentById(id); c != nil {
			return &component.ComponentRef{Component: c, ComponentId: id}
		}
	}
	return &component.ComponentRef{ComponentId: id}
}

func refOrgUnit(m model.Model, id uuid.UUID) *iam.OrgUnitRef {
	if id == uuid.Nil {
		return nil
	}
	if m != nil {
		if ou := m.GetOrgUnitById(id); ou != nil {
			return &iam.OrgUnitRef{OrgUnit: ou, OrgUnitId: id}
		}
	}
	return &iam.OrgUnitRef{OrgUnitId: id}
}

func apiRefsFromUUIDs(m model.Model, ids []openapi_types.UUID) []mdlapi.ApiRef {
	out := make([]mdlapi.ApiRef, 0, len(ids))
	for _, x := range ids {
		id := uuid.UUID(x)
		if id == uuid.Nil {
			continue
		}
		ref := refAPI(m, id)
		if ref == nil {
			log.Printf("WARNING: skipping unresolvable API ref %s", id)
			continue
		}
		out = append(out, *ref)
	}
	return out
}

func versionFromDto(v *Version) common.Version {
	if v == nil {
		return common.Version{}
	}
	out := common.Version{Version: v.Version}
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

func parseAPIType(v interface{}) mdlapi.ApiType {
	s, _ := v.(string)
	r, _ := mdlapi.ParseApiType(strings.TrimSpace(s))
	return r
}

// MergeAnnotationsFromDto copies OpenAPI annotations into dst.
func MergeAnnotationsFromDto(dst annotations.Annotations, src *[]Annotation) {
	if src == nil {
		return
	}
	for _, a := range *src {
		dst.Add(a.Key, a.Value)
	}
}

// ResourceTypeFromWireField decodes Finding.ResourceRef.resourceType after JSON decoding.
func ResourceTypeFromWireField(v interface{}) events.ResourceType {
	if v == nil {
		return events.UnknownResourceType
	}
	if s, ok := v.(string); ok {
		if rt := events.ParseResourceType(s); rt != events.UnknownResourceType {
			return rt
		}
	}
	switch n := v.(type) {
	case float64:
		return events.ResourceType(int(n))
	case int:
		return events.ResourceType(n)
	}
	return events.ParseResourceType(fmt.Sprint(v))
}
