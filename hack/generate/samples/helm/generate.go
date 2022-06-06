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
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/pkg"
	"github.com/operator-framework/operator-sdk/internal/plugins"
	helmv1 "github.com/operator-framework/operator-sdk/internal/plugins/helm/v1"
	manifestsv2 "github.com/operator-framework/operator-sdk/internal/plugins/manifests/v2"
	scorecardv2 "github.com/operator-framework/operator-sdk/internal/plugins/scorecard/v2"
	"github.com/operator-framework/operator-sdk/testutils/command"
	"github.com/operator-framework/operator-sdk/testutils/sample"
	clisample "github.com/operator-framework/operator-sdk/testutils/sample/cli-sample"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kubebuilder/v3/pkg/cli"
	cfgv3 "sigs.k8s.io/kubebuilder/v3/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	kustomizev1 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v1"
)

func GenerateMemcachedSamples(binaryPath, rootPath, helmChartPath string) []sample.Sample {
	bundleImage := "quay.io/example/memcached-operator:v0.0.1"

	helmCC := command.NewGenericCommandContext(
		command.WithEnv("GO111MODULE=on", "KUBECONFIG=broken_so_we_generate_static_default_rules"),
		command.WithDir(filepath.Join(rootPath, "helm")),
	)

	memcachedGVK := schema.GroupVersionKind{
		Group:   "cache",
		Version: "v1alpha1",
		Kind:    "Memcached",
	}

	// Create a custom CLI to run the helm scaffolding
	helmBundle, _ := plugin.NewBundle("helm"+plugins.DefaultNameQualifier, plugin.Version{Number: 1},
		kustomizev1.Plugin{},
		helmv1.Plugin{},
		manifestsv2.Plugin{},
		scorecardv2.Plugin{},
	)
	helmCli, err := cli.New(
		cli.WithCommandName("helm-test-cli"),
		cli.WithVersion("v0.0.0"),
		cli.WithPlugins(
			helmBundle,
		),
		cli.WithDefaultPlugins(cfgv3.Version, helmBundle),
		cli.WithDefaultProjectVersion(cfgv3.Version),
		cli.WithCompletion(),
	)

	helmMemcached, err := clisample.NewCliSample(
		clisample.WithCLI(helmCli),
		clisample.WithCommandContext(helmCC),
		clisample.WithDomain("example.com"),
		clisample.WithPlugins("helm"),
		clisample.WithGvk(memcachedGVK),
		clisample.WithExtraApiOptions("--helm-chart", helmChartPath),
		clisample.WithName("memcached-operator"),
	)

	// remove sample directory if it already exists
	err = os.RemoveAll(helmMemcached.Dir())
	pkg.CheckError("attempting to remove sample dir", err)

	gen := sample.NewGenerator(
		sample.WithNoWebhook(),
	)

	err = gen.GenerateSamples(helmMemcached)
	pkg.CheckError("generating helm samples", err)

	ImplementMemcached(helmMemcached, bundleImage)

	return []sample.Sample{helmMemcached}
}
