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

// Adapted from https://github.com/operator-framework/operator-registry/blob/v1.17.1/test/e2e/e2e_suite_test.go
package integration

import (
	"os"

	"github.com/operator-framework/operator-sdk/internal/testutils"
)

const (
	defaultLocalRegistryName     = "localhost"
	defaultInClusterRegistryName = "kind-registry"
)

var (
	localRegistryHost     = os.Getenv("DOCKER_REGISTRY_LOCAL_HOST")
	inClusterRegistryHost = os.Getenv("DOCKER_REGISTRY_IN_CLUSTER_HOST")

	registryPort = "443"
)

func configureRegistry(testutils.TestContext) func() {
	if localRegistryHost == "" {
		localRegistryHost = defaultLocalRegistryName + ":" + registryPort
	}
	if inClusterRegistryHost == "" {
		inClusterRegistryHost = defaultInClusterRegistryName + ":" + registryPort
	}

	return func() {
	}
}
