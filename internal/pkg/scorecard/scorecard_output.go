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
	schelpers "github.com/operator-framework/operator-sdk/internal/pkg/scorecard/helpers"
	scapi "github.com/operator-framework/operator-sdk/pkg/apis/scorecard"
	scapiv1alpha1 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha1"
)

func printPluginOutputs(pluginOutputs []scapiv1alpha1.ScorecardOutput) error {

	var list scapi.ScorecardFormatter
	if schelpers.IsV1alpha2() {
		//list = schelpers.ConvertScorecardOutputV1ToV2(pluginOutputs)
		list = scapi.ConvertScorecardOutputV1ToV2(pluginOutputs)
	} else {
		v1list := scapiv1alpha1.ScorecardOutputList{}
		v1list.Items = pluginOutputs
		list = v1list
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
		bytes, err := list.MarshalJSONOutput(logReadWriter)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", string(bytes))
		return nil
	}

	return nil
}
