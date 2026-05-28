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
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/google/uuid"
)

//nolint:unused
func acceptsHTML(ctx context.Context) bool {
	return strings.EqualFold(ctx.Value(ctxKeyNegotiatedContentType).(string),
		string(CONTENT_TYPE_HTML))
}

//nolint:unused
func renderHTML(resp any, template *template.Template) (io.Reader, int64) {
	body := new(strings.Builder)

	if err := template.Execute(body, resp); err != nil {
		return nil, 0
	}

	return strings.NewReader(body.String()), int64(body.Len())
}

// Generic helper to build instance list responses
type hasIdAndName interface {
	GetResourceId() uuid.UUID
	GetResourceName() string
}

func buildInstanceList[T hasIdAndName](baseURL, path string, items []T) []InstanceListItem {
	result := make([]InstanceListItem, 0, len(items))
	for _, item := range items {
		id := item.GetResourceId()
		name := item.GetResourceName()
		ref := fmt.Sprintf("%s%s/%s", baseURL, path, id.String())
		result = append(result, InstanceListItem{
			InstanceId:  &id,
			DisplayName: &name,
			Reference:   &ref,
		})
	}
	return result
}
