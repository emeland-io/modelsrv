//go:build load_n_scale

// Package loadscale_test exercises modelsrv with large numbers of resources and
// changes, to observe how memory consumption grows.
//
// It is gated behind the "load_n_scale" build tag (Go build constraints do not
// allow hyphens, so "load-n-scale" is not a valid tag name) so it never runs as
// part of the normal test suite; invoke it explicitly via `make test-load-scale`.
package loadscale_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"go.emeland.io/modelsrv/pkg/backend"
	"go.emeland.io/modelsrv/pkg/model"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/artifact"
	mdlcapability "go.emeland.io/modelsrv/pkg/model/capability"
	mdlcap "go.emeland.io/modelsrv/pkg/model/capacity"
	"go.emeland.io/modelsrv/pkg/model/component"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/finding"
	"go.emeland.io/modelsrv/pkg/model/iam"
	"go.emeland.io/modelsrv/pkg/model/node"
	mdlparameter "go.emeland.io/modelsrv/pkg/model/parameter"
	mdlproduct "go.emeland.io/modelsrv/pkg/model/product"
	"go.emeland.io/modelsrv/pkg/model/system"
)

// Environment variables used to configure the test. Each *_COUNTS variable
// takes a comma-separated list of positive integers, sorted least to
// greatest (e.g. "10,100,1000").
const (
	envInstanceCounts = "LOAD_SCALE_INSTANCE_COUNTS"
	envChangeCounts   = "LOAD_SCALE_CHANGE_COUNTS"
	envReportPath     = "LOAD_SCALE_REPORT_PATH"

	defaultInstanceCounts = "10,100,1000"
	defaultChangeCounts   = "10,100,1000"
	defaultReportPath     = "load_scale_report.md"
)

// parseCountList reads a comma-separated list of positive integers from the
// given environment variable (or defaultCSV if unset), and requires the
// values to be sorted least to greatest.
func parseCountList(t *testing.T, envVar, defaultCSV string) []int {
	t.Helper()

	raw := os.Getenv(envVar)
	if raw == "" {
		raw = defaultCSV
	}

	var counts []int
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			t.Fatalf("%s: invalid integer %q: %v", envVar, part, err)
		}
		if n <= 0 {
			t.Fatalf("%s: values must be positive, got %d", envVar, n)
		}
		counts = append(counts, n)
	}

	if len(counts) == 0 {
		t.Fatalf("%s: at least one value is required", envVar)
	}
	if !sort.IntsAreSorted(counts) {
		t.Fatalf("%s: values must be sorted least to greatest, got %v", envVar, counts)
	}
	return counts
}

// loadScaleCapacityContextNamespace deterministically derives a per-Capacity-instance
// Context id from the Capacity's own id (see the "Capacity" resourceKind below).
var loadScaleCapacityContextNamespace = uuid.MustParse("c9f3a1b2-6d4e-4a7f-9b1c-2e3d4f5a6b7c")

// fixtureIDs are the ids of a single, shared set of prerequisite resources
// that every generated instance may reference. Using one fixed set (rather
// than cross-referencing other generated instances) keeps every resource
// kind's scaling independent of the others and avoids creation-order
// dependencies between kinds.
type fixtureIDs struct {
	contextType          uuid.UUID
	context              uuid.UUID
	system               uuid.UUID
	nodeType             uuid.UUID
	findingType          uuid.UUID
	orgUnit              uuid.UUID
	group                uuid.UUID
	permissionSpec       uuid.UUID
	roleSpec             uuid.UUID
	permission           uuid.UUID
	role                 uuid.UUID
	api                  uuid.UUID
	component            uuid.UUID
	capacityResourceType uuid.UUID
}

