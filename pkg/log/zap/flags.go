// Copyright 2018 The Operator-SDK Authors
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
	"fmt"
	"strconv"

	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func (f *Factory) FlagSet() *pflag.FlagSet {
	flagSet := pflag.NewFlagSet("zap", pflag.ExitOnError)
	flagSet.BoolVar(&f.development, "zap-devel", false, "Enable zap development mode (changes defaults to console encoder, debug log level, and disables sampling)")
	flagSet.Var(&f.encoderValue, "zap-encoder", "Zap log encoding ('json' or 'console')")
	flagSet.Var(&f.levelValue, "zap-level", "Zap log level (one of 'debug', 'info', 'warn', 'error', 'dpanic', 'panic', 'fatal')")
	flagSet.Var(&f.sampleValue, "zap-sample", "Enable zap log sampling")
	return flagSet
}

type encoderValue struct {
	set     bool
	encoder zapcore.Encoder
	str     string
}

func (v *encoderValue) Set(e string) error {
	v.set = true
	switch e {
	case "json":
		v.encoder = jsonEncoder()
	case "console":
		v.encoder = consoleEncoder()
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

func jsonEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	return zapcore.NewJSONEncoder(encoderConfig)
}

func consoleEncoder() zapcore.Encoder {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	return zapcore.NewConsoleEncoder(encoderConfig)
}

type levelValue struct {
	set   bool
	level zapcore.Level
}

func (v *levelValue) Set(l string) error {
	v.set = true
	return v.level.Set(l)
}

func (v levelValue) String() string {
	return v.level.String()
}

func (v levelValue) Type() string {
	return "level"
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
