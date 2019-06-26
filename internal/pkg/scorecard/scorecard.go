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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	schelpers "github.com/operator-framework/operator-sdk/internal/pkg/scorecard/helpers"
	scplugins "github.com/operator-framework/operator-sdk/internal/pkg/scorecard/plugins"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	scapiv1alpha1 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha1"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const DefaultConfigFile = ".osdk-scorecard"

const (
	ConfigOpt                 = "config"
	OutputFormatOpt           = "output"
	PluginDirOpt              = "plugin-dir"
	JSONOutputFormat          = "json"
	HumanReadableOutputFormat = "human-readable"
)

// make a global logger for scorecard
var (
	logReadWriter io.ReadWriter
	log           = logrus.New()
)

var rootDir string

func getPlugins() []Plugin {
	var plugins []Plugin
	// Add internal plugins
	if viper.GetBool(scplugins.BasicTestsOpt) {
		plugins = append(plugins, basicTestsPlugin)
	}
	if viper.GetBool(scplugins.OLMTestsOpt) {
		plugins = append(plugins, olmTestsPlugin)
	}
	// find external plugins
	pluginDir := viper.GetString(PluginDirOpt)
	if dir, err := os.Stat(pluginDir); err != nil || !dir.IsDir() {
		log.Warnf("Plugin directory not found; skipping external plugins: %v", err)
		return plugins
	}
	if err := os.Chdir(pluginDir); err != nil {
		log.Warnf("Failed to chdir into scorecard plugin directory: %v", err)
		return plugins
	}
	files, err := ioutil.ReadDir("bin")
	if err != nil {
		log.Errorf("Failed to list files in %s/bin; skipping external plugin tests: %v", pluginDir, err)
		return plugins
	}
	for _, f := range files {
		plugins = append(plugins, genericPlugin{filepath.Join("./bin", f.Name())})
	}
	return plugins
}

func ScorecardTests(cmd *cobra.Command, args []string) error {
	if err := initConfig(); err != nil {
		return err
	}
	if err := validateScorecardFlags(); err != nil {
		return err
	}
	cmd.SilenceUsage = true
	// declare err var to prevent redeclaration of global rootDir var
	var err error
	rootDir, err = os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %v", err)
	}
	var pluginOutputs []scapiv1alpha1.ScorecardOutput
	for _, plugin := range getPlugins() {
		pluginOutputs = append(pluginOutputs, plugin.Run())
	}
	totalScore := 0.0
	// Update the state for the tests
	for _, suite := range pluginOutputs {
		for idx, res := range suite.Results {
			suite.Results[idx] = schelpers.UpdateSuiteStates(res)
		}
	}
	if viper.GetString(OutputFormatOpt) == HumanReadableOutputFormat {
		numSuites := 0
		for _, plugin := range pluginOutputs {
			for _, suite := range plugin.Results {
				fmt.Printf("%s:\n", suite.Name)
				for _, result := range suite.Tests {
					fmt.Printf("\t%s: %d/%d\n", result.Name, result.EarnedPoints, result.MaximumPoints)
				}
				totalScore += float64(suite.TotalScore)
				numSuites++
			}
		}
		totalScore = totalScore / float64(numSuites)
		fmt.Printf("\nTotal Score: %.0f%%\n", totalScore)
		// TODO: We can probably use some helper functions to clean up these quadruple nested loops
		// Print suggestions
		for _, plugin := range pluginOutputs {
			for _, suite := range plugin.Results {
				for _, result := range suite.Tests {
					for _, suggestion := range result.Suggestions {
						// 33 is yellow (specifically, the same shade of yellow that logrus uses for warnings)
						fmt.Printf("\x1b[%dmSUGGESTION:\x1b[0m %s\n", 33, suggestion)
					}
				}
			}
		}
		// Print errors
		for _, plugin := range pluginOutputs {
			for _, suite := range plugin.Results {
				for _, result := range suite.Tests {
					for _, err := range result.Errors {
						// 31 is red (specifically, the same shade of red that logrus uses for errors)
						fmt.Printf("\x1b[%dmERROR:\x1b[0m %s\n", 31, err)
					}
				}
			}
		}
	}
	if viper.GetString(OutputFormatOpt) == JSONOutputFormat {
		log, err := ioutil.ReadAll(logReadWriter)
		if err != nil {
			return fmt.Errorf("failed to read log buffer: %v", err)
		}
		scTest := schelpers.CombineScorecardOutput(pluginOutputs, string(log))
		// Pretty print so users can also read the json output
		bytes, err := json.MarshalIndent(scTest, "", "  ")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", string(bytes))
	}
	return nil
}

func initConfig() error {
	// viper/cobra already has flags parsed at this point; we can check if a config file flag is set
	if viper.GetString(ConfigOpt) != "" {
		// Use config file from the flag.
		viper.SetConfigFile(viper.GetString(ConfigOpt))
	} else {
		viper.AddConfigPath(projutil.MustGetwd())
		// using SetConfigName allows users to use a .yaml, .json, or .toml file
		viper.SetConfigName(DefaultConfigFile)
	}

	if err := viper.ReadInConfig(); err == nil {
		// configure logger output before logging anything
		err := configureLogger()
		if err != nil {
			return err
		}
		log.Info("Using config file: ", viper.ConfigFileUsed())
	} else {
		err := configureLogger()
		if err != nil {
			return err
		}
		log.Warn("Could not load config file; using flags")
	}
	return nil
}

func configureLogger() error {
	if viper.GetString(OutputFormatOpt) == HumanReadableOutputFormat {
		logReadWriter = os.Stdout
	} else if viper.GetString(OutputFormatOpt) == JSONOutputFormat {
		logReadWriter = &bytes.Buffer{}
	} else {
		return fmt.Errorf("invalid output format: %s", viper.GetString(OutputFormatOpt))
	}
	log.SetOutput(logReadWriter)
	return nil
}

func validateScorecardFlags() error {
	// this is already being checked in configure logger; may be unnecessary
	outputFormat := viper.GetString(OutputFormatOpt)
	if outputFormat != HumanReadableOutputFormat && outputFormat != JSONOutputFormat {
		return fmt.Errorf("invalid output format (%s); valid values: %s, %s", outputFormat, HumanReadableOutputFormat, JSONOutputFormat)
	}
	return nil
}
