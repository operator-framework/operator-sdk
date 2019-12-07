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
	scapiv1alpha1 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha1"
	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
)

var _ ScorecardFormatter = &scapiv1alpha1.ScorecardOutput{}
var _ ScorecardFormatter = &scapiv1alpha2.ScorecardOutput{}

type ScorecardFormatter interface { //nolint:golint
	// todo(camilamacedo86): The no lint here is for pkg/apis/scorecard/formatting.go:25:6: type name will be used as scorecard.ScorecardFormatter by other packages, and that stutters; consider calling this Formatter (golint)
	// However, was decided to not move forward with it now in order to not introduce breakchanges with the task to add the linter. We should to do it after.
	MarshalText() (string, error)
}
