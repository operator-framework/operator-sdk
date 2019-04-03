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

	scorecardCmd.Flags().String(scorecard.ConfigOpt, "", "config file (default is <project_dir>/.osdk-yaml)")
	scorecardCmd.Flags().String(scorecard.NamespaceOpt, "", "Namespace of custom resource created in cluster")
	scorecardCmd.Flags().String(scorecard.KubeconfigOpt, "", "Path to kubeconfig of custom resource created in cluster")
	scorecardCmd.Flags().Int(scorecard.InitTimeoutOpt, 10, "Timeout for status block on CR to be created in seconds")
	scorecardCmd.Flags().Bool(scorecard.OlmDeployedOpt, false, "The OLM has deployed the operator. Use only the CSV for test data")
	scorecardCmd.Flags().String(scorecard.CSVPathOpt, "", "Path to CSV being tested")
	scorecardCmd.Flags().Bool(scorecard.BasicTestsOpt, true, "Enable basic operator checks")
	scorecardCmd.Flags().Bool(scorecard.OLMTestsOpt, true, "Enable OLM integration checks")
	scorecardCmd.Flags().Bool(scorecard.TenantTestsOpt, false, "Enable good tenant checks")
	scorecardCmd.Flags().String(scorecard.NamespacedManifestOpt, "", "Path to manifest for namespaced resources (e.g. RBAC and Operator manifest)")
	scorecardCmd.Flags().String(scorecard.GlobalManifestOpt, "", "Path to manifest for Global resources (e.g. CRD manifests)")
	scorecardCmd.Flags().String(scorecard.CRManifestOpt, "", "Path to manifest for Custom Resource (required)")
	scorecardCmd.Flags().String(scorecard.ProxyImageOpt, fmt.Sprintf("quay.io/operator-framework/scorecard-proxy:%s", strings.TrimSuffix(version.Version, "+git")), "Image name for scorecard proxy")
	scorecardCmd.Flags().String(scorecard.ProxyPullPolicyOpt, "Always", "Pull policy for scorecard proxy image")
	scorecardCmd.Flags().String(scorecard.CRDsDirOpt, scaffold.CRDsDir, "Directory containing CRDs (all CRD manifest filenames must have the suffix 'crd.yaml')")
	scorecardCmd.Flags().Bool(scorecard.VerboseOpt, false, "Enable verbose logging")

	if err := viper.BindPFlags(scorecardCmd.Flags()); err != nil {
		log.Fatalf("Failed to bind scorecard flags to viper: %v", err)
	}

	return scorecardCmd
}
