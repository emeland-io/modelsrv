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
		kind: "System", idField: "systemId",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the system"},
			{name: "abstract", specKey: "abstract", usage: "Whether the system is abstract", isBool: true},
			{name: "parent", specKey: "parent", usage: "Parent system UUID"},
		},
	},
	{
		use: "api", short: "Create an API resource",
		kind: "API", idField: "apiId",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the API"},
			{name: "type", specKey: "type", usage: "API type (OpenAPI, GraphQL, GRPC, Other)"},
			{name: "system", specKey: "system", usage: "System UUID this API belongs to"},
		},
	},
	{
		use: "component", short: "Create a Component resource",
		kind: "Component", idField: "componentId",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the component"},
			{name: "system", specKey: "system", usage: "System UUID this component belongs to"},
		},
	},
	{
		use: "context", short: "Create a Context resource",
		kind: "Context", idField: "contextId",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the context"},
			{name: "parent", specKey: "parent", usage: "Parent context UUID"},
			{name: "type", specKey: "type", usage: "Context type UUID"},
		},
	},
	{
		use: "context-type", short: "Create a ContextType resource",
		kind: "ContextType", idField: "contextTypeId",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the context type"},
		},
	},
	{
		use: "node", short: "Create a Node resource",
		kind: "Node", idField: "nodeId",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the node"},
			{name: "node-type", specKey: "nodeType", usage: "Node type UUID"},
		},
	},
	{
		use: "node-type", short: "Create a NodeType resource",
		kind: "NodeType", idField: "nodeTypeId",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the node type"},
		},
	},
	{
		use: "finding", short: "Create a Finding resource",
		kind: "Finding", idField: "findingId", nameField: "summary",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the finding"},
		},
	},
	{
		use: "finding-type", short: "Create a FindingType resource",
		kind: "FindingType", idField: "findingTypeId",
		flags: []flagDef{
			{name: "desc", specKey: "description", usage: "Description of the finding type"},
		},
	},
	{
		use: "system-instance", short: "Create a SystemInstance resource",
		kind: "SystemInstance", idField: "instanceId",
		flags: []flagDef{
			{name: "system", specKey: "system", usage: "System UUID this instance refers to"},
			{name: "context", specKey: "context", usage: "Context UUID for this instance"},
		},
	},
	{
		use: "component-instance", short: "Create a ComponentInstance resource",
		kind: "ComponentInstance", idField: "instanceId",
		flags: []flagDef{
			{name: "component", specKey: "component", usage: "Component UUID this instance refers to"},
			{name: "system-instance", specKey: "systemInstance", usage: "SystemInstance UUID"},
		},
	},
	{
		use: "api-instance", short: "Create an ApiInstance resource",
		kind: "ApiInstance", idField: "instanceId",
		flags: []flagDef{
			{name: "api", specKey: "api", usage: "API UUID this instance refers to"},
			{name: "system-instance", specKey: "systemInstance", usage: "SystemInstance UUID"},
		},
	},
}

func init() {
	for _, def := range resourceTypes {
		registerResourceCmd(def)
	}
}
