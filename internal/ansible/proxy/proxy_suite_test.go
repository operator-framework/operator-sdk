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
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-sdk/internal/ansible/proxy/controllermap"
	kcorev1 "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var testMgr manager.Manager

var testClient client.Client

func TestProxy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Proxy Test Suite")
}

var _ = BeforeSuite(func() {
	if testing.Short() {
		return
	}
	var err error
	testMgr, err = manager.New(config.GetConfigOrDie(), manager.Options{Namespace: "default"})
	if err != nil {
		Fail(fmt.Sprintf("Failed to instantiate manager: %v", err))
	}
	done := make(chan error)
	cMap := controllermap.NewControllerMap()
	err = Run(done, Options{
		Address:           "localhost",
		Port:              8888,
		KubeConfig:        testMgr.GetConfig(),
		Cache:             nil,
		RESTMapper:        testMgr.GetRESTMapper(),
		ControllerMap:     cMap,
		WatchedNamespaces: []string{"test-watched-namespace"},
		OwnerInjection:    true,
	})
	if err != nil {
		Fail(fmt.Sprintf("Error starting proxy: %v", err))
	}
	testClient, err = client.New(testMgr.GetConfig(), client.Options{})
	if err != nil {
		Fail(fmt.Sprintf("Failed to create the client: %v", err))
	}
	_, err = createTestNamespace("test-watched-namespace", testClient)
	if err != nil {
		Fail(fmt.Sprintf("Failed to create watched namespace: %v", err))
	}
})

var _ = AfterSuite(func() {
	if testing.Short() {
		return
	}
	err := testClient.Delete(context.Background(), &kcorev1.Namespace{
		ObjectMeta: kmetav1.ObjectMeta{
			Name: "test-watched-namespace",
			Labels: map[string]string{
				"test-label": "test-watched-namespace",
			},
		},
	})

	if err != nil {
		Fail(fmt.Sprintf("Failed to clean up namespace: %v:", err))
	}
})

func createTestNamespace(name string, cl client.Client) (client.Object, error) {
	ns := &kcorev1.Namespace{
		ObjectMeta: kmetav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"test-label": name,
			},
		},
	}
	if err := cl.Create(context.Background(), ns); err != nil {
		return nil, err
	}
	return ns, nil
}

func createTestPod(name, namespace string, cl client.Client) (client.Object, error) {
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
	if err := cl.Create(context.Background(), pod); err != nil {
		return nil, err
	}
	return pod, nil
}
