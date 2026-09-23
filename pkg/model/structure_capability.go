package model

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	mdlcapability "go.emeland.io/modelsrv/pkg/model/capability"
	"go.emeland.io/modelsrv/pkg/model/common"
)

func validateCapability(c mdlcapability.Capability, m *modelData) error {
	if c == nil {
		return fmt.Errorf("capability is nil")
	}
	if c.GetCapabilityId() == uuid.Nil {
		return common.ErrUUIDNotSet
	}
	for _, ref := range c.GetVersions() {
		vid := ref.CapabilityVersionId
		if ref.CapabilityVersion != nil {
			vid = ref.CapabilityVersion.GetCapabilityVersionId()
		}
		if vid == uuid.Nil {
			return fmt.Errorf("capability versions entry has nil capabilityVersionId")
		}
		if m.GetCapabilityVersionById(vid) == nil {
			return common.ErrCapabilityVersionNotFound
		}
	}
	return nil
}

func validateCapabilityVersion(cv mdlcapability.CapabilityVersion, m *modelData) error {
	if cv == nil {
		return fmt.Errorf("capability version is nil")
	}
	if cv.GetCapabilityVersionId() == uuid.Nil {
		return common.ErrUUIDNotSet
	}
	capID := cv.GetCapabilityId()
	if capID == uuid.Nil {
		return fmt.Errorf("capabilityRef is required")
	}
	if m.GetCapabilityById(capID) == nil {
		return common.ErrCapabilityNotFound
	}
	return nil
}

func validateParameterValueSet(pvs mdlcapability.ParameterValueSet, m *modelData) error {
	if pvs.ParameterId == uuid.Nil {
		return fmt.Errorf("parameterId is required in parameter value set")
	}
	if m.GetParameterById(pvs.ParameterId) == nil {
		return common.ErrParameterNotFound
	}
	for _, vid := range pvs.ValidValueIds {
		vv := m.GetValidValueById(vid)
		if vv == nil {
			return common.ErrValidValueNotFound
		}
		if vv.GetParameterId() != pvs.ParameterId {
			return fmt.Errorf("valid value %s does not belong to parameter %s", vid, pvs.ParameterId)
		}
	}
	return nil
}

func validValueIDsContainAll(provided, required []uuid.UUID) bool {
	set := make(map[uuid.UUID]struct{}, len(provided))
	for _, id := range provided {
		set[id] = struct{}{}
	}
	for _, id := range required {
		if _, ok := set[id]; !ok {
			return false
		}
	}
	return true
}

func variantProvidesRequiredSuperset(v mdlcapability.Variant, required []mdlcapability.ParameterValueSet) bool {
	byParam := make(map[uuid.UUID][]uuid.UUID, len(v.GetProvides()))
	for _, pvs := range v.GetProvides() {
		byParam[pvs.ParameterId] = pvs.ValidValueIds
	}
	for _, req := range required {
		provided, ok := byParam[req.ParameterId]
		if !ok || !validValueIDsContainAll(provided, req.ValidValueIds) {
			return false
		}
	}
	return true
}

func variantDependencySatisfied(dep mdlcapability.VariantDependency, m *modelData) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, v := range m.variantsByUUID {
		cv := m.capabilityVersionsByUUID[v.GetCapabilityVersionId()]
		if cv == nil {
			continue
		}
		if cv.GetCapabilityId() != dep.CapabilityId {
			continue
		}
		if variantProvidesRequiredSuperset(v, dep.Required) {
			return true
		}
	}
	return false
}

func capabilityOwnerOfVariantUnlocked(v mdlcapability.Variant, m *modelData) uuid.UUID {
	cv := m.capabilityVersionsByUUID[v.GetCapabilityVersionId()]
	if cv == nil {
		return uuid.Nil
	}
	return cv.GetCapabilityId()
}

