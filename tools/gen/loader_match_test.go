package main

import (
	"reflect"
	"testing"
)

// TestLoaderMatchesHandWritten is the hard contract for ADR step 6 ("Path B"): every
// annotated struct in testdata, loaded and enriched exactly as init() enriches the
// hand-written specs (buildTypeSpec + applyWiring + enrichDomainMeta + enrichWireMeta),
// must be reflect.DeepEqual to its hand-written allTypes entry. When this passes, the
// struct loader fully reproduces the []TypeSpec literal, so the literal and the
// wire_meta.go / domain_meta.go lookup maps can be deleted in a later step.
func TestLoaderMatchesHandWritten(t *testing.T) {
	loaded, err := loadTypeSpecs("testdata")
	if err != nil {
		t.Fatalf("loadTypeSpecs: %v", err)
	}

	// Oracle: the retained hand-written literal, enriched the same way. allTypes is now
	// built from the loader, so comparing against it would be tautological.
	oracle := make([]TypeSpec, len(handwrittenTypes))
	copy(oracle, handwrittenTypes)
	byName := map[string]TypeSpec{}
	for i := range oracle {
		s := oracle[i]
		enrichDomainMeta(&s)
		enrichWireMeta(&s)
		byName[s.Name] = s
	}

	if len(loaded) != len(handwrittenTypes) {
		t.Fatalf("loaded %d types, want %d (hand-written)", len(loaded), len(handwrittenTypes))
	}

	seen := map[string]bool{}
	for i := range loaded {
		spec := loaded[i]
		enrichDomainMeta(&spec)
		enrichWireMeta(&spec)
		seen[spec.Name] = true

		want, ok := byName[spec.Name]
		if !ok {
			t.Errorf("%s: loaded struct has no hand-written allTypes entry", spec.Name)
			continue
		}

		if reflect.DeepEqual(spec, want) {
			continue
		}

		// Report the specific fields that differ to make failures actionable.
		gv := reflect.ValueOf(spec)
		wv := reflect.ValueOf(want)
		tt := gv.Type()
		for f := 0; f < tt.NumField(); f++ {
			fld := tt.Field(f)
			if fld.PkgPath != "" {
				continue // unexported
			}
			got := gv.Field(f).Interface()
			exp := wv.Field(f).Interface()
			if !reflect.DeepEqual(got, exp) {
				t.Errorf("%s: field %s differs:\n  loader: %#v\n  want:   %#v", spec.Name, fld.Name, got, exp)
			}
		}
	}

	// Every hand-written type must be produced by the loader.
	for name := range byName {
		if !seen[name] {
			t.Errorf("hand-written type %s not produced by the loader", name)
		}
	}
}
