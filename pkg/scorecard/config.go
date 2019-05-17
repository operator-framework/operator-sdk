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

package scorecard

import (
	"errors"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	ScorecardConfigOpt = "scorecard-config"
	// scorecard-config keys.
	NamespaceOpt          = ScorecardConfigOpt + "." + "namespace"
	KubeconfigPathOpt     = ScorecardConfigOpt + "." + "kubeconfig"
	InitTimeoutOpt        = ScorecardConfigOpt + "." + "init-timeout"
	OLMDeployedOpt        = ScorecardConfigOpt + "." + "olm-deployed"
	CSVPathOpt            = ScorecardConfigOpt + "." + "csv-path"
	BasicTestsOpt         = ScorecardConfigOpt + "." + "basic-tests"
	OLMTestsOpt           = ScorecardConfigOpt + "." + "olm-tests"
	TenantTestsOpt        = ScorecardConfigOpt + "." + "good-tenant-tests"
	NamespacedManifestOpt = ScorecardConfigOpt + "." + "namespaced-manifest"
	GlobalManifestOpt     = ScorecardConfigOpt + "." + "global-manifest"
	CRManifestOpt         = ScorecardConfigOpt + "." + "cr-manifest"
	ProxyImageOpt         = ScorecardConfigOpt + "." + "proxy-image"
	ProxyPullPolicyOpt    = ScorecardConfigOpt + "." + "proxy-pull-policy"
	CRDsDirOpt            = ScorecardConfigOpt + "." + "crds-dir"
	OutputFormatOpt       = ScorecardConfigOpt + "." + "output"
	PluginDirOpt          = ScorecardConfigOpt + "." + "plugin-dir"
)

const (
	JSONOutputFormat          = "json"
	HumanReadableOutputFormat = "human-readable"
)

type ScorecardCmd struct {
	// General scorecard configuration.
	Namespace          string
	KubeconfigPath     string
	CSVPath            string
	NamespacedManifest string
	GlobalManifest     string
	ProxyImage         string
	ProxyPullPolicy    string
	CRDsDir            string
	OutputFormat       string
	PluginDir          string
	CRManifest         []string
	InitTimeout        int

	// Test types.
	OLMDeployed bool
	BasicTests  bool
	OLMTests    bool
	TenantTests bool

	// logReadWriter is configured before Run() is invoked, so it should be
	// configured during command setup. It is injected into a ResourceConfig
	// at runtime.
	logReadWriter io.ReadWriter
	// Logger for ScorecardCmd methods.
	log *logrus.Logger
}

func (c *ScorecardCmd) validateFlags() error {
	if !viper.GetBool(OLMDeployedOpt) && len(viper.GetStringSlice(CRManifestOpt)) == 0 {
		return errors.New("cr-manifest config option must be set")
	}
	if !viper.GetBool(BasicTestsOpt) && !viper.GetBool(OLMTestsOpt) {
		return errors.New("at least one test type must be set")
	}
	if viper.GetBool(OLMTestsOpt) && viper.GetString(CSVPathOpt) == "" {
		return fmt.Errorf("csv-path must be set if olm-tests is enabled")
	}
	if viper.GetBool(OLMDeployedOpt) && viper.GetString(CSVPathOpt) == "" {
		return fmt.Errorf("csv-path must be set if olm-deployed is enabled")
	}
	pullPolicy := viper.GetString(ProxyPullPolicyOpt)
	if pullPolicy != "Always" && pullPolicy != "Never" && pullPolicy != "PullIfNotPresent" {
		return fmt.Errorf("invalid proxy pull policy: (%s); valid values: Always, Never, PullIfNotPresent", pullPolicy)
	}
	// this is already being checked in configure logger; may be unnecessary
	outputFmt := viper.GetString(OutputFormatOpt)
	if outputFmt != HumanReadableOutputFormat && outputFmt != JSONOutputFormat {
		return fmt.Errorf("invalid output format (%s); valid values: human-readable, json", outputFmt)
	}
	return nil
}

func (c *ScorecardCmd) setInGlobal() error {
	if c.Namespace != "" {
		viper.Set(NamespaceOpt, c.Namespace)
	}
	if c.KubeconfigPath != "" {
		viper.Set(KubeconfigPathOpt, c.KubeconfigPath)
	}
	if c.CSVPath != "" {
		viper.Set(CSVPathOpt, c.CSVPath)
	}
	if c.NamespacedManifest != "" {
		viper.Set(NamespacedManifestOpt, c.NamespacedManifest)
	}
	if c.GlobalManifest != "" {
		viper.Set(GlobalManifestOpt, c.GlobalManifest)
	}
	if c.ProxyImage != "" {
		viper.Set(ProxyImageOpt, c.ProxyImage)
	}
	if c.ProxyPullPolicy != "" {
		viper.Set(ProxyPullPolicyOpt, c.ProxyPullPolicy)
	}
	if c.CRDsDir != "" {
		viper.Set(CRDsDirOpt, c.CRDsDir)
	}
	if c.OutputFormat != "" {
		viper.Set(OutputFormatOpt, c.OutputFormat)
	}
	if len(c.CRManifest) != 0 {
		viper.Set(CRManifestOpt, c.CRManifest)
	}
	if c.InitTimeout != 0 {
		viper.Set(InitTimeoutOpt, c.InitTimeout)
	}
	if c.OLMDeployed {
		viper.Set(OLMDeployedOpt, c.OLMDeployed)
	}
	if c.BasicTests {
		viper.Set(BasicTestsOpt, c.BasicTests)
	}
	if c.OLMTests {
		viper.Set(OLMTestsOpt, c.OLMTests)
	}
	if c.TenantTests {
		viper.Set(TenantTestsOpt, c.TenantTests)
	}
	return nil
}
