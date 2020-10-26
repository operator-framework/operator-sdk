// Copyright 2018 The Operator-SDK Authors
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
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/generate/bundle"
	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/generate/kustomize"
	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/generate/packagemanifests"
)

// NewCmd returns the 'generate' command configured for the new project layout.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate <generator>",
		Short: "Invokes a specific generator",
		Long: `The 'operator-sdk generate' command invokes a specific generator to generate
code or manifests.`,
	}

	cmd.AddCommand(
		kustomize.NewCmd(),
		bundle.NewCmd(),
		packagemanifests.NewCmd(),
	)
	return cmd
}
