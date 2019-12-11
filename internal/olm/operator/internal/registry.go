// Copyright 2019 The Operator-SDK Authors
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

package olm

import (
	"context"
	"fmt"

	olmclient "github.com/operator-framework/operator-sdk/internal/olm/client"

	"github.com/operator-framework/operator-registry/pkg/registry"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
)

// SDKLabels are used to identify certain operator-sdk resources.
var SDKLabels = map[string]string{
	"owner": "operator-sdk",
}

// RegistryResources configures creation/deletion of internal registry-related
// resources.
type RegistryResources struct {
	Client  *olmclient.Client
	Pkg     registry.PackageManifest
	Bundles []*registry.Bundle
}

// FEAT(estroz): allow users to specify labels for registry objects.

// CreateRegistryManifests creates all registry objects required to serve
// manifests from m.manifests in namespace.
func (m *RegistryResources) CreateRegistryManifests(ctx context.Context, namespace string) error {
	pkgName := m.Pkg.PackageName
	binaryKeyValues, err := createConfigMapBinaryData(m.Pkg, m.Bundles)
	if err != nil {
		return fmt.Errorf("error creating registry ConfigMap binary data: %w", err)
	}
	cm := newRegistryConfigMap(pkgName, namespace,
		withBinaryData(binaryKeyValues),
	)
	volName := getRegistryVolumeName(pkgName)
	dep := newRegistryDeployment(pkgName, namespace,
		withRegistryGRPCContainer(pkgName),
		withVolumeConfigMap(volName, cm.GetName()),
		withContainerVolumeMounts(volName, []string{containerManifestsDir}),
	)
	service := newRegistryService(pkgName, namespace,
		withTCPPort("grpc", registryGRPCPort),
	)
	if err = m.Client.DoCreate(ctx, cm, dep, service); err != nil {
		return fmt.Errorf("error creating operator %q registry-server objects: %w", pkgName, err)
	}
	depKey := types.NamespacedName{
		Name:      dep.GetName(),
		Namespace: namespace,
	}
	log.Infof("Waiting for Deployment %q rollout to complete", depKey)
	if err = m.Client.DoRolloutWait(ctx, depKey); err != nil {
		return fmt.Errorf("error waiting for Deployment %q to roll out: %w", depKey, err)
	}
	return nil
}

// DeleteRegistryManifests deletes all registry objects serving manifests
// from m.manifests in namespace.
func (m *RegistryResources) DeleteRegistryManifests(ctx context.Context, namespace string) error {
	pkgName := m.Pkg.PackageName
	cm := newRegistryConfigMap(pkgName, namespace)
	dep := newRegistryDeployment(pkgName, namespace)
	service := newRegistryService(pkgName, namespace)
	err := m.Client.DoDelete(ctx, dep, cm, service)
	if err != nil {
		return fmt.Errorf("error deleting operator %q registry-server objects: %w", pkgName, err)
	}
	return nil
}

// GetRegistryServiceAddr returns a Service's DNS name + port for a given
// pkgName and namespace.
func GetRegistryServiceAddr(pkgName, namespace string) string {
	name := getRegistryServerName(pkgName)
	return fmt.Sprintf("%s.%s.svc.cluster.local:%d", name, namespace, registryGRPCPort)
}
