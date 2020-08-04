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

package internal

import (
	"context"
	"fmt"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/operator-framework/operator-sdk/internal/operator"
)

type OperatorInstaller struct {
	CatalogSourceName string
	PackageName       string
	StartingCSV       string
	Channel           string
	InstallMode       InstallMode
	CatalogCreator    CatalogCreator

	cfg *operator.Configuration
}

func NewOperatorInstaller(cfg *operator.Configuration) *OperatorInstaller {
	return &OperatorInstaller{cfg: cfg}
}

func (o OperatorInstaller) InstallOperator(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	cs, err := o.CatalogCreator.CreateCatalog(ctx, o.CatalogSourceName)
	if err != nil {
		return nil, fmt.Errorf("create catalog: %v", err)
	}
	_ = cs

	fmt.Printf("OperatorInstaller.CatalogSourceName: %q\n", o.CatalogSourceName)
	fmt.Printf("OperatorInstaller.PackageName:       %q\n", o.PackageName)
	fmt.Printf("OperatorInstaller.StartingCSV:       %q\n", o.StartingCSV)
	fmt.Printf("OperatorInstaller.Channel:           %q\n", o.Channel)
	fmt.Printf("OperatorInstaller.InstallMode:       %q\n", o.InstallMode)
	todo := &v1alpha1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{Name: o.StartingCSV},
	}

	// Ensure Operator Group
	// Create Subscription
	// Approve Install Plan (if necessary)
	// Wait for successfully installed CSV

	return todo, nil
}
