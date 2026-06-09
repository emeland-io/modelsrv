package main

import (
	"fmt"
	"strings"
	"text/template"
)

func specByName(name string) TypeSpec {
	for _, s := range allTypes {
		if s.Name == name {
			return s
		}
	}
	return TypeSpec{}
}

func handlerMethodName(spec TypeSpec) string {
	if spec.HandlerMethodSuffix != "" {
		return spec.HandlerMethodSuffix
	}
	return spec.Name
}

func storeDeleteCall(spec TypeSpec) string {
	idExpr := fmt.Sprintf(`testIDs["%s"]`, spec.Name)
	if spec.HandlerDeleteName != "" {
		return spec.HandlerDeleteName + "(" + idExpr + ")"
	}
	return "Delete" + handlerMethodName(spec) + "ById(" + idExpr + ")"
}

func storeIDGetter(spec TypeSpec) string {
	return "Get" + spec.IDField + "()"
}

func setupObjectVar(testSetup string) string {
	first := strings.TrimSpace(strings.Split(strings.TrimSpace(testSetup), "\n")[0])
	parts := strings.SplitN(first, ":=", 2)
	if len(parts) < 2 {
		return "obj"
	}
	return strings.TrimSpace(parts[0])
}

func replicationObjectsArg(spec TypeSpec) string {
	return "[]any{" + setupObjectVar(spec.TestSetup) + "}"
}

func applyObjectSetup(spec TypeSpec) string {
	lines := strings.Split(strings.TrimSpace(spec.TestSetup), "\n")
	var out []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "require.NoError(t, m.Add") {
			continue
		}
		line = strings.ReplaceAll(line, fmt.Sprintf(`testIDs["%s"]`, spec.Name), "resourceID")
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func buildTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"lower": strings.ToLower,
		"refParamType": func(r RefByRefSpec) string {
			if r.ParamGoType != "" {
				return r.ParamGoType
			}
			return r.ResourceTypeName
		},
		"handlerQualifier": func(s TypeSpec) string {
			if s.HandlerPkgAlias != "" {
				return s.HandlerPkgAlias
			}
			return s.Dir
		},
		"handlerMethod":         handlerMethodName,
		"storeDeleteCall":       storeDeleteCall,
		"storeIDGetter":         storeIDGetter,
		"setupObjectVar":        setupObjectVar,
		"applyObjectSetup":      applyObjectSetup,
		"replicationObjectsArg": replicationObjectsArg,
	}
}
