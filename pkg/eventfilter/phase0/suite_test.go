package phase0_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPhase0(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "pkg/eventfilter/phase0 Suite")
}