// setupFixtures creates one instance of every resource kind that other
// generated resources reference, and returns their ids.
func setupFixtures(t *testing.T, m model.Model) fixtureIDs {
	t.Helper()

	f := fixtureIDs{
		contextType:          uuid.New(),
		context:              uuid.New(),
		system:               uuid.New(),
		nodeType:             uuid.New(),
		findingType:          uuid.New(),
		orgUnit:              uuid.New(),
		group:                uuid.New(),
		permissionSpec:       uuid.New(),
		roleSpec:             uuid.New(),
		permission:           uuid.New(),
		role:                 uuid.New(),
		api:                  uuid.New(),
		component:            uuid.New(),
		capacityResourceType: uuid.New(),
	}

	ct := mdlctx.NewContextType(f.contextType)
	ct.SetDisplayName("fixture ContextType")
	require.NoError(t, m.AddContextType(ct))

	c := mdlctx.NewContext(f.context)
	c.SetDisplayName("fixture Context")
	c.SetContextTypeById(f.contextType)
	require.NoError(t, m.AddContext(c))

	sys := system.NewSystem(f.system)
	sys.SetDisplayName("fixture System")
	require.NoError(t, m.AddSystem(sys))

	nt := node.NewNodeType(f.nodeType)
	nt.SetDisplayName("fixture NodeType")
	require.NoError(t, m.AddNodeType(nt))

	ft := finding.NewFindingType(f.findingType)
	ft.SetDisplayName("fixture FindingType")
	require.NoError(t, m.AddFindingType(ft))

	ou := iam.NewOrgUnit(f.orgUnit)
	ou.SetDisplayName("fixture OrgUnit")
	require.NoError(t, m.AddOrgUnit(ou))

	g := iam.NewGroup(f.group)
	g.SetDisplayName("fixture Group")
	require.NoError(t, m.AddGroup(g))

	ps := iam.NewPermissionSpec(f.permissionSpec)
	ps.SetDisplayName("fixture PermissionSpec")
	require.NoError(t, m.AddPermissionSpec(ps))

	rs := iam.NewRoleSpec(f.roleSpec)
	rs.SetDisplayName("fixture RoleSpec")
	rs.SetPermissions([]*iam.PermissionSpecRef{{PermissionSpecId: f.permissionSpec}})
	require.NoError(t, m.AddRoleSpec(rs))

	p := iam.NewPermission(f.permission)
	p.SetDisplayName("fixture Permission")
	p.SetPermissionSpecById(f.permissionSpec)
	require.NoError(t, m.AddPermission(p))

	r := iam.NewRole(f.role)
	r.SetDisplayName("fixture Role")
	r.SetRoleSpecById(f.roleSpec)
	r.SetContextRef(&mdlctx.ContextRef{ContextId: f.context})
	r.SetPermissions([]*iam.PermissionRef{{PermissionId: f.permission}})
	require.NoError(t, m.AddRole(r))

	a := mdlapi.NewAPI(f.api)
	a.SetDisplayName("fixture API")
	a.SetSystem(&system.SystemRef{SystemId: f.system})
	require.NoError(t, m.AddApi(a))

	comp := component.NewComponent(f.component)
	comp.SetDisplayName("fixture Component")
	comp.SetSystem(&system.SystemRef{SystemId: f.system})
	require.NoError(t, m.AddComponent(comp))

	crt := mdlcap.NewCapacityResourceType(f.capacityResourceType)
	crt.SetDisplayName("fixture CapacityResourceType")
	crt.SetUnit("cores")
	require.NoError(t, m.AddCapacityResourceType(crt))

	return f
}

// resourceKind pairs a human-readable name with a function that creates or
// updates (depending on whether id already exists in the model) one instance
// of that resource kind.
type resourceKind struct {
	Name  string
	Apply func(m model.Model, id uuid.UUID, displayName string) error
}

