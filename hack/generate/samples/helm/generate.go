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

package helm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/pkg"
	"github.com/operator-framework/operator-sdk/testutils/command"
	"github.com/operator-framework/operator-sdk/testutils/sample"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GenerateMemcachedSamples(binaryPath, rootPath, helmChartPath string) []sample.Sample {
	bundleImageBase := "bundle"

	helmCC := command.NewGenericCommandContext(
		command.WithEnv("GO111MODULE=on", "KUBECONFIG=broken_so_we_generate_static_default_rules"),
		command.WithDir(filepath.Join(rootPath, "helm")),
	)

	memcachedGVK := schema.GroupVersionKind{
		Group:   "cache",
		Version: "v1alpha1",
		Kind:    "Memcached",
	}

	helmMemcached := sample.NewGenericSample(
		sample.WithBinary(binaryPath),
		sample.WithCommandContext(helmCC),
		sample.WithDomain("example.com"),
		sample.WithPlugins("helm"),
		sample.WithGvk(memcachedGVK),
		sample.WithExtraApiOptions("--helm-chart", helmChartPath),
		sample.WithName("memcached-operator"),
	)

	// remove sample directory if it already exists
	err := os.RemoveAll(helmMemcached.Dir())
	pkg.CheckError("attempting to remove sample dir", err)

	gen := sample.NewGenerator(
		sample.WithNoWebhook(),
	)

	err = gen.GenerateSamples(helmMemcached)
	pkg.CheckError("generating helm samples", err)

	ImplementMemcached(helmMemcached, fmt.Sprintf("%s-%s", bundleImageBase, helmMemcached.Name()))

	return []sample.Sample{helmMemcached}
}
