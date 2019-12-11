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
	"fmt"
	"io"
	"os"
	"strings"

	schelpers "github.com/operator-framework/operator-sdk/internal/scorecard/helpers"
	scplugins "github.com/operator-framework/operator-sdk/internal/scorecard/plugins"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	scapiv1alpha1 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha1"
	"github.com/operator-framework/operator-sdk/version"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const DefaultConfigFile = ".osdk-scorecard"

const (
	ConfigOpt        = "config"
	OutputFormatOpt  = "output"
	JSONOutputFormat = "json"
	TextOutputFormat = "text"
	SelectorOpt      = "selector"
	BundleOpt        = "bundle"
	ListOpt          = "list"
)

// make a global logger for scorecard
var (
	logReadWriter io.ReadWriter
	log           = logrus.New()
)

var scViper *viper.Viper

type pluginConfig struct {
	Basic    *scplugins.BasicAndOLMPluginConfig `mapstructure:"basic,omitempty"`
	Olm      *scplugins.BasicAndOLMPluginConfig `mapstructure:"olm,omitempty"`
	External *externalPluginConfig              `mapstructure:"external,omitempty"`
}

func getPlugins(version string, selector labels.Selector) ([]Plugin, error) {
	kubeconfig := ""
	if scViper.IsSet(scplugins.KubeconfigOpt) {
		kubeconfig = scViper.GetString(scplugins.KubeconfigOpt)
	}

	// Add plugins from config
	var plugins []Plugin
	configs := []pluginConfig{}
	// set ErrorUnused to true in decoder to fail if an unknown field is set by the user
	if err := scViper.UnmarshalKey("plugins", &configs, func(c *mapstructure.DecoderConfig) { c.ErrorUnused = true }); err != nil {
		return nil, errors.Wrap(err, "Could not load plugin configurations")
	}
	for idx, plugin := range configs {
		if err := validateConfig(plugin, idx, version); err != nil {
			return nil, fmt.Errorf("error validating plugin config: %v", err)
		}
		var newPlugin Plugin
		if plugin.Basic != nil {
			pluginConfig := plugin.Basic
			pluginConfig.Version = version
			pluginConfig.Selector = selector
			pluginConfig.ListOpt = scViper.GetBool(ListOpt)
			pluginConfig.Bundle = scViper.GetString(BundleOpt)
			setConfigDefaults(pluginConfig, kubeconfig)
			newPlugin = basicOrOLMPlugin{pluginType: scplugins.BasicOperator, config: *pluginConfig}
		} else if plugin.Olm != nil {
			pluginConfig := plugin.Olm
			pluginConfig.Version = version
			pluginConfig.Selector = selector
			pluginConfig.ListOpt = scViper.GetBool(ListOpt)
			pluginConfig.Bundle = scViper.GetString(BundleOpt)
			setConfigDefaults(pluginConfig, kubeconfig)
			newPlugin = basicOrOLMPlugin{pluginType: scplugins.OLMIntegration, config: *pluginConfig}
		} else {
			pluginConfig := plugin.External
			if kubeconfig != "" {
				// put the kubeconfig flag first in case user is overriding it with an env var in config file
				pluginConfig.Env = append([]externalPluginEnv{{Name: "KUBECONFIG", Value: kubeconfig}}, pluginConfig.Env...)
			}
			newPlugin = externalPlugin{config: *pluginConfig}
		}
		plugins = append(plugins, newPlugin)
	}
	return plugins, nil
}

func Tests(cmd *cobra.Command, args []string) error {
	if err := initConfig(); err != nil {
		return err
	}
	if err := validateScorecardConfig(); err != nil {
		return err
	}

	selector, err := labels.Parse(scViper.GetString(SelectorOpt))
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true
	plugins, err := getPlugins(scViper.GetString(schelpers.VersionOpt), selector)
	if err != nil {
		return err
	}

	var pluginOutputs []scapiv1alpha1.ScorecardOutput
	for _, plugin := range plugins {
		if scViper.GetBool(ListOpt) {
			pluginOutputs = append(pluginOutputs, plugin.List())
		} else {
			pluginOutputs = append(pluginOutputs, plugin.Run())
		}
	}

	// Update the state for the tests
	for _, suite := range pluginOutputs {
		for idx, res := range suite.Results {
			suite.Results[idx] = schelpers.UpdateSuiteStates(res)
		}
	}

	if err := printPluginOutputs(scViper.GetString(schelpers.VersionOpt), pluginOutputs); err != nil {
		return err
	}

	apiVersion := scViper.GetString(schelpers.VersionOpt)
	if schelpers.IsV1alpha2(apiVersion) {
		for _, scorecardOutput := range pluginOutputs {
			for _, result := range scorecardOutput.Results {
				if result.Fail > 0 || result.PartialPass > 0 {
					os.Exit(1)
				}
			}
		}
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
		makeSCViper()
		// configure logger output before logging anything
		err := configureLogger()
		if err != nil {
			return err
		}
		log.Info("Using config file: ", viper.ConfigFileUsed())
	} else {
		return fmt.Errorf("could not read config file: %v\nSee %s for more information about the scorecard config file", err, configDocLink())
	}
	return nil
}

func configureLogger() error {
	if !scViper.IsSet(OutputFormatOpt) {
		scViper.Set(OutputFormatOpt, TextOutputFormat)
	}
	format := scViper.GetString(OutputFormatOpt)
	if format == TextOutputFormat {
		logReadWriter = os.Stdout
	} else if format == JSONOutputFormat {
		logReadWriter = &bytes.Buffer{}
	} else {
		return fmt.Errorf("invalid output format: %s", format)
	}
	log.SetOutput(logReadWriter)
	return nil
}

func validateScorecardConfig() error {
	// this is already being checked in configure logger; may be unnecessary
	outputFormat := scViper.GetString(OutputFormatOpt)
	if outputFormat != TextOutputFormat && outputFormat != JSONOutputFormat {
		return fmt.Errorf("invalid output format (%s); valid values: %s, %s", outputFormat, TextOutputFormat, JSONOutputFormat)
	}

	version := scViper.GetString(schelpers.VersionOpt)
	err := schelpers.ValidateVersion(version)
	if err != nil {
		return err
	}

	if !schelpers.IsV1alpha2(version) && scViper.GetBool(ListOpt) {
		return fmt.Errorf("list flag is not supported on v1alpha1")
	}

	return nil

}

func makeSCViper() {
	scViper = viper.Sub("scorecard")
	// this is a workaround for the fact that nested flags don't persist on viper.Sub
	scViper.Set(OutputFormatOpt, viper.GetString("scorecard."+OutputFormatOpt))
	scViper.Set(scplugins.KubeconfigOpt, viper.GetString("scorecard."+scplugins.KubeconfigOpt))
	scViper.Set(schelpers.VersionOpt, viper.GetString("scorecard."+schelpers.VersionOpt))
	scViper.Set(SelectorOpt, viper.GetString("scorecard."+SelectorOpt))
	scViper.Set(BundleOpt, viper.GetString("scorecard."+BundleOpt))
	scViper.Set(ListOpt, viper.GetString("scorecard."+ListOpt))

}

func configDocLink() string {
	if strings.HasSuffix(version.Version, "+git") {
		return "https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/scorecard.md"
	}
	return fmt.Sprintf("https://github.com/operator-framework/operator-sdk/blob/%s/doc/test-framework/scorecard.md", version.Version)
}
