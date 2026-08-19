package ingress

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"go.emeland.io/modelsrv/pkg/events"
)

// DefaultCSVVersion is used when ParseOptions.Version is empty and no version column is mapped.
const DefaultCSVVersion = "emeland.io/v1"

// Spec path sentinels for CSV column mapping.
const (
	csvPathKind    = "kind"
	csvPathVersion = "version"
	// csvPathID expands to the primary UUID field for the row's resource type (e.g. contextId).
	csvPathID = "id"
)

// DefaultCSVColumns maps a generic CSV header layout onto Document fields.
// uuid → id is resolved per row from resourcetype.
// annotations expects a JSON object string, e.g. {"owner":"ops"}.
var DefaultCSVColumns = map[string]string{
	"resourcetype": "kind",
	"uuid":         csvPathID,
	"displayname":  "displayName",
	"description":  "description",
	"annotations":  "annotations",
}

// primaryIDField is the spec key that holds the resource's primary UUID for each kind.
var primaryIDField = map[events.ResourceType]string{
	events.NodeResource:                 "nodeId",
	events.NodeTypeResource:             "nodeTypeId",
	events.ContextResource:              "contextId",
	events.ContextTypeResource:          "contextTypeId",
	events.SystemResource:               "systemId",
	events.SystemInstanceResource:       "instanceId",
	events.APIResource:                  "apiId",
	events.APIInstanceResource:          "instanceId",
	events.ComponentResource:            "componentId",
	events.ComponentInstanceResource:    "instanceId",
	events.OrgUnitResource:              "orgUnitId",
	events.GroupResource:                "groupId",
	events.IdentityResource:             "identityId",
	events.PermissionSpecResource:       "permissionSpecId",
	events.RoleSpecResource:             "roleSpecId",
	events.PermissionResource:           "permissionId",
	events.RoleResource:                 "roleId",
	events.BindingResource:              "bindingId",
	events.ProductResource:              "productId",
	events.FindingResource:              "findingId",
	events.FindingTypeResource:          "findingTypeId",
	events.ArtifactResource:             "artifactId",
	events.ArtifactInstanceResource:     "artifactInstanceId",
	events.FilterRuleResource:           "ruleId",
	events.MergeRuleResource:            "ruleId",
	events.CapabilityResource:           "capabilityId",
	events.ParameterResource:            "parameterId",
	events.CapacityResourceTypeResource: "capacityResourceTypeId",
	events.CapacityResource:             "capacityId",
}

// ValidateCSVOptions reports config errors that do not require reading a file.
// CSV needs ParseOptions.Kind and/or a column mapped to "kind"; empty Columns
// uses [DefaultCSVColumns]. Call this at sensor Open so a columns block that
// omits kind fails at startup instead of at first parse.
func ValidateCSVOptions(opts ParseOptions) error {
	if opts.Kind != events.UnknownResourceType {
		if _, ok := documentKinds[opts.Kind]; !ok {
			return fmt.Errorf("CSV unsupported kind %q", opts.Kind)
		}
		return nil
	}
	columns := opts.Columns
	if len(columns) == 0 {
		columns = DefaultCSVColumns
	}
	for _, path := range columns {
		if strings.TrimSpace(path) == csvPathKind {
			return nil
		}
	}
	return fmt.Errorf("CSV requires ParseOptions.Kind or a column mapped to %q", csvPathKind)
}

