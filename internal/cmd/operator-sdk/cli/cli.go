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

package cli

import (
	quarkusv1 "github.com/operator-framework/java-operator-plugins/pkg/quarkus/v1alpha"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/kubebuilder/v3/pkg/cli"
	cfgv2 "sigs.k8s.io/kubebuilder/v3/pkg/config/v2"
	cfgv3 "sigs.k8s.io/kubebuilder/v3/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	kustomizev1 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v1"
	declarativev1 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/declarative/v1"

	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang"
	golangv2 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2"
	golangv3 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3"

	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/alpha/config3alphato3"
	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/bundle"
	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/cleanup"
	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/generate"
	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/olm"
	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/pkgmantobundle"
	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/run"
	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/scorecard"
	"github.com/operator-framework/operator-sdk/internal/flags"
	"github.com/operator-framework/operator-sdk/internal/plugins"
	ansiblev1 "github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1"
	envtestv1 "github.com/operator-framework/operator-sdk/internal/plugins/envtest/v1"
	helmv1 "github.com/operator-framework/operator-sdk/internal/plugins/helm/v1"
	manifestsv2 "github.com/operator-framework/operator-sdk/internal/plugins/manifests/v2"
	scorecardv2 "github.com/operator-framework/operator-sdk/internal/plugins/scorecard/v2"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

var (
	commands = []*cobra.Command{
		bundle.NewCmd(),
		cleanup.NewCmd(),
		generate.NewCmd(),
		olm.NewCmd(),
		run.NewCmd(),
		scorecard.NewCmd(),
		pkgmantobundle.NewCmd(),
	}
	alphaCommands = []*cobra.Command{
		config3alphato3.NewCmd(),
	}
)

func Run() error {
	c, _ := GetPluginsCLIAndRoot()
	return c.Run()
}

// GetPluginsCLIAndRoot returns the plugins based CLI configured to use operator-sdk as the root command
// This CLI can run kubebuilder commands and certain SDK specific commands that are aligned for
// the kubebuilder project layout
func GetPluginsCLIAndRoot() (*cli.CLI, *cobra.Command) {
	// todo: Export the bundles KB and then change here to use the bundles exported instead
	// more info: https://github.com/kubernetes-sigs/kubebuilder/pull/2112
	gov2Bundle, _ := plugin.NewBundle(golang.DefaultNameQualifier, golangv2.Plugin{}.Version(),
		golangv2.Plugin{},
		envtestv1.Plugin{},
		manifestsv2.Plugin{},
		scorecardv2.Plugin{},
	)
	gov3Bundle, _ := plugin.NewBundle(golang.DefaultNameQualifier, golangv3.Plugin{}.Version(),
		kustomizev1.Plugin{},
		golangv3.Plugin{},
		manifestsv2.Plugin{},
		scorecardv2.Plugin{},
	)
	ansibleBundle, _ := plugin.NewBundle("ansible"+plugins.DefaultNameQualifier, plugin.Version{Number: 1},
		kustomizev1.Plugin{},
		ansiblev1.Plugin{},
		manifestsv2.Plugin{},
		scorecardv2.Plugin{},
	)
	helmBundle, _ := plugin.NewBundle("helm"+plugins.DefaultNameQualifier, plugin.Version{Number: 1},
		kustomizev1.Plugin{},
		helmv1.Plugin{},
		manifestsv2.Plugin{},
		scorecardv2.Plugin{},
	)
	c, err := cli.New(
		cli.WithCommandName("operator-sdk"),
		cli.WithVersion(makeVersionString()),
		cli.WithPlugins(
			ansibleBundle,
			gov2Bundle,
			gov3Bundle,
			helmBundle,
			kustomizev1.Plugin{},
			declarativev1.Plugin{},
			&quarkusv1.Plugin{},
		),
		cli.WithDefaultPlugins(cfgv2.Version, gov2Bundle),
		cli.WithDefaultPlugins(cfgv3.Version, gov3Bundle),
		cli.WithDefaultProjectVersion(cfgv3.Version),
		cli.WithExtraCommands(commands...),
		cli.WithExtraAlphaCommands(alphaCommands...),
		cli.WithCompletion(),
	)
	if err != nil {
		log.Fatal(err)
	}

	// We can get the whole CLI for doc-gen/completion from the root of any
	// command added to a CLI.
	root := commands[0].Root()

	// Configure --verbose globally.
	// TODO(estroz): upstream PR for global --verbose.
	root.PersistentFlags().Bool(flags.VerboseOpt, false, "Enable verbose logging")
	if err := viper.BindPFlags(root.PersistentFlags()); err != nil {
		log.Fatalf("Failed to bind %s flags: %v", root.Name(), err)
	}
	root.PersistentPreRun = rootPersistentPreRun

	return c, root
}

func rootPersistentPreRun(cmd *cobra.Command, args []string) {
	if viper.GetBool(flags.VerboseOpt) {
		if err := projutil.SetGoVerbose(); err != nil {
			log.Fatalf("Could not set GOFLAGS: (%v)", err)
		}
		log.SetLevel(log.DebugLevel)
		log.Debug("Debug logging is set")
	}

	config3alphato3.RootPersistentPreRun(cmd, args)
}
