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

	scplugins "github.com/operator-framework/operator-sdk/internal/pkg/scorecard/plugins"
	scapiv1alpha1 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha1"

	"github.com/spf13/viper"
)

type Plugin interface {
	Name() string
	Run() scapiv1alpha1.ScorecardOutput
}

type genericPlugin struct {
	binPath string
}

type internalPlugin struct {
	name       string
	pluginType scplugins.PluginType
}

func (p genericPlugin) Name() string {
	return filepath.Base(p.binPath)
}

func (p genericPlugin) Run() scapiv1alpha1.ScorecardOutput {
	// This should never error since we chdir in getPlugins()
	if err := os.Chdir(viper.GetString(PluginDirOpt)); err != nil {
		name := fmt.Sprintf("Failed Plugin: %s", filepath.Base(p.binPath))
		description := fmt.Sprintf("Plugin with file name `%s` failed", filepath.Base(p.binPath))
		logs := fmt.Sprintf("failed to chdir into scorecard plugin directory: %v", err)
		// output error to main logger as well for human-readable output
		log.Errorf("Failed to chdir into scorecard plugin directory: %v", err)
		return failedPlugin(name, description, logs)
	}
	cmd := exec.Command(p.binPath)
	stdout := &bytes.Buffer{}
	cmd.Stdout = stdout
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		name := fmt.Sprintf("Failed Plugin: %s", filepath.Base(p.binPath))
		description := fmt.Sprintf("Plugin with file name `%s` failed", filepath.Base(p.binPath))
		logs := fmt.Sprintf("%s:\nStdout: %s\nStderr: %s", err, string(stdout.Bytes()), string(stderr.Bytes()))
		// output error to main logger as well for human-readable output
		log.Errorf("Plugin `%s` failed with error (%v)", filepath.Base(p.binPath), err)
		return failedPlugin(name, description, logs)
	}
	// parse output and add to suites
	result := scapiv1alpha1.ScorecardOutput{}
	err = json.Unmarshal(stdout.Bytes(), &result)
	if err != nil {
		name := fmt.Sprintf("Plugin output invalid: %s", filepath.Base(p.binPath))
		description := fmt.Sprintf("Plugin with file name %s did not produce valid ScorecardOutput JSON", filepath.Base(p.binPath))
		logs := fmt.Sprintf("%s:\nStdout: %s\nStderr: %s", err, string(stdout.Bytes()), string(stderr.Bytes()))
		// output error to main logger as well for human-readable output
		log.Errorf("Output from plugin `%s` failed to unmarshal with error (%v)", filepath.Base(p.binPath), err)
		return failedPlugin(name, description, logs)
	}
	stderrString := string(stderr.Bytes())
	if len(stderrString) != 0 {
		log.Warn(stderrString)
	}
	return result
}

var basicTestsPlugin = internalPlugin{
	name:       scplugins.BasicTestsOpt,
	pluginType: scplugins.BasicOperator,
}

var olmTestsPlugin = internalPlugin{
	name:       scplugins.OLMTestsOpt,
	pluginType: scplugins.OLMIntegration,
}

func (p internalPlugin) Name() string {
	return p.name
}

func (p internalPlugin) Run() scapiv1alpha1.ScorecardOutput {
	// This shouldn't error since we started in the rootDir
	if err := os.Chdir(rootDir); err != nil {
		name := fmt.Sprintf("Failed Plugin: %s", p.name)
		description := fmt.Sprintf("Internal plugin `%s` failed", p.name)
		logs := fmt.Sprintf("failed to chdir into project root directory: %v", err)
		// output error to main logger as well for human-readable output
		log.Errorf("Failed to chdir into project root directory: %v", err)
		return failedPlugin(name, description, logs)
	}
	// TODO: make individual viper configs
	pluginLogs := &bytes.Buffer{}
	res, err := scplugins.RunInternalPlugin(p.pluginType, viper.GetViper(), pluginLogs)
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

func failedPlugin(name, desc, log string) scapiv1alpha1.ScorecardOutput {
	return scapiv1alpha1.ScorecardOutput{
		Results: []scapiv1alpha1.ScorecardSuiteResult{{
			Name:        name,
			Description: desc,
			Error:       1,
			Log:         log,
		},
		},
	}
}
