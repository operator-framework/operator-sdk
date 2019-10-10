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
	v1 "k8s.io/api/core/v1"
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

type BasicAndOLMPluginConfig struct {
	Namespace          string        `mapstructure:"namespace"`
	Kubeconfig         string        `mapstructure:"kubeconfig"`
	InitTimeout        int           `mapstructure:"init-timeout"`
	OLMDeployed        bool          `mapstructure:"olm-deployed"`
	NamespacedManifest string        `mapstructure:"namespaced-manifest"`
	GlobalManifest     string        `mapstructure:"global-manifest"`
	CRManifest         []string      `mapstructure:"cr-manifest"`
	CSVManifest        string        `mapstructure:"csv-path"`
	ProxyImage         string        `mapstructure:"proxy-image"`
	ProxyPullPolicy    v1.PullPolicy `mapstructure:"proxy-pull-policy"`
	CRDsDir            string        `mapstructure:"crds-dir"`
	DeployDir          string        `mapstructure:"deploy-dir"`
}

func validateScorecardPluginFlags(config BasicAndOLMPluginConfig, pluginType PluginType) error {
	if !config.OLMDeployed && len(config.CRManifest) == 0 {
		return errors.New("cr-manifest config option must be set")
	}
	if pluginType == OLMIntegration && config.CSVManifest == "" {
		return fmt.Errorf("csv-path must be set if olm-tests is enabled")
	}
	if config.OLMDeployed && config.CSVManifest == "" {
		return fmt.Errorf("csv-path must be set if olm-deployed is enabled")
	}
	pullPolicy := config.ProxyPullPolicy
	if pullPolicy != v1.PullAlways && pullPolicy != v1.PullNever && pullPolicy != v1.PullIfNotPresent {
		return fmt.Errorf("invalid proxy pull policy: (%s); valid values: %s, %s, %s", pullPolicy, v1.PullAlways, v1.PullNever, v1.PullIfNotPresent)
	}
	return nil
}
