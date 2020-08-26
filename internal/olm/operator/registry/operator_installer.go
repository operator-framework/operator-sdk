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

package registry

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmclient "github.com/operator-framework/operator-sdk/internal/olm/client"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
)

type OperatorInstaller struct {
	CatalogSourceName string
	PackageName       string
	StartingCSV       string
	Channel           string
	InstallMode       operator.InstallMode
	CatalogCreator    CatalogCreator

	cfg *operator.Configuration
}

func NewOperatorInstaller(cfg *operator.Configuration) *OperatorInstaller {
	return &OperatorInstaller{cfg: cfg}
}

func (o OperatorInstaller) InstallOperator(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	log.Info("Creating CatalogSource")
	cs, err := o.CatalogCreator.CreateCatalog(ctx, o.CatalogSourceName)
	if err != nil {
		return nil, fmt.Errorf("create catalog: %v", err)
	}

	// TODO: OLM doesn't appear to propagate the "READY" connection status to the catalogsource in a timely manner
	// even though its catalog-operator reports a connection almost immediately. This condition either needs to
	// be propagated more quickly by OLM or we need to find a different resource to probe for readiness.
	//
	// if err := o.waitForCatalogSource(ctx, cs); err != nil {
	// 	return nil, err
	// }

	log.Infof("OperatorInstaller.CatalogSourceName: %q\n", o.CatalogSourceName)
	log.Infof("OperatorInstaller.PackageName:       %q\n", o.PackageName)
	log.Infof("OperatorInstaller.StartingCSV:       %q\n", o.StartingCSV)
	log.Infof("OperatorInstaller.Channel:           %q\n", o.Channel)
	log.Infof("OperatorInstaller.InstallMode:       %q\n", o.InstallMode)

	// Ensure Operator Group
	if err = o.createOperatorGroup(ctx); err != nil {
		return nil, err
	}

	// Create Subscription
	if err = o.createSubscription(ctx, cs); err != nil {
		return nil, err
	}

	// Approve Install Plan (if necessary)
	if approver, ok := o.CatalogCreator.(InstallPlanApprover); ok {
		if err = approver.Approve(ctx, o.PackageName); err != nil {
			return nil, err
		}
	}

	// Wait for successfully installed CSV
	csv, err := o.getInstalledCSV(ctx)
	if err != nil {
		return nil, err
	}

	log.Infof("OLM has successfully installed %q", o.StartingCSV)

	return csv, nil
}

//nolint:unused
func (o OperatorInstaller) waitForCatalogSource(ctx context.Context, cs *v1alpha1.CatalogSource) error {
	catSrcKey, err := client.ObjectKeyFromObject(cs)
	if err != nil {
		return fmt.Errorf("error in getting catalog source key: %v", err)
	}

	// verify that catalog source connection status is READY
	catSrcCheck := wait.ConditionFunc(func() (done bool, err error) {
		if err := o.cfg.Client.Get(ctx, catSrcKey, cs); err != nil {
			return false, err
		}
		if cs.Status.GRPCConnectionState != nil {
			fmt.Println("grpc connection state:", cs.Status.GRPCConnectionState.LastObservedState)
			if cs.Status.GRPCConnectionState.LastObservedState == "READY" {
				return true, nil
			}
		} else {
			fmt.Println("grpc connection state: <nil>")
		}
		return false, nil
	})

	if err := wait.PollImmediateUntil(200*time.Millisecond, catSrcCheck, ctx.Done()); err != nil {
		return fmt.Errorf("catalog source connection is not ready: %v", err)
	}

	return nil
}

