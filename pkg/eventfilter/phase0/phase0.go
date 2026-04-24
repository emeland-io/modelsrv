// Package phase0 provides an [eventfilter.FilterFunc] that infers findings for
// phase-0 resources (Node, NodeType, Context, ContextType).
//
// For every relevant Create, Update, or Delete event the filter checks
// referential integrity and upserts or removes a [finding.Finding] in the model:
//
//   - [finding.ContextTypeMissing]    – Context has no type, or its ContextType UUID
//     is not registered in the model.
//   - [finding.ContextParentNotFound] – Context references a parent UUID that is
//     not registered in the model.
//   - [finding.NodeTypeMissing]       – Node has no NodeType assigned.
//
// The original event is always forwarded unchanged; findings are side-effects
// written directly to the model via AddFinding / DeleteFindingById.  Each
// inferred finding references a [finding.FindingType] that is ensured to exist
// in the model (matched by display name, created on demand) so FindingType
// events are emitted alongside findings for replication.
package phase0

import (
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/eventfilter"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/common"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/finding"
	"go.emeland.io/modelsrv/pkg/model/node"
)

// phase0Namespace is the UUID v5 namespace used to derive deterministic finding
// IDs.  Mixing in the FindingKind string ensures that different finding kinds
// for the same subject resource receive distinct UUIDs and can coexist.
var phase0Namespace = uuid.MustParse("7a3f2c1e-4b8d-5e9f-a0b1-c2d3e4f56789")

// findingID computes a deterministic UUID for a finding from the subject
// resource UUID and the finding kind.
func findingID(subjectID uuid.UUID, kind finding.FindingKind) uuid.UUID {
	key := append(subjectID[:], []byte(kind)...)
	return uuid.NewSHA1(phase0Namespace, key)
}

// ensureFindingType returns the id of a [finding.FindingType] for kind.  If the
// model already contains a type whose display name equals the kind string, that
// type's id is used; otherwise a new type is added with the well-known id from
// [finding.TypeIDForKind] and the same string as display name.
func ensureFindingType(m model.Model, kind finding.FindingKind) uuid.UUID {
	name := string(kind)
	if ft := m.GetFindingTypeByName(name); ft != nil {
		return ft.GetFindingTypeId()
	}

	id := finding.TypeIDForKind(kind)
	ft := finding.NewFindingType(m.GetSink(), id)
	ft.SetDisplayName(name)
	if err := m.AddFindingType(ft); err != nil {
		log.Printf("phase0: AddFindingType kind=%s id=%s: %v", kind, id, err)
	}
	return id
}

// upsertFinding creates or replaces a finding in m.
func upsertFinding(m model.Model, kind finding.FindingKind, summary string, resources []*common.ResourceRef) {
	// Derive a stable ID so repeated events produce an upsert, not duplicates.
	subjectID := resources[0].ResourceId
	id := findingID(subjectID, kind)

	f := finding.NewFinding(m.GetSink(), id)
	f.SetFindingTypeById(ensureFindingType(m, kind))
	f.SetSummary(summary)
	f.SetResources(resources)

	if err := m.AddFinding(f, summary); err != nil {
		log.Printf("phase0: AddFinding id=%s kind=%s: %v", id, kind, err)
	}
}

// deleteFinding removes a finding (if present) from m.
func deleteFinding(m model.Model, subjectID uuid.UUID, kind finding.FindingKind) {
	id := findingID(subjectID, kind)
	if m.GetFindingById(id) == nil {
		return
	}
	if err := m.DeleteFindingById(id); err != nil && !errors.Is(err, common.ErrFindingNotFound) {
		log.Printf("phase0: DeleteFindingById id=%s kind=%s: %v", id, kind, err)
	}
}

