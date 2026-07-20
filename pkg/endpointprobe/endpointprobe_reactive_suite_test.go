package endpointprobe

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEndpointprobeReactive(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "pkg/endpointprobe Reactive Suite")
}
