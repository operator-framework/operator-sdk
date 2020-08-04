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

package bundle

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/operator-framework/operator-sdk/internal/operator"
	"github.com/operator-framework/operator-sdk/internal/operator/internal"
)

type Uninstall struct {
	BundleImage string

	*internal.OperatorUninstaller
}

func NewUninstall(cfg *operator.Configuration) Uninstall {
	u := Uninstall{
		OperatorUninstaller: internal.NewOperatorUninstaller(cfg),
	}
	return u
}

func (u *Uninstall) BindFlags(fs *pflag.FlagSet) {
	// TODO: compare with packagemanifests cleanup flags. How does it work now?
	fs.BoolVarP(&u.DeleteAll, "delete-all", "X", false, "Enable all deletion flags")
	fs.BoolVar(&u.DeleteCRDs, "delete-crds", false, "Delete CRDs (and CRs) before cleaning up operator")
	fs.BoolVar(&u.DeleteOperatorGroup, "delete-operator-group", false, "Delete Operator Group if no other subscriptions exist in this namespace")
}

func (u Uninstall) Run(ctx context.Context) error {
	if err := u.setup(ctx); err != nil {
		return err
	}
	return u.UninstallOperator(ctx)
}

func (u *Uninstall) setup(ctx context.Context) error {
	labels, _, err := loadBundle(ctx, u.BundleImage)
	if err != nil {
		return fmt.Errorf("load bundle: %v", err)
	}

	u.PackageName = labels["operators.operatorframework.io.bundle.package.v1"]
	if u.DeleteAll {
		u.DeleteOperatorGroup = true
		u.DeleteCRDs = true
	}
	return nil
}
