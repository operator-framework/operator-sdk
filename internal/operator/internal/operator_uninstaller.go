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

	"github.com/operator-framework/operator-sdk/internal/operator"
)

type OperatorUninstaller struct {
	PackageName         string
	DeleteCRDs          bool
	DeleteOperatorGroup bool
	DeleteAll           bool

	cfg *operator.Configuration
}

func NewOperatorUninstaller(cfg *operator.Configuration) *OperatorUninstaller {
	return &OperatorUninstaller{
		cfg: cfg,
	}
}

func (u OperatorUninstaller) UninstallOperator(ctx context.Context) error {
	fmt.Printf("OperatorUninstaller.PackageName:         %q\n", u.PackageName)
	fmt.Printf("OperatorUninstaller.DeleteCRDs:          %v\n", u.DeleteCRDs)
	fmt.Printf("OperatorUninstaller.DeleteOperatorGroup: %v\n", u.DeleteOperatorGroup)
	fmt.Printf("OperatorUninstaller.DeleteAll:           %v\n", u.DeleteAll)

	// Delete Subscription

	// Delete CRDs (if delete-crds option set)

	// Delete CSV

	// Delete OperatorGroup (if delete-operator-group option set
	// and no more subscriptions remaining in NS)
	return nil
}