// resourceKinds returns every resource kind defined by
// [go.emeland.io/modelsrv/pkg/events.ResourceType], excluding
// FilterRuleResource, MergeRuleResource, and AnnotationsResource (which are
// pipeline-visibility / value-object kinds rather than model resources), and
// UnknownResourceType.
func resourceKinds(f fixtureIDs) []resourceKind {
	return []resourceKind{
		{"Node", func(m model.Model, id uuid.UUID, dn string) error {
			n := node.NewNode(id)
			n.SetDisplayName(dn)
			n.SetTypeRef(&node.NodeTypeRef{NodeTypeId: f.nodeType})
			return m.AddNode(n)
		}},
		{"NodeType", func(m model.Model, id uuid.UUID, dn string) error {
			nt := node.NewNodeType(id)
			nt.SetDisplayName(dn)
			return m.AddNodeType(nt)
		}},
		{"Context", func(m model.Model, id uuid.UUID, dn string) error {
			c := mdlctx.NewContext(id)
			c.SetDisplayName(dn)
			c.SetContextTypeById(f.contextType)
			return m.AddContext(c)
		}},
		{"ContextType", func(m model.Model, id uuid.UUID, dn string) error {
			ct := mdlctx.NewContextType(id)
			ct.SetDisplayName(dn)
			return m.AddContextType(ct)
		}},
		{"System", func(m model.Model, id uuid.UUID, dn string) error {
			sys := system.NewSystem(id)
			sys.SetDisplayName(dn)
			return m.AddSystem(sys)
		}},
		{"SystemInstance", func(m model.Model, id uuid.UUID, dn string) error {
			si := system.NewSystemInstance(id)
			si.SetDisplayName(dn)
			si.SetSystemRef(&system.SystemRef{SystemId: f.system})
			return m.AddSystemInstance(si)
		}},
		{"API", func(m model.Model, id uuid.UUID, dn string) error {
			a := mdlapi.NewAPI(id)
			a.SetDisplayName(dn)
			a.SetSystem(&system.SystemRef{SystemId: f.system})
			return m.AddApi(a)
		}},
		{"APIInstance", func(m model.Model, id uuid.UUID, dn string) error {
			ai := mdlapi.NewApiInstance(id)
			ai.SetDisplayName(dn)
			ai.SetApiRef(&mdlapi.ApiRef{ApiID: f.api})
			return m.AddApiInstance(ai)
		}},
		{"Component", func(m model.Model, id uuid.UUID, dn string) error {
			comp := component.NewComponent(id)
			comp.SetDisplayName(dn)
			comp.SetSystem(&system.SystemRef{SystemId: f.system})
			return m.AddComponent(comp)
		}},
		{"ComponentInstance", func(m model.Model, id uuid.UUID, dn string) error {
			ci := component.NewComponentInstance(id)
			ci.SetDisplayName(dn)
			ci.SetComponentRef(&component.ComponentRef{ComponentId: f.component})
			return m.AddComponentInstance(ci)
		}},
		{"OrgUnit", func(m model.Model, id uuid.UUID, dn string) error {
			ou := iam.NewOrgUnit(id)
			ou.SetDisplayName(dn)
			return m.AddOrgUnit(ou)
		}},
		{"Group", func(m model.Model, id uuid.UUID, dn string) error {
			g := iam.NewGroup(id)
			g.SetDisplayName(dn)
			return m.AddGroup(g)
		}},
		{"Identity", func(m model.Model, id uuid.UUID, dn string) error {
			idy := iam.NewIdentity(id)
			idy.SetDisplayName(dn)
			return m.AddIdentity(idy)
		}},
		{"PermissionSpec", func(m model.Model, id uuid.UUID, dn string) error {
			ps := iam.NewPermissionSpec(id)
			ps.SetDisplayName(dn)
			return m.AddPermissionSpec(ps)
		}},
		{"RoleSpec", func(m model.Model, id uuid.UUID, dn string) error {
			rs := iam.NewRoleSpec(id)
			rs.SetDisplayName(dn)
			rs.SetPermissions([]*iam.PermissionSpecRef{{PermissionSpecId: f.permissionSpec}})
			return m.AddRoleSpec(rs)
		}},
		{"Permission", func(m model.Model, id uuid.UUID, dn string) error {
			p := iam.NewPermission(id)
			p.SetDisplayName(dn)
			p.SetPermissionSpecById(f.permissionSpec)
			return m.AddPermission(p)
		}},
		{"Role", func(m model.Model, id uuid.UUID, dn string) error {
			r := iam.NewRole(id)
			r.SetDisplayName(dn)
			r.SetRoleSpecById(f.roleSpec)
			r.SetContextRef(&mdlctx.ContextRef{ContextId: f.context})
			r.SetPermissions([]*iam.PermissionRef{{PermissionId: f.permission}})
			return m.AddRole(r)
		}},
		{"Binding", func(m model.Model, id uuid.UUID, dn string) error {
			b := iam.NewBinding(id)
			b.SetDisplayName(dn)
			b.SetRole(&iam.RoleRef{RoleId: f.role})
			b.SetSubject(&iam.SubjectRef{Group: &iam.GroupRef{GroupId: f.group}})
			return m.AddBinding(b)
		}},
		{"Product", func(m model.Model, id uuid.UUID, dn string) error {
			p := mdlproduct.NewProduct(id)
			p.SetDisplayName(dn)
			p.SetVendor(&iam.OrgUnitRef{OrgUnitId: f.orgUnit})
			return m.AddProduct(p)
		}},
		{"Finding", func(m model.Model, id uuid.UUID, dn string) error {
			fnd := finding.NewFinding(id)
			fnd.SetDisplayName(dn)
			fnd.SetFindingTypeById(f.findingType)
			return m.AddFinding(fnd)
		}},
		{"FindingType", func(m model.Model, id uuid.UUID, dn string) error {
			ft := finding.NewFindingType(id)
			ft.SetDisplayName(dn)
			return m.AddFindingType(ft)
		}},
		{"Artifact", func(m model.Model, id uuid.UUID, dn string) error {
			a := artifact.NewArtifact(id)
			a.SetDisplayName(dn)
			return m.AddArtifact(a)
		}},
		{"ArtifactInstance", func(m model.Model, id uuid.UUID, dn string) error {
			ai := artifact.NewArtifactInstance(id)
			ai.SetDisplayName(dn)
			return m.AddArtifactInstance(ai)
		}},
		{"Capability", func(m model.Model, id uuid.UUID, dn string) error {
			cap := mdlcapability.NewCapability(id)
			cap.SetDisplayName(dn)
			return m.AddCapability(cap)
		}},
		{"Parameter", func(m model.Model, id uuid.UUID, dn string) error {
			param := mdlparameter.NewParameter(id)
			param.SetDisplayName(dn)
			param.SetValues([]string{"val1", "val2"})
			return m.AddParameter(param)
		}},
		{"CapacityResourceType", func(m model.Model, id uuid.UUID, dn string) error {
			crt := mdlcap.NewCapacityResourceType(id)
			crt.SetDisplayName(dn)
			crt.SetUnit("cores")
			return m.AddCapacityResourceType(crt)
		}},
		{"Capacity", func(m model.Model, id uuid.UUID, dn string) error {
			// Capacity uniqueness is (contextId, capacityResourceTypeId, category), not id alone
			// (see pkg/model/structure_capacity.go), so each instance needs its own Context;
			// reusing the shared fixture Context would collide after the first instance.
			ctxID := uuid.NewSHA1(loadScaleCapacityContextNamespace, id[:])
			if m.GetContextById(ctxID) == nil {
				ctx := mdlctx.NewContext(ctxID)
				ctx.SetDisplayName("capacity-context-" + id.String())
				ctx.SetContextTypeById(f.contextType)
				if err := m.AddContext(ctx); err != nil {
					return err
				}
			}
			cap := mdlcap.NewCapacity(id)
			cap.SetDisplayName(dn)
			cap.SetCapacityResourceTypeById(f.capacityResourceType)
			cap.SetContextById(ctxID)
			cap.SetCategory(mdlcap.CategoryProvided)
			cap.SetAmount(mdlcap.Amount("64"))
			return m.AddCapacity(cap)
		}},
	}
}

