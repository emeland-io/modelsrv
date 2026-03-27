package context_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCtx(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Context (model) Suite")
}
