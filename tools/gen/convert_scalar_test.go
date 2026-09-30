package main

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
)

// renderConvert renders the embedded convert_from / convert_to templates against a
// single synthetic spec and returns (fromDto, toDto) source.
func renderConvert(t *testing.T, spec convertGenSpec) (string, string) {
	t.Helper()

	data := convertGenData{Specs: []convertGenSpec{spec}}

	fromTmpl := template.Must(template.New("convert_from").Parse(convertFromTemplate))
	toTmpl := template.Must(template.New("convert_to").Parse(convertToTemplate))

	var fromBuf, toBuf bytes.Buffer
	if err := fromTmpl.Execute(&fromBuf, data); err != nil {
		t.Fatalf("convert_from execute: %v", err)
	}
	if err := toTmpl.Execute(&toBuf, data); err != nil {
		t.Fatalf("convert_to execute: %v", err)
	}
	return fromBuf.String(), toBuf.String()
}

// baseSpec returns a minimal convertGenSpec sufficient for the convert templates.
func baseSpec() convertGenSpec {
	return convertGenSpec{
		TypeSpec: TypeSpec{
			Name:               "Widget",
			DomainPkgAlias:     "widget",
			DomainTypeName:     "Widget",
			OapiWireTypeName:   "Widget",
			WireIDField:        "WidgetId",
			WireDomainIDGetter: "GetWidgetId",
			FromDtoFuncName:    "WidgetFromDto",
			ToDtoFuncName:      "WidgetToDto",
			ConvertNilLabel:    "widget",
		},
		HasDisplayName: true,
		HasAnnotations: true,
	}
}

func TestConvertGen_RequiredScalar(t *testing.T) {
	spec := baseSpec()
	spec.ExtraScalarFields = []scalarField{{Name: "Owner", GoType: "string", Optional: false}}

	from, to := renderConvert(t, spec)

	// FromDto: required scalar assigned directly (no nil check).
	if !strings.Contains(from, "v.SetOwner(o.Owner)") {
		t.Errorf("required FromDto missing direct assignment; got:\n%s", from)
	}
	if strings.Contains(from, "if o.Owner != nil") {
		t.Errorf("required FromDto should not emit a nil check; got:\n%s", from)
	}

	// ToDto: required scalar set inside the struct literal.
	if !strings.Contains(to, "Owner: v.GetOwner(),") {
		t.Errorf("required ToDto missing struct-literal assignment; got:\n%s", to)
	}
}

func TestConvertGen_OptionalScalar(t *testing.T) {
	spec := baseSpec()
	spec.ExtraScalarFields = []scalarField{{Name: "Owner", GoType: "string", Optional: true}}

	from, to := renderConvert(t, spec)

	// FromDto: optional scalar guarded by a nil check and dereferenced.
	if !strings.Contains(from, "if o.Owner != nil {") || !strings.Contains(from, "v.SetOwner(*o.Owner)") {
		t.Errorf("optional FromDto missing nil-safe deref; got:\n%s", from)
	}

	// ToDto: optional scalar emitted only when non-zero, as a pointer.
	if !strings.Contains(to, `if val := v.GetOwner(); val != "" {`) || !strings.Contains(to, "out.Owner = &val") {
		t.Errorf("optional ToDto missing non-zero pointer assignment; got:\n%s", to)
	}
	// Optional field must NOT appear inside the struct literal (would be a *string mismatch).
	if strings.Contains(to, "Owner: v.GetOwner(),") {
		t.Errorf("optional ToDto must not assign in struct literal; got:\n%s", to)
	}
}

func TestConvertGen_OptionalNumericZero(t *testing.T) {
	spec := baseSpec()
	spec.ExtraScalarFields = []scalarField{
		{Name: "Weight", GoType: "int", Optional: true},
		{Name: "Enabled", GoType: "bool", Optional: true},
	}

	_, to := renderConvert(t, spec)

	if !strings.Contains(to, "if val := v.GetWeight(); val != 0 {") {
		t.Errorf("optional int ToDto should compare against 0; got:\n%s", to)
	}
	if !strings.Contains(to, "if val := v.GetEnabled(); val != false {") {
		t.Errorf("optional bool ToDto should compare against false; got:\n%s", to)
	}
}

func TestIsScalarConvertType(t *testing.T) {
	scalars := []string{"string", "bool", "int", "int32", "int64", "uint", "float64"}
	for _, s := range scalars {
		if !isScalarConvertType(s) {
			t.Errorf("expected %q to be a scalar convert type", s)
		}
	}
	nonScalars := []string{"*string", "annotations.Annotations", "*NodeTypeRef", "[]string", "SystemRef"}
	for _, s := range nonScalars {
		if isScalarConvertType(s) {
			t.Errorf("expected %q NOT to be a scalar convert type", s)
		}
	}
}

func TestScalarFieldZero(t *testing.T) {
	cases := map[string]string{
		"string":  `""`,
		"bool":    "false",
		"int":     "0",
		"int64":   "0",
		"float64": "0",
	}
	for goType, want := range cases {
		got := scalarField{GoType: goType}.Zero()
		if got != want {
			t.Errorf("Zero() for %q = %q, want %q", goType, got, want)
		}
	}
}
