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
	"fmt"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/internal/pkg"
	"github.com/operator-framework/operator-sdk/testutils/command"
	"github.com/operator-framework/operator-sdk/testutils/sample"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const bundleImageBase = "bundle"

var memcachedGVK = schema.GroupVersionKind{
	Group:   "cache",
	Version: "v1alpha1",
	Kind:    "Memcached",
}

func GenerateMemcachedSamples(binaryPath, rootPath string) {
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
		sample.WithName("ansible-memcached-operator"),
	)

	// remove sample directory if it already exists
	err := os.RemoveAll(ansibleMemcached.Dir())
	pkg.CheckError("attempting to remove sample dir", err)

	gen := sample.NewGenerator(
		sample.WithNoWebhook(),
	)

	err = gen.GenerateSamples(ansibleMemcached)
	pkg.CheckError("generating ansible samples", err)

	ImplementMemcached(ansibleMemcached, fmt.Sprintf("%s-%s", bundleImageBase, ansibleMemcached.Name()))
}

// GenerateMoleculeSample will call all actions to create the directory and generate the sample
// The Context to run the samples are not the same in the e2e test. In this way, note that it should NOT
// be called in the e2e tests since it will call the Prepare() to set the sample context and generate the files
// in the testdata directory. The e2e tests only ought to use the Run() method with the TestContext.
func GenerateMoleculeSample(binaryPath, samplesPath string) {
	ansibleCC := command.NewGenericCommandContext(
		command.WithEnv("GO111MODULE=on"),
		command.WithDir(filepath.Join(samplesPath, "molecule-operator")),
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
			schema.GroupVersionKind{
				Group:   "ignore",
				Version: "v1",
				Kind:    "Secret",
			},
		),
		sample.WithPlugins("ansible"),
		sample.WithExtraApiOptions("--generate-role", "--generate-playbook"),
		sample.WithName("ansible-molecule-memcached-operator"),
	)

	// remove sample directory if it already exists
	err := os.RemoveAll(ansibleMoleculeMemcached.Dir())
	pkg.CheckError("attempting to remove sample dir", err)

	gen := sample.NewGenerator(sample.WithNoWebhook())

	err = gen.GenerateSamples(ansibleMoleculeMemcached)

	pkg.CheckError("generating ansible molecule sample", err)

	ImplementMemcachedMolecule(ansibleMoleculeMemcached, fmt.Sprintf("%s-%s", bundleImageBase, ansibleMoleculeMemcached.Name()))
}

// GenerateAdvancedMoleculeSample will call all actions to create the directory and generate the sample
// The Context to run the samples are not the same in the e2e test. In this way, note that it should NOT
// be called in the e2e tests since it will call the Prepare() to set the sample context and generate the files
// in the testdata directory. The e2e tests only ought to use the Run() method with the TestContext.
func GenerateAdvancedMoleculeSample(binaryPath, samplesPath string) {
	ansibleCC := command.NewGenericCommandContext(
		command.WithEnv("GO111MODULE=on"),
		command.WithDir(filepath.Join(samplesPath, "advanced-molecule-operator")),
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
		sample.WithName("ansible-advanced-molecule-memcached-operator"),
	)

	// remove sample directory if it already exists
	err := os.RemoveAll(advancedMoleculeMemcached.Dir())
	pkg.CheckError("attempting to remove sample dir", err)

	gen := sample.NewGenerator(sample.WithNoWebhook())

	err = gen.GenerateSamples(advancedMoleculeMemcached)
	pkg.CheckError("generating ansible advanced molecule sample", err)

	ImplementAdvancedMolecule(advancedMoleculeMemcached, fmt.Sprintf("%s-%s", bundleImageBase, advancedMoleculeMemcached.Name()))
}