// loadResult captures the outcome of a single (instances, changes) combination.
type loadResult struct {
	Instances        int
	Changes          int
	TotalEvents      int
	Duration         time.Duration
	HeapAllocBefore  uint64
	HeapAllocAfter   uint64
	TotalAllocDelta  uint64
	HeapObjectsAfter uint64
}

// TestLoadScale builds, for every (instances, changes) combination in the
// configured lists, a fresh production-wired backend (Model + EventManager),
// creates `instances` instances of every resource kind (see resourceKinds),
// and applies `changes` total Create/Update operations to each instance (the
// first operation is the Create, the rest are Updates). It measures process
// memory before and after each combination and writes a markdown report.
func TestLoadScale(t *testing.T) {
	instanceCounts := parseCountList(t, envInstanceCounts, defaultInstanceCounts)
	changeCounts := parseCountList(t, envChangeCounts, defaultChangeCounts)

	reportPath := os.Getenv(envReportPath)
	if reportPath == "" {
		reportPath = defaultReportPath
	}

	kindCount := len(resourceKinds(fixtureIDs{}))

	var results []loadResult

	for _, instances := range instanceCounts {
		for _, changes := range changeCounts {
			t.Run(fmt.Sprintf("instances=%d/changes=%d", instances, changes), func(t *testing.T) {
				totalEvents := kindCount * instances * changes
				t.Logf("building %d resource kinds x %d instances x %d changes = %d events", kindCount, instances, changes, totalEvents)

				runtime.GC()
				var before runtime.MemStats
				runtime.ReadMemStats(&before)

				start := time.Now()

				b, err := backend.New()
				require.NoError(t, err)
				m := b.GetModel()
				f := setupFixtures(t, m)

				for _, kind := range resourceKinds(f) {
					for i := 0; i < instances; i++ {
						id := uuid.New()
						for c := 0; c < changes; c++ {
							dn := fmt.Sprintf("%s-%d-change-%d", kind.Name, i, c)
							require.NoErrorf(t, kind.Apply(m, id, dn), "%s instance %d change %d", kind.Name, i, c)
						}
					}
				}

				elapsed := time.Since(start)

				runtime.GC()
				var after runtime.MemStats
				runtime.ReadMemStats(&after)
				runtime.KeepAlive(b)

				results = append(results, loadResult{
					Instances:        instances,
					Changes:          changes,
					TotalEvents:      totalEvents,
					Duration:         elapsed,
					HeapAllocBefore:  before.HeapAlloc,
					HeapAllocAfter:   after.HeapAlloc,
					TotalAllocDelta:  after.TotalAlloc - before.TotalAlloc,
					HeapObjectsAfter: after.HeapObjects,
				})

				t.Logf("instances=%d changes=%d: heap alloc after=%s (delta %s), duration=%s",
					instances, changes, formatBytes(after.HeapAlloc), formatSignedBytes(int64(after.HeapAlloc)-int64(before.HeapAlloc)), elapsed.Round(time.Millisecond))
			})
		}
	}

	writeReport(t, reportPath, kindCount, instanceCounts, changeCounts, results)
}

