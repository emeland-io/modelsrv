package common

import "fmt"

var (
	ErrNodeNotFound              = fmt.Errorf("Node not found")
	ErrNodeTypeNotFound          = fmt.Errorf("Node Type not found")
	ErrContextNotFound           = fmt.Errorf("Context not found")
	ErrContextTypeNotFound       = fmt.Errorf("Context Type not found")
	ErrSystemNotFound            = fmt.Errorf("System not found")
	ErrSystemInstanceNotFound    = fmt.Errorf("System Instance not found")
	ErrApiNotFound               = fmt.Errorf("API not found")
	ErrApiInstanceNotFound       = fmt.Errorf("API Instance not found")
	ErrComponentNotFound         = fmt.Errorf("Component not found")
	ErrComponentInstanceNotFound = fmt.Errorf("Component Instance not found")
	ErrFindingNotFound           = fmt.Errorf("Finding not found")
	ErrFindingTypeNotFound       = fmt.Errorf("Finding Type not found")

	ErrUUIDNotSet = fmt.Errorf("resource identifier UUID not set")
)
