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

package olm

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Officially supported OLM versions by Operator SDK
var (
	OLMSupportedVersions = "unknown"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "olm",
		Short: fmt.Sprintf(`Manage the Operator Lifecycle Manager installation in your cluster.

Operator SDK officially supports the following OLM versions: %s.
Any other version installed with this command may work but is not officially tested.`, OLMSupportedVersions),
	}
	cmd.AddCommand(
		newInstallCmd(),
		newStatusCmd(),
		newUninstallCmd(),
	)
	return cmd
}
