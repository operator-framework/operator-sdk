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
	"log"

	"github.com/operator-framework/operator-sdk/internal/scorecard"
	schelpers "github.com/operator-framework/operator-sdk/internal/scorecard/helpers"
	scplugins "github.com/operator-framework/operator-sdk/internal/scorecard/plugins"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmd() *cobra.Command {
	scorecardCmd := &cobra.Command{
		Use:   "scorecard",
		Short: "Run scorecard tests",
		Long: `Runs blackbox scorecard tests on an operator
`,
		RunE: scorecard.Tests,
	}

	scorecardCmd.Flags().String(scorecard.ConfigOpt, "", fmt.Sprintf("config file (default is '<project_dir>/%s'; the config file's extension and format can be .yaml, .json, or .toml)", scorecard.DefaultConfigFile))
	scorecardCmd.Flags().String(scplugins.KubeconfigOpt, "", "Path to kubeconfig of custom resource created in cluster")
	scorecardCmd.Flags().StringP(scorecard.OutputFormatOpt, "o", scorecard.TextOutputFormat, fmt.Sprintf("Output format for results. Valid values: %s, %s", scorecard.TextOutputFormat, scorecard.JSONOutputFormat))
	scorecardCmd.Flags().String(schelpers.VersionOpt, schelpers.DefaultScorecardVersion, "scorecard version. Valid values: v1alpha1, v1alpha2")
	scorecardCmd.Flags().StringP(scorecard.SelectorOpt, "l", "", "selector (label query) to filter tests on (only valid when version is v1alpha2)")
	scorecardCmd.Flags().BoolP(scorecard.ListOpt, "L", false, "If true, only print the test names that would be run based on selector filtering (only valid when version is v1alpha2)")
	scorecardCmd.Flags().StringP(scorecard.BundleOpt, "b", "", "OLM bundle directory path, when specified runs bundle validation")

	// TODO: make config file global and make this a top level flag
	if err := viper.BindPFlag(scorecard.ConfigOpt, scorecardCmd.Flags().Lookup(scorecard.ConfigOpt)); err != nil {
		log.Fatalf("Unable to add config :%v", err)
	}
	if err := viper.BindPFlag("scorecard."+scplugins.KubeconfigOpt, scorecardCmd.Flags().Lookup(scplugins.KubeconfigOpt)); err != nil {
		log.Fatalf("Unable to add kubeconfig :%v", err)
	}
	if err := viper.BindPFlag("scorecard."+scorecard.OutputFormatOpt, scorecardCmd.Flags().Lookup(scorecard.OutputFormatOpt)); err != nil {
		log.Fatalf("Unable to add output format :%v", err)
	}
	if err := viper.BindPFlag("scorecard."+schelpers.VersionOpt, scorecardCmd.Flags().Lookup(schelpers.VersionOpt)); err != nil {
		log.Fatalf("Unable to add version :%v", err)
	}
	if err := viper.BindPFlag("scorecard."+scorecard.SelectorOpt, scorecardCmd.Flags().Lookup(scorecard.SelectorOpt)); err != nil {
		log.Fatalf("Unable to add selector :%v", err)
	}
	if err := viper.BindPFlag("scorecard."+scorecard.ListOpt, scorecardCmd.Flags().Lookup(scorecard.ListOpt)); err != nil {
		log.Fatalf("Unable to add list :%v", err)
	}
	if err := viper.BindPFlag("scorecard."+scorecard.BundleOpt, scorecardCmd.Flags().Lookup(scorecard.BundleOpt)); err != nil {
		log.Fatalf("Unable to add bundle :%v", err)
	}
	return scorecardCmd
}
