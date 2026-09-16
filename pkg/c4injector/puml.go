package c4injector

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// RenderContext emits C4-PlantUML for level 1 (System Context).
func RenderContext(v ContextView) string {
	var b strings.Builder
	b.WriteString("@startuml\n")
	b.WriteString("!include <C4/C4_Context>\n")
	fmt.Fprintf(&b, "title %s — Level 1: System Context\n\n", escapeTitle(v.Landscape.Name))

	fmt.Fprintf(&b, "Enterprise_Boundary(landscape, %s) {\n", quote(v.Landscape.Name))
	for _, root := range v.Roots {
		writeContextBoundary(&b, root, 1)
	}
	b.WriteString("}\n\n")

	if desc := strings.TrimSpace(v.Landscape.Description); desc != "" {
		b.WriteString("note as landscape_summary\n")
		for _, line := range strings.Split(desc, "\n") {
			fmt.Fprintf(&b, "  %s\n", line)
		}
		b.WriteString("end note\n")
	}
	b.WriteString("@enduml\n")
	return b.String()
}

func writeContextBoundary(b *strings.Builder, n ContextNode, indent int) {
	pad := strings.Repeat("  ", indent)
	typeName := n.TypeName
	if typeName == "" {
		typeName = "Context"
	}
	fmt.Fprintf(b, "%sBoundary(%s, %s, %s) {\n",
		pad, alias("ctx", n.ID), quote(n.DisplayName), quote(typeName))
	for _, child := range n.Children {
		writeContextBoundary(b, child, indent+1)
	}
	fmt.Fprintf(b, "%s}\n", pad)
}

// RenderContainer emits C4-PlantUML for level 2: EmELand Components as C4
// Containers, grouped in a System_Boundary per concrete System.
func RenderContainer(v ContainerView) string {
	var b strings.Builder
	b.WriteString("@startuml\n")
	b.WriteString("!include <C4/C4_Container>\n")
	b.WriteString("title Level 2: Container Diagram\n\n")

	for _, e := range v.Externals {
		fmt.Fprintf(&b, "System_Ext(%s, %s, %s)\n",
			alias("sys", e.ID), quote(e.DisplayName), quote(e.Description))
	}

	for _, s := range v.Systems {
		fmt.Fprintf(&b, "System_Boundary(%s, %s) {\n", alias("sys", s.ID), quote(s.DisplayName))
		for _, c := range s.Containers {
			fmt.Fprintf(&b, "  Container(%s, %s, %s, %s)\n",
				alias("container", c.ID), quote(c.DisplayName), quote(c.Technology),
				quoteLines(c.Description, instanceCountLabel(c.InstanceCount)))
		}
		b.WriteString("}\n")
	}
	b.WriteString("\n")

	for _, e := range v.Edges {
		from := aliasForKind(e.FromKind, e.FromID)
		to := aliasForKind(e.ToKind, e.ToID)
		if e.Tech != "" {
			fmt.Fprintf(&b, "Rel(%s, %s, %s, %s)\n", from, to, quote(e.Label), quote(e.Tech))
		} else {
			fmt.Fprintf(&b, "Rel(%s, %s, %s)\n", from, to, quote(e.Label))
		}
	}
	b.WriteString("@enduml\n")
	return b.String()
}

// instanceCountLabel renders the deployed-instance count for a level-2 Container.
// Instances themselves stay off the type-level diagram; only the count crosses over,
// and a zero count is stated explicitly so an undeployed Component is obvious.
func instanceCountLabel(count int) string {
	switch count {
	case 0:
		return "[no instances]"
	case 1:
		return "[1 instance]"
	default:
		return fmt.Sprintf("[%d instances]", count)
	}
}

