// Copyright 2018 The Operator-SDK Authors
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
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"

	kcorev1 "k8s.io/api/core/v1"
)

var _ = Describe("proxyTests", func() {
	t := GinkgoT()

	It("should retrieve resources from the cache", func() {
		if testing.Short() {
			Skip("skipping ansible proxy testing in short mode")
		}
		po, err := createTestPod("test", "test-watched-namespace", testClient)
		if err != nil {
			t.Fatalf("Failed to create the pod: %v", err)
		}
		defer func() {
			if err := testClient.Delete(context.Background(), po); err != nil {
				t.Fatalf("Failed to delete the pod: %v", err)
			}
		}()

		resp, err := http.Get("http://localhost:8888/api/v1/namespaces/test-watched-namespace/pods/test")
		if err != nil {
			t.Fatalf("Error getting pod from proxy: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
				t.Errorf("Failed to close response body: (%v)", err)
			}
		}()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Error reading response body: %v", err)
		}
		// Should only be one string from 'X-Cache' header (explicitly set to HIT in proxy)
		if resp.Header["X-Cache"] == nil {
			t.Fatalf("Object was not retrieved from cache")
			if resp.Header["X-Cache"][0] != "HIT" {
				t.Fatalf("Cache response header found but got [%v], expected [HIT]", resp.Header["X-Cache"][0])
			}
		}
		data := kcorev1.Pod{}
		err = json.Unmarshal(body, &data)
		if err != nil {
			t.Fatalf("Error parsing response: %v", err)
		}
		if data.Name != "test" {
			t.Fatalf("Got unexpected pod name: %#v", data.Name)
		}
	})
})
