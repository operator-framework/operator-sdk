// Copyright © 2018 The Operator-SDK Authors
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

package completion

import (
	kbutil "github.com/operator-framework/operator-sdk/internal/util/kubebuilder"

	"github.com/spf13/cobra"
)

// KB_INTEGRATION_TODO(estroz): wire this into kubebuilder's CLI.

func NewCmd() *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion",
		Short: "Generators for shell completions",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			// This command is superceded by a kubebuilder equivalent.
			kbutil.DieIfCmdNotAllowed(true)
		},
		Hidden: kbutil.IsConfigExist(),
	}
	completionCmd.AddCommand(newZshCmd())
	completionCmd.AddCommand(newBashCmd())
	return completionCmd
}
