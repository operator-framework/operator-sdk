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

	"github.com/operator-framework/operator-sdk/internal/pkg/scorecard"
	schelpers "github.com/operator-framework/operator-sdk/internal/pkg/scorecard/helpers"
	scplugins "github.com/operator-framework/operator-sdk/internal/pkg/scorecard/plugins"

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
	scorecardCmd.Flags().String(scplugins.KubeconfigOpt, "", "Path to kubeconfig of custom resource created in cluster")
	scorecardCmd.Flags().StringP(scorecard.OutputFormatOpt, "o", scorecard.TextOutputFormat, fmt.Sprintf("Output format for results. Valid values: %s, %s", scorecard.TextOutputFormat, scorecard.JSONOutputFormat))
	scorecardCmd.Flags().String(schelpers.VersionOpt, schelpers.DefaultScorecardVersion, fmt.Sprintf("scorecard version (tech preview version is '%s'", schelpers.LatestScorecardVersion))

	// TODO: make config file global and make this a top level flag
	viper.BindPFlag(scorecard.ConfigOpt, scorecardCmd.Flags().Lookup(scorecard.ConfigOpt))

	viper.BindPFlag("scorecard."+scplugins.KubeconfigOpt, scorecardCmd.Flags().Lookup(scplugins.KubeconfigOpt))
	viper.BindPFlag("scorecard."+scorecard.OutputFormatOpt, scorecardCmd.Flags().Lookup(scorecard.OutputFormatOpt))
	viper.BindPFlag("scorecard."+schelpers.VersionOpt, scorecardCmd.Flags().Lookup(schelpers.VersionOpt))

	return scorecardCmd
}
