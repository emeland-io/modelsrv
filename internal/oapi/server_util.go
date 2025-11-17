/*
Copyright 2025 Lutz Behnke <lutz.behnke@gmx.de>.

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

package oapi

import (
	"context"
	"io"
	"maps"
	"strings"
	"text/template"
)

func acceptsHTML(ctx context.Context) bool {
	return strings.EqualFold(ctx.Value(HEADER_ACCEPT).(string),
		string(CONTENT_TYPE_HTML))
}

func renderHTML(resp any, template *template.Template) (io.Reader, int64) {
	body := new(strings.Builder)

	template.Execute(body, resp)
	return strings.NewReader(body.String()), int64(body.Len())
}

func cloneAnnotations(annos map[string]string) *[]Annotation {
	retval := make([]Annotation, 0)
	for key, value := range maps.All(annos) {
		retval = append(retval, Annotation{Key: key, Value: value})
	}
	return &retval
}
