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

package operator

import (
	"context"
	"fmt"
	"sort"
	"strings"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

type Uninstall struct {
	config *Configuration

	Package string
}

func NewUninstall(cfg *Configuration) *Uninstall {
	return &Uninstall{
		config: cfg,
	}
}

func (u *Uninstall) Run(ctx context.Context) error {
	subs := v1alpha1.SubscriptionList{}
	if err := u.config.Client.List(ctx, &subs, client.InNamespace(u.config.Namespace)); err != nil {
		return fmt.Errorf("list subscriptions: %v", err)
	}

	var sub *v1alpha1.Subscription
	for _, s := range subs.Items {
		if u.Package == s.Spec.Package {
			sub = &s
			break
		}
	}
	if sub == nil {
		return fmt.Errorf("operator package %q not found", u.Package)
	}

	catsrcKey := types.NamespacedName{
		Namespace: sub.Spec.CatalogSourceNamespace,
		Name:      sub.Spec.CatalogSource,
	}
	catsrc := &v1alpha1.CatalogSource{}
	if err := u.config.Client.Get(ctx, catsrcKey, catsrc); err != nil {
		return fmt.Errorf("get catalog source: %v", err)
	}

	installPlanKey := types.NamespacedName{
		Namespace: sub.Status.InstallPlanRef.Namespace,
		Name:      sub.Status.InstallPlanRef.Name,
	}

	// Since the install plan is owned by the subscription, we need to
	// read all of the resource references from the install plan before
	// deleting the subscription.
	deleteObjs, err := u.getInstallPlanResources(ctx, installPlanKey)
	if err != nil {
		return err
	}

	// Delete the subscription first, so that no further installs or upgrades
	// of the operator occur while we're cleaning up.
	if err := u.config.Client.Delete(ctx, sub); err != nil {
		return fmt.Errorf("delete subscription %q: %v", sub.Name, err)
	}
	fmt.Printf("subscription %q deleted\n", sub.Name)

	// Ensure CustomResourceDefinitions are deleted first, so that the operator
	// has a chance to handle CRs that have finalizers.
	sort.SliceStable(deleteObjs, func(i, j int) bool {
		return deleteObjs[i].GetObjectKind().GroupVersionKind().Kind == "CustomResourceDefinition"
	})
	for _, obj := range deleteObjs {
		err := u.config.Client.Delete(ctx, obj)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		if err == nil {
			fmt.Printf("%s %q deleted\n", strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind), obj.GetName())
		}
	}

	// Delete the catalog source. This assumes that all underlying resources related
	// to this catalog source have an owner reference to this catalog source so that
	// they are automatically garbage-collected.
	if err := u.config.Client.Delete(ctx, catsrc); err != nil {
		return fmt.Errorf("delete catalog source: %v", err)
	}
	fmt.Printf("catalogsource %q deleted\n", catsrc.Name)

	// If this was the last subscription in the namespace and the operator group is
	// the one we created, delete it
	if len(subs.Items) == 1 {
		ogs := v1.OperatorGroupList{}
		if err := u.config.Client.List(ctx, &ogs, client.InNamespace(u.config.Namespace)); err != nil {
			return fmt.Errorf("list operatorgroups: %v", err)
		}
		for _, og := range ogs.Items {
			if og.GetName() == SDKOperatorGroupName {
				if err := u.config.Client.Delete(ctx, &og); err != nil {
					return fmt.Errorf("delete operatorgroup %q: %v", og.Name, err)
				}
				fmt.Printf("operatorgroup %q deleted\n", og.Name)
			}
		}
	}

	return nil
}

func (u *Uninstall) getInstallPlanResources(ctx context.Context, installPlanKey types.NamespacedName) ([]controllerutil.Object, error) {
	installPlan := &v1alpha1.InstallPlan{}
	if err := u.config.Client.Get(ctx, installPlanKey, installPlan); err != nil {
		return nil, fmt.Errorf("get install plan: %v", err)
	}

	var objs []controllerutil.Object
	for _, step := range installPlan.Status.Plan {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{}}
		lowerKind := strings.ToLower(step.Resource.Kind)
		if err := yaml.Unmarshal([]byte(step.Resource.Manifest), &obj.Object); err != nil {
			return nil, fmt.Errorf("parse %s manifest %q: %v", lowerKind, step.Resource.Name, err)
		}
		obj.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   step.Resource.Group,
			Version: step.Resource.Version,
			Kind:    step.Resource.Kind,
		})
		objs = append(objs, obj)
	}
	return objs, nil
}
