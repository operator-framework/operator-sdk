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
	"io"
	"os"
	"strings"

	schelpers "github.com/operator-framework/operator-sdk/internal/scorecard/helpers"
	scplugins "github.com/operator-framework/operator-sdk/internal/scorecard/plugins"
	scapiv1alpha1 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha1"
	"github.com/operator-framework/operator-sdk/version"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/sirupsen/logrus"
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
	LogReadWriter io.ReadWriter
	Log           = logrus.New()
)

type Config struct {
	ListOpt         bool
	OutputFormatOpt string
	VersionOpt      string
	ConfigOpt       string
	SelectorOpt     string
	Selector        labels.Selector
	BundleOpt       string
	Kubeconfig      string
	Something       string
	Plugins         []Plugin
	PluginConfigs   []PluginConfig
}

type PluginConfig struct {
	Basic    *scplugins.BasicAndOLMPluginConfig `mapstructure:"basic,omitempty"`
	Olm      *scplugins.BasicAndOLMPluginConfig `mapstructure:"olm,omitempty"`
	External *externalPluginConfig              `mapstructure:"external,omitempty"`
}

func (s Config) GetPlugins(configs []PluginConfig) ([]Plugin, error) {

	// Add plugins from config
	var plugins []Plugin
	for _, plugin := range configs {
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

func (s Config) RunTests() error {
	for idx, plugin := range s.PluginConfigs {
		if err := validateConfig(plugin, idx, s.VersionOpt); err != nil {
			return fmt.Errorf("error validating plugin config: %v", err)
		}
	}

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

	if err := s.printPluginOutputs(pluginOutputs); err != nil {
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

func ConfigDocLink() string {
	if strings.HasSuffix(version.Version, "+git") {
		return "https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/scorecard.md"
	}
	return fmt.Sprintf("https://github.com/operator-framework/operator-sdk/blob/%s/doc/test-framework/scorecard.md", version.Version)
}
