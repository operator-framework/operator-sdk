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

package genutil

import (
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/operator-framework/operator-sdk/internal/generate/collector"
)

// GetManifestObjects returns all objects to be written to a manifests directory from collector.Manifests.
func GetManifestObjects(c *collector.Manifests) (objs []controllerutil.Object) {
	// All CRDs passed in should be written.
	for i := range c.V1CustomResourceDefinitions {
		objs = append(objs, &c.V1CustomResourceDefinitions[i])
	}
	for i := range c.V1beta1CustomResourceDefinitions {
		objs = append(objs, &c.V1beta1CustomResourceDefinitions[i])
	}

	// All ServiceAccounts passed in should be written.
	for i := range c.ServiceAccounts {
		objs = append(objs, &c.ServiceAccounts[i])
	}

	// All non-webhook Services passed in should be written.
	for i := range c.Services {
		svc := &c.Services[i]
		removeWebhookServicePorts(c, svc)
		if len(svc.Spec.Ports) > 0 {
			objs = append(objs, svc)
		}
	}

	// Add all other supported kinds
	for i := range c.Others {
		obj := &c.Others[i]
		if supported, _ := bundle.IsSupported(obj.GroupVersionKind().Kind); supported {
			objs = append(objs, obj)
		}
	}

	// RBAC objects that are not a part of the CSV should be written.
	_, roleObjs := c.SplitCSVPermissionsObjects()
	objs = append(objs, roleObjs...)
	_, clusterRoleObjs := c.SplitCSVClusterPermissionsObjects()
	objs = append(objs, clusterRoleObjs...)

	removeNamespace(objs)
	return objs
}

// removeNamespace removes the namespace field of resources intended to be inserted into
// an OLM manifests directory.
//
// This is required to pass OLM validations which require that namespaced resources do
// not include explicit namespace settings. OLM automatically installs namespaced
// resources in the same namespace that the operator is installed in, which is determined
// at runtime, not bundle/packagemanifests creation time.
func removeNamespace(objs []controllerutil.Object) {
	for _, obj := range objs {
		obj.SetNamespace("")
	}
}

// removeWebhookServicePorts filters ports out of svc that correspond to ports used
// by Validating and Mutating Webhooks present in c.
func removeWebhookServicePorts(c *collector.Manifests, svc *corev1.Service) {
	filter := func(ref *admissionregv1.ServiceReference, s *corev1.Service) {
		if ref == nil {
			return
		}
		if ref.Namespace == s.Namespace && ref.Name == s.Name {
			// Port 443 is the default.
			// See: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#servicereference-v1-admissionregistration-k8s-io
			webhookPort := int32(443)
			if ref.Port != nil {
				webhookPort = *ref.Port
			}
			keep := s.Spec.Ports[:0]
			for _, p := range s.Spec.Ports {
				if p.Port != webhookPort {
					keep = append(keep, p)
				}
			}
			s.Spec.Ports = keep
		}
	}
	for _, w := range c.ValidatingWebhooks {
		filter(w.ClientConfig.Service, svc)
	}
	for _, w := range c.MutatingWebhooks {
		filter(w.ClientConfig.Service, svc)
	}
}
