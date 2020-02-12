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
	"encoding/json"
	"fmt"
	"io/ioutil"

	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
)

func (cfg Config) printPluginOutputs(pluginOutputs []scapiv1alpha2.ScorecardOutput) error {

	var scorecardOutput scapiv1alpha2.ScorecardOutput
	var err error
	scorecardOutput, err = cfg.combinePluginOutput(pluginOutputs)
	if err != nil {
		return err
	}

	if cfg.List {
		for i := 0; i < len(scorecardOutput.Results); i++ {
			scorecardOutput.Results[i].State = scapiv1alpha2.NotRunState
		}
	}

	switch format := cfg.OutputFormat; format {
	case TextOutputFormat:
		output, err := scorecardOutput.MarshalText()
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", output)
	case JSONOutputFormat:
		bytes, err := json.MarshalIndent(scorecardOutput, "", "  ")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", string(bytes))
	}

	return nil
}

func (cfg Config) combinePluginOutput(pluginOutputs []scapiv1alpha2.
	ScorecardOutput) (scapiv1alpha2.ScorecardOutput, error) {
	output := scapiv1alpha2.ScorecardOutput{}
	output.Results = make([]scapiv1alpha2.ScorecardTestResult, 0)
	for _, v := range pluginOutputs {
		output.Results = append(output.Results, v.Results...)
	}

	if cfg.OutputFormat == JSONOutputFormat {
		log, err := ioutil.ReadAll(cfg.LogReadWriter)
		if err != nil {
			return output, fmt.Errorf("failed to read log buffer: %v", err)
		}
		output.Log = string(log)
	}

	return output, nil
}
