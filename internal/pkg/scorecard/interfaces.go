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
	"os"
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
	Name() string
	Run() scapiv1alpha1.ScorecardOutput
}

type externalPlugin struct {
	name   string
	config externalPluginConfig
}

type basicOrOLMPlugin struct {
	name       string
	pluginType scplugins.PluginType
	config     scplugins.BasicAndOLMPluginConfig
}

func (p externalPlugin) Name() string {
	return p.name
}

func (p externalPlugin) Run() scapiv1alpha1.ScorecardOutput {
	// This should never error since we chdir in getPlugins()
	if err := os.Chdir(filepath.Join(rootDir, scViper.GetString(PluginDirOpt))); err != nil {
		name := fmt.Sprintf("Failed Plugin: %s", p.name)
		description := fmt.Sprintf("Plugin with file name `%s` failed", filepath.Base(p.config.Command))
		logs := fmt.Sprintf("failed to chdir into scorecard plugin directory: %v", err)
		// output error to main logger as well for human-readable output
		log.Errorf("Failed to chdir into scorecard plugin directory: %v", err)
		return failedPlugin(name, description, logs)
	}
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
		name := fmt.Sprintf("Failed Plugin: %s", p.name)
		description := fmt.Sprintf("Plugin with file name `%s` failed", filepath.Base(p.config.Command))
		logs := fmt.Sprintf("%s:\nStdout: %s\nStderr: %s", err, string(stdout.Bytes()), string(stderr.Bytes()))
		// output error to main logger as well for human-readable output
		log.Errorf("Plugin `%s` failed with error (%v)", p.name, err)
		return failedPlugin(name, description, logs)
	}
	// parse output and add to suites
	result := scapiv1alpha1.ScorecardOutput{}
	err = json.Unmarshal(stdout.Bytes(), &result)
	if err != nil {
		name := fmt.Sprintf("Plugin output invalid: %s", p.name)
		description := fmt.Sprintf("Plugin with file name %s did not produce valid ScorecardOutput JSON", filepath.Base(p.config.Command))
		logs := fmt.Sprintf("%s:\nStdout: %s\nStderr: %s", err, string(stdout.Bytes()), string(stderr.Bytes()))
		// output error to main logger as well for human-readable output
		log.Errorf("Output from plugin `%s` failed to unmarshal with error (%v)", p.name, err)
		return failedPlugin(name, description, logs)
	}
	stderrString := string(stderr.Bytes())
	if len(stderrString) != 0 {
		log.Warnf(stderrString)
	}
	return result
}

var basicTestsPlugin = basicOrOLMPlugin{
	name:       scplugins.BasicTestsOpt,
	pluginType: scplugins.BasicOperator,
}

var olmTestsPlugin = basicOrOLMPlugin{
	name:       scplugins.OLMTestsOpt,
	pluginType: scplugins.OLMIntegration,
}

func (p basicOrOLMPlugin) Name() string {
	return p.name
}

func (p basicOrOLMPlugin) Run() scapiv1alpha1.ScorecardOutput {
	// This shouldn't error since we started in the rootDir
	if err := os.Chdir(rootDir); err != nil {
		name := fmt.Sprintf("Failed Plugin: %s", p.name)
		description := fmt.Sprintf("Internal plugin `%s` failed", p.name)
		logs := fmt.Sprintf("failed to chdir into project root directory: %v", err)
		// output error to main logger as well for human-readable output
		log.Errorf("Failed to chdir into project root directory: %v", err)
		return failedPlugin(name, description, logs)
	}
	pluginLogs := &bytes.Buffer{}
	res, err := scplugins.RunInternalPlugin(p.pluginType, p.config, pluginLogs)
	if err != nil {
		name := fmt.Sprintf("Failed Plugin: %s", p.name)
		description := fmt.Sprintf("Internal plugin `%s` failed", p.name)
		logs := fmt.Sprintf("%s:\nLogs: %s", err, pluginLogs.String())
		// output error to main logger as well for human-readable output
		log.Errorf("Plugin `%s` failed with error (%v)", p.name, err)
		return failedPlugin(name, description, logs)
	}
	stderrString := pluginLogs.String()
	if len(stderrString) != 0 {
		log.Warn(stderrString)
	}
	return res
}

// updateConfig sets certain config fields to default values if they are not set
func updateConfig(config *scplugins.BasicAndOLMPluginConfig, kubeconfig string) {
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

func failedPlugin(name, desc, log string) scapiv1alpha1.ScorecardOutput {
	return scapiv1alpha1.ScorecardOutput{
		Results: []scapiv1alpha1.ScorecardSuiteResult{{
			Name:        name,
			Description: desc,
			Error:       1,
			Log:         log,
			Tests: []scapiv1alpha1.ScorecardTestResult{
				{
					State:       scapiv1alpha1.ErrorState,
					Suggestions: []string{},
					Errors:      []string{},
				},
			},
		}},
	}
}
