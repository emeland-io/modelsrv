package oapi

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/node"
)

func nodeTypeViewFromModel(m model.Model, typeID uuid.UUID) NodeTypeView {
	out := NodeTypeView{
		NodeTypeId: uuidToOpenAPI(typeID),
		Resource:   NodeTypeViewResourceNodeType,
	}
	if typeID != uuid.Nil {
		if nt := m.GetNodeTypeById(typeID); nt != nil {
			out.DisplayName = nt.GetDisplayName()
		}
	}
	return out
}

func nodeSummaryToView(m model.Model, baseURL string, n node.Node) NodeSummaryView {
	if n == nil {
		return NodeSummaryView{}
	}
	id := n.GetNodeId()
	out := NodeSummaryView{
		NodeId:      uuidToOpenAPI(id),
		DisplayName: n.GetDisplayName(),
		Reference:   fmt.Sprintf("%s/landscape/nodes/%s", baseURL, id.String()),
	}
	if desc := n.GetDescription(); desc != "" {
		out.Description = &desc
	}
	if typeID := n.GetNodeTypeId(); typeID != uuid.Nil {
		out.NodeType = nodeTypeViewFromModel(m, typeID)
	}
	return out
}

func nodeToView(m model.Model, baseURL string, n node.Node) NodeView {
	summary := nodeSummaryToView(m, baseURL, n)
	return NodeView{
		NodeId:      summary.NodeId,
		DisplayName: summary.DisplayName,
		Description: summary.Description,
		Reference:   summary.Reference,
		NodeType:    summary.NodeType,
		Annotations: AnnotationsToDto(n.GetAnnotations()),
	}
}
