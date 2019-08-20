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

package zap

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.uber.org/zap/zapcore"
)

func TestLevel(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		shouldErr bool
		expStr    string
		expSet    bool
		expLevel  zapcore.Level
	}{
		{
			name:     "debug level set",
			input:    "debug",
			expStr:   "debug",
			expSet:   true,
			expLevel: zapcore.DebugLevel,
		},
		{
			name:     "info level set",
			input:    "info",
			expStr:   "info",
			expSet:   true,
			expLevel: zapcore.InfoLevel,
		},
		{
			name:     "error level set",
			input:    "error",
			expStr:   "error",
			expSet:   true,
			expLevel: zapcore.ErrorLevel,
		},
		{
			name:      "negative number should error",
			input:     "-10",
			shouldErr: true,
			expSet:    false,
		},
		{
			name:     "positive number level results in negative level set",
			input:    "8",
			expStr:   "Level(-8)",
			expSet:   true,
			expLevel: zapcore.Level(int8(-8)),
		},
		{
			name:      "non-integer should cause error",
			input:     "invalid",
			shouldErr: true,
			expSet:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lvl := levelValue{}
			err := lvl.Set(tc.input)
			if err != nil && !tc.shouldErr {
				t.Fatalf("Unknown error - %v", err)
			}
			if err != nil && tc.shouldErr {
				return
			}
			assert.Equal(t, tc.expSet, lvl.set)
			assert.Equal(t, tc.expLevel, lvl.level)
			assert.Equal(t, "level", lvl.Type())
			assert.Equal(t, tc.expStr, lvl.String())
		})
	}
}

func TestSample(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		shouldErr bool
		expStr    string
		expSet    bool
		expValue  bool
	}{
		{
			name:     "enable sampling",
			input:    "true",
			expStr:   "true",
			expSet:   true,
			expValue: true,
		},
		{
			name:     "disable sampling",
			input:    "false",
			expStr:   "false",
			expSet:   true,
			expValue: false,
		},
		{
			name:      "invalid input",
			input:     "notaboolean",
			shouldErr: true,
			expSet:    false,
		},
		{
			name:     "UPPERCASE true input",
			input:    "true",
			expStr:   "true",
			expSet:   true,
			expValue: true,
		},
		{
			name:      "MiXeDCase true input",
			input:     "tRuE",
			shouldErr: true,
			expSet:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sample := sampleValue{}
			err := sample.Set(tc.input)
			if err != nil && !tc.shouldErr {
				t.Fatalf("Unknown error - %v", err)
			}
			if err != nil && tc.shouldErr {
				return
			}
			assert.Equal(t, tc.expSet, sample.set)
			assert.Equal(t, tc.expValue, sample.sample)
			assert.Equal(t, "sample", sample.Type())
			assert.Equal(t, tc.expStr, sample.String())
			assert.True(t, sample.IsBoolFlag())
		})
	}
}
func TestEncoder(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		shouldErr  bool
		expStr     string
		expSet     bool
		expEncoder zapcore.Encoder
	}{
		{
			name:       "json encoder",
			input:      "json",
			expStr:     "json",
			expSet:     true,
			expEncoder: newJSONEncoder(),
		},
		{
			name:       "console encoder",
			input:      "console",
			expStr:     "console",
			expSet:     true,
			expEncoder: newConsoleEncoder(),
		},
		{
			name:       "unknown encoder",
			input:      "unknown",
			shouldErr:  true,
			expEncoder: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoder := encoderValue{}
			err := encoder.Set(tc.input)
			if err != nil && !tc.shouldErr {
				t.Fatalf("Unknown error - %v", err)
			}
			if err != nil && tc.shouldErr {
				return
			}
			assert.Equal(t, tc.expSet, encoder.set)
			assert.Equal(t, "encoder", encoder.Type())
			assert.Equal(t, tc.expStr, encoder.String())
			assert.ObjectsAreEqual(tc.expEncoder, encoder.newEncoder)
		})
	}
}
func TestTimeEncoder(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		shouldErr bool
		expStr    string
		expSet    bool
	}{
		{
			name:   "iso8601 time encoding",
			input:  "iso8601",
			expStr: "iso8601",
			expSet: true,
		},
		{
			name:   "millis time encoding",
			input:  "millis",
			expStr: "millis",
			expSet: true,
		},
		{
			name:   "nanos time encoding",
			input:  "nanos",
			expStr: "nanos",
			expSet: true,
		},
		{
			name:   "epoch time encoding",
			input:  "epoch",
			expStr: "epoch",
			expSet: true,
		},
		{
			name:      "invalid time encoding",
			input:     "invalid",
			expStr:    "epoch",
			expSet:    true,
			shouldErr: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			te := timeEncodingValue{}
			err := te.Set(tc.input)
			if err != nil && !tc.shouldErr {
				t.Fatalf("Unknown error - %v", err)
			}
			if tc.shouldErr {
				assert.Error(t, err)
			}
			assert.Equal(t, tc.expSet, te.set)
			assert.Equal(t, "timeEncoding", te.Type())
			assert.Equal(t, tc.expStr, te.String())
		})
	}
}
