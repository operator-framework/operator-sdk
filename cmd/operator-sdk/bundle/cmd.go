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
	"github.com/spf13/cobra"
)

//nolint:structcheck
type bundleCmd struct {
	directory      string
	packageName    string
	imageTag       string
	imageBuilder   string
	defaultChannel string
	channels       string
	generateOnly   bool
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Manage operator bundle metadata",
		Long: `Manage bundle builds, bundle metadata generation, and bundle validation.
An operator bundle is a portable operator packaging format understood by Kubernetes
native software, like the Operator Lifecycle Manager.

More information about operator bundles and metadata:
https://github.com/operator-framework/operator-registry#manifest-format.

More information about the integration with OLM via SDK:
https://sdk.operatorframework.io/docs/olm-integration/
`,
	}

	cmd.AddCommand(
		newCreateCmd(),
		newValidateCmd(),
	)
	return cmd
}
