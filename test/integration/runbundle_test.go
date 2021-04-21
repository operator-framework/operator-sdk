package integration

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Runbundle test - local", func() {

	var (
		PortNumber    int
		runBundleImg  string
		kubeNamespace string
		operatorImage string
	)

	Describe("run bundle test", func() {
		BeforeEach(func() {
			_ = os.Getenv("LOCAL_IMAGE_REGISTRY")
			PortNumber, _ = strconv.Atoi(os.Getenv("PORT_NUMBER"))
			runBundleImg = os.Getenv("BUNDLE_IMAGE_NAME")
			kubeNamespace = os.Getenv("KUBECTL_NAMESPACE")
			operatorImage = os.Getenv("OPERATOR_IMAGE_NAME")
		})

		It("should be a pleasant experience", func() {
			fmt.Printf("PORT_NUMBER = %+v\n", PortNumber)

			By("push the image to a local registry")
			err := tc.Make("docker-push", "IMG="+runBundleImg)
			Expect(err).NotTo(HaveOccurred())

			By("push the image to a local registry")
			err = tc.Make("docker-push", "IMG="+operatorImage)
			Expect(err).NotTo(HaveOccurred())

			By("running the operator bundle using `run bundle` command")
			runBundleCmd := exec.Command(tc.BinaryName, "run", "bundle", runBundleImg, "--namespace", kubeNamespace, "--timeout", "4m")
			_, err = tc.Run(runBundleCmd)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
