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
	"bytes"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/stretchr/testify/assert"
)

func TestGetConfig(t *testing.T) {
	var opts []zap.Option
	type fields struct {
		name              string
		inEncoder         *encoderValue
		inLevel           *levelValue
		inSample          *sampleValue
		inTimeEncoding    *timeEncodingValue
		expected          *config
		inDevel           bool
		inStackTraceLevel *stackLevelValue
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "development on",
			fields: fields{inDevel: true,
				inEncoder: &encoderValue{
					set: false,
				},
				inLevel: &levelValue{
					set: false,
				},
				inSample: &sampleValue{
					set: false,
				},
				inTimeEncoding: &timeEncodingValue{
					set: false,
				},
				inStackTraceLevel: &stackLevelValue{
					set: false,
				},
				expected: &config{
					encoder: newConsoleEncoder(),
					level:   zap.NewAtomicLevelAt(zap.DebugLevel),
					opts:    append(opts, zap.Development(), zap.AddStacktrace(zap.ErrorLevel)),
					sample:  false,
				}},
		},
		{
			name: "development off",
			fields: fields{inDevel: false,
				inEncoder: &encoderValue{
					set: false,
				},
				inLevel: &levelValue{
					set: false,
				},
				inSample: &sampleValue{
					set: false,
				},
				inTimeEncoding: &timeEncodingValue{
					set: false,
				},
				inStackTraceLevel: &stackLevelValue{
					set: false,
				},
				expected: &config{
					encoder: newJSONEncoder(),
					level:   zap.NewAtomicLevelAt(zap.InfoLevel),
					opts:    append(opts, zap.AddStacktrace(zap.WarnLevel)),
					sample:  true,
				}},
		},
		{
			name: "set encoder",
			fields: fields{inDevel: false,
				inEncoder: &encoderValue{
					set:        true,
					newEncoder: newConsoleEncoder,
				},
				inLevel: &levelValue{
					set: false,
				},
				inSample: &sampleValue{
					set: false,
				},
				inTimeEncoding: &timeEncodingValue{
					set: false,
				},
				inStackTraceLevel: &stackLevelValue{
					set: false,
				},
				expected: &config{
					encoder: newConsoleEncoder(),
					level:   zap.NewAtomicLevelAt(zap.InfoLevel),
					opts:    append(opts, zap.AddStacktrace(zap.WarnLevel)),
					sample:  true,
				}},
		},
		{
			fields: fields{name: "set level using level constant",
				inDevel: false,
				inEncoder: &encoderValue{
					set: false,
				},
				inLevel: &levelValue{
					set:   true,
					level: zapcore.ErrorLevel,
				},
				inSample: &sampleValue{
					set: false,
				},
				inTimeEncoding: &timeEncodingValue{
					set: false,
				},
				inStackTraceLevel: &stackLevelValue{
					set: false,
				},
				expected: &config{
					encoder: newJSONEncoder(),
					level:   zap.NewAtomicLevelAt(zap.ErrorLevel),
					opts:    append(opts, zap.AddStacktrace(zap.WarnLevel)),
					sample:  true,
				}},
		},
		{
			name: "set level using custom level",
			fields: fields{inDevel: false,
				inEncoder: &encoderValue{
					set: false,
				},
				inLevel: &levelValue{
					set:   true,
					level: zapcore.Level(-10),
				},
				inSample: &sampleValue{
					set: false,
				},
				inTimeEncoding: &timeEncodingValue{
					set: false,
				},
				inStackTraceLevel: &stackLevelValue{
					set: false,
				},
				expected: &config{
					encoder: newJSONEncoder(),
					level:   zap.NewAtomicLevelAt(zapcore.Level(-10)),
					opts:    append(opts, zap.AddStacktrace(zap.WarnLevel)),
					sample:  false,
				}},
		},
		{
			name: "set sampling",
			fields: fields{inDevel: false,
				inEncoder: &encoderValue{
					set: false,
				},
				inLevel: &levelValue{
					set: false,
				},
				inSample: &sampleValue{
					set:    true,
					sample: false,
				},
				inTimeEncoding: &timeEncodingValue{
					set: false,
				},
				inStackTraceLevel: &stackLevelValue{
					set: false,
				},
				expected: &config{
					encoder: newJSONEncoder(),
					level:   zap.NewAtomicLevelAt(zap.InfoLevel),
					opts:    append(opts, zap.AddStacktrace(zap.WarnLevel)),
					sample:  false,
				}},
		},
		{
			name: "set level using custom level, sample override not possible",
			fields: fields{inDevel: false,
				inEncoder: &encoderValue{
					set: false,
				},
				inLevel: &levelValue{
					set:   true,
					level: zapcore.Level(-10),
				},
				inSample: &sampleValue{
					set:    true,
					sample: true,
				},
				inTimeEncoding: &timeEncodingValue{
					set: false,
				},
				inStackTraceLevel: &stackLevelValue{
					set: false,
				},
				expected: &config{
					encoder: newJSONEncoder(),
					level:   zap.NewAtomicLevelAt(zapcore.Level(-10)),
					opts:    append(opts, zap.AddStacktrace(zap.WarnLevel)),
					sample:  false,
				}},
		},
		{
			fields: fields{name: "set time encoding",
				inDevel: false,
				inEncoder: &encoderValue{
					set: false,
				},
				inLevel: &levelValue{
					set: false,
				},
				inSample: &sampleValue{
					set: false,
				},
				inTimeEncoding: &timeEncodingValue{
					set:         true,
					timeEncoder: zapcore.EpochMillisTimeEncoder,
				},
				inStackTraceLevel: &stackLevelValue{
					set: false,
				},
				expected: &config{
					encoder: newJSONEncoder(withTimeEncoding(zapcore.EpochMillisTimeEncoder)),
					level:   zap.NewAtomicLevelAt(zap.InfoLevel),
					opts:    append(opts, zap.AddStacktrace(zap.WarnLevel)),
					sample:  true,
				}},
		},
		{
			name: "set stacktrace generation on 'panic' level only",
			fields: fields{
				inDevel: false,
				inEncoder: &encoderValue{
					set: false,
				},
				inLevel: &levelValue{
					set:   true,
					level: zapcore.Level(-10),
				},
				inSample: &sampleValue{
					set:    true,
					sample: true,
				},
				inTimeEncoding: &timeEncodingValue{
					set: false,
				},
				inStackTraceLevel: &stackLevelValue{
					set:   true,
					level: zapcore.Level(zapcore.PanicLevel),
				},
				expected: &config{
					encoder: newJSONEncoder(),
					level:   zap.NewAtomicLevelAt(zapcore.Level(-10)),
					opts:    append(opts, zap.AddStacktrace(zap.PanicLevel)),
					sample:  false,
				}},
		},
	}

	entry := zapcore.Entry{
		Level:      levelVal.level,
		Time:       time.Now(),
		LoggerName: "TestLogger",
		Message:    "Test message",
		Caller: zapcore.EntryCaller{
			Defined: true,
			File:    "dummy_file.go",
			Line:    10,
		},
		Stack: "Sample stack",
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			development = tc.fields.inDevel
			encoderVal = *tc.fields.inEncoder
			levelVal = *tc.fields.inLevel
			sampleVal = *tc.fields.inSample
			timeEncodingVal = *tc.fields.inTimeEncoding
			stacktraceLevel = *tc.fields.inStackTraceLevel

			cfg := getConfig()
			assert.Equal(t, tc.fields.expected.level, cfg.level)
			assert.Equal(t, len(tc.fields.expected.opts), len(cfg.opts))
			assert.Equal(t, tc.fields.expected.sample, cfg.sample)

			// Test that the encoder returned by getConfig encodes an entry
			// the same way that the expected encoder does. In addition to
			// testing that the correct entry encoding (json vs. console) is
			// used, this also tests that the correct time encoding is used.
			expectedEncoderOut, err := tc.fields.expected.encoder.EncodeEntry(entry, []zapcore.Field{{Key: "fieldKey",
				Type: zapcore.StringType, String: "fieldValue"}})
			if err != nil {
				t.Fatalf("Unexpected error encoding entry with expected encoder: %s", err)
			}
			actualEncoderOut, err := cfg.encoder.EncodeEntry(entry, []zapcore.Field{{Key: "fieldKey",
				Type: zapcore.StringType, String: "fieldValue"}})
			if err != nil {
				t.Fatalf("Unexpected error encoding entry with actual encoder: %s", err)
			}
			assert.Equal(t, expectedEncoderOut.String(), actualEncoderOut.String())

			// This test helps ensure that we disable sampling for verbose log
			// levels. Logging at V(10) should never panic, which would happen
			// if sampling is enabled at this level.
			assert.NotPanics(t, func() {
				out := &bytes.Buffer{}
				dalog := createLogger(cfg, out)
				dalog.V(10).Info("This should not panic")
			})
		})
	}
}
