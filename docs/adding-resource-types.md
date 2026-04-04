# Adding Resource Types

As may be obvious in the multiple empty pages in the EmELand book, that there are still a number of resource types missing from the model.

This document should help you when adding an additional resource type.

1. Update the resource type enum in the `ResourceRef` resource of the OpenAPI spec in the `api/EmergingEnterpriseLandscape-0.1.0-oapi-3.0.3.yaml` file.
1. Add endpoints for listing and retrieving the resources of the new type by Id to the same OAPI file.
1. add the type to the list of resource types in `pkg/events/events.go`
1. add the type to the documentKinds map in  `pkg/filesensor/parse.go`
