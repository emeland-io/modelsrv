package main

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// findSpec returns the hand-written TypeSpec (post-enrichment, as init() left it) by name.
func findSpec(t *testing.T, name string) TypeSpec {
	t.Helper()
	for _, s := range allTypes {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("hand-written spec %q not found in allTypes", name)
	return TypeSpec{}
}

// enrichLoaded applies the same enrichment init() applies to hand-written specs, so the
// loader output is comparable to the enriched allTypes entry.
func enrichLoaded(spec TypeSpec) TypeSpec {
	enrichDomainMeta(&spec)
	enrichWireMeta(&spec)
	return spec
}

// normalizeSetupVar rewrites the leading "x := ..." variable in a TestSetup block to a
// fixed name so two semantically identical setups with different local variable names
// compare equal. The variable name is arbitrary generator input; setupObjectVar in
// template_funcs.go derives whatever name is used.
func normalizeSetupVar(setup string) string {
	setup = strings.TrimSpace(setup)
	firstLine := strings.SplitN(setup, "\n", 2)[0]
	name, _, ok := strings.Cut(firstLine, ":=")
	if !ok {
		return setup
	}
	varName := strings.TrimSpace(name)
	// Replace whole-word occurrences of the variable with "v".
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(varName) + `\b`)
	normalized := re.ReplaceAllString(setup, "v")
	// Collapse leading whitespace on each line so indentation differences don't matter.
	lines := strings.Split(normalized, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	return strings.Join(lines, "\n")
}

func TestLoader_NodeTypeMatchesHandWritten(t *testing.T) {
	loaded, err := loadTypeSpecs("testdata")
	if err != nil {
		t.Fatalf("loadTypeSpecs: %v", err)
	}

	var got *TypeSpec
	for i := range loaded {
		if loaded[i].Name == "NodeType" {
			got = &loaded[i]
			break
		}
	}
	if got == nil {
		t.Fatal("loader did not produce a NodeType spec")
	}

	want := findSpec(t, "NodeType")
	haveEnriched := enrichLoaded(*got)

	// Compare the core fields the loader is responsible for. Enrichment-derived fields are
	// covered transitively because both specs are run through the same enrich functions.
	type check struct {
		name     string
		got      any
		want     any
	}
	checks := []check{
		{"Name", haveEnriched.Name, want.Name},
		{"Dir", haveEnriched.Dir, want.Dir},
		{"EventType", haveEnriched.EventType, want.EventType},
		{"IDField", haveEnriched.IDField, want.IDField},
		{"NameField", haveEnriched.NameField, want.NameField},
		{"HasHandler", haveEnriched.HasHandler, want.HasHandler},
		{"NotFoundErr", haveEnriched.NotFoundErr, want.NotFoundErr},
		{"ExtraImports", haveEnriched.ExtraImports, want.ExtraImports},
		{"HasClientTest", haveEnriched.HasClientTest, want.HasClientTest},
		{"GenClientMethods", haveEnriched.GenClientMethods, want.GenClientMethods},
		{"ClientListMethod", haveEnriched.ClientListMethod, want.ClientListMethod},
		{"ClientGetByIdMethod", haveEnriched.ClientGetByIdMethod, want.ClientGetByIdMethod},
		{"ClientListOapiMethod", haveEnriched.ClientListOapiMethod, want.ClientListOapiMethod},
		{"ClientGetByIdOapiMethod", haveEnriched.ClientGetByIdOapiMethod, want.ClientGetByIdOapiMethod},
		{"OapiTypeName", haveEnriched.OapiTypeName, want.OapiTypeName},
		{"TestDisplayName", haveEnriched.TestDisplayName, want.TestDisplayName},
		{"TestIDAssertExpr", haveEnriched.TestIDAssertExpr, want.TestIDAssertExpr},
		{"TestNameAssertExpr", haveEnriched.TestNameAssertExpr, want.TestNameAssertExpr},
		{"NotFoundSentinel", haveEnriched.NotFoundSentinel, want.NotFoundSentinel},
		{"WireKind", haveEnriched.WireKind, want.WireKind},
		// Enrichment-derived, asserted for confidence the whole chain matches:
		{"WireIDField", haveEnriched.WireIDField, want.WireIDField},
		{"FromDtoFuncName", haveEnriched.FromDtoFuncName, want.FromDtoFuncName},
		{"ToDtoFuncName", haveEnriched.ToDtoFuncName, want.ToDtoFuncName},
		{"RestListPath", haveEnriched.RestListPath, want.RestListPath},
		{"EventsResource", haveEnriched.EventsResource, want.EventsResource},
		{"DomainPkgAlias", haveEnriched.DomainPkgAlias, want.DomainPkgAlias},
		{"BackendGetByIdMethod", haveEnriched.BackendGetByIdMethod, want.BackendGetByIdMethod},
	}
	for _, c := range checks {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("%s: loader=%#v hand-written=%#v", c.name, c.got, c.want)
		}
	}

	// Fields compared explicitly (Name/Type/Optional/HasAnnotations).
	if !reflect.DeepEqual(haveEnriched.Fields, want.Fields) {
		t.Errorf("Fields differ:\nloader=%#v\nhand-written=%#v", haveEnriched.Fields, want.Fields)
	}

	// TestSetup compared after normalizing the local variable name and indentation.
	if normalizeSetupVar(haveEnriched.TestSetup) != normalizeSetupVar(want.TestSetup) {
		t.Errorf("TestSetup differs:\nloader=%q\nhand-written=%q",
			normalizeSetupVar(haveEnriched.TestSetup), normalizeSetupVar(want.TestSetup))
	}
}