// createOperatorGroup creates an OperatorGroup using package name if an OperatorGroup does not exist.
// If one exists in the desired namespace and it's target namespaces do not match the desired set,
// createOperatorGroup will return an error.
func (o OperatorInstaller) createOperatorGroup(ctx context.Context) error {
	targetNamespaces := make([]string, len(o.InstallMode.TargetNamespaces), cap(o.InstallMode.TargetNamespaces))
	copy(targetNamespaces, o.InstallMode.TargetNamespaces)
	// Check OperatorGroup existence, since we cannot create a second OperatorGroup in namespace.
	og, ogFound, err := o.getOperatorGroup(ctx)
	if err != nil {
		return err
	}
	// TODO: we may need to poll for status updates, since status.namespaces may not be updated immediately.
	if ogFound {
		// targetNamespaces will always be initialized, but the operator group's namespaces may not be
		// (required for comparison).
		if og.Status.Namespaces == nil {
			og.Status.Namespaces = []string{}
		}
		// Simple check for OperatorGroup compatibility: if namespaces are not an exact match,
		// the user must manage the resource themselves.
		sort.Strings(og.Status.Namespaces)
		sort.Strings(targetNamespaces)
		if !reflect.DeepEqual(og.Status.Namespaces, targetNamespaces) {
			msg := fmt.Sprintf("namespaces %+q do not match desired namespaces %+q", og.Status.Namespaces, targetNamespaces)
			if og.GetName() == operator.SDKOperatorGroupName {
				return fmt.Errorf("existing SDK-managed operator group's %s, "+
					"please clean up existing operators `operator-sdk cleanup` before running package %q", msg, o.PackageName)
			}
			return fmt.Errorf("existing operator group %q's %s, "+
				"please ensure it has the exact namespace set before running package %q", og.GetName(), msg, o.PackageName)
		}
		log.Infof("Using existing operator group %q", og.GetName())
	} else {
		// New SDK-managed OperatorGroup.
		og = newSDKOperatorGroup(o.cfg.Namespace,
			withTargetNamespaces(targetNamespaces...))
		log.Info("Creating OperatorGroup")
		if err = o.cfg.Client.Create(ctx, og); err != nil {
			return fmt.Errorf("error creating OperatorGroup: %w", err)
		}
	}
	return nil
}

// getOperatorGroup returns true if an OperatorGroup in the desired namespace was found.
// If more than one operator group exists in namespace, this function will return an error
// since CSVs in namespace will have an error status in that case.
func (o OperatorInstaller) getOperatorGroup(ctx context.Context) (*v1.OperatorGroup, bool, error) {
	ogList := &v1.OperatorGroupList{}
	if err := o.cfg.Client.List(ctx, ogList, client.InNamespace(o.cfg.Namespace)); err != nil {
		return nil, false, err
	}
	if len(ogList.Items) == 0 {
		return nil, false, nil
	}
	if len(ogList.Items) != 1 {
		var names []string
		for _, og := range ogList.Items {
			names = append(names, og.GetName())
		}
		return nil, true, fmt.Errorf("more than one operator group in namespace %s: %+q", o.cfg.Namespace, names)
	}
	return &ogList.Items[0], true, nil
}

func (o OperatorInstaller) createSubscription(ctx context.Context, cs *v1alpha1.CatalogSource) error {
	sub := newSubscription(o.StartingCSV, o.cfg.Namespace,
		withPackageChannel(o.PackageName, o.Channel, o.StartingCSV),
		withCatalogSource(cs.GetName(), o.cfg.Namespace))
	log.Info("Creating Subscription")
	if err := o.cfg.Client.Create(ctx, sub); err != nil {
		return fmt.Errorf("error creating OperatorGroup: %w", err)
	}
	return nil
}

func (o OperatorInstaller) getInstalledCSV(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	c, err := olmclient.NewClientForConfig(o.cfg.RESTConfig)
	if err != nil {
		return nil, err
	}

	// BUG(estroz): if namespace is not contained in targetNamespaces,
	// DoCSVWait will fail because the CSV is not deployed in namespace.
	nn := types.NamespacedName{
		Name:      o.StartingCSV,
		Namespace: o.cfg.Namespace,
	}
	log.Printf("Waiting for ClusterServiceVersion %q to reach 'Succeeded' phase", nn)
	if err = c.DoCSVWait(ctx, nn); err != nil {
		return nil, fmt.Errorf("error waiting for CSV to install: %w", err)
	}

	// TODO: check status of all resources in the desired bundle/package.
	csv := &v1alpha1.ClusterServiceVersion{}
	if err = o.cfg.Client.Get(ctx, nn, csv); err != nil {
		return nil, fmt.Errorf("error getting installed CSV: %w", err)
	}
	return csv, nil
}
