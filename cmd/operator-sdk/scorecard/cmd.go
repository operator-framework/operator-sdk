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

package scorecard

import (
	"fmt"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/config"
	"github.com/operator-framework/operator-sdk/pkg/scorecard"
	"github.com/operator-framework/operator-sdk/version"

	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	c := &scorecard.ScorecardCmd{}

	scorecardCmd := &cobra.Command{
		Use:   "scorecard",
		Short: "Run scorecard tests",
		Long:  `Runs blackbox scorecard tests on an operator`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.ConfigureLogger()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			return c.Run()
		},
	}

	scorecardCmd.Flags().String(stripPrefix(scorecard.NamespaceOpt), "", "Namespace of custom resource created in cluster")
	scorecardCmd.Flags().String(stripPrefix(scorecard.KubeconfigPathOpt), "", "Path to kubeconfig of custom resource created in cluster")
	scorecardCmd.Flags().Int(stripPrefix(scorecard.InitTimeoutOpt), 60, "Timeout for status block on CR to be created in seconds")
	scorecardCmd.Flags().Bool(stripPrefix(scorecard.OLMDeployedOpt), false, "The OLM has deployed the operator. Use only the CSV for test data")
	scorecardCmd.Flags().String(stripPrefix(scorecard.CSVPathOpt), "", "Path to CSV being tested")
	scorecardCmd.Flags().Bool(stripPrefix(scorecard.BasicTestsOpt), true, "Enable basic operator checks")
	scorecardCmd.Flags().Bool(stripPrefix(scorecard.OLMTestsOpt), true, "Enable OLM integration checks")
	scorecardCmd.Flags().Bool(stripPrefix(scorecard.TenantTestsOpt), false, "Enable good tenant checks")
	scorecardCmd.Flags().String(stripPrefix(scorecard.NamespacedManifestOpt), "", "Path to manifest for namespaced resources (e.g. RBAC and Operator manifest)")
	scorecardCmd.Flags().String(stripPrefix(scorecard.GlobalManifestOpt), "", "Path to manifest for Global resources (e.g. CRD manifests)")
	scorecardCmd.Flags().StringSlice(stripPrefix(scorecard.CRManifestOpt), nil, "Path to manifest for Custom Resource (required) (specify flag multiple times for multiple CRs)")
	scorecardCmd.Flags().String(stripPrefix(scorecard.ProxyImageOpt), fmt.Sprintf("quay.io/operator-framework/scorecard-proxy:%s", strings.TrimSuffix(version.Version, "+git")), "Image name for scorecard proxy")
	scorecardCmd.Flags().String(stripPrefix(scorecard.ProxyPullPolicyOpt), "Always", "Pull policy for scorecard proxy image")
	scorecardCmd.Flags().String(stripPrefix(scorecard.CRDsDirOpt), "", "Directory containing CRDs (all CRD manifest filenames must have the suffix 'crd.yaml')")
	scorecardCmd.Flags().StringP(stripPrefix(scorecard.OutputFormatOpt), "o", scorecard.HumanReadableOutputFormat, fmt.Sprintf("Output format for results. Valid values: %s, %s", scorecard.HumanReadableOutputFormat, scorecard.JSONOutputFormat))
	scorecardCmd.Flags().String(stripPrefix(scorecard.PluginDirOpt), "scorecard", "Scorecard plugin directory (plugin exectuables must be in a \"bin\" subdirectory")
	config.BindFlagsWithPrefix(scorecardCmd.Flags(), scorecard.ScorecardConfigOpt)

	return scorecardCmd
}

func stripPrefix(k string) string {
	return strings.TrimPrefix(k, scorecard.ScorecardConfigOpt+".")
}
