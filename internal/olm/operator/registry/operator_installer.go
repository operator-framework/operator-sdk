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

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"

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

func (o OperatorInstaller) createOperatorGroup(ctx context.Context) error {
	og := newSDKOperatorGroup(o.cfg.Namespace,
		withTargetNamespaces(o.InstallMode.TargetNamespaces...))
	log.Info("Creating OperatorGroup")
	if err := o.cfg.Client.Create(ctx, og); err != nil {
		return fmt.Errorf("error creating OperatorGroup: %w", err)
	}
	// TODO: ensure operator group created successfully and no 2 operator groups exist.
	// https://github.com/operator-framework/operator-sdk/pull/3689
	return nil
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
