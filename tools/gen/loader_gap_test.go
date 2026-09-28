package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// TestLoaderGapReport is a diagnostic (not a pass/fail contract yet): it loads every
// annotated struct, enriches it the same way init() enriches the hand-written specs, and
// reports which TypeSpec fields differ from the current hand-written allTypes entry. It
// exists to drive ADR step 6 (replacing the []TypeSpec literal). Run with:
//
//	go test ./tools/gen -run TestLoaderGapReport -v
//
// It only fails if a struct has no matching hand-written spec (a structural problem);
// field differences are logged, not failed, so we can enumerate the remaining work.
func TestLoaderGapReport(t *testing.T) {
	loaded, err := loadTypeSpecs("testdata")
	if err != nil {
		t.Fatalf("loadTypeSpecs: %v", err)
	}

	byName := map[string]TypeSpec{}
	for i := range handwrittenTypes {
		s := handwrittenTypes[i]
		enrichDomainMeta(&s)
		enrichWireMeta(&s)
		byName[s.Name] = s
	}

	// Fields we already know the loader is not responsible for reproducing in step 6, or
	// that are populated purely by enrichment; compared anyway but listed for context.
	fieldDiffCounts := map[string]int{}
	var missing []string

	for i := range loaded {
		spec := loaded[i]
		enrichDomainMeta(&spec)
		enrichWireMeta(&spec)

		want, ok := byName[spec.Name]
		if !ok {
			missing = append(missing, spec.Name)
			continue
		}

		diffs := diffTypeSpec(spec, want)
		if len(diffs) > 0 {
			t.Logf("%s: %d field(s) differ: %s", spec.Name, len(diffs), strings.Join(diffs, ", "))
			for _, d := range diffs {
				fieldDiffCounts[d]++
			}
		}
	}

	if len(missing) > 0 {
		t.Errorf("loaded structs with no hand-written spec: %v", missing)
	}

	// Summary: which fields differ across how many types.
	type fc struct {
		field string
		count int
	}
	var summary []fc
	for f, c := range fieldDiffCounts {
		summary = append(summary, fc{f, c})
	}
	sort.Slice(summary, func(i, j int) bool {
		if summary[i].count != summary[j].count {
			return summary[i].count > summary[j].count
		}
		return summary[i].field < summary[j].field
	})
	t.Logf("=== field-diff summary (field: #types differing) ===")
	for _, s := range summary {
		t.Logf("  %-28s %d", s.field, s.count)
	}
	t.Logf("loaded=%d handwritten=%d", len(loaded), len(allTypes))
}

// diffTypeSpec returns the names of exported TypeSpec fields whose values differ.
func diffTypeSpec(got, want TypeSpec) []string {
	var diffs []string
	gv := reflect.ValueOf(got)
	wv := reflect.ValueOf(want)
	tt := gv.Type()
	for i := 0; i < tt.NumField(); i++ {
		f := tt.Field(i)
		if f.PkgPath != "" {
			continue // unexported
		}
		if !reflect.DeepEqual(gv.Field(i).Interface(), wv.Field(i).Interface()) {
			diffs = append(diffs, f.Name)
		}
	}
	return diffs
}

// helper to keep fmt imported for future detailed dumps.
var _ = fmt.Sprintf
