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
	"io"
	"os"

	scplugins "github.com/operator-framework/operator-sdk/internal/scorecard/plugins"
	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
)

const DefaultConfigFile = ".osdk-scorecard"

const (
	JSONOutputFormat = "json"
	TextOutputFormat = "text"
)

var (
	Log = logrus.New()
)

type Config struct {
	List          bool
	OutputFormat  string
	Version       string
	Selector      labels.Selector
	Bundle        string
	Kubeconfig    string
	Plugins       []Plugin
	PluginConfigs []PluginConfig
	LogReadWriter io.ReadWriter
}

type PluginConfig struct {
	Basic *scplugins.BasicAndOLMPluginConfig `mapstructure:"basic,omitempty"`
	Olm   *scplugins.BasicAndOLMPluginConfig `mapstructure:"olm,omitempty"`
}

func (s Config) GetPlugins(configs []PluginConfig) ([]Plugin, error) {

	// Add plugins from config
	var plugins []Plugin

	for _, plugin := range configs {
		var newPlugin Plugin
		if plugin.Basic != nil {
			pluginConfig := plugin.Basic
			pluginConfig.Version = s.Version
			pluginConfig.Selector = s.Selector
			pluginConfig.ListOpt = s.List
			pluginConfig.Bundle = s.Bundle
			setConfigDefaults(pluginConfig, s.Kubeconfig)
			newPlugin = basicOrOLMPlugin{pluginType: scplugins.BasicOperator, config: *pluginConfig}
		} else if plugin.Olm != nil {
			pluginConfig := plugin.Olm
			pluginConfig.Version = s.Version
			pluginConfig.Selector = s.Selector
			pluginConfig.ListOpt = s.List
			pluginConfig.Bundle = s.Bundle
			setConfigDefaults(pluginConfig, s.Kubeconfig)
			newPlugin = basicOrOLMPlugin{pluginType: scplugins.OLMIntegration, config: *pluginConfig}
		}
		plugins = append(plugins, newPlugin)
	}
	return plugins, nil
}

func (s Config) RunTests() error {

	var pluginOutputs []scapiv1alpha2.ScorecardOutput
	for _, plugin := range s.Plugins {
		if s.List {
			pluginOutputs = append(pluginOutputs, plugin.List())
		} else {
			pluginOutputs = append(pluginOutputs, plugin.Run())
		}
	}

	if err := s.printPluginOutputs(pluginOutputs); err != nil {
		return err
	}

	for _, scorecardOutput := range pluginOutputs {
		for _, result := range scorecardOutput.Results {
			if result.State != scapiv1alpha2.PassState {
				os.Exit(1)
			}
		}
	}

	return nil
}

func ConfigDocLink() string {
	return "https://sdk.operatorframework.io/docs/scorecard/"
}
