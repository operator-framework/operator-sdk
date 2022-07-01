// Copyright 2022 The Operator-SDK Authors
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
	"fmt"
	"testing"

	openapi_v2 "github.com/google/gnostic/openapiv2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiconfigv1 "github.com/openshift/api/config/v1"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	disfake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/openapi"
	restclient "k8s.io/client-go/rest"
	cgotesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	crfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestProxyUtility(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Proxy Utility")
}

var _ = Describe("GetProxyVars", func() {
	var (
		cfg         *operator.Configuration
		scheme      *runtime.Scheme
		discov      discovery.DiscoveryInterface
		prox        *apiconfigv1.Proxy
		expectedEnv []corev1.EnvVar
	)
	BeforeEach(func() {
		scheme = runtime.NewScheme()
		discov = &disfake.FakeDiscovery{
			Fake: &cgotesting.Fake{
				Resources: []*metav1.APIResourceList{
					{
						GroupVersion: apiconfigv1.GroupVersion.String(),
					},
				},
			},
		}
		prox = &apiconfigv1.Proxy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Spec: apiconfigv1.ProxySpec{
				HTTPProxy:  "httpProxy",
				HTTPSProxy: "httpsProxy",
				NoProxy:    "noProxy",
			},
			Status: apiconfigv1.ProxyStatus{
				HTTPProxy:  "httpProxy",
				HTTPSProxy: "httpsProxy",
				NoProxy:    "noProxy",
			},
		}
		cfg = &operator.Configuration{
			Scheme: scheme,
			Client: validFakeClient(scheme, prox),
		}

		expectedEnv = []corev1.EnvVar{
			{
				Name:  "HTTP_PROXY",
				Value: prox.Status.HTTPProxy,
			},
			{
				Name:  "http_proxy",
				Value: prox.Status.HTTPProxy,
			},
			{
				Name:  "HTTPS_PROXY",
				Value: prox.Status.HTTPSProxy,
			},
			{
				Name:  "https_proxy",
				Value: prox.Status.HTTPSProxy,
			},
			{
				Name:  "NO_PROXY",
				Value: prox.Status.NoProxy,
			},
			{
				Name:  "no_proxy",
				Value: prox.Status.NoProxy,
			},
		}
	})

	It("should return the proxy", func() {
		p, err := GetProxyVars(cfg, discov)
		Expect(err).Should(BeNil())

		Expect(p).ShouldNot(BeNil())
		Expect(p).Should(Equal(expectedEnv))
	})

	It("should return nil if client can not find the proxy object", func() {
		cfg.Client = noProxyFakeClient(scheme)
		p, err := GetProxyVars(cfg, discov)
		Expect(err).Should(BeNil())
		Expect(p).Should(BeNil())
	})

	It("should return an error if the client errors when the error does not meet the IgnoreNotFound criteria", func() {
		cfg.Client = invalidFakeClient()
		p, err := GetProxyVars(cfg, discov)
		Expect(err).ShouldNot(BeNil())
		Expect(p).Should(BeNil())
	})

	It("should return nil if the discovery client does not find the proxy api", func() {
		discov = &disfake.FakeDiscovery{
			Fake: &cgotesting.Fake{
				Resources: []*metav1.APIResourceList{},
			},
		}
		p, err := GetProxyVars(cfg, discov)
		Expect(err).Should(BeNil())
		Expect(p).Should(BeNil())
	})

	It("should return an error if the cfg parameter is nil", func() {
		p, err := GetProxyVars(nil, discov)
		Expect(err).ShouldNot(BeNil())
		Expect(err.Error()).Should(Equal("configuration can not be nil when trying to get the Proxy config"))
		Expect(p).Should(BeNil())
	})

	It("should return an error if the discovery client returns an error other than the resource not being found", func() {
		errString := fmt.Sprintf("encountered an error getting server resources for GroupVersion `%s`:", apiconfigv1.GroupVersion.Identifier())
		p, err := GetProxyVars(cfg, &errorDiscovery{})
		Expect(err).ShouldNot(BeNil())
		Expect(err.Error()).Should(ContainSubstring(errString))
		Expect(p).Should(BeNil())
	})
})

func validFakeClient(scheme *runtime.Scheme, prox *apiconfigv1.Proxy) client.Client {
	scheme.AddKnownTypeWithName(apiconfigv1.SchemeGroupVersion.WithKind("Proxy"), prox)

	bldr := crfake.NewClientBuilder()
	bldr.WithScheme(scheme)
	bldr.WithObjects(prox)
	return bldr.Build()
}

func noProxyFakeClient(scheme *runtime.Scheme) client.Client {
	return crfake.NewClientBuilder().WithScheme(scheme).Build()
}

func invalidFakeClient() client.Client {
	return crfake.NewClientBuilder().Build()
}

// Implement the discovery.DiscoveryInterface to test ServerResourcesForGroupVersion() error case
// ----------------------------------------------
type errorDiscovery struct{}

func (c *errorDiscovery) OpenAPISchema() (*openapi_v2.Document, error) { return nil, nil }
func (c *errorDiscovery) OpenAPIV3() openapi.Client                    { return nil }
func (c *errorDiscovery) RESTClient() restclient.Interface             { return nil }
func (c *errorDiscovery) ServerGroups() (*metav1.APIGroupList, error)  { return nil, nil }
func (c *errorDiscovery) ServerVersion() (*version.Info, error)        { return nil, nil }

func (c *errorDiscovery) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	return nil, nil, nil
}
func (c *errorDiscovery) ServerPreferredNamespacedResources() ([]*metav1.APIResourceList, error) {
	return nil, nil
}
func (c *errorDiscovery) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	return nil, nil
}
func (c *errorDiscovery) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	return nil, fmt.Errorf("test")
}

// ----------------------------------------------
