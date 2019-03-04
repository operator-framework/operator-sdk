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
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/pkg/ansible/proxy/controllermap"

	kcorev1 "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ansible proxy testing in short mode")
	}
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{Namespace: "default"})
	if err != nil {
		t.Fatalf("Failed to instantiate manager: %v", err)
	}
	done := make(chan error)
	cMap := controllermap.NewControllerMap()
	err = Run(done, Options{
		Address:           "localhost",
		Port:              8888,
		KubeConfig:        mgr.GetConfig(),
		Cache:             nil,
		RESTMapper:        mgr.GetRESTMapper(),
		ControllerMap:     cMap,
		WatchedNamespaces: []string{"default"},
	})
	if err != nil {
		t.Fatalf("Error starting proxy: %v", err)
	}

	po := createPod("test", "default", mgr.GetConfig())

	resp, err := http.Get("http://localhost:8888/api/v1/namespaces/default/pods/test")
	if err != nil {
		t.Fatalf("Error getting pod from proxy: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil && !fileutil.IsClosedError(err) {
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
	if data.Name != "test" {
		t.Fatalf("Got unexpected pod name: %#v", data.Name)
	}
	deletePod(po, mgr.GetConfig())
}

func createPod(name, namespace string, cfg *rest.Config) runtime.Object {
	three := int64(3)
	pod := &kcorev1.Pod{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"test-label": name,
			},
		},
		Spec: kcorev1.PodSpec{
			Containers:            []kcorev1.Container{{Name: "nginx", Image: "nginx"}},
			RestartPolicy:         "Always",
			ActiveDeadlineSeconds: &three,
		},
	}
	cl, err := client.New(cfg, client.Options{})
	err = cl.Create(context.Background(), pod)
	if err != nil {
		return nil
	}
	return pod
}

func deletePod(pod runtime.Object, cfg *rest.Config) {
	cl, err := client.New(cfg, client.Options{})
	err = cl.Delete(context.Background(), pod)
	if err != nil {
		return
	}
}
