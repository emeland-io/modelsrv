package filesensor_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFilesensor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Filesensor Suite")
}
