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

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: stratDep.Permissions[0].ServiceAccountName,
		},
		Rules: stratDep.Permissions[0].Rules,
	}
	roleBytes, err := yaml.Marshal(role)
	if err != nil {
		return nil, err
	}
	cRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: stratDep.ClusterPermissions[0].ServiceAccountName,
		},
		Rules: stratDep.ClusterPermissions[0].Rules,
	}
	cRoleBytes, err := yaml.Marshal(cRole)
	if err != nil {
		return nil, err
	}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: stratDep.DeploymentSpecs[0].Name,
		},
		Spec: stratDep.DeploymentSpecs[0].Spec,
	}
	depBytes, err := yaml.Marshal(dep)
	if err != nil {
		return nil, err
	}

	_, err = man.Write(yamlutil.CombineManifests(roleBytes, cRoleBytes, depBytes))
	if err != nil {
		return nil, err
	}

	return man, nil
}
