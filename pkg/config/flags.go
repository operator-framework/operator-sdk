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

package config

import (
	"log"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// global command-line flags
const (
	RepoOpt    = "repo"
	VerboseOpt = "verbose"
	ConfigOpt  = "config"

	DeployDirOpt     = "deploy-dir"
	CRDsDirOpt       = "crds-dir"
	APIsDirOpt       = "apis-dir"
	OLMCatalogDirOpt = "olm-catalog-dir"
)

type pflagValue struct {
	flag *pflag.Flag
}

func (p pflagValue) HasChanged() bool    { return p.flag.Changed }
func (p pflagValue) Name() string        { return p.flag.Name }
func (p pflagValue) ValueString() string { return p.flag.Value.String() }
func (p pflagValue) ValueType() string   { return p.flag.Value.Type() }

func BindFlagWithPrefix(f *pflag.Flag, prefix string) {
	err := viper.BindFlagValue(prefix+"."+f.Name, pflagValue{f})
	if err != nil {
		log.Fatalf("Failed to bind flag %s to viper: %v", f.Name, err)
	}
}

func BindFlagsWithPrefix(flagSet *pflag.FlagSet, prefix string) {
	flagSet.VisitAll(func(f *pflag.Flag) {
		BindFlagWithPrefix(f, prefix)
	})
}
