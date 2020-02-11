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

// Code adapted from:
// https://github.com/kubernetes-sigs/controller-tools/blob/6eef398/cmd/controller-gen/main.go
package gen

import (
	"fmt"
	"sync"

	"github.com/spf13/afero"
	crdgen "sigs.k8s.io/controller-tools/pkg/crd"
	"sigs.k8s.io/controller-tools/pkg/genall"
	"sigs.k8s.io/controller-tools/pkg/markers"
)

// Generator can generate artifacts using data contained in the Generator.
type Generator interface {
	// Generate invokes the Generator, usually writing a file to disk or memory
	// depending on what output rules are set.
	Generate() error
}

// Runner runs a generator.
type Runner interface {
	// AddOutputRule associates an OutputRule with a definition name in the
	// Runner's generator.
	AddOutputRule(string, genall.OutputRule)
	// Run creates a generator runtime by passing in rawOpts, a raw set of option
	// strings, and invokes the runtime.
	Run([]string) error
}

// cache contains all generated files from calls to cachedRunner.Run().
// A global cache is necessary because individual output rules that use
// an in-memory filesystem set by the caller cannot propagate that filesystem
// instance to the underlying generator runtime. Only type information is
// propagated; all other fields are set by the runtime's parser. cache can
// be used by output rules defined here to access generated artifacts.
var cache afero.Fs

func init() {
	cache = afero.NewMemMapFs()
}

// GetCache gets the singleton cache instance.
func GetCache() afero.Fs {
	return cache
}

// cachedRunner contains a set of pre-defined markers for generators and
// output rules from controller-tools used to configure a generator runtime.
// cachedRunner caches each file generated in a global cache, to both
// speed up generation for commands and work around the output rule system.
type cachedRunner struct {
	// optionsRegistry contains all the marker definitions used to process
	// option strings.
	optionsRegistry *markers.Registry
	// allGenerators maintains the list of all known generators.
	allGenerators map[string]genall.Generator
	// allOutputRules defines the list of all known output rules.
	// Each output rule turns into two command line options:
	// - output:<generator>:<form> (per-generator output)
	// - output:<form> (default output)
	allOutputRules map[string]genall.OutputRule

	once sync.Once
}

// For lazy initialization.
func (g *cachedRunner) init() {
	g.once.Do(func() {
		for genName, gen := range g.allGenerators {
			// make the generator options marker itself
			defn := markers.Must(markers.MakeDefinition(genName, markers.DescribesPackage, gen))
			if err := g.optionsRegistry.Register(defn); err != nil {
				panic(err)
			}
			// make per-generation output rule markers
			for ruleName, rule := range g.allOutputRules {
				ruleMarker := markers.Must(markers.MakeDefinition(fmt.Sprintf("output:%s:%s", genName,
					ruleName), markers.DescribesPackage, rule))
				if err := g.optionsRegistry.Register(ruleMarker); err != nil {
					panic(err)
				}
			}
		}
		// make "default output" output rule markers
		for ruleName, rule := range g.allOutputRules {
			ruleMarker := markers.Must(markers.MakeDefinition("output:"+ruleName, markers.DescribesPackage, rule))
			if err := g.optionsRegistry.Register(ruleMarker); err != nil {
				panic(err)
			}
		}
		// add in the common options markers
		if err := genall.RegisterOptionsMarkers(g.optionsRegistry); err != nil {
			panic(err)
		}
	})
}

// NewCachedRunner returns a cachedRunner with a set of default
// generators and output rules. The returned cachedRunner is lazily
// initialized.
func NewCachedRunner() Runner {
	return &cachedRunner{
		optionsRegistry: &markers.Registry{},
		allGenerators: map[string]genall.Generator{
			"crd": crdgen.Generator{},
		},
		allOutputRules: map[string]genall.OutputRule{
			"dir": genall.OutputToDirectory(""),
		},
	}
}

// AddOutputRule adds a output rule definition to g's options registry.
func (g *cachedRunner) AddOutputRule(defName string, rule genall.OutputRule) {
	ruleMarker := markers.Must(markers.MakeDefinition(defName, markers.DescribesPackage, rule))
	if err := g.optionsRegistry.Register(ruleMarker); err != nil {
		panic(err)
	}
}

// Run creates a generator runtime by passing in rawOpts, a raw set of option
// strings, and invokes the runtime. rawOpts must contain CLI options that
// can be consumed by controller-gen. Note that only a subset of generators
// and rules that controller-gen supports are implemented.
func (g *cachedRunner) Run(rawOpts []string) error {
	g.init()
	rt, err := genall.FromOptions(g.optionsRegistry, rawOpts)
	if err != nil {
		return err
	}
	if len(rt.Generators) == 0 {
		return fmt.Errorf("no generators specified")
	}
	if hadErrs := rt.Run(); hadErrs {
		return fmt.Errorf("not all generators ran successfully")
	}
	return nil
}
