package eventfilter_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEventfilter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "pkg/eventfilter Suite")
}
