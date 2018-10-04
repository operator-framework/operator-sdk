// Modified from github.com/kubernetes-sigs/controller-tools/pkg/scaffold/input/input_suite_test.go

package input_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestInput(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Input Suite")
}
