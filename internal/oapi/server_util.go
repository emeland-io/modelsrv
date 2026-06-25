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
	"fmt"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/iam"
)

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

func cloneResourceRefs(resourceRef []*common.ResourceRef) []ResourceRef {
	respArr := make([]ResourceRef, 0, len(resourceRef))
	for _, resRef := range resourceRef {
		respArr = append(respArr, ResourceRef{
			ResourceType: resRef.ResourceType.String(),
			ResourceId:   resRef.ResourceId,
		})
	}
	return respArr
}

func iamPermissionSpecRefsToOpenAPIUUIDs(refs []*iam.PermissionSpecRef) *[]openapi_types.UUID {
	if len(refs) == 0 {
		return nil
	}
	out := make([]openapi_types.UUID, 0, len(refs))
	for _, ref := range refs {
		out = append(out, openapi_types.UUID(ref.EffectivePermissionSpecID()))
	}
	return &out
}

func iamPermissionRefsToOpenAPIUUIDs(refs []*iam.PermissionRef) *[]openapi_types.UUID {
	if len(refs) == 0 {
		return nil
	}
	out := make([]openapi_types.UUID, 0, len(refs))
	for _, ref := range refs {
		out = append(out, openapi_types.UUID(ref.EffectivePermissionID()))
	}
	return &out
}

func iamResourceRefsOptional(refs []*common.ResourceRef) *[]ResourceRef {
	if len(refs) == 0 {
		return nil
	}
	dup := cloneResourceRefs(refs)
	return &dup
}

func iamSubjectToAPISubjectRef(sub *iam.SubjectRef) SubjectRef {
	if sub == nil {
		return SubjectRef{}
	}
	switch sub.EffectiveKind() {
	case iam.SubjectKindGroup:
		id := openapi_types.UUID(sub.EffectiveGroupID())
		return SubjectRef{GroupId: &id}
	case iam.SubjectKindIdentity:
		id := openapi_types.UUID(sub.EffectiveIdentityID())
		return SubjectRef{IdentityId: &id}
	default:
		return SubjectRef{}
	}
}
