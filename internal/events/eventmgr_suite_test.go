package eventmgr_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEventmgr(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "internal/events (eventmgr) Suite")
}
