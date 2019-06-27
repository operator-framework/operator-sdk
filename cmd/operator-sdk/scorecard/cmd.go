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

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/pkg/scorecard"
	scplugins "github.com/operator-framework/operator-sdk/internal/pkg/scorecard/plugins"
	"github.com/operator-framework/operator-sdk/version"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmd() *cobra.Command {
	scorecardCmd := &cobra.Command{
		Use:   "scorecard",
		Short: "Run scorecard tests",
		Long: `Runs blackbox scorecard tests on an operator
`,
		RunE: scorecard.ScorecardTests,
	}

	scorecardCmd.Flags().String(scorecard.ConfigOpt, "", fmt.Sprintf("config file (default is '<project_dir>/%s'; the config file's extension and format can be .yaml, .json, or .toml)", scorecard.DefaultConfigFile))
	scorecardCmd.Flags().String(scplugins.NamespaceOpt, "", "Namespace of custom resource created in cluster")
	scorecardCmd.Flags().String(scplugins.KubeconfigOpt, "", "Path to kubeconfig of custom resource created in cluster")
	scorecardCmd.Flags().Int(scplugins.InitTimeoutOpt, 60, "Timeout for status block on CR to be created in seconds")
	scorecardCmd.Flags().Bool(scplugins.OlmDeployedOpt, false, "The OLM has deployed the operator. Use only the CSV for test data")
	scorecardCmd.Flags().String(scplugins.CSVPathOpt, "", "Path to CSV being tested")
	scorecardCmd.Flags().Bool(scplugins.BasicTestsOpt, true, "Enable basic operator checks")
	scorecardCmd.Flags().Bool(scplugins.OLMTestsOpt, true, "Enable OLM integration checks")
	scorecardCmd.Flags().String(scplugins.NamespacedManifestOpt, "", "Path to manifest for namespaced resources (e.g. RBAC and Operator manifest)")
	scorecardCmd.Flags().String(scplugins.GlobalManifestOpt, "", "Path to manifest for Global resources (e.g. CRD manifests)")
	scorecardCmd.Flags().StringSlice(scplugins.CRManifestOpt, nil, "Path to manifest for Custom Resource (required) (specify flag multiple times for multiple CRs)")
	scorecardCmd.Flags().String(scplugins.ProxyImageOpt, fmt.Sprintf("quay.io/operator-framework/scorecard-proxy:%s", strings.TrimSuffix(version.Version, "+git")), "Image name for scorecard proxy")
	scorecardCmd.Flags().String(scplugins.ProxyPullPolicyOpt, "Always", "Pull policy for scorecard proxy image")
	scorecardCmd.Flags().String(scplugins.CRDsDirOpt, scaffold.CRDsDir, "Directory containing CRDs (all CRD manifest filenames must have the suffix 'crd.yaml')")
	scorecardCmd.Flags().StringP(scorecard.OutputFormatOpt, "o", scorecard.HumanReadableOutputFormat, fmt.Sprintf("Output format for results. Valid values: %s, %s", scorecard.HumanReadableOutputFormat, scorecard.JSONOutputFormat))
	scorecardCmd.Flags().String(scorecard.PluginDirOpt, "scorecard", "Scorecard plugin directory (plugin exectuables must be in a \"bin\" subdirectory")

	if err := viper.BindPFlags(scorecardCmd.Flags()); err != nil {
		log.Fatalf("Failed to bind scorecard flags to viper: %v", err)
	}

	return scorecardCmd
}
