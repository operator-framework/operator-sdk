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
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/klog"
)

var (
	zapFlagSet *pflag.FlagSet

	development     bool
	encoderVal      encoderValue
	levelVal        levelValue
	sampleVal       sampleValue
	timeEncodingVal timeEncodingValue
	stacktraceLevel stackLevelValue
)

func init() {
	zapFlagSet = pflag.NewFlagSet("zap", pflag.ExitOnError)
	zapFlagSet.BoolVar(&development, "zap-devel", false, "Enable zap development mode"+
		" (changes defaults to console encoder, debug log level, disables sampling and stacktrace from 'warning' level)")
	zapFlagSet.Var(&encoderVal, "zap-encoder", "Zap log encoding ('json' or 'console')")
	zapFlagSet.Var(&levelVal, "zap-level", "Zap log level (one of 'debug', 'info', 'error' or any integer value > 0)")
	zapFlagSet.Var(&sampleVal, "zap-sample",
		"Enable zap log sampling. Sampling will be disabled for integer log levels > 1")
	zapFlagSet.Var(&timeEncodingVal, "zap-time-encoding",
		"Sets the zap time format ('epoch', 'millis', 'nano', or 'iso8601')")
	zapFlagSet.Var(&stacktraceLevel, "zap-stacktrace-level",
		"Set the minimum log level that triggers stacktrace generation")
}

// FlagSet - The zap logging flagset.
func FlagSet() *pflag.FlagSet {
	return zapFlagSet
}

type encoderConfigFunc func(*zapcore.EncoderConfig)

type encoderValue struct {
	set        bool
	newEncoder func(...encoderConfigFunc) zapcore.Encoder
	str        string
}

func (v *encoderValue) Set(e string) error {
	v.set = true
	switch e {
	case "json":
		v.newEncoder = newJSONEncoder
	case "console":
		v.newEncoder = newConsoleEncoder
	default:
		return fmt.Errorf("unknown encoder \"%s\"", e)
	}
	v.str = e
	return nil
}

func (v encoderValue) String() string {
	return v.str
}

func (v encoderValue) Type() string {
	return "encoder"
}

func newJSONEncoder(ecfs ...encoderConfigFunc) zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	for _, f := range ecfs {
		f(&encoderConfig)
	}
	return zapcore.NewJSONEncoder(encoderConfig)
}

func newConsoleEncoder(ecfs ...encoderConfigFunc) zapcore.Encoder {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	for _, f := range ecfs {
		f(&encoderConfig)
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

type levelValue struct {
	set   bool
	level zapcore.Level
}

func (v *levelValue) Set(l string) error {
	v.set = true
	lvl, err := intLogLevel(l)
	if err != nil {
		return err
	}

	v.level = zapcore.Level(int8(lvl))
	// If log level is greater than debug, set glog/klog level to that level.
	if lvl < -3 {
		fs := flag.NewFlagSet("", flag.ContinueOnError)
		klog.InitFlags(fs)
		err := fs.Set("v", fmt.Sprintf("%v", -1*lvl))
		if err != nil {
			return err
		}
	}
	return nil
}

func (v levelValue) String() string {
	return v.level.String()
}

func (v levelValue) Type() string {
	return "level"
}

type stackLevelValue struct {
	set   bool
	level zapcore.Level
}

func (v *stackLevelValue) Set(l string) error {
	v.set = true
	lvl, err := intLogLevel(l)
	if err != nil {
		return err
	}

	v.level = zapcore.Level(int8(lvl))
	return nil
}

func (v stackLevelValue) String() string {
	if v.set {
		return v.level.String()
	}

	return "error"
}

func (v stackLevelValue) Type() string {
	return "level"
}

func intLogLevel(l string) (int, error) {
	lower := strings.ToLower(l)
	var lvl int
	switch lower {
	case "debug":
		lvl = -1
	case "info":
		lvl = 0
	case "error":
		lvl = 2
	default:
		i, err := strconv.Atoi(lower)
		if err != nil {
			return lvl, fmt.Errorf("invalid log level \"%s\"", l)
		}

		if i > 0 {
			lvl = -1 * i
		} else {
			return lvl, fmt.Errorf("invalid log level \"%s\"", l)
		}
	}
	return lvl, nil
}

type sampleValue struct {
	set    bool
	sample bool
}

func (v *sampleValue) Set(s string) error {
	var err error
	v.set = true
	v.sample, err = strconv.ParseBool(s)
	return err
}

func (v sampleValue) String() string {
	return strconv.FormatBool(v.sample)
}

func (v sampleValue) IsBoolFlag() bool {
	return true
}

func (v sampleValue) Type() string {
	return "sample"
}

type timeEncodingValue struct {
	set         bool
	timeEncoder zapcore.TimeEncoder
	str         string
}

func (v *timeEncodingValue) Set(s string) error {
	v.set = true

	// As of zap v1.9.1, UnmarshalText does not return an error. Instead, it
	// uses the epoch time encoding when unknown strings are unmarshalled.
	//
	// Set s to "epoch" if it doesn't match one of the known formats, so that
	// it aligns with the default time encoder function.
	//
	// TODO: remove this entire switch statement if UnmarshalText is ever
	// refactored to return an error.
	switch s {
	case "iso8601", "ISO8601", "millis", "nanos":
	default:
		s = "epoch"
	}

	v.str = s
	return v.timeEncoder.UnmarshalText([]byte(s))
}

func (v timeEncodingValue) String() string {
	return v.str
}

func (v timeEncodingValue) IsBoolFlag() bool {
	return false
}

func (v timeEncodingValue) Type() string {
	return "timeEncoding"
}

func withTimeEncoding(te zapcore.TimeEncoder) encoderConfigFunc {
	return func(ec *zapcore.EncoderConfig) {
		ec.EncodeTime = te
	}
}
