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

	// RBAC objects that are not a part of the CSV should be written.
	_, roleObjs := c.SplitCSVPermissionsObjects()
	objs = append(objs, roleObjs...)
	_, clusterRoleObjs := c.SplitCSVClusterPermissionsObjects()
	objs = append(objs, clusterRoleObjs...)

	return objs
}
