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

package operator

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubectl/pkg/util/slice"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

type Uninstall struct {
	config *Configuration

	Package                  string
	DeleteAll                bool
	DeleteCRDs               bool
	DeleteOperatorGroups     bool
	DeleteOperatorGroupNames []string

	Logf func(string, ...interface{})
}

func NewUninstall(cfg *Configuration) *Uninstall {
	return &Uninstall{
		config: cfg,
	}
}

type ErrPackageNotFound struct {
	PackageName string
}

func (e ErrPackageNotFound) Error() string {
	return fmt.Sprintf("package %q not found", e.PackageName)
}

func (u *Uninstall) Run(ctx context.Context) error {
	if u.DeleteAll {
		u.DeleteCRDs = true
		u.DeleteOperatorGroups = true
	}

	subs := v1alpha1.SubscriptionList{}
	if err := u.config.Client.List(ctx, &subs, client.InNamespace(u.config.Namespace)); err != nil {
		return fmt.Errorf("list subscriptions: %v", err)
	}

	var sub *v1alpha1.Subscription
	for i := range subs.Items {
		s := subs.Items[i]
		if u.Package == s.Spec.Package {
			sub = &s
			break
		}
	}

	catsrc := &v1alpha1.CatalogSource{}
	catsrc.SetGroupVersionKind(v1alpha1.SchemeGroupVersion.WithKind(v1alpha1.CatalogSourceKind))
	if sub != nil {
		catsrcKey := types.NamespacedName{
			Namespace: sub.Spec.CatalogSourceNamespace,
			Name:      sub.Spec.CatalogSource,
		}
		if err := u.config.Client.Get(ctx, catsrcKey, catsrc); err != nil {
			return fmt.Errorf("get catalog source: %v", err)
		}

		// Since the install plan is owned by the subscription, we need to
		// read all of the resource references from the install plan before
		// deleting the subscription.
		var crds, csvs, others []controllerutil.Object
		if sub.Status.InstallPlanRef != nil {
			ipKey := types.NamespacedName{
				Namespace: sub.Status.InstallPlanRef.Namespace,
				Name:      sub.Status.InstallPlanRef.Name,
			}
			var err error
			crds, csvs, others, err = u.getInstallPlanResources(ctx, ipKey)
			if err != nil {
				return fmt.Errorf("get install plan resources: %v", err)
			}
		}

		// Delete the subscription first, so that no further installs or upgrades
		// of the operator occur while we're cleaning up.
		if err := u.deleteObjects(ctx, false, sub); err != nil {
			return err
		}

		if u.DeleteCRDs {
			// Ensure CustomResourceDefinitions are deleted next, so that the operator
			// has a chance to handle CRs that have finalizers.
			if err := u.deleteObjects(ctx, true, crds...); err != nil {
				return err
			}
		}

		// Delete CSVs and all other objects created by the install plan.
		objects := append(csvs, others...)
		if err := u.deleteObjects(ctx, true, objects...); err != nil {
			return err
		}
	} else {
		catsrc.SetNamespace(u.config.Namespace)
		catsrc.SetName(CatalogNameForPackage(u.Package))
	}

	// Delete the catalog source. This assumes that all underlying resources related
	// to this catalog source have an owner reference to this catalog source so that
	// they are automatically garbage-collected.
	if err := u.deleteObjects(ctx, true, catsrc); err != nil {
		return err
	}

	// If this was the last subscription in the namespace and the operator group is
	// the one we created, delete it
	if u.DeleteOperatorGroups {
		if err := u.config.Client.List(ctx, &subs, client.InNamespace(u.config.Namespace)); err != nil {
			return fmt.Errorf("list subscriptions: %v", err)
		}
		if len(subs.Items) == 0 {
			ogs := v1.OperatorGroupList{}
			if err := u.config.Client.List(ctx, &ogs, client.InNamespace(u.config.Namespace)); err != nil {
				return fmt.Errorf("list operatorgroups: %v", err)
			}
			for _, og := range ogs.Items {
				og := og
				if len(u.DeleteOperatorGroupNames) == 0 || slice.ContainsString(u.DeleteOperatorGroupNames, og.GetName(), nil) {
					if err := u.deleteObjects(ctx, false, &og); err != nil {
						return err
					}
				}
			}
		}
	}
	if sub == nil {
		return &ErrPackageNotFound{u.Package}
	}
	return nil
}

func (u *Uninstall) deleteObjects(ctx context.Context, waitForDelete bool, objs ...controllerutil.Object) error {
	for _, obj := range objs {
		obj := obj
		lowerKind := strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind)
		if err := u.config.Client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete %s %q: %v", lowerKind, obj.GetName(), err)
		} else if err == nil {
			u.Logf("%s %q deleted", lowerKind, obj.GetName())
		}
		if waitForDelete {
			key, err := client.ObjectKeyFromObject(obj)
			if err != nil {
				return fmt.Errorf("get %s key: %v", lowerKind, err)
			}
			if err := wait.PollImmediateUntil(250*time.Millisecond, func() (bool, error) {
				if err := u.config.Client.Get(ctx, key, obj); apierrors.IsNotFound(err) {
					return true, nil
				} else if err != nil {
					return false, err
				}
				return false, nil
			}, ctx.Done()); err != nil {
				return fmt.Errorf("wait for %s deleted: %v", lowerKind, err)
			}
		}
	}
	return nil
}

func (u *Uninstall) getInstallPlanResources(ctx context.Context, installPlanKey types.NamespacedName) (crds, csvs, others []controllerutil.Object, err error) {
	installPlan := &v1alpha1.InstallPlan{}
	if err := u.config.Client.Get(ctx, installPlanKey, installPlan); err != nil {
		return nil, nil, nil, fmt.Errorf("get install plan: %v", err)
	}

	for _, step := range installPlan.Status.Plan {
		lowerKind := strings.ToLower(step.Resource.Kind)
		obj := &unstructured.Unstructured{Object: map[string]interface{}{}}
		if err := yaml.Unmarshal([]byte(step.Resource.Manifest), &obj.Object); err != nil {
			return nil, nil, nil, fmt.Errorf("parse %s manifest %q: %v", lowerKind, step.Resource.Name, err)
		}
		obj.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   step.Resource.Group,
			Version: step.Resource.Version,
			Kind:    step.Resource.Kind,
		})

		// TODO(joelanford): This seems necessary for namespaced resources
		//    See: https://github.com/operator-framework/operator-lifecycle-manager/blob/c9405d035bc50d9aa290220cb8d75b0402e72707/pkg/controller/registry/resolver/rbac.go#L133
		if supported, namespaced := bundle.IsSupported(step.Resource.Kind); supported && bool(namespaced) {
			obj.SetNamespace(installPlanKey.Namespace)
		}

		switch step.Resource.Kind {
		case "CustomResourceDefinition":
			crds = append(crds, obj)
		case "ClusterServiceVersion":
			csvs = append(csvs, obj)
		default:
			// Skip non-CRD/non-CSV resources in the install plan that were not created by the install plan.
			// This means we avoid deleting things like the default service account.
			if step.Status != v1alpha1.StepStatusCreated {
				continue
			}
			others = append(others, obj)
		}
	}
	return crds, csvs, others, nil
}