func capabilityDependencyEdges(m *modelData, including mdlcapability.Variant) map[uuid.UUID][]uuid.UUID {
	edges := make(map[uuid.UUID][]uuid.UUID)
	add := func(v mdlcapability.Variant) {
		from := capabilityOwnerOfVariantUnlocked(v, m)
		if from == uuid.Nil {
			return
		}
		for _, dep := range v.GetDependencies() {
			if dep.CapabilityId == uuid.Nil {
				continue
			}
			edges[from] = append(edges[from], dep.CapabilityId)
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, v := range m.variantsByUUID {
		if including != nil && v.GetVariantId() == including.GetVariantId() {
			continue
		}
		add(v)
	}
	if including != nil {
		add(including)
	}
	return edges
}

func capabilityDependencyGraphHasCycle(edges map[uuid.UUID][]uuid.UUID) bool {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[uuid.UUID]int)
	var visit func(uuid.UUID) bool
	visit = func(n uuid.UUID) bool {
		color[n] = gray
		for _, to := range edges[n] {
			switch color[to] {
			case gray:
				return true
			case white:
				if visit(to) {
					return true
				}
			}
		}
		color[n] = black
		return false
	}
	for n := range edges {
		if color[n] == white {
			if visit(n) {
				return true
			}
		}
	}
	// also visit nodes that only appear as targets
	for _, tos := range edges {
		for _, to := range tos {
			if color[to] == white {
				if visit(to) {
					return true
				}
			}
		}
	}
	return false
}

func validateVariant(v mdlcapability.Variant, m *modelData) error {
	if v == nil {
		return fmt.Errorf("variant is nil")
	}
	if v.GetVariantId() == uuid.Nil {
		return common.ErrUUIDNotSet
	}
	cvID := v.GetCapabilityVersionId()
	if cvID == uuid.Nil {
		return fmt.Errorf("capabilityVersionRef is required")
	}
	if m.GetCapabilityVersionById(cvID) == nil {
		return common.ErrCapabilityVersionNotFound
	}
	for _, pvs := range v.GetProvides() {
		if err := validateParameterValueSet(pvs, m); err != nil {
			return err
		}
	}
	for _, dep := range v.GetDependencies() {
		if dep.CapabilityId == uuid.Nil {
			return fmt.Errorf("dependency capabilityId is required")
		}
		if m.GetCapabilityById(dep.CapabilityId) == nil {
			return common.ErrCapabilityNotFound
		}
		for _, pvs := range dep.Required {
			if err := validateParameterValueSet(pvs, m); err != nil {
				return err
			}
		}
		if !variantDependencySatisfied(dep, m) {
			return common.ErrVariantDependencyUnsatisfied
		}
	}
	edges := capabilityDependencyEdges(m, v)
	if capabilityDependencyGraphHasCycle(edges) {
		return common.ErrVariantDependencyCycle
	}
	return nil
}

// AddCapability implements [Model].
func (m *modelData) AddCapability(capability mdlcapability.Capability) error {
	if err := validateCapability(capability, m); err != nil {
		return err
	}
	return addEventEnabled(m, capability, mdlcapability.Capability.GetCapabilityId,
		func(x mdlcapability.Capability, s events.EventSink) { x.Register(s) },
		m.capabilitiesByUUID, events.CapabilityResource)
}

// DeleteCapabilityById implements [Model].
func (m *modelData) DeleteCapabilityById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.capabilitiesByUUID, events.CapabilityResource, common.ErrCapabilityNotFound)
}

// GetCapabilityById implements [Model].
func (m *modelData) GetCapabilityById(id uuid.UUID) mdlcapability.Capability {
	return getEventEnabled(m, id, m.capabilitiesByUUID)
}

// GetCapabilities implements [Model].
func (m *modelData) GetCapabilities() ([]mdlcapability.Capability, error) {
	return getAllEventEnabled(m, m.capabilitiesByUUID)
}

// AddCapabilityVersion implements [Model].
func (m *modelData) AddCapabilityVersion(cv mdlcapability.CapabilityVersion) error {
	if err := validateCapabilityVersion(cv, m); err != nil {
		return err
	}
	return addEventEnabled(m, cv, mdlcapability.CapabilityVersion.GetCapabilityVersionId,
		func(x mdlcapability.CapabilityVersion, s events.EventSink) { x.Register(s) },
		m.capabilityVersionsByUUID, events.CapabilityVersionResource)
}

// DeleteCapabilityVersionById implements [Model].
func (m *modelData) DeleteCapabilityVersionById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.capabilityVersionsByUUID, events.CapabilityVersionResource, common.ErrCapabilityVersionNotFound)
}

// GetCapabilityVersionById implements [Model].
func (m *modelData) GetCapabilityVersionById(id uuid.UUID) mdlcapability.CapabilityVersion {
	return getEventEnabled(m, id, m.capabilityVersionsByUUID)
}

// GetCapabilityVersions implements [Model].
func (m *modelData) GetCapabilityVersions() ([]mdlcapability.CapabilityVersion, error) {
	return getAllEventEnabled(m, m.capabilityVersionsByUUID)
}

// AddVariant implements [Model].
func (m *modelData) AddVariant(v mdlcapability.Variant) error {
	if err := validateVariant(v, m); err != nil {
		return err
	}
	return addEventEnabled(m, v, mdlcapability.Variant.GetVariantId,
		func(x mdlcapability.Variant, s events.EventSink) { x.Register(s) },
		m.variantsByUUID, events.VariantResource)
}

// DeleteVariantById implements [Model].
func (m *modelData) DeleteVariantById(id uuid.UUID) error {
	return deleteEventEnabled(m, id, m.variantsByUUID, events.VariantResource, common.ErrVariantNotFound)
}

// GetVariantById implements [Model].
func (m *modelData) GetVariantById(id uuid.UUID) mdlcapability.Variant {
	return getEventEnabled(m, id, m.variantsByUUID)
}

// GetVariants implements [Model].
func (m *modelData) GetVariants() ([]mdlcapability.Variant, error) {
	return getAllEventEnabled(m, m.variantsByUUID)
}