// NewFilterFunc returns a [eventfilter.FilterFunc] that infers phase-0
// findings.  It always passes the triggering event through unchanged.
func NewFilterFunc() eventfilter.FilterFunc {
	return func(m model.Model, ev events.Event) []events.Event {
		switch ev.Operation {
		case events.CreateOperation, events.UpdateOperation:
			switch ev.ResourceType {
			case events.ContextResource:
				if len(ev.Objects) > 0 {
					if ctx, ok := ev.Objects[0].(mdlctx.Context); ok {
						checkContext(m, ctx)
						reconcileChildContextsForParent(m, ctx.GetContextId())
					}
				}
			case events.NodeResource:
				if len(ev.Objects) > 0 {
					if n, ok := ev.Objects[0].(node.Node); ok {
						checkNode(m, n)
					}
				}
			case events.ContextTypeResource:
				if typeID, ok := contextTypeIDFromEvent(ev); ok {
					reconcileContextsReferencingContextType(m, typeID)
				}
			case events.NodeTypeResource:
				if typeID, ok := nodeTypeIDFromEvent(ev); ok {
					reconcileNodesReferencingNodeType(m, typeID)
				}
			}
		case events.DeleteOperation:
			switch ev.ResourceType {
			case events.ContextResource:
				reconcileChildContextsAfterParentDeleted(m, ev.ResourceId)
			case events.ContextTypeResource:
				reconcileContextsReferencingContextType(m, ev.ResourceId)
			case events.NodeTypeResource:
				reconcileNodesReferencingNodeType(m, ev.ResourceId)
			}
		}

		return []events.Event{ev}
	}
}

func contextTypeIDFromEvent(ev events.Event) (uuid.UUID, bool) {
	if len(ev.Objects) > 0 {
		if ct, ok := ev.Objects[0].(mdlctx.ContextType); ok {
			return ct.GetContextTypeId(), true
		}
	}
	if ev.ResourceId != uuid.Nil {
		return ev.ResourceId, true
	}
	return uuid.Nil, false
}

func nodeTypeIDFromEvent(ev events.Event) (uuid.UUID, bool) {
	if len(ev.Objects) > 0 {
		if nt, ok := ev.Objects[0].(node.NodeType); ok {
			return nt.GetNodeTypeId(), true
		}
	}
	if ev.ResourceId != uuid.Nil {
		return ev.ResourceId, true
	}
	return uuid.Nil, false
}

func reconcileContextsReferencingContextType(m model.Model, typeID uuid.UUID) {
	if typeID == uuid.Nil {
		return
	}
	contexts, err := m.GetContexts()
	if err != nil {
		log.Printf("phase0: GetContexts for ContextType %s: %v", typeID, err)
		return
	}
	for _, c := range contexts {
		if c.GetContextTypeId() != typeID {
			continue
		}
		if cur := m.GetContextById(c.GetContextId()); cur != nil {
			checkContext(m, cur)
		}
	}
}

func reconcileChildContextsForParent(m model.Model, parentID uuid.UUID) {
	if parentID == uuid.Nil {
		return
	}
	contexts, err := m.GetContexts()
	if err != nil {
		log.Printf("phase0: GetContexts for parent %s: %v", parentID, err)
		return
	}
	for _, c := range contexts {
		if c.GetParentId() != parentID {
			continue
		}
		if cur := m.GetContextById(c.GetContextId()); cur != nil {
			checkContextParent(m, cur)
		}
	}
}

func reconcileChildContextsAfterParentDeleted(m model.Model, deletedParentID uuid.UUID) {
	if deletedParentID == uuid.Nil {
		return
	}
	contexts, err := m.GetContexts()
	if err != nil {
		log.Printf("phase0: GetContexts after parent delete %s: %v", deletedParentID, err)
		return
	}
	for _, c := range contexts {
		if c.GetParentId() != deletedParentID {
			continue
		}
		if cur := m.GetContextById(c.GetContextId()); cur != nil {
			checkContextParent(m, cur)
		}
	}
}

