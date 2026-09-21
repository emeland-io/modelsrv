/*
Copyright © 2025 Lutz Behnke

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// dateInputLayouts are the layouts accepted for the lifecycle date flags and
// interactive prompts. Values are always emitted as RFC3339 so they round-trip
// through the modelsrv ingress (pkg/ingress parseOptionalTime expects RFC3339).
var dateInputLayouts = []string{time.RFC3339, "2006-01-02"}

// capabilityVersionInput collects the per-version values gathered from the
// command line before prompting and YAML generation.
type capabilityVersionInput struct {
	version        string
	availableFrom  string
	deprecatedFrom string
	terminatedFrom string
}

// capabilityArgs holds everything parsed from the raw capability command line.
type capabilityArgs struct {
	displayName string
	annotations []string
	versions    []capabilityVersionInput
	outputDir   string // overrides parent -d when set
	outputFile  string // overrides parent -o when set
	dirSet      bool
	fileSet     bool
	showHelp    bool
}

// newCapabilityCmd builds the bespoke "create capability" subcommand.
//
// Unlike the generic resource commands, a Capability carries a list of versions,
// each with its own lifecycle dates. The --version flag may be given multiple
// times and every lifecycle date flag is scoped to the most recent --version:
//
//	emelandctl create capability "Mail Service" \
//	    --version 1.0.0 --available-from 2026-01-01 --deprecated-from 2027-01-01 \
//	    --version 2.0.0 --available-from 2027-01-01
//
// Any value not supplied on the command line is asked for interactively when
// stdin is a terminal. Flag parsing is done by hand (DisableFlagParsing) so the
// date flags can be grouped under the preceding --version.
func newCapabilityCmd(def resourceDef, outputDir, outputFile *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   def.use + " [displayName]",
		Short: def.short,
		Long: def.short + `.

The --version flag may be repeated. Lifecycle date flags (--available-from,
--deprecated-from, --terminated-from) are scoped to the most recent --version.
Dates accept YYYY-MM-DD or RFC3339 and are stored as RFC3339.

Any value not provided on the command line is prompted for interactively.

Flags:
  -n, --name string             Display name of the capability
      --annotation key=value    Annotation (repeatable)
      --version string          Version string (repeatable)
      --available-from date     Version available date, scopes to last --version
      --deprecated-from date    Version deprecated date, scopes to last --version
      --terminated-from date    Version terminated date, scopes to last --version
  -d, --output-dir string       Output directory (inherited from create)
  -o, --output string           Output file, or - for stdout (inherited from create)`,
		// Parse the args ourselves to support repeatable, scoped --version.
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			parsed, err := parseCapabilityArgs(args)
			if err != nil {
				return err
			}
			if parsed.showHelp {
				return cmd.Help()
			}

			displayName := parsed.displayName
			if displayName == "" {
				if isInteractive() {
					displayName, _ = promptLine(cmd, "Display name", "")
				}
				if displayName == "" {
					return fmt.Errorf("display name is required (positional argument or --name/-n flag)")
				}
			}

			versions, err := promptForVersions(cmd, parsed.versions)
			if err != nil {
				return err
			}

			id := uuid.New()
			spec := buildCapabilitySpec(id, displayName, versions)
			if len(parsed.annotations) > 0 {
				spec["annotations"] = parseAnnotations(parsed.annotations)
			}

			dir := *outputDir
			if parsed.dirSet {
				dir = parsed.outputDir
			}
			file := *outputFile
			if parsed.fileSet {
				file = parsed.outputFile
			}

			r := Resource{Version: resourceVersion, Kind: def.kind, Spec: spec}
			return writeResource(r, id, dir, file)
		},
	}

	return cmd
}

// parseCapabilityArgs performs manual flag parsing for the capability command.
// It supports the long/short forms below, inline (--flag=value) and separate
// (--flag value) syntax, and groups the lifecycle date flags under the most
// recent --version.
func parseCapabilityArgs(args []string) (capabilityArgs, error) {
	var out capabilityArgs
	cur := -1 // index of the version currently in scope

	// next consumes the inline value or the following token.
	i := 0
	next := func(name, inline string, hasInline bool) (string, error) {
		if hasInline {
			return inline, nil
		}
		i++
		if i >= len(args) {
			return "", fmt.Errorf("--%s requires a value", name)
		}
		return args[i], nil
	}

	for i = 0; i < len(args); i++ {
		tok := args[i]

		// Short flags.
		switch tok {
		case "-h", "--help":
			out.showHelp = true
			continue
		case "-n":
			v, err := next("name", "", false)
			if err != nil {
				return out, err
			}
			out.displayName = v
			continue
		case "-d":
			v, err := next("output-dir", "", false)
			if err != nil {
				return out, err
			}
			out.outputDir, out.dirSet = v, true
			continue
		case "-o":
			v, err := next("output", "", false)
			if err != nil {
				return out, err
			}
			out.outputFile, out.fileSet = v, true
			continue
		}

		if !strings.HasPrefix(tok, "--") {
			// Positional: first one is the display name.
			if out.displayName == "" {
				out.displayName = tok
			} else {
				return out, fmt.Errorf("unexpected argument %q", tok)
			}
			continue
		}

		name, inline, hasInline := splitLongFlag(tok)
		switch name {
		case "name":
			v, err := next(name, inline, hasInline)
			if err != nil {
				return out, err
			}
			out.displayName = v
		case "annotation":
			v, err := next(name, inline, hasInline)
			if err != nil {
				return out, err
			}
			out.annotations = append(out.annotations, v)
		case "output-dir":
			v, err := next(name, inline, hasInline)
			if err != nil {
				return out, err
			}
			out.outputDir, out.dirSet = v, true
		case "output":
			v, err := next(name, inline, hasInline)
			if err != nil {
				return out, err
			}
			out.outputFile, out.fileSet = v, true
		case "version":
			v, err := next(name, inline, hasInline)
			if err != nil {
				return out, err
			}
			out.versions = append(out.versions, capabilityVersionInput{version: v})
			cur = len(out.versions) - 1
		case "available-from", "deprecated-from", "terminated-from":
			v, err := next(name, inline, hasInline)
			if err != nil {
				return out, err
			}
			if cur < 0 {
				return out, fmt.Errorf("--%s must follow a --version", name)
			}
			switch name {
			case "available-from":
				out.versions[cur].availableFrom = v
			case "deprecated-from":
				out.versions[cur].deprecatedFrom = v
			case "terminated-from":
				out.versions[cur].terminatedFrom = v
			}
		default:
			return out, fmt.Errorf("unknown flag --%s", name)
		}
	}

	// Validate & normalize dates now so errors surface before prompting/writing.
	for idx := range out.versions {
		if err := normalizeVersionDates(&out.versions[idx]); err != nil {
			return out, fmt.Errorf("version %q: %w", out.versions[idx].version, err)
		}
	}

	return out, nil
}

// splitLongFlag parses "--name" or "--name=value".
func splitLongFlag(tok string) (name, val string, hasInline bool) {
	body := strings.TrimPrefix(tok, "--")
	if eq := strings.IndexByte(body, '='); eq >= 0 {
		return body[:eq], body[eq+1:], true
	}
	return body, "", false
}

// normalizeVersionDates validates and rewrites the version's date strings to
// RFC3339. Empty values are left empty.
func normalizeVersionDates(v *capabilityVersionInput) error {
	for _, f := range []struct {
		label string
		ptr   *string
	}{
		{"available-from", &v.availableFrom},
		{"deprecated-from", &v.deprecatedFrom},
		{"terminated-from", &v.terminatedFrom},
	} {
		if *f.ptr == "" {
			continue
		}
		norm, err := normalizeDate(*f.ptr)
		if err != nil {
			return fmt.Errorf("%s: %w", f.label, err)
		}
		*f.ptr = norm
	}
	return nil
}

// normalizeDate parses a date in any accepted layout and returns it as RFC3339.
func normalizeDate(s string) (string, error) {
	s = strings.TrimSpace(s)
	for _, layout := range dateInputLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC().Format(time.RFC3339), nil
		}
	}
	return "", fmt.Errorf("invalid date %q (use YYYY-MM-DD or RFC3339)", s)
}

// buildCapabilitySpec assembles the YAML spec map for a Capability.
//
// The versions carry the real, ingress-backed fields (capabilityVersionId +
// version{version,availableFrom,deprecatedFrom,terminatedFrom}). Each version
// also gets a scaffolded single variant and an empty dependencies list. The
// Variant and Dependency entities are not yet part of the modelsrv model (see
// emeland-io/modelsrv#177); the modelsrv ingress currently ignores these keys,
// so they act as forward-looking scaffolding rather than functional data.
func buildCapabilitySpec(id uuid.UUID, displayName string, versions []capabilityVersionInput) map[string]any {
	if len(versions) == 0 {
		// Keep a single empty version so the construct is always "full".
		versions = []capabilityVersionInput{{}}
	}

	verList := make([]any, 0, len(versions))
	for _, v := range versions {
		ver := map[string]any{"version": v.version}
		if v.availableFrom != "" {
			ver["availableFrom"] = v.availableFrom
		}
		if v.deprecatedFrom != "" {
			ver["deprecatedFrom"] = v.deprecatedFrom
		}
		if v.terminatedFrom != "" {
			ver["terminatedFrom"] = v.terminatedFrom
		}

		entry := map[string]any{
			"capabilityVersionId": uuid.New().String(),
			"version":             ver,
			// Scaffold: not yet modelled by modelsrv (emeland-io/modelsrv#177).
			"dependencies": []any{},
			"variants": []any{
				map[string]any{
					"variantId":    uuid.New().String(),
					"displayName":  "default",
					"dependencies": []any{},
				},
			},
		}
		verList = append(verList, entry)
	}

	return map[string]any{
		"capabilityId": id.String(),
		"displayName":  displayName,
		"versions":     verList,
	}
}

// promptForVersions asks interactively for at least one version when none were
// supplied and stdin is a terminal. When not interactive, missing values are
// left as-is (the scaffold still produces a valid, if sparse, construct).
func promptForVersions(cmd *cobra.Command, versions []capabilityVersionInput) ([]capabilityVersionInput, error) {
	if !isInteractive() {
		return versions, nil
	}

	if len(versions) == 0 {
		ver, err := promptLine(cmd, "Version string (e.g. 1.0.0)", "")
		if err != nil {
			return nil, err
		}
		versions = append(versions, capabilityVersionInput{version: ver})
	}

	for i := range versions {
		v := &versions[i]
		if v.version == "" {
			s, err := promptLine(cmd, fmt.Sprintf("Version #%d string", i+1), "")
			if err != nil {
				return nil, err
			}
			v.version = s
		}
		if err := promptDate(cmd, "  available from (YYYY-MM-DD)", &v.availableFrom); err != nil {
			return nil, err
		}
		if err := promptDate(cmd, "  deprecated from (YYYY-MM-DD)", &v.deprecatedFrom); err != nil {
			return nil, err
		}
		if err := promptDate(cmd, "  terminated from (YYYY-MM-DD)", &v.terminatedFrom); err != nil {
			return nil, err
		}
	}
	return versions, nil
}

// promptDate prompts for a date only if the value is currently empty, and
// normalizes it to RFC3339. Empty input leaves the value unset.
func promptDate(cmd *cobra.Command, label string, dst *string) error {
	if *dst != "" {
		return nil
	}
	s, err := promptLine(cmd, label, "")
	if err != nil {
		return err
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	norm, err := normalizeDate(s)
	if err != nil {
		return err
	}
	*dst = norm
	return nil
}

// promptLine writes a prompt to the command's output and reads one line from
// stdin. Returns the trimmed input (or the default if empty).
func promptLine(cmd *cobra.Command, label, def string) (string, error) {
	out := cmd.OutOrStdout()
	if def != "" {
		_, _ = fmt.Fprintf(out, "%s [%s]: ", label, def)
	} else {
		_, _ = fmt.Fprintf(out, "%s: ", label)
	}
	reader := bufio.NewReader(cmdInput(cmd))
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def, nil
	}
	return line, nil
}

// cmdInput returns the reader used for interactive prompts. It honors a reader
// set on the command (used in tests) and otherwise falls back to os.Stdin.
func cmdInput(cmd *cobra.Command) io.Reader {
	if in := cmd.InOrStdin(); in != nil {
		return in
	}
	return os.Stdin
}

// isInteractive reports whether stdin is a terminal, so that non-interactive
// invocations (tests, pipes, /dev/null, CI) never block on or emit a prompt.
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
