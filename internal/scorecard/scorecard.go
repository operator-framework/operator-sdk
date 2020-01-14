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

type Config struct {
	ListOpt         bool
	OutputFormatOpt string
	VersionOpt      string
	SelectorOpt     string
	Selector        labels.Selector
	BundleOpt       string
	Kubeconfig      string
	Plugins         []Plugin
}

type pluginConfig struct {
	Basic    *scplugins.BasicAndOLMPluginConfig `mapstructure:"basic,omitempty"`
	Olm      *scplugins.BasicAndOLMPluginConfig `mapstructure:"olm,omitempty"`
	External *externalPluginConfig              `mapstructure:"external,omitempty"`
}

func (s Config) getPlugins(scViper *viper.Viper) ([]Plugin, error) {

	// Add plugins from config
	var plugins []Plugin
	configs := []pluginConfig{}
	// set ErrorUnused to true in decoder to fail if an unknown field is set by the user
	if err := scViper.UnmarshalKey("plugins", &configs, func(c *mapstructure.DecoderConfig) { c.ErrorUnused = true }); err != nil {
		return nil, errors.Wrap(err, "Could not load plugin configurations")
	}
	for idx, plugin := range configs {
		if err := validateConfig(plugin, idx, s.VersionOpt); err != nil {
			return nil, fmt.Errorf("error validating plugin config: %v", err)
		}
		var newPlugin Plugin
		if plugin.Basic != nil {
			pluginConfig := plugin.Basic
			pluginConfig.Version = s.VersionOpt
			pluginConfig.Selector = s.Selector
			pluginConfig.ListOpt = s.ListOpt
			pluginConfig.Bundle = s.BundleOpt
			setConfigDefaults(pluginConfig, s.Kubeconfig)
			newPlugin = basicOrOLMPlugin{pluginType: scplugins.BasicOperator, config: *pluginConfig}
		} else if plugin.Olm != nil {
			pluginConfig := plugin.Olm
			pluginConfig.Version = s.VersionOpt
			pluginConfig.Selector = s.Selector
			pluginConfig.ListOpt = s.ListOpt
			pluginConfig.Bundle = s.BundleOpt
			setConfigDefaults(pluginConfig, s.Kubeconfig)
			newPlugin = basicOrOLMPlugin{pluginType: scplugins.OLMIntegration, config: *pluginConfig}
		} else {
			pluginConfig := plugin.External
			if s.Kubeconfig != "" {
				// put the kubeconfig flag first in case user is overriding it with an env var in config file
				pluginConfig.Env = append([]externalPluginEnv{{Name: "KUBECONFIG", Value: s.Kubeconfig}}, pluginConfig.Env...)
			}
			newPlugin = externalPlugin{config: *pluginConfig}
		}
		plugins = append(plugins, newPlugin)
	}
	return plugins, nil
}

func Tests(cmd *cobra.Command, args []string) error {

	scViper, err := initConfig()
	if err != nil {
		return err
	}

	outputFormat := scViper.GetString(OutputFormatOpt)
	if outputFormat != TextOutputFormat && outputFormat != JSONOutputFormat {
		return fmt.Errorf("invalid output format (%s); valid values: %s, %s", outputFormat, TextOutputFormat, JSONOutputFormat)
	}

	version := scViper.GetString(schelpers.VersionOpt)
	err = schelpers.ValidateVersion(version)
	if err != nil {
		return err
	}

	if !schelpers.IsV1alpha2(version) && scViper.GetBool(ListOpt) {
		return fmt.Errorf("list flag is not supported on v1alpha1")
	}

	c := Config{}
	c.ListOpt = scViper.GetBool(ListOpt)
	c.OutputFormatOpt = scViper.GetString(OutputFormatOpt)
	c.VersionOpt = scViper.GetString(schelpers.VersionOpt)
	c.SelectorOpt = scViper.GetString(SelectorOpt)
	c.BundleOpt = scViper.GetString(BundleOpt)

	c.Kubeconfig = ""
	if scViper.IsSet(scplugins.KubeconfigOpt) {
		c.Kubeconfig = scViper.GetString(scplugins.KubeconfigOpt)
	}

	c.Selector, err = labels.Parse(c.SelectorOpt)
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true
	c.Plugins, err = c.getPlugins(scViper)
	if err != nil {
		return err
	}

	return c.RunTests()

}

func (s Config) RunTests() error {

	var pluginOutputs []scapiv1alpha1.ScorecardOutput
	for _, plugin := range s.Plugins {
		if s.ListOpt {
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

	if err := printPluginOutputs(s, pluginOutputs); err != nil {
		return err
	}

	if schelpers.IsV1alpha2(s.VersionOpt) {
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

func initConfig() (*viper.Viper, error) {
	// viper/cobra already has flags parsed at this point; we can check if a config file flag is set
	if viper.GetString(ConfigOpt) != "" {
		// Use config file from the flag.
		viper.SetConfigFile(viper.GetString(ConfigOpt))
	} else {
		viper.AddConfigPath(projutil.MustGetwd())
		// using SetConfigName allows users to use a .yaml, .json, or .toml file
		viper.SetConfigName(DefaultConfigFile)
	}

	var scViper *viper.Viper
	if err := viper.ReadInConfig(); err == nil {
		scViper = viper.Sub("scorecard")
		// this is a workaround for the fact that nested flags don't persist on viper.Sub
		scViper.Set(OutputFormatOpt, viper.GetString("scorecard."+OutputFormatOpt))
		scViper.Set(scplugins.KubeconfigOpt, viper.GetString("scorecard."+scplugins.KubeconfigOpt))
		scViper.Set(schelpers.VersionOpt, viper.GetString("scorecard."+schelpers.VersionOpt))
		scViper.Set(SelectorOpt, viper.GetString("scorecard."+SelectorOpt))
		scViper.Set(BundleOpt, viper.GetString("scorecard."+BundleOpt))
		scViper.Set(ListOpt, viper.GetString("scorecard."+ListOpt))

		// configure logger output before logging anything
		if !scViper.IsSet(OutputFormatOpt) {
			scViper.Set(OutputFormatOpt, TextOutputFormat)
		}
		format := scViper.GetString(OutputFormatOpt)
		if format == TextOutputFormat {
			logReadWriter = os.Stdout
		} else if format == JSONOutputFormat {
			logReadWriter = &bytes.Buffer{}
		} else {
			return nil, fmt.Errorf("invalid output format: %s", format)
		}
		log.SetOutput(logReadWriter)
		if err != nil {
			return nil, err
		}
		log.Info("Using config file: ", viper.ConfigFileUsed())
	} else {
		return nil, fmt.Errorf("could not read config file: %v\nSee %s for more information about the scorecard config file", err, configDocLink())
	}
	return scViper, nil
}

func configDocLink() string {
	if strings.HasSuffix(version.Version, "+git") {
		return "https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/scorecard.md"
	}
	return fmt.Sprintf("https://github.com/operator-framework/operator-sdk/blob/%s/doc/test-framework/scorecard.md", version.Version)
}
