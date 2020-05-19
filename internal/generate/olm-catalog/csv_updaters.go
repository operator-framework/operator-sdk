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

package olmcatalog

import (
	"bytes"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
)

const olmTNMeta = "metadata.annotations['olm.targetNamespaces']"

func handleWatchNamespaces(csv *operatorsv1alpha1.ClusterServiceVersion) {
	for _, dep := range csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
		setWatchNamespacesEnv(&dep.Spec)
		// Make sure "olm.targetNamespaces" is referenced somewhere in dep,
		// and emit a warning of not.
		if !depHasOLMNamespaces(dep.Spec) {
			log.Warnf(`No WATCH_NAMESPACE environment variable nor reference to "%s"`+
				` detected in operator Deployment. For OLM compatibility, your operator`+
				` MUST watch namespaces defined in "%s"`, olmTNMeta, olmTNMeta)
		}
	}
}

// setWatchNamespacesEnv sets WATCH_NAMESPACE to olmTNString in dep if
// WATCH_NAMESPACE exists in a pod spec container's env list.
func setWatchNamespacesEnv(dep *appsv1.DeploymentSpec) {
	envVar := newEnvVar(k8sutil.WatchNamespaceEnvVar, olmTNMeta)
	overwriteContainerEnvVar(dep, k8sutil.WatchNamespaceEnvVar, envVar)
}

func overwriteContainerEnvVar(dep *appsv1.DeploymentSpec, name string, ev corev1.EnvVar) {
	for _, c := range dep.Template.Spec.Containers {
		for i := 0; i < len(c.Env); i++ {
			if c.Env[i].Name == name {
				c.Env[i] = ev
			}
		}
	}
}

func newEnvVar(name, fieldPath string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: fieldPath,
			},
		},
	}
}

// OLM places the set of target namespaces for the operator in
// "metadata.annotations['olm.targetNamespaces']". This value should be
// referenced in either:
//	- The DeploymentSpec's pod spec WATCH_NAMESPACE env variable.
//	- Some other DeploymentSpec pod spec field.
func depHasOLMNamespaces(dep appsv1.DeploymentSpec) bool {
	b, err := dep.Template.Marshal()
	if err != nil {
		// Something is wrong with the deployment manifest, not with CLI inputs.
		log.Fatalf("Marshal Deployment spec: %v", err)
	}
	return bytes.Contains(b, []byte(olmTNMeta))
}
