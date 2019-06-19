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

package scplugins

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	NamespaceOpt          = "namespace"
	KubeconfigOpt         = "kubeconfig"
	InitTimeoutOpt        = "init-timeout"
	OlmDeployedOpt        = "olm-deployed"
	CSVPathOpt            = "csv-path"
	NamespacedManifestOpt = "namespaced-manifest"
	GlobalManifestOpt     = "global-manifest"
	CRManifestOpt         = "cr-manifest"
	ProxyImageOpt         = "proxy-image"
	ProxyPullPolicyOpt    = "proxy-pull-policy"
	CRDsDirOpt            = "crds-dir"
	DeployDirOpt          = "deploy-dir"
	BasicTestsOpt         = "basic-tests"
	OLMTestsOpt           = "olm-tests"
)

func validateScorecardPluginFlags(config *viper.Viper) error {
	if !config.GetBool(OlmDeployedOpt) && len(config.GetStringSlice(CRManifestOpt)) == 0 {
		return errors.New("cr-manifest config option must be set")
	}
	if !config.GetBool(BasicTestsOpt) && !config.GetBool(OLMTestsOpt) {
		return errors.New("at least one test type must be set")
	}
	if config.GetBool(OLMTestsOpt) && config.GetString(CSVPathOpt) == "" {
		return fmt.Errorf("csv-path must be set if olm-tests is enabled")
	}
	if config.GetBool(OlmDeployedOpt) && config.GetString(CSVPathOpt) == "" {
		return fmt.Errorf("csv-path must be set if olm-deployed is enabled")
	}
	pullPolicy := config.GetString(ProxyPullPolicyOpt)
	if pullPolicy != "Always" && pullPolicy != "Never" && pullPolicy != "PullIfNotPresent" {
		return fmt.Errorf("invalid proxy pull policy: (%s); valid values: Always, Never, PullIfNotPresent", pullPolicy)
	}
	return nil
}
