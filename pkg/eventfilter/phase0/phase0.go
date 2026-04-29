// Package phase0 provides a phase-0 [eventfilter.FilterFunc] for referential
// checks on Context, ContextType, Node, and NodeType, as [finding.Finding] values.
//
// [finding.ContextTypeMissing], [finding.ContextParentNotFound], [finding.NodeTypeMissing]
// cover missing/unknown type, missing parent, and missing/invalid node type.
//
// The incoming event is always returned as-is. Findings are created or removed
// with AddFinding / DeleteFindingById. [finding.FindingType] is ensured
// (by name or [finding.TypeIDForKind]) for replication.
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

// SHA-1 namespace so each (subject id, kind) maps to one finding id.
var phase0Namespace = uuid.MustParse("7a3f2c1e-4b8d-5e9f-a0b1-c2d3e4f56789")

func findingID(subjectID uuid.UUID, kind finding.FindingKind) uuid.UUID {
	key := append(subjectID[:], []byte(kind)...)
	return uuid.NewSHA1(phase0Namespace, key)
}

// FindingType id for kind: existing match by name, else create with [finding.TypeIDForKind].
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

func upsertFinding(m model.Model, kind finding.FindingKind, summary string, resources []*common.ResourceRef) {
	subjectID := resources[0].ResourceId // idempotent: same subject+kind reuses id
	id := findingID(subjectID, kind)

	f := finding.NewFinding(m.GetSink(), id)
	f.SetFindingTypeById(ensureFindingType(m, kind))
	f.SetSummary(summary)
	f.SetResources(resources)

	if err := m.AddFinding(f, summary); err != nil {
		log.Printf("phase0: AddFinding id=%s kind=%s: %v", id, kind, err)
	}
}

func deleteFinding(m model.Model, subjectID uuid.UUID, kind finding.FindingKind) {
	id := findingID(subjectID, kind)
	if m.GetFindingById(id) == nil {
		return
	}
	if err := m.DeleteFindingById(id); err != nil && !errors.Is(err, common.ErrFindingNotFound) {
		log.Printf("phase0: DeleteFindingById id=%s kind=%s: %v", id, kind, err)
	}
}

// NewFilterFunc returns the phase-0 filter; the trigger event is always passed through.
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
		deleteFinding(m, ctxID, finding.ContextParentNotFound) // no parent is valid
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
		deleteFinding(m, nodeID, finding.NodeTypeMissing) // embedded type matches id before registry
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