// RenderDeployment emits C4-PlantUML for the deployment (instance) diagram.
func RenderDeployment(v DeploymentView) string {
	var b strings.Builder
	b.WriteString("@startuml\n")
	b.WriteString("!include <C4/C4_Container>\n")
	b.WriteString(endpointTagDef)
	b.WriteString("title Deployment Diagram (instances)\n\n")

	for _, ctx := range v.Contexts {
		ctxAlias := "ctx_unscoped"
		if ctx.ID != uuid.Nil {
			ctxAlias = alias("ctx", ctx.ID)
		}
		typeName := ctx.TypeName
		if typeName == "" {
			typeName = "Context"
		}
		fmt.Fprintf(&b, "Boundary(%s, %s, %s) {\n", ctxAlias, quote(ctx.DisplayName), quote(typeName))
		for _, si := range ctx.SystemInstances {
			label := si.DisplayName
			if si.SystemName != "" {
				label = si.DisplayName + " [" + si.SystemName + "]"
			}
			fmt.Fprintf(&b, "  System_Boundary(%s, %s) {\n", alias("si", si.ID), quote(label))
			for _, c := range si.Containers {
				fmt.Fprintf(&b, "    Container(%s, %s, %s, %s)\n",
					alias("ci", c.ID), quote(c.DisplayName), quote(""), quote(c.Description))
			}
			for _, e := range si.Endpoints {
				writeEndpoint(&b, e, "    ")
			}
			b.WriteString("  }\n")
		}
		for _, c := range ctx.LooseContainers {
			fmt.Fprintf(&b, "  Container(%s, %s, %s, %s)\n",
				alias("ci", c.ID), quote(c.DisplayName), quote(""), quote(c.Description))
		}
		for _, e := range ctx.LooseEndpoints {
			writeEndpoint(&b, e, "  ")
		}
		b.WriteString("}\n")
	}
	b.WriteString("\n")

	for _, e := range v.Edges {
		from := alias("ci", e.FromID)
		to := alias("ci", e.ToID)
		if e.Tech != "" {
			fmt.Fprintf(&b, "Rel(%s, %s, %s, %s)\n", from, to, quote(e.Label), quote(e.Tech))
		} else {
			fmt.Fprintf(&b, "Rel(%s, %s, %s)\n", from, to, quote(e.Label))
		}
	}
	b.WriteString("@enduml\n")
	return b.String()
}

// endpointTagDef styles ApiInstances distinctly from ComponentInstances, which
// share the Container primitive on the deployment diagram.
const endpointTagDef = `AddElementTag("endpoint", $bgColor="#08427B", $fontColor="#ffffff", $borderColor="#052E56")` + "\n"

// writeEndpoint emits an ApiInstance. The label carries the API type name when the
// ref resolves, so an ApiInstance pointing at a missing API is visibly incomplete.
func writeEndpoint(b *strings.Builder, e InstanceEndpoint, indent string) {
	label := e.DisplayName
	if e.APIName != "" {
		label += " [" + e.APIName + "]"
	}
	fmt.Fprintf(b, "%sContainer(%s, %s, %s, %s, $tags=\"endpoint\")\n",
		indent, alias("ai", e.ID), quote(label), quote(e.Technology), quote(e.Address))
}

func aliasForKind(kind string, id uuid.UUID) string {
	switch kind {
	case "container":
		return alias("container", id)
	default:
		return alias("sys", id)
	}
}

// alias builds a PlantUML-safe identifier from a prefix and UUID.
func alias(prefix string, id uuid.UUID) string {
	s := strings.ReplaceAll(id.String(), "-", "_")
	return prefix + "_" + s
}

// escapePUML escapes backslashes and double quotes for a PlantUML string body.
func escapePUML(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// quote wraps s in double quotes and escapes embedded quotes for PlantUML.
func quote(s string) string {
	return `"` + escapePUML(s) + `"`
}

// quoteLines joins non-empty parts into one quoted string separated by PlantUML
// line breaks. The separator is written after escaping so it stays a line break
// rather than becoming a literal backslash-n.
func quoteLines(parts ...string) string {
	escaped := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		escaped = append(escaped, escapePUML(p))
	}
	return `"` + strings.Join(escaped, `\n\n`) + `"`
}

func escapeTitle(s string) string {
	return strings.ReplaceAll(s, "\n", " ")
}
