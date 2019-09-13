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
	"regexp"

	olmresourceclient "github.com/operator-framework/operator-sdk/internal/olm/client"
	registryutil "github.com/operator-framework/operator-sdk/internal/util/operator-registry"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
)

// RegistryResources configures creation/deletion of internal registry-related
// resources.
type RegistryResources struct {
	Client    *olmresourceclient.Client
	Manifests registryutil.ManifestsStore
}

// FEAT(estroz): allow users to specify labels for registry objects.

// CreateRegistryManifests creates all registry objects required to serve
// manifests from m.manifests in namespace.
func (m *RegistryResources) CreateRegistryManifests(ctx context.Context, namespace string) error {
	pkg := m.Manifests.GetPackageManifest()
	pkgName := pkg.PackageName
	bundles := m.Manifests.GetBundles()
	binaryKeyValues, err := createConfigMapBinaryData(pkg, bundles)
	if err != nil {
		return errors.Wrap(err, "error creating registry ConfigMap binary data")
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
		return errors.Wrapf(err, "error creating operator %q registry-server objects", pkgName)
	}
	depKey := types.NamespacedName{
		Name:      dep.GetName(),
		Namespace: namespace,
	}
	log.Infof("Waiting for Deployment %q rollout to complete", depKey)
	if err = m.Client.DoRolloutWait(ctx, depKey); err != nil {
		return errors.Wrapf(err, "error waiting for Deployment %q to roll out", depKey)
	}
	return nil
}

// DeleteRegistryManifests deletes all registry objects serving manifests
// from m.manifests in namespace.
func (m *RegistryResources) DeleteRegistryManifests(ctx context.Context, namespace string) error {
	pkgName := m.Manifests.GetPackageManifest().PackageName
	cm := newRegistryConfigMap(pkgName, namespace)
	dep := newRegistryDeployment(pkgName, namespace)
	service := newRegistryService(pkgName, namespace)
	err := m.Client.DoDelete(ctx, dep, cm, service)
	if err != nil {
		return errors.Wrapf(err, "error deleting operator %q registry-server objects", pkgName)
	}
	return nil
}

// GetRegistryServiceAddr returns a Service's DNS name + port for a given
// pkgName and namespace.
func GetRegistryServiceAddr(pkgName, namespace string) string {
	name := getRegistryServerName(pkgName)
	return fmt.Sprintf("%s.%s.svc.cluster.local:%d", name, namespace, registryGRPCPort)
}

// formatOperatorNameDNS1123 ensures name is DNS1123 label-compliant by
// replacing all non-compliant UTF-8 characters with "-".
func formatOperatorNameDNS1123(name string) string {
	if len(validation.IsDNS1123Label(name)) != 0 {
		replacer := regexp.MustCompile("[^a-zA-Z0-9]+")
		return replacer.ReplaceAllString(name, "-")
	}
	return name
}
