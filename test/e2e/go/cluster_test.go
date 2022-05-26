// Copyright 2020 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Modified from https://github.com/kubernetes-sigs/kubebuilder/tree/39224f0/test/e2e/v3

package e2e_go_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-sdk/internal/testutils"
	"github.com/operator-framework/operator-sdk/testutils/e2e"
	"github.com/operator-framework/operator-sdk/testutils/e2e/metrics"
	"github.com/operator-framework/operator-sdk/testutils/e2e/operator"
)

var _ = Describe("operator-sdk", func() {
	var metricsClusterRoleBindingName string

	Context("built with operator-sdk", func() {

		BeforeEach(func() {
			metricsClusterRoleBindingName = fmt.Sprintf("%s-metrics-reader", goSample.Name())

			By("installing the CRDs on the cluster")
			Expect(operator.InstallCRDs(goSample)).To(Succeed())

			By("deploying project on the cluster")
			Expect(operator.DeployOperator(goSample, image)).To(Succeed())
		})

		AfterEach(func() {
			By("cleaning up metrics")
			Expect(metrics.CleanUpMetrics(kctl, metricsClusterRoleBindingName)).To(Succeed())

			By("cleaning up created API objects during test process")
			// TODO(estroz): go/v2 does not have this target, so generalize once tests are refactored.
			Expect(operator.UndeployOperator(goSample)).To(Succeed())

			By("ensuring that the namespace was deleted")
			testutils.WrapWarnOutput(kctl.Wait(false, "namespace", "foo", "--for", "delete", "--timeout", "2m"))
		})

		It("should run correctly in a cluster", func() {
			By("checking if the Operator project Pod is running")
			verifyControllerUp := func() error {
				return operator.EnsureOperatorRunning(kctl, 1, "controller-manager", "controller-manager")
			}
			Eventually(verifyControllerUp, 2*time.Minute, time.Second).Should(Succeed())

			By("ensuring the created ServiceMonitor for the manager")
			_, err := kctl.Get(
				true,
				"ServiceMonitor",
				fmt.Sprintf("%s-controller-manager-metrics-monitor", goSample.Name()))
			Expect(err).NotTo(HaveOccurred())

			By("ensuring the created metrics Service for the manager")
			_, err = kctl.Get(
				true,
				"Service",
				fmt.Sprintf("%s-controller-manager-metrics-service", goSample.Name()))
			Expect(err).NotTo(HaveOccurred())

			By("creating an instance of CRs")
			// currently controller-runtime doesn't provide a readiness probe, we retry a few times
			// we can change it to probe the readiness endpoint after CR supports it.
			Eventually(func() error {
				return e2e.CreateCustomResources(goSample, kctl)
			}, time.Minute, time.Second).Should(Succeed())

			_ = metrics.GetMetrics(goSample, kctl, metricsClusterRoleBindingName)

			// The controller updates memcacheds' status.nodes with a list of pods it is replicated across
			// on a successful reconcile.
			By("validating that the created resource object gets reconciled in the controller")
			var status string
			getStatus := func() error {
				status, err = kctl.Get(true, "memcacheds", "memcached-sample", "-o", "jsonpath={.status.nodes}")
				if err == nil && strings.TrimSpace(status) == "" {
					err = errors.New("empty status, continue")
				}
				return err
			}
			Eventually(getStatus, 1*time.Minute, time.Second).Should(Succeed())
			var nodes []string
			Expect(json.Unmarshal([]byte(status), &nodes)).To(Succeed())
			Expect(len(nodes)).To(BeNumerically(">", 0))
		})
	})
})
