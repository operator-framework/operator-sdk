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
	"io"
	"os"
	"time"

	"github.com/go-logr/zapr"
	"github.com/spf13/pflag"

	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"go.uber.org/zap"
)

type Factory struct {
	development  bool
	encoderValue encoderValue
	levelValue   levelValue
	sampleValue  sampleValue
}

func (f Factory) Logger() logr.Logger {
	return f.LoggerTo(os.Stderr)
}

func (f Factory) LoggerTo(destWriter io.Writer) logr.Logger {
	sink := zapcore.AddSync(destWriter)
	conf := f.getConfig(destWriter)

	conf.encoder = &logf.KubeAwareEncoder{Encoder: conf.encoder, Verbose: conf.level.Level() < 0}
	if conf.sample {
		conf.opts = append(conf.opts, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewSampler(core, time.Second, 100, 100)
		}))
	}
	conf.opts = append(conf.opts, zap.AddCallerSkip(1), zap.ErrorOutput(sink))
	log := zap.New(zapcore.NewCore(conf.encoder, sink, conf.level))
	log = log.WithOptions(conf.opts...)
	return zapr.NewLogger(log)
}

type config struct {
	encoder zapcore.Encoder
	level   zap.AtomicLevel
	sample  bool
	opts    []zap.Option
}

func (f *Factory) getConfig(destWriter io.Writer) config {
	var c config

	// Set the defaults depending on the log mode (development vs. production)
	if f.development {
		c.encoder = consoleEncoder()
		c.level = zap.NewAtomicLevelAt(zap.DebugLevel)
		c.opts = append(c.opts, zap.Development(), zap.AddStacktrace(zap.ErrorLevel))
		c.sample = false
	} else {
		c.encoder = jsonEncoder()
		c.level = zap.NewAtomicLevelAt(zap.InfoLevel)
		c.opts = append(c.opts, zap.AddStacktrace(zap.WarnLevel))
		c.sample = true
	}

	// Override the defaults if the flags were set explicitly on the command line
	if f.encoderValue.set {
		c.encoder = f.encoderValue.encoder
	}
	if f.levelValue.set {
		c.level = zap.NewAtomicLevelAt(f.levelValue.level)
	}
	if f.sampleValue.set {
		c.sample = f.sampleValue.sample
	}

	return c
}

func FactoryForFlags(flagSet *pflag.FlagSet) *Factory {
	f := &Factory{}
	flagSet.AddFlagSet(f.FlagSet())
	return f
}
