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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubectl/pkg/util/slice"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	crdKind = "CustomResourceDefinition"
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

func (u *Uninstall) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&u.DeleteCRDs, "delete-crds", false, "If set to true, owned CRDs and CRs will be deleted")
	fs.BoolVar(&u.DeleteAll, "delete-all", true, "If set to true, all other delete options will be enabled")
	fs.BoolVar(&u.DeleteOperatorGroups, "delete-operator-groups", false, "If set to true, operator groups will be deleted")
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

	// Use nil objects to determine if the underlying was found.
	// If it was, the object will != nil.
	var subObj, csvObj, csObj client.Object
	var sub *v1alpha1.Subscription
	var crds []client.Object
	catsrc := &v1alpha1.CatalogSource{}
	catsrc.SetNamespace(u.config.Namespace)
	catsrc.SetName(CatalogNameForPackage(u.Package))

	for i := range subs.Items {
		s := subs.Items[i]
		if u.Package == s.Spec.Package {
			sub = &s
			break
		}
	}

	catsrcKey := client.ObjectKeyFromObject(catsrc)
	if sub != nil {
		subObj = sub
		// Use the subscription's catalog source data only if available.
		keyFromSpec := types.NamespacedName{
			Namespace: sub.Spec.CatalogSourceNamespace,
			Name:      sub.Spec.CatalogSource,
		}
		if catsrcKey.Name != "" && catsrcKey.Namespace != "" {
			catsrcKey = keyFromSpec
		}

		// CSV name may either be the installed or current name in a subscription's status,
		// depending on installation state.
		csvKey := types.NamespacedName{
			Name:      sub.Status.InstalledCSV,
			Namespace: u.config.Namespace,
		}
		if csvKey.Name == "" {
			csvKey.Name = sub.Status.CurrentCSV
		}
		// This value can be empty which will cause errors.
		if csvKey.Name != "" {
			csv := &v1alpha1.ClusterServiceVersion{}
			if err := u.config.Client.Get(ctx, csvKey, csv); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("error getting installed CSV %q: %v", csvKey.Name, err)
			} else if err == nil {
				crds = getCRDs(csv)
			}
			csvObj = csv
		}
	}

	// Get the catalog source to make sure the correct error is returned.
	if err := u.config.Client.Get(ctx, catsrcKey, catsrc); err == nil {
		csObj = catsrc
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("error get catalog source: %v", err)
	}

	// Deletion order:
	//
	// 1. Subscription to prevent further installs or upgrades of the operator while cleaning up.
	// 2. CustomResourceDefinitions so the operator has a chance to handle CRs that have finalizers.
	// 3. ClusterServiceVersion. OLM puts an ownerref on every namespaced resource to the CSV,
	//    and an owner label on every cluster scoped resource so they get gc'd on deletion.
	// 4. CatalogSource. All other resources installed by OLM or operator-sdk related to this
	//    package will be gc'd.

	// Subscriptions can be deleted asynchronously.
	if err := u.deleteObjects(ctx, false, subObj); err != nil {
		return err
	}
	var objs []client.Object

	if u.DeleteCRDs {
		objs = append(objs, crds...)
	} else {
		log.Info("Skipping CRD deletion")

	}

	objs = append(objs, csvObj, csObj)
	// These objects may have owned resources/finalizers, so block on deletion.
	if err := u.deleteObjects(ctx, true, objs...); err != nil {
		return err
	}

	// If the last subscription in the namespace was deleted and the operator group is
	// the one operator-sdk created, delete it.
	if u.DeleteOperatorGroups {
		if err := u.deleteOperatorGroup(ctx); err != nil {
			return err
		}
	} else {
		log.Info("Skipping Operator Groups deletion")
	}

	// If no objects were cleaned up, the package was not found.
	if subObj == nil && csObj == nil && csvObj == nil && len(crds) == 0 {
		return &ErrPackageNotFound{u.Package}
	}
	return nil
}

func (u *Uninstall) deleteOperatorGroup(ctx context.Context) error {
	subs := v1alpha1.SubscriptionList{}
	if err := u.config.Client.List(ctx, &subs, client.InNamespace(u.config.Namespace)); err != nil {
		return fmt.Errorf("list subscriptions: %v", err)
	}
	if len(subs.Items) != 0 {
		return nil
	}
	ogs := v1.OperatorGroupList{}
	if err := u.config.Client.List(ctx, &ogs, client.InNamespace(u.config.Namespace)); err != nil {
		return fmt.Errorf("list operatorgroups: %v", err)
	}
	for _, og := range ogs.Items {
		if len(u.DeleteOperatorGroupNames) == 0 || slice.ContainsString(u.DeleteOperatorGroupNames, og.GetName(), nil) {
			if err := u.deleteObjects(ctx, false, &og); err != nil {
				return err
			}
		}
	}
	return nil
}

func (u *Uninstall) deleteObjects(ctx context.Context, waitForDelete bool, objs ...client.Object) error {
	for _, obj := range objs {
		if obj == nil {
			continue
		}
		gvks, _, err := u.config.Scheme.ObjectKinds(obj)
		if err != nil {
			return err
		}
		lowerKind := strings.ToLower(gvks[0].Kind)
		if err := u.config.Client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete %s %q: %v", lowerKind, obj.GetName(), err)
		} else if err == nil {
			u.Logf("%s %q deleted", lowerKind, obj.GetName())
		}
		if waitForDelete {
			key := client.ObjectKeyFromObject(obj)
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

// getCRDs returns the list of CRDs required by a CSV.
func getCRDs(csv *v1alpha1.ClusterServiceVersion) (crds []client.Object) {
	for _, resource := range csv.Status.RequirementStatus {
		if resource.Kind == crdKind {
			obj := &unstructured.Unstructured{}
			obj.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   resource.Group,
				Version: resource.Version,
				Kind:    resource.Kind,
			})
			obj.SetName(resource.Name)
			crds = append(crds, obj)
		}
	}
	return
}
