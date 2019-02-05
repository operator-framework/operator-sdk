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

package scorecard

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"

	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olminstall "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func generateCombinedNamespacedManifestFromCSV(csv *olmapiv1alpha1.ClusterServiceVersion) (*os.File, error) {
	man, err := ioutil.TempFile("", "namespaced-manifest.yaml")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := man.Close(); err != nil && !fileutil.IsClosedError(err) {
			log.Errorf("Failed to close file %s: (%v)", man.Name(), err)
		}
	}()
	var resolver *olminstall.StrategyResolver
	strat, err := resolver.UnmarshalStrategy(csv.Spec.InstallStrategy)
	if err != nil {
		return nil, err
	}
	stratDep, ok := strat.(*olminstall.StrategyDetailsDeployment)
	if !ok {
		return nil, fmt.Errorf("expected StrategyDetailsDeployment, got strategy of type %T", strat)
	}

	var manBytes []byte
	if perms := stratDep.Permissions; len(perms) > 0 {
		role := &rbacv1.Role{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "Role",
			},
			ObjectMeta: metav1.ObjectMeta{Name: perms[0].ServiceAccountName},
			Rules:      perms[0].Rules,
		}
		roleBytes, err := yaml.Marshal(role)
		if err != nil {
			return nil, err
		}
		// Create dummy role binding and service accounts for testing.
		sa := corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "ServiceAccount",
			},
			ObjectMeta: metav1.ObjectMeta{Name: perms[0].ServiceAccountName},
		}
		saBytes, err := yaml.Marshal(sa)
		if err != nil {
			return nil, err
		}
		rb := &rbacv1.RoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "RoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{Name: perms[0].ServiceAccountName},
			Subjects:   []rbacv1.Subject{{Kind: sa.Kind, Name: sa.Name}},
			RoleRef: rbacv1.RoleRef{
				Kind:     role.Kind,
				Name:     role.Name,
				APIGroup: rbacv1.GroupName,
			},
		}
		rbBytes, err := yaml.Marshal(rb)
		if err != nil {
			return nil, err
		}

		manBytes = yamlutil.CombineManifests(manBytes, saBytes, roleBytes, rbBytes)
	}
	if cperms := stratDep.ClusterPermissions; len(cperms) > 0 {
		cRole := &rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{Name: cperms[0].ServiceAccountName},
			Rules:      cperms[0].Rules,
		}
		cRoleBytes, err := yaml.Marshal(cRole)
		if err != nil {
			return nil, err
		}
		manBytes = yamlutil.CombineManifests(manBytes, cRoleBytes)
	}
	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: stratDep.DeploymentSpecs[0].Name,
		},
		Spec: stratDep.DeploymentSpecs[0].Spec,
	}
	depBytes, err := yaml.Marshal(dep)
	if err != nil {
		return nil, err
	}

	_, err = man.Write(yamlutil.CombineManifests(manBytes, depBytes))
	if err != nil {
		return nil, err
	}

	return man, nil
}
