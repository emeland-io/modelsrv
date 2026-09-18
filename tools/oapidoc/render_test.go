package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestRenderGolden(t *testing.T) {
	out := t.TempDir()
	if err := render("testdata/tiny.yaml", out, "v9.9.9"); err != nil {
		t.Fatalf("render: %v", err)
	}

	goldenRoot := "testdata/golden"
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.RemoveAll(goldenRoot); err != nil {
			t.Fatalf("clear golden: %v", err)
		}
		if err := copyTree(out, goldenRoot); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Log("updated golden files")
		return
	}

	err := filepath.WalkDir(goldenRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(goldenRoot, path)
		if err != nil {
			return err
		}
		gotPath := filepath.Join(out, rel)
		want, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		got, err := os.ReadFile(gotPath)
		if err != nil {
			t.Errorf("missing rendered file %s: %v", rel, err)
			return nil
		}
		if string(got) != string(want) {
			t.Errorf("mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", rel, got, want)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk golden: %v", err)
	}

	// Every rendered file must also exist in golden (no extras).
	err = filepath.WalkDir(out, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(out, path)
		if err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(goldenRoot, rel)); err != nil {
			t.Errorf("unexpected rendered file not in golden: %s", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk out: %v", err)
	}
}

func TestRenderRealSpecLinkIntegrity(t *testing.T) {
	spec := filepath.Join("..", "..", "api", "openapi", "EmergingEnterpriseLandscape-0.1.0-oapi-3.0.3.yaml")
	if _, err := os.Stat(spec); err != nil {
		t.Skipf("real spec not found: %v", err)
	}
	out := t.TempDir()
	if err := render(spec, out, "v0.0.0-test"); err != nil {
		t.Fatalf("render real spec: %v", err)
	}

	files := map[string]bool{}
	empty := []string{}
	err := filepath.WalkDir(out, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(out, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = true
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if len(strings.TrimSpace(string(data))) == 0 {
			empty = append(empty, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(empty) > 0 {
		t.Fatalf("empty files: %v", empty)
	}
	if !files["README.md"] {
		t.Fatal("missing README.md")
	}
	if len(files) < 10 {
		t.Fatalf("expected many rendered files, got %d", len(files))
	}

	linkRe := regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
	for rel := range files {
		data, err := os.ReadFile(filepath.Join(out, rel))
		if err != nil {
			t.Fatal(err)
		}
		baseDir := filepath.Dir(rel)
		for _, m := range linkRe.FindAllStringSubmatch(string(data), -1) {
			target := m[1]
			if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
				continue
			}
			if strings.HasPrefix(target, "#") {
				continue
			}
			// Strip optional anchors.
			if i := strings.IndexByte(target, '#'); i >= 0 {
				target = target[:i]
			}
			resolved := filepath.ToSlash(filepath.Clean(filepath.Join(baseDir, target)))
			if !files[resolved] {
				t.Errorf("%s: dangling link to %s (resolved %s)", rel, m[1], resolved)
			}
		}
	}
}

func TestFirstSentence(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"A pet in the demo catalog (e.g. a dog). More.", "A pet in the demo catalog (e.g. a dog)."},
		{"One line.\nTwo.", "One line."},
		{"No period here", "No period here"},
		{"Ends with period.", "Ends with period."},
	}
	for _, c := range cases {
		if got := firstSentence(c.in); got != c.want {
			t.Errorf("firstSentence(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
