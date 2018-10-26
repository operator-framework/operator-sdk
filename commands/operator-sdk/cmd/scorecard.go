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

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-sdk/commands/operator-sdk/cmd/scorecard"
)

func NewScorecardCmd() *cobra.Command {
	scorecardCmd := &cobra.Command{
		Use:   "scorecard",
		Short: "Run scorecard tests",
		Long: `Runs blackbox scorecard tests on an operator
`,
		RunE: scorecard.ScorecardTests,
	}

	scorecardCmd.Flags().StringVar(&scorecard.SCConf.Namespace, "namespace", "", "Namespace of custom resource created in cluster")
	scorecardCmd.Flags().StringVar(&scorecard.SCConf.KubeconfigPath, "kubeconfig", "", "Path to kubeconfig of custom resource created in cluster")
	scorecardCmd.Flags().IntVar(&scorecard.SCConf.InitTimeout, "init-timeout", 10, "Timeout for status block on CR to be created in seconds")
	scorecardCmd.Flags().StringVar(&scorecard.SCConf.CSVPath, "csv-path", "", "Path to CSV being tested")
	scorecardCmd.Flags().BoolVar(&scorecard.SCConf.BasicTests, "basic-tests", false, "Enable basic operator checks")
	scorecardCmd.Flags().BoolVar(&scorecard.SCConf.OlmTests, "olm-tests", false, "Enable OLM integration checks")
	scorecardCmd.Flags().StringVar(&scorecard.SCConf.NamespacedMan, "namespaced-manifest", "", "Path to manifest for namespaced resources (e.g. RBAC and Operator manifest)")
	scorecardCmd.Flags().StringVar(&scorecard.SCConf.GlobalMan, "global-manifest", "", "Path to manifest for Global resources (e.g. CRD manifests)")
	scorecardCmd.Flags().StringVar(&scorecard.SCConf.CrMan, "cr-manifest", "", "Path to manifest for Custom Resource")
	// Since it's difficult to handle multiple CRs, we will require users to specify what CR they want to test; we can handle this better in the future
	scorecardCmd.MarkFlagRequired("cr-manifest")

	return scorecardCmd
}
