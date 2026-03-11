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
	"go.emeland.io/modelsrv/pkg/model"
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

func cloneAnnotations(annos model.Annotations) *[]Annotation {
	retval := make([]Annotation, 0)
	for key := range annos.GetKeys() {
		retval = append(retval, Annotation{Key: key, Value: annos.GetValue(key)})
	}
	return &retval
}

// Generic helper to build instance list responses
type hasIdAndName interface {
	GetInstanceId() uuid.UUID
	GetDisplayName() string
}

func buildInstanceList[T hasIdAndName](baseURL, path string, items []T) []InstanceListItem {
	result := make([]InstanceListItem, 0, len(items))
	for _, item := range items {
		id := item.GetInstanceId()
		name := item.GetDisplayName()
		ref := fmt.Sprintf("%s%s/%s", baseURL, path, id.String())
		result = append(result, InstanceListItem{
			InstanceId:  &id,
			DisplayName: &name,
			Reference:   &ref,
		})
	}
	return result
}

// Specialized version for APIs (uses GetApiId instead of GetInstanceId)
func buildApiList(baseURL string, items []model.API) []InstanceListItem {
	result := make([]InstanceListItem, 0, len(items))
	for _, item := range items {
		id := item.GetApiId()
		name := item.GetDisplayName()
		ref := fmt.Sprintf("%s/landscape/apis/%s", baseURL, id.String())
		result = append(result, InstanceListItem{
			InstanceId:  &id,
			DisplayName: &name,
			Reference:   &ref,
		})
	}
	return result
}

// Specialized version for Components (uses GetComponentId)
func buildComponentList(baseURL string, items []model.Component) []InstanceListItem {
	result := make([]InstanceListItem, 0, len(items))
	for _, item := range items {
		id := item.GetComponentId()
		name := item.GetDisplayName()
		ref := fmt.Sprintf("%s/landscape/components/%s", baseURL, id.String())
		result = append(result, InstanceListItem{
			InstanceId:  &id,
			DisplayName: &name,
			Reference:   &ref,
		})
	}
	return result
}

// Specialized version for Findings (uses GetFindingId and GetSummary)
func buildFindingList(baseURL string, items []model.Finding) []InstanceListItem {
	result := make([]InstanceListItem, 0, len(items))
	for _, item := range items {
		id := item.GetFindingId()
		summary := item.GetSummary()
		ref := fmt.Sprintf("%s/landscape/findings/%s", baseURL, id.String())
		result = append(result, InstanceListItem{
			InstanceId:  &id,
			DisplayName: &summary,
			Reference:   &ref,
		})
	}
	return result
}

/*
cloneResourceRefs creates a deep copy of the given ResourceRef slice.

	Warning: note the change in the type of the items from reference to value.
*/
func cloneResourceRefs(resourceRef []*model.ResourceRef) []ResourceRef {
	respArr := make([]ResourceRef, 0, len(resourceRef))
	for _, resRef := range resourceRef {
		respArr = append(respArr, ResourceRef{
			ResourceType: resRef.ResourceType,
			ResourceId:   resRef.ResourceId,
		})
	}
	return respArr
}
