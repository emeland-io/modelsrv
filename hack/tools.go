//go:build tools
// +build tools

package tools

import (
	_ "github.com/golang/mock/mockgen"                           //
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen" //
)
