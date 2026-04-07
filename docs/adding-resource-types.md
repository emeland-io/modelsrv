# Adding Resource Types

As may be obvious in the multiple empty pages in the EmELand book, that there are still a number of resource types missing from the model.

This document should help you when adding an additional resource type.

1. Update the resource type enum in the `ResourceRef` resource of the OpenAPI spec in the `api/EmergingEnterpriseLandscape-0.1.0-oapi-3.0.3.yaml` file.
1. Add endpoints for listing and retrieving the resources of the new type by Id to the same OAPI file.
1. add the type to the list of resource types in `pkg/events/events.go`
1. add the type to the documentKinds map in  `pkg/filesensor/parse.go`
1. implement missing methods for the type `ApiServer` in `internal/oapi/server.go`. You can start with auto-generated functions that simply call `panic(unimplemented)`, but fulfill the interface requirement.
1. define the new resource type in `tools/gen/specs.go`.  Ensure that you not only the fields of the resource, but also define any required Ref structures.
1. define any missing Ref types that are missing in the sub-package of `pkg/model` in which you have just added the new resource type.
1. create a Model sub-interface for the new resource type and add to full model in `pkg/model/structure.go`
1. add the Id-to-resource maps to the modelData structure and add required initialization code to the `NewModel`function in `pkg/model/structure.go`
1. implement the missing methods for the compound `Model` interface in `pkg/model`
1. Add error codes for missing resources to `pkg/model/common/errors.go`
