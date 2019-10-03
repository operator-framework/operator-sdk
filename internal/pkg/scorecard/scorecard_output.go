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
	schelpers "github.com/operator-framework/operator-sdk/internal/pkg/scorecard/helpers"
	scapi "github.com/operator-framework/operator-sdk/pkg/apis/scorecard"
	scapiv1alpha1 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha1"
	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
	"io/ioutil"
)

func printPluginOutputs(pluginOutputs []scapiv1alpha1.ScorecardOutput) error {

	var list scapi.ScorecardFormatter
	list = combinePluginOutput(pluginOutputs)

	if schelpers.IsV1alpha2() {
		list = scapi.ConvertScorecardOutputV1ToV2(list.(scapiv1alpha1.ScorecardOutput))
	}

	// produce text output
	if scViper.GetString(OutputFormatOpt) == TextOutputFormat {
		output, err := list.MarshalText()
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", output)

		return nil
	}

	// produce json output
	if scViper.GetString(OutputFormatOpt) == JSONOutputFormat {

		log, err := ioutil.ReadAll(logReadWriter)
		if err != nil {
			return fmt.Errorf("failed to read log buffer: %v", err)
		}
		if schelpers.IsV1alpha2() {
			temp := list.(scapiv1alpha2.ScorecardOutput)
			temp.Log = string(log)
			bytes, err := json.MarshalIndent(temp, "", "  ")
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", string(bytes))
			return nil
		}

		temp := list.(scapiv1alpha1.ScorecardOutput)
		temp.Log = string(log)
		bytes, err := json.MarshalIndent(temp, "", "  ")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", string(bytes))
		return nil

	}

	return nil
}

func combinePluginOutput(pluginOutputs []scapiv1alpha1.ScorecardOutput) scapiv1alpha1.ScorecardOutput {
	output := scapiv1alpha1.ScorecardOutput{}
	output.Results = make([]scapiv1alpha1.ScorecardSuiteResult, 0)
	for _, v := range pluginOutputs {
		for _, r := range v.Results {
			output.Results = append(output.Results, r)
		}
	}

	return output
}
