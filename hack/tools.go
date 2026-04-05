//go:build tools
// +build tools

package hack

import (
	_ "github.com/golang/mock/mockgen"                           //
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen" //
)