// DecodeCSVDocuments turns CSV rows into [Document] values using opts for column mapping.
//
// Kind may come from ParseOptions.Kind and/or a column mapped to "kind" (e.g. resourcetype).
// At least one source is required; per-row kind wins when present.
// Version defaults to [DefaultCSVVersion] when unset and no version column is present.
// A column mapped to "id" is written to that kind's primary UUID field (see primaryIDField).
func DecodeCSVDocuments(data []byte, opts ParseOptions) ([]Document, error) {
	if err := ValidateCSVOptions(opts); err != nil {
		return nil, err
	}

	columns := opts.Columns
	if len(columns) == 0 {
		columns = DefaultCSVColumns
	}

	delim := opts.Delimiter
	if delim == 0 {
		delim = ','
	}

	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = delim
	r.TrimLeadingSpace = true
	r.ReuseRecord = true

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV decode: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("no CSV data rows found")
	}

	headers := records[0]
	headerIndex := make(map[string]int, len(headers))
	for i, h := range headers {
		headerIndex[strings.TrimSpace(h)] = i
	}

	kindCol := -1
	versionCol := -1
	idCol := -1
	specMaps := make([]struct {
		col  int
		path string
	}, 0, len(columns))

	for header, path := range columns {
		idx, ok := headerIndex[header]
		if !ok {
			// Mapped headers are optional so DefaultCSVColumns can include
			// annotations/description without requiring every file to have them.
			continue
		}
		switch strings.TrimSpace(path) {
		case csvPathKind:
			kindCol = idx
		case csvPathVersion:
			versionCol = idx
		case csvPathID:
			idCol = idx
		default:
			specMaps = append(specMaps, struct {
				col  int
				path string
			}{col: idx, path: path})
		}
	}

	if opts.Kind == events.UnknownResourceType && kindCol < 0 {
		return nil, fmt.Errorf("CSV file has no column mapped to %q", csvPathKind)
	}

	defaultVersion := strings.TrimSpace(opts.Version)
	if defaultVersion == "" {
		defaultVersion = DefaultCSVVersion
	}

	docs := make([]Document, 0, len(records)-1)
	for rowIdx, row := range records[1:] {
		if isEmptyCSVRow(row) {
			continue
		}
		kind := opts.Kind
		if kindCol >= 0 && kindCol < len(row) {
			s := strings.TrimSpace(row[kindCol])
			if s != "" {
				rt := events.ParseResourceType(s)
				if rt == events.UnknownResourceType {
					return nil, fmt.Errorf("row %d: unsupported kind %q", rowIdx+1, s)
				}
				if _, ok := documentKinds[rt]; !ok {
					return nil, fmt.Errorf("row %d: unsupported kind %q", rowIdx+1, s)
				}
				kind = rt
			}
		}
		if kind == events.UnknownResourceType {
			return nil, fmt.Errorf("row %d: missing kind (empty %q and no ParseOptions.Kind)", rowIdx+1, csvPathKind)
		}

		ver := defaultVersion
		if versionCol >= 0 && versionCol < len(row) {
			s := strings.TrimSpace(row[versionCol])
			if s != "" {
				ver = s
			}
		}

		spec := map[string]any{}
		if idCol >= 0 && idCol < len(row) {
			val := strings.TrimSpace(row[idCol])
			if val != "" {
				idField, ok := primaryIDField[kind]
				if !ok {
					return nil, fmt.Errorf("row %d: no primary id field for kind %s", rowIdx+1, kind)
				}
				spec[idField] = coerceCSVValue(val)
			}
		}
		for _, m := range specMaps {
			if m.col >= len(row) {
				continue
			}
			val := strings.TrimSpace(row[m.col])
			if val == "" {
				continue
			}
			if err := setSpecPath(spec, m.path, coerceCSVValue(val)); err != nil {
				return nil, fmt.Errorf("row %d: %w", rowIdx+1, err)
			}
		}

		docs = append(docs, Document{
			Version: ver,
			Kind:    DocumentKind(kind),
			Spec:    spec,
		})
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("no CSV data rows found")
	}
	return docs, nil
}

func isEmptyCSVRow(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func coerceCSVValue(s string) any {
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		var v any
		if err := json.Unmarshal([]byte(trimmed), &v); err == nil {
			return v
		}
	}
	if b, err := strconv.ParseBool(s); err == nil && (s == "true" || s == "false") {
		return b
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}

// setSpecPath writes value into dest at a dotted path (e.g. "annotations.owner").
func setSpecPath(dest map[string]any, path string, value any) error {
	parts := strings.Split(path, ".")
	if len(parts) == 0 || parts[0] == "" {
		return fmt.Errorf("empty column path")
	}
	cur := dest
	for i, p := range parts {
		if i == len(parts)-1 {
			cur[p] = value
			return nil
		}
		next, ok := cur[p]
		if !ok {
			m := map[string]any{}
			cur[p] = m
			cur = m
			continue
		}
		m, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("path %q collides with non-object at %q", path, p)
		}
		cur = m
	}
	return nil
}
