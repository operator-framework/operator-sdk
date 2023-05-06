// Copyright 2021 The Operator-SDK Authors
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

package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/operator-framework/operator-sdk/internal/ansible/proxy/kubeconfig"
)

var _ = Describe("injectOwnerReferenceHandler", func() {

	Describe("ServeHTTP", func() {
		It("Should inject ownerReferences even when namespace is not explicitly set", func() {
			if testing.Short() {
				Skip("skipping ansible owner reference injection testing in short mode")
			}
			cm := corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-owner-ref-injection",
				},
				Data: map[string]string{
					"hello": "world",
				},
			}

			body, err := json.Marshal(cm)
			if err != nil {
				Fail("Failed to marshal body")
			}

			po, err := createTestPod("test-injection", "default", testClient)
			if err != nil {
				Fail(fmt.Sprintf("Failed to create pod: %v", err))
			}
			defer func() {
				if err := testClient.Delete(context.Background(), po); err != nil {
					Fail(fmt.Sprintf("Failed to delete the pod: %v", err))
				}
			}()

			req, err := http.NewRequest("POST", "http://localhost:8888/api/v1/namespaces/default/configmaps", bytes.NewReader(body))
			if err != nil {
				Fail(fmt.Sprintf("Failed to create http request: %v", err))
			}

			username, err := kubeconfig.EncodeOwnerRef(
				metav1.OwnerReference{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       po.GetName(),
					UID:        po.GetUID(),
				}, "default")
			if err != nil {
				Fail("Failed to encode owner reference")
			}
			req.SetBasicAuth(username, "unused")

			httpClient := http.Client{}

			defer func() {
				cleanupReq, err := http.NewRequest("DELETE", "http://localhost:8888/api/v1/namespaces/default/configmaps/test-owner-ref-injection", bytes.NewReader([]byte{}))
				if err != nil {
					Fail(fmt.Sprintf("Failed to delete configmap: %v", err))
				}
				_, err = httpClient.Do(cleanupReq)
				if err != nil {
					Fail(fmt.Sprintf("Failed to delete configmap: %v", err))
				}
			}()

			resp, err := httpClient.Do(req)
			if err != nil {
				Fail(fmt.Sprintf("Failed to create configmap: %v", err))
			}
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				Fail(fmt.Sprintf("Failed to read response body: %v", err))
			}
			var modifiedCM corev1.ConfigMap
			err = json.Unmarshal(respBody, &modifiedCM)
			if err != nil {
				Fail(fmt.Sprintf("Failed to unmarshal configmap: %v", err))
			}
			ownerRefs := modifiedCM.ObjectMeta.OwnerReferences

			Expect(ownerRefs).To(HaveLen(1))

			ownerRef := ownerRefs[0]

			Expect(ownerRef.APIVersion).To(Equal("v1"))
			Expect(ownerRef.Kind).To(Equal("Pod"))
			Expect(ownerRef.Name).To(Equal(po.GetName()))
			Expect(ownerRef.UID).To(Equal(po.GetUID()))
		})
	})
})
