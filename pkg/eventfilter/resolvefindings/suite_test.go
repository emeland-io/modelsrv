package resolvefindings_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestResolveFindings(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "pkg/eventfilter/resolvefindings Suite")
}
