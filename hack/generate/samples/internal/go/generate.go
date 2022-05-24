// Copyright 2020 The Operator-SDK Authors
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

package golang

import (
	"fmt"
	"os"
	"path/filepath"

	golangv2 "github.com/operator-framework/operator-sdk/hack/generate/samples/internal/go/v2"
	golangv3 "github.com/operator-framework/operator-sdk/hack/generate/samples/internal/go/v3"
	"github.com/operator-framework/operator-sdk/hack/generate/samples/internal/pkg"
	"github.com/operator-framework/operator-sdk/testutils/command"
	"github.com/operator-framework/operator-sdk/testutils/sample"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GenerateMemcachedSamples(binaryPath, rootPath string) {
	golangv2.GenerateMemcachedSample(binaryPath, filepath.Join(rootPath, "go", "v2"))

	bundleImageBase := "bundle"

	goV3CC := command.NewGenericCommandContext(
		command.WithEnv("GO111MODULE=on"),
		command.WithDir(filepath.Join(rootPath, "go", "v3")),
	)

	memcachedGVK := schema.GroupVersionKind{
		Group:   "cache",
		Version: "v1alpha1",
		Kind:    "Memcached",
	}

	goV3Memcached := sample.NewGenericSample(
		sample.WithBinary(binaryPath),
		sample.WithCommandContext(goV3CC),
		sample.WithDomain("example.com"),
		sample.WithGvk(memcachedGVK),
		sample.WithPlugins("go/v3"),
		sample.WithRepository("github.com/example/memcached-operator"),
		sample.WithExtraInitOptions("--project-version", "3"),
		sample.WithExtraApiOptions("--controller", "--resource"),
		sample.WithExtraWebhookOptions("--defaulting"),
		sample.WithName("gov3-memcached-operator"),
	)

	// remove sample directory if it already exists
	err := os.RemoveAll(goV3Memcached.Dir())
	pkg.CheckError("attempting to remove sample dir", err)

	gen := sample.NewGenerator()

	err = gen.GenerateSamples(goV3Memcached)
	pkg.CheckError("generating go/v3 samples", err)

	// Perform implementation logic
	golangv3.ImplementMemcached(goV3Memcached, fmt.Sprintf("%s-%s", bundleImageBase, goV3Memcached.Name()))
}
