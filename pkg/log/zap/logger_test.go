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
	"io"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/assert"
)

func TestGetConfig(t *testing.T) {
	var opts []zap.Option

	testCases := []struct {
		name      string
		inDevel   bool
		inEncoder encoderValue
		inLevel   levelValue
		inSample  sampleValue
		expected  config
	}{
		{
			name:    "development on",
			inDevel: true,
			inEncoder: encoderValue{
				set: false,
			},
			inLevel: levelValue{
				set: false,
			},
			inSample: sampleValue{
				set: false,
			},
			expected: config{
				encoder: consoleEncoder(),
				level:   zap.NewAtomicLevelAt(zap.DebugLevel),
				opts:    append(opts, zap.Development(), zap.AddStacktrace(zap.ErrorLevel)),
				sample:  false,
			},
		},
		{
			name:    "development off",
			inDevel: false,
			inEncoder: encoderValue{
				set: false,
			},
			inLevel: levelValue{
				set: false,
			},
			inSample: sampleValue{
				set: false,
			},
			expected: config{
				encoder: jsonEncoder(),
				level:   zap.NewAtomicLevelAt(zap.InfoLevel),
				opts:    append(opts, zap.AddStacktrace(zap.WarnLevel)),
				sample:  true,
			},
		},
		{
			name:    "set encoder",
			inDevel: false,
			inEncoder: encoderValue{
				set:     true,
				encoder: consoleEncoder(),
			},
			inLevel: levelValue{
				set: false,
			},
			inSample: sampleValue{
				set: false,
			},
			expected: config{
				encoder: jsonEncoder(),
				level:   zap.NewAtomicLevelAt(zap.InfoLevel),
				opts:    append(opts, zap.AddStacktrace(zap.WarnLevel)),
				sample:  true,
			},
		},
		{
			name:    "set level using level constant",
			inDevel: false,
			inEncoder: encoderValue{
				set: false,
			},
			inLevel: levelValue{
				set:   true,
				level: zapcore.ErrorLevel,
			},
			inSample: sampleValue{
				set: false,
			},
			expected: config{
				encoder: jsonEncoder(),
				level:   zap.NewAtomicLevelAt(zap.ErrorLevel),
				opts:    append(opts, zap.AddStacktrace(zap.WarnLevel)),
				sample:  true,
			},
		},
		{
			name:    "set level using custom level",
			inDevel: false,
			inEncoder: encoderValue{
				set: false,
			},
			inLevel: levelValue{
				set:   true,
				level: zapcore.Level(-10),
			},
			inSample: sampleValue{
				set: false,
			},
			expected: config{
				encoder: jsonEncoder(),
				level:   zap.NewAtomicLevelAt(zapcore.Level(-10)),
				opts:    append(opts, zap.AddStacktrace(zap.WarnLevel)),
				sample:  false,
			},
		},
		{
			name:    "set level using custom level, sample override not possible",
			inDevel: false,
			inEncoder: encoderValue{
				set: false,
			},
			inLevel: levelValue{
				set:   true,
				level: zapcore.Level(-10),
			},
			inSample: sampleValue{
				set:    true,
				sample: true,
			},
			expected: config{
				encoder: jsonEncoder(),
				level:   zap.NewAtomicLevelAt(zapcore.Level(-10)),
				opts:    append(opts, zap.AddStacktrace(zap.WarnLevel)),
				sample:  false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			development = tc.inDevel
			encoderVal = tc.inEncoder
			levelVal = tc.inLevel
			sampleVal = tc.inSample

			cfg := getConfig()
			assert.Equal(t, tc.expected.level, cfg.level)
			assert.Equal(t, len(tc.expected.opts), len(cfg.opts))
			assert.Equal(t, tc.expected.sample, cfg.sample)

			dalog := createLogger(cfg, os.Stderr)
			dalog.V(10).Info("This should not panic")
		})
	}
}

func createLogger(cfg config, dest io.Writer) logr.Logger {
	syncer := zapcore.AddSync(dest)
	if cfg.sample {
		cfg.opts = append(cfg.opts, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewSampler(core, time.Second, 100, 100)
		}))
	}
	cfg.opts = append(cfg.opts, zap.AddCallerSkip(1), zap.ErrorOutput(syncer))
	log := zap.New(zapcore.NewCore(cfg.encoder, syncer, cfg.level))
	log = log.WithOptions(cfg.opts...)
	return zapr.NewLogger(log)
}
