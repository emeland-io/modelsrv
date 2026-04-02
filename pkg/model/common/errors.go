package common

import "fmt"

var (
	ErrNodeNotFound              = fmt.Errorf("node not found")
	ErrNodeTypeNotFound          = fmt.Errorf("node Type not found")
	ErrContextNotFound           = fmt.Errorf("context not found")
	ErrContextTypeNotFound       = fmt.Errorf("context Type not found")
	ErrSystemNotFound            = fmt.Errorf("system not found")
	ErrSystemInstanceNotFound    = fmt.Errorf("system Instance not found")
	ErrApiNotFound               = fmt.Errorf("api not found")
	ErrApiInstanceNotFound       = fmt.Errorf("api Instance not found")
	ErrComponentNotFound         = fmt.Errorf("component not found")
	ErrComponentInstanceNotFound = fmt.Errorf("component Instance not found")
	ErrFindingNotFound           = fmt.Errorf("finding not found")
	ErrFindingTypeNotFound       = fmt.Errorf("finding Type not found")

	ErrUUIDNotSet = fmt.Errorf("resource identifier UUID not set")
)
