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

package generate

import (
	"fmt"

	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/internal/genutil"

	"github.com/spf13/cobra"
)

func newGenerateOpenshiftCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "openshift",
		Short: "Generates Openshift operator specific manifest files",
		Long: `openshift generator generates the needed manifest files for 
the operator to be deployed on Openshift. 
Example:
	$ operator-sdk generate openshift
	$ tree deploy/openshift/
		deploy/openshift/
		├── crds
		│   ├── app_v1alpha1_memcached_cr.yaml
		│   └── app_v1alpha1_memcached_crd.yaml
		├── metrics
		│   ├── service-monitor.yaml
		│   └── service.yaml
		├── operator.yaml
		└── rbac
			├── role.yaml
			├── role_binding.yaml
			└── service_account.yaml
`,
		RunE: openshiftFunc,
	}
}

func openshiftFunc(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
	}

	return genutil.OpenshiftGen()
}
