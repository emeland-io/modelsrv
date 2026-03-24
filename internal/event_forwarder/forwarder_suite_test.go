package eventforwarder

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEventForwarder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Event Forwarder Suite")
}
