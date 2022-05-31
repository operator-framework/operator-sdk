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

package ansible

import (
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/pkg"
	"github.com/operator-framework/operator-sdk/testutils/command"
	"github.com/operator-framework/operator-sdk/testutils/e2e"
	"github.com/operator-framework/operator-sdk/testutils/sample"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const bundleImage = "quay.io/example/memcached-operator:v0.0.1"

var memcachedGVK = schema.GroupVersionKind{
	Group:   "cache",
	Version: "v1alpha1",
	Kind:    "Memcached",
}

func GenerateMemcachedSamples(binaryPath, rootPath string) []sample.Sample {
	ansibleCC := command.NewGenericCommandContext(
		command.WithEnv("GO111MODULE=on"),
		command.WithDir(filepath.Join(rootPath, "ansible")),
	)

	ansibleMemcached := sample.NewGenericSample(
		sample.WithBinary(binaryPath),
		sample.WithCommandContext(ansibleCC),
		sample.WithDomain("example.com"),
		sample.WithGvk(memcachedGVK),
		sample.WithPlugins("ansible"),
		sample.WithExtraApiOptions("--generate-role", "--generate-playbook"),
		sample.WithName("memcached-operator"),
	)

	// remove sample directory if it already exists
	err := os.RemoveAll(ansibleMemcached.Dir())
	pkg.CheckError("attempting to remove sample dir", err)

	gen := sample.NewGenerator(
		sample.WithNoWebhook(),
	)

	err = gen.GenerateSamples(ansibleMemcached)
	pkg.CheckError("generating ansible samples", err)

	ImplementMemcached(ansibleMemcached, bundleImage)
	return []sample.Sample{ansibleMemcached}
}

// GenerateMoleculeSample will call all actions to create the directory and generate the sample
// The Context to run the samples are not the same in the e2e test. In this way, note that it should NOT
// be called in the e2e tests since it will call the Prepare() to set the sample context and generate the files
// in the testdata directory. The e2e tests only ought to use the Run() method with the TestContext.
func GenerateMoleculeSample(binaryPath, samplesPath string) {
	ansibleCC := command.NewGenericCommandContext(
		command.WithEnv("GO111MODULE=on"),
		command.WithDir(filepath.Join(samplesPath, "")),
	)

	ansibleMoleculeMemcached := sample.NewGenericSample(
		sample.WithBinary(binaryPath),
		sample.WithCommandContext(ansibleCC),
		sample.WithDomain("example.com"),
		sample.WithGvk(
			memcachedGVK,
			schema.GroupVersionKind{
				Group:   memcachedGVK.Group,
				Version: memcachedGVK.Version,
				Kind:    "Foo",
			},
			schema.GroupVersionKind{
				Group:   memcachedGVK.Group,
				Version: memcachedGVK.Version,
				Kind:    "Memfin",
			},
		),
		sample.WithPlugins("ansible"),
		sample.WithExtraApiOptions("--generate-role", "--generate-playbook"),
		sample.WithName("memcached-molecule-operator"),
	)

	addIgnore := sample.NewGenericSample(
		sample.WithBinary(ansibleMoleculeMemcached.Binary()),
		sample.WithCommandContext(ansibleMoleculeMemcached.CommandContext()),
		sample.WithName(ansibleMoleculeMemcached.Name()),
		sample.WithPlugins("ansible"),
		sample.WithGvk(schema.GroupVersionKind{
			Group:   "ignore",
			Version: "v1",
			Kind:    "Secret",
		}),
		sample.WithExtraApiOptions("--generate-role"),
	)

	// remove sample directory if it already exists
	err := os.RemoveAll(ansibleMoleculeMemcached.Dir())
	pkg.CheckError("attempting to remove sample dir", err)

	gen := sample.NewGenerator(
		sample.WithNoWebhook(),
	)

	err = gen.GenerateSamples(ansibleMoleculeMemcached)
	pkg.CheckError("generating ansible molecule sample", err)

	log.Infof("enabling multigroup support")
	err = e2e.AllowProjectBeMultiGroup(ansibleMoleculeMemcached)
	pkg.CheckError("updating PROJECT file", err)

	ignoreGen := sample.NewGenerator(sample.WithNoInit(), sample.WithNoWebhook())
	err = ignoreGen.GenerateSamples(addIgnore)
	pkg.CheckError("generating ansible molecule sample - ignore", err)

	ImplementMemcached(ansibleMoleculeMemcached, bundleImage)

	ImplementMemcachedMolecule(ansibleMoleculeMemcached, bundleImage)
}

// GenerateAdvancedMoleculeSample will call all actions to create the directory and generate the sample
// The Context to run the samples are not the same in the e2e test. In this way, note that it should NOT
// be called in the e2e tests since it will call the Prepare() to set the sample context and generate the files
// in the testdata directory. The e2e tests only ought to use the Run() method with the TestContext.
func GenerateAdvancedMoleculeSample(binaryPath, samplesPath string) {
	ansibleCC := command.NewGenericCommandContext(
		command.WithEnv("GO111MODULE=on"),
		command.WithDir(filepath.Join(samplesPath, "")),
	)

	gv := schema.GroupVersion{
		Group:   "test",
		Version: "v1alpha1",
	}

	kinds := []string{
		"ArgsTest",
		"CaseTest",
		"CollectionTest",
		"ClusterAnnotationTest",
		"FinalizerConcurrencyTest",
		"ReconciliationTest",
		"SelectorTest",
		"SubresourcesTest",
	}

	var gvks []schema.GroupVersionKind

	for _, kind := range kinds {
		gvks = append(gvks, schema.GroupVersionKind{Group: gv.Group, Version: gv.Version, Kind: kind})
	}

	advancedMoleculeMemcached := sample.NewGenericSample(
		sample.WithBinary(binaryPath),
		sample.WithCommandContext(ansibleCC),
		sample.WithDomain("example.com"),
		sample.WithGvk(gvks...),
		sample.WithPlugins("ansible"),
		sample.WithExtraInitOptions("--group", gv.Group, "--version", gv.Version, "--kind", "InventoryTest", "--generate-role", "--generate-playbook"),
		sample.WithExtraApiOptions("--generate-playbook"),
		sample.WithName("advanced-molecule-operator"),
	)

	// remove sample directory if it already exists
	err := os.RemoveAll(advancedMoleculeMemcached.Dir())
	pkg.CheckError("attempting to remove sample dir", err)

	gen := sample.NewGenerator(sample.WithNoWebhook())

	err = gen.GenerateSamples(advancedMoleculeMemcached)
	pkg.CheckError("generating ansible advanced molecule sample", err)

	ImplementAdvancedMolecule(advancedMoleculeMemcached, bundleImage)
}