func writeReport(t *testing.T, path string, kindCount int, instanceCounts, changeCounts []int, results []loadResult) {
	t.Helper()

	var sb strings.Builder

	sb.WriteString("# modelsrv Load & Scale Report\n\n")
	fmt.Fprintf(&sb, "Generated: %s\n\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(&sb, "Go runtime: %s, GOMAXPROCS: %d\n\n", runtime.Version(), runtime.GOMAXPROCS(0))
	fmt.Fprintf(&sb, "Resource kinds exercised per combination: **%d** "+
		"(every `pkg/events.ResourceType` except `UnknownResourceType`, `FilterRuleResource`, "+
		"`MergeRuleResource`, and `AnnotationsResource`).\n\n", kindCount)
	fmt.Fprintf(&sb, "Instance counts (`%s`): %v\n\n", envInstanceCounts, instanceCounts)
	fmt.Fprintf(&sb, "Change counts (`%s`): %v\n\n", envChangeCounts, changeCounts)
	sb.WriteString("Each row builds a fresh `backend.Backend` (the same `Model` + `EventManager` wiring " +
		"modelsrv uses in production, including the in-memory event history kept for replication). " +
		"For every resource kind it creates N instances and applies C total Create/Update operations " +
		"to each instance (the first operation is the Create, the remaining C-1 are Updates), so " +
		"`Total Events` = resource kinds × instances × changes. Memory is sampled with " +
		"`runtime.ReadMemStats` immediately after a forced `runtime.GC()`, once before and once after " +
		"the run.\n\n")

	sb.WriteString("| Instances/Resource | Changes/Instance | Total Events | Duration | Heap Alloc (after) | Heap Alloc Δ | Total Alloc (churn) | Heap Objects (after) |\n")
	sb.WriteString("|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, r := range results {
		delta := int64(r.HeapAllocAfter) - int64(r.HeapAllocBefore)
		fmt.Fprintf(&sb, "| %d | %d | %d | %s | %s | %s | %s | %d |\n",
			r.Instances, r.Changes, r.TotalEvents, r.Duration.Round(time.Millisecond),
			formatBytes(r.HeapAllocAfter), formatSignedBytes(delta),
			formatBytes(r.TotalAllocDelta), r.HeapObjectsAfter)
	}

	if err := os.MkdirAll(filepath.Dir(abs(path)), 0o755); err != nil {
		t.Fatalf("creating report directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("writing report to %s: %v", path, err)
	}
	t.Logf("load-scale report written to %s", path)
}

// abs returns path made absolute against the current working directory,
// falling back to path itself if that fails (e.g. an already-absolute path).
func abs(path string) string {
	if a, err := filepath.Abs(path); err == nil {
		return a
	}
	return path
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatSignedBytes(b int64) string {
	if b < 0 {
		return "-" + formatBytes(uint64(-b))
	}
	return formatBytes(uint64(b))
}
