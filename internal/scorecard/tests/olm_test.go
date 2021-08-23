// Copyright 2021 The Operator-SDK Authors
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

package tests

import (
	"testing"

	scapiv1alpha3 "github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
)

func Test_checkSize(t *testing.T) {
	type args struct {
		size int64
		r    *scapiv1alpha3.TestResult
	}
	tests := []struct {
		name string
		args args
		want scapiv1alpha3.State
	}{
		{
			name: "fail on bundle size too large",
			args: args{
				size: 1048577,
				r:    new(scapiv1alpha3.TestResult),
			},
			want: scapiv1alpha3.FailState,
		},
		{
			name: "pass on bundle size small enough",
			args: args{
				size: 1048576,
				r:    new(scapiv1alpha3.TestResult),
			},
			want: scapiv1alpha3.PassState,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkSize(tt.args.size, *tt.args.r)
			if got.State != tt.want {
				t.Errorf("BundleSizeTest CheckSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
