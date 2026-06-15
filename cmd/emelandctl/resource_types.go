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

// resourceTypes defines all resource types available via "emelandctl create <type>".
// Each entry is turned into a cobra subcommand by registerResourceCmd.
var resourceTypes = []resourceDef{
	{
		use: "system", short: "Create a System resource",
		kind: "System", idField: "systemId", listPath: "/landscape/systems",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the system"},
			{name: "abstract", specKey: "abstract", usage: "Whether the system is abstract", isBool: true},
			{name: "parent", specKey: "parent", usage: "Parent system UUID"},
		},
	},
	{
		use: "api", short: "Create an API resource",
		kind: "API", idField: "apiId", listPath: "/landscape/apis",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the API"},
			{name: "type", specKey: "type", usage: "API type (OpenAPI, GraphQL, GRPC, Other)"},
			{name: "system", specKey: "system", usage: "System UUID this API belongs to"},
		},
	},
	{
		use: "component", short: "Create a Component resource",
		kind: "Component", idField: "componentId", listPath: "/landscape/components",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the component"},
			{name: "system", specKey: "system", usage: "System UUID this component belongs to"},
		},
	},
	{
		use: "context", short: "Create a Context resource",
		kind: "Context", idField: "contextId", listPath: "/landscape/contexts",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the context"},
			{name: "parent", specKey: "parent", usage: "Parent context UUID"},
			{name: "type", specKey: "type", usage: "Context type UUID"},
		},
	},
	{
		use: "context-type", short: "Create a ContextType resource",
		kind: "ContextType", idField: "contextTypeId", listPath: "/landscape/contextTypes",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the context type"},
		},
	},
	{
		use: "node", short: "Create a Node resource",
		kind: "Node", idField: "nodeId", listPath: "/landscape/nodes",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the node"},
			{name: "node-type", specKey: "nodeType", usage: "Node type UUID"},
		},
	},
	{
		use: "node-type", short: "Create a NodeType resource",
		kind: "NodeType", idField: "nodeTypeId", listPath: "/landscape/nodeTypes",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the node type"},
		},
	},
	{
		use: "finding", short: "Create a Finding resource",
		kind: "Finding", idField: "findingId", nameField: "summary", listPath: "/landscape/findings",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the finding"},
		},
	},
	{
		use: "finding-type", short: "Create a FindingType resource",
		kind: "FindingType", idField: "findingTypeId", listPath: "/landscape/findingTypes",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the finding type"},
		},
	},
	{
		use: "system-instance", short: "Create a SystemInstance resource",
		kind: "SystemInstance", idField: "instanceId", listPath: "/landscape/system-instances",
		flags: []flagDef{
			{name: "system", specKey: "system", usage: "System UUID this instance refers to"},
			{name: "context", specKey: "context", usage: "Context UUID for this instance"},
		},
	},
	{
		use: "component-instance", short: "Create a ComponentInstance resource",
		kind: "ComponentInstance", idField: "instanceId", listPath: "/landscape/component-instances",
		flags: []flagDef{
			{name: "component", specKey: "component", usage: "Component UUID this instance refers to"},
			{name: "system-instance", specKey: "systemInstance", usage: "SystemInstance UUID"},
		},
	},
	{
		use: "api-instance", short: "Create an ApiInstance resource",
		kind: "ApiInstance", idField: "instanceId", listPath: "/landscape/api-instances",
		flags: []flagDef{
			{name: "api", specKey: "api", usage: "API UUID this instance refers to"},
			{name: "system-instance", specKey: "systemInstance", usage: "SystemInstance UUID"},
		},
	},
	{
		use: "product", short: "Create a Product resource",
		kind: "Product", idField: "productId", listPath: "/landscape/products",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the product"},
			{name: "vendor", specKey: "vendor", usage: "Vendor org unit UUID"},
		},
	},
	{
		use: "artifact", short: "Create an Artifact resource",
		kind: "Artifact", idField: "artifactId", listPath: "/landscape/artifacts",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the artifact"},
		},
	},
	{
		use: "artifact-instance", short: "Create an ArtifactInstance resource",
		kind: "ArtifactInstance", idField: "artifactInstanceId", listPath: "/landscape/artifactInstances",
		flags: []flagDef{
			{name: "artifact", specKey: "artifact", usage: "Artifact UUID this instance refers to"},
		},
	},
	{
		use: "org-unit", short: "Create an OrgUnit resource",
		kind: "OrgUnit", idField: "orgUnitId", listPath: "/landscape/orgUnits",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the org unit"},
			{name: "parent", specKey: "parent", usage: "Parent org unit UUID"},
		},
	},
	{
		use: "group", short: "Create a Group resource",
		kind: "Group", idField: "groupId", listPath: "/landscape/groups",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the group"},
		},
	},
	{
		use: "identity", short: "Create an Identity resource",
		kind: "Identity", idField: "identityId", listPath: "/landscape/identities",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the identity"},
		},
	},
	{
		use: "permission-spec", short: "Create a PermissionSpec resource",
		kind: "PermissionSpec", idField: "permissionSpecId", listPath: "/landscape/permissionSpecs",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the permission spec"},
		},
	},
	{
		use: "role-spec", short: "Create a RoleSpec resource",
		kind: "RoleSpec", idField: "roleSpecId", listPath: "/landscape/roleSpecs",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the role spec"},
		},
	},
	{
		use: "permission", short: "Create a Permission resource",
		kind: "Permission", idField: "permissionId", listPath: "/landscape/permissions",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the permission"},
		},
	},
	{
		use: "role", short: "Create a Role resource",
		kind: "Role", idField: "roleId", listPath: "/landscape/roles",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the role"},
		},
	},
	{
		use: "binding", short: "Create a Binding resource",
		kind: "Binding", idField: "bindingId", listPath: "/landscape/bindings",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the binding"},
		},
	},
}