func reconcileNodesReferencingNodeType(m model.Model, typeID uuid.UUID) {
	if typeID == uuid.Nil {
		return
	}
	nodes, err := m.GetNodes()
	if err != nil {
		log.Printf("phase0: GetNodes for NodeType %s: %v", typeID, err)
		return
	}
	for _, n := range nodes {
		if n.GetNodeTypeId() != typeID {
			continue
		}
		if cur := m.GetNodeById(n.GetNodeId()); cur != nil {
			checkNode(m, cur)
		}
	}
}

// checkContext evaluates referential integrity for a Context and upserts or
// removes the relevant findings.
func checkContext(m model.Model, ctx mdlctx.Context) {
	checkContextType(m, ctx)
	checkContextParent(m, ctx)
}

func checkContextType(m model.Model, ctx mdlctx.Context) {
	ctxID := ctx.GetContextId()
	typeID := ctx.GetContextTypeId()

	switch {
	case typeID == uuid.Nil:
		upsertFinding(m, finding.ContextTypeMissing,
			fmt.Sprintf("ContextTypeMissing: context %s has no type assigned", ctxID),
			[]*common.ResourceRef{
				{ResourceId: ctxID, ResourceType: events.ContextResource},
			},
		)
	case m.GetContextTypeById(typeID) == nil:
		upsertFinding(m, finding.ContextTypeMissing,
			fmt.Sprintf("ContextTypeMissing: context %s references type %s which does not exist", ctxID, typeID),
			[]*common.ResourceRef{
				{ResourceId: ctxID, ResourceType: events.ContextResource},
				{ResourceId: typeID, ResourceType: events.ContextTypeResource},
			},
		)
	default:
		deleteFinding(m, ctxID, finding.ContextTypeMissing)
	}
}

func checkContextParent(m model.Model, ctx mdlctx.Context) {
	ctxID := ctx.GetContextId()
	parentID := ctx.GetParentId()

	if parentID == uuid.Nil {
		// No parent is not a violation; remove the finding if it was previously set.
		deleteFinding(m, ctxID, finding.ContextParentNotFound)
		return
	}

	if m.GetContextById(parentID) == nil {
		upsertFinding(m, finding.ContextParentNotFound,
			fmt.Sprintf("ContextParentNotFound: context %s references parent %s which does not exist", ctxID, parentID),
			[]*common.ResourceRef{
				{ResourceId: ctxID, ResourceType: events.ContextResource},
				{ResourceId: parentID, ResourceType: events.ContextResource},
			},
		)
	} else {
		deleteFinding(m, ctxID, finding.ContextParentNotFound)
	}
}

func checkNode(m model.Model, n node.Node) {
	nodeID := n.GetNodeId()
	typeID := n.GetNodeTypeId()
	embedded, _ := n.GetNodeType()

	switch {
	case typeID == uuid.Nil:
		if embedded == nil {
			upsertFinding(m, finding.NodeTypeMissing,
				fmt.Sprintf("NodeTypeMissing: node %s has no type assigned", nodeID),
				[]*common.ResourceRef{
					{ResourceId: nodeID, ResourceType: events.NodeResource},
				},
			)
		} else {
			deleteFinding(m, nodeID, finding.NodeTypeMissing)
		}
	case m.GetNodeTypeById(typeID) != nil:
		deleteFinding(m, nodeID, finding.NodeTypeMissing)
	case embedded != nil && embedded.GetNodeTypeId() == typeID:
		// In-memory type object matches the id (e.g. not yet registered via AddNodeType).
		deleteFinding(m, nodeID, finding.NodeTypeMissing)
	default:
		upsertFinding(m, finding.NodeTypeMissing,
			fmt.Sprintf("NodeTypeMissing: node %s references type %s which does not exist", nodeID, typeID),
			[]*common.ResourceRef{
				{ResourceId: nodeID, ResourceType: events.NodeResource},
				{ResourceId: typeID, ResourceType: events.NodeTypeResource},
			},
		)
	}
}
