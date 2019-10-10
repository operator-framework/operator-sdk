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
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	scplugins "github.com/operator-framework/operator-sdk/internal/pkg/scorecard/plugins"
	scapiv1alpha1 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha1"
	"github.com/operator-framework/operator-sdk/version"
	v1 "k8s.io/api/core/v1"
)

type Plugin interface {
	Run() scapiv1alpha1.ScorecardOutput
}

type externalPlugin struct {
	config externalPluginConfig
}

type basicOrOLMPlugin struct {
	pluginType scplugins.PluginType
	config     scplugins.BasicAndOLMPluginConfig
}

func (p externalPlugin) Run() scapiv1alpha1.ScorecardOutput {
	cmd := exec.Command(p.config.Command, p.config.Args...)
	for _, env := range p.config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}
	stdout := &bytes.Buffer{}
	cmd.Stdout = stdout
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		name := filepath.Base(p.config.Command)
		logs := fmt.Sprintf("%s\nError: %s\nStdout: %s\nStderr: %s", p.config, err, stdout.String(), stderr.String())
		// output error to main logger as well for human-readable output
		log.Errorf("Plugin `%s` failed\nLogs: %s", filepath.Base(p.config.Command), logs)
		return failedPlugin(name, logs)
	}
	// parse output and add to suites
	result := scapiv1alpha1.ScorecardOutput{}
	err = json.Unmarshal(stdout.Bytes(), &result)
	if err != nil {
		name := filepath.Base(p.config.Command)
		logs := fmt.Sprintf("%s\nError: %s\nStdout: %s\nStderr: %s", p.config, err, stdout.String(), stderr.String())
		// output error to main logger as well for human-readable output
		log.Errorf("Output from plugin `%s` failed to unmarshal\nLogs: %s", filepath.Base(p.config.Command), logs)
		return failedPlugin(name, logs)
	}
	stderrString := stderr.String()
	if len(stderrString) != 0 {
		log.Warn(stderrString)
	}
	return result
}

var basicTestsPlugin = basicOrOLMPlugin{
	pluginType: scplugins.BasicOperator,
}

var olmTestsPlugin = basicOrOLMPlugin{
	pluginType: scplugins.OLMIntegration,
}

func (p basicOrOLMPlugin) Run() scapiv1alpha1.ScorecardOutput {
	pluginLogs := &bytes.Buffer{}
	res, err := scplugins.RunInternalPlugin(p.pluginType, p.config, pluginLogs)
	if err != nil {
		var name string
		if p.pluginType == scplugins.BasicOperator {
			name = fmt.Sprintf("Basic Tests")
		} else if p.pluginType == scplugins.OLMIntegration {
			name = fmt.Sprintf("OLM Integration")
		}
		logs := fmt.Sprintf("%s:\nLogs: %s", err, pluginLogs.String())
		// output error to main logger as well for human-readable output
		log.Errorf("Plugin `%s` failed with error (%v)", name, err)
		return failedPlugin(name, logs)
	}
	stderrString := pluginLogs.String()
	if len(stderrString) != 0 {
		log.Warn(stderrString)
	}
	return res
}

// setConfigDefaults sets certain config fields to default values if they are not set
func setConfigDefaults(config *scplugins.BasicAndOLMPluginConfig, kubeconfig string) {
	if config.InitTimeout == 0 {
		config.InitTimeout = 60
	}
	if config.ProxyImage == "" {
		config.ProxyImage = fmt.Sprintf("quay.io/operator-framework/scorecard-proxy:%s", strings.TrimSuffix(version.Version, "+git"))
	}
	if config.ProxyPullPolicy == "" {
		config.ProxyPullPolicy = v1.PullAlways
	}
	if config.CRDsDir == "" {
		config.CRDsDir = scaffold.CRDsDir
	}
	if config.Kubeconfig == "" {
		config.Kubeconfig = kubeconfig
	}
}

func failedPlugin(name, log string) scapiv1alpha1.ScorecardOutput {
	return scapiv1alpha1.ScorecardOutput{
		Results: []scapiv1alpha1.ScorecardSuiteResult{{
			Name:  name,
			Error: 1,
			Log:   log,
		}},
	}
}
