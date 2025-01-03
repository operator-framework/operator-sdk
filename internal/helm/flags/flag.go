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

package flags

import (
	"crypto/tls"
	"runtime"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// Flags - Options to be used by a helm operator
type Flags struct {
	ReconcilePeriod         time.Duration
	WatchesFile             string
	MetricsBindAddress      string
	LeaderElection          bool
	LeaderElectionID        string
	LeaderElectionNamespace string
	MaxConcurrentReconciles int
	ProbeAddr               string
	SuppressOverrideValues  bool
	EnableHTTP2             bool
	SecureMetrics           bool
	MetricsRequireRBAC      bool

	// If not nil, used to deduce which flags were set in the CLI.
	flagSet *pflag.FlagSet
}

// AddTo - Add the helm operator flags to the flagset
func (f *Flags) AddTo(flagSet *pflag.FlagSet) {
	// Store flagset internally to be used for lookups later.
	f.flagSet = flagSet

	// Helm flags.
	flagSet.StringVar(&f.WatchesFile,
		"watches-file",
		"./watches.yaml",
		"Path to the watches file to use",
	)

	// Controller flags.
	flagSet.DurationVar(&f.ReconcilePeriod,
		"reconcile-period",
		time.Minute,
		"Default reconcile period for controllers",
	)
	flagSet.IntVar(&f.MaxConcurrentReconciles,
		"max-concurrent-reconciles",
		runtime.NumCPU(),
		"Maximum number of concurrent reconciles for controllers.",
	)

	_ = flagSet.MarkDeprecated("config",
		`controller-runtime has deprecated the ComponentConfig package 
and as such, the ability to load the configuation from a file. Since the helm operator relies on controller-runtime
this flag will be removed when upgrading to a version of controller-runtime where the ComponentConfig package has been removed.
see https://github.com/kubernetes-sigs/controller-runtime/issues/895 for more information.`)

	// TODO(2.0.0): remove
	flagSet.StringVar(&f.MetricsBindAddress,
		"metrics-addr",
		":8080",
		"The address the metric endpoint binds to",
	)
	_ = flagSet.MarkDeprecated("metrics-addr", "use --metrics-bind-address instead")
	flagSet.StringVar(&f.MetricsBindAddress,
		"metrics-bind-address",
		":8080",
		"The address the metric endpoint binds to",
	)
	// TODO(2.0.0): for Go/Helm the port used is: 8081
	// update it to keep the project aligned to the other
	flagSet.StringVar(&f.ProbeAddr,
		"health-probe-bind-address",
		":8081",
		"The address the probe endpoint binds to.",
	)
	// TODO(2.0.0): remove
	flagSet.BoolVar(&f.LeaderElection,
		"enable-leader-election",
		false,
		"Enable leader election for controller manager. Enabling this will"+
			" ensure there is only one active controller manager.",
	)
	_ = flagSet.MarkDeprecated("enable-leader-election", "use --leader-elect instead.")
	flagSet.BoolVar(&f.LeaderElection,
		"leader-elect",
		false,
		"Enable leader election for controller manager. Enabling this will"+
			" ensure there is only one active controller manager.",
	)
	flagSet.StringVar(&f.LeaderElectionID,
		"leader-election-id",
		"",
		"Name of the configmap that is used for holding the leader lock.",
	)
	flagSet.StringVar(&f.LeaderElectionNamespace,
		"leader-election-namespace",
		"",
		"Namespace in which to create the leader election configmap for"+
			" holding the leader lock (required if running locally with leader"+
			" election enabled).",
	)
	flagSet.BoolVar(&f.SuppressOverrideValues,
		"suppress-override-values",
		false,
		"Silences the override-value for OverrideValuesInUse events",
	)
	flagSet.BoolVar(&f.EnableHTTP2,
		"enable-http2",
		false,
		"enables HTTP/2 on the webhook and metrics servers",
	)
	flagSet.BoolVar(&f.SecureMetrics,
		"metrics-secure",
		false,
		"enables secure serving of the metrics endpoint",
	)
	flagSet.BoolVar(&f.MetricsRequireRBAC,
		"metrics-require-rbac",
		false,
		"enables protection of the metrics endpoint with RBAC-based authn/authz."+
			"see https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/metrics/filters#WithAuthenticationAndAuthorization for more info")
}

// ToManagerOptions uses the flag set in f to configure options.
// Values of options take precedence over flag defaults,
// as values are assume to have been explicitly set.
func (f *Flags) ToManagerOptions(options manager.Options) manager.Options {
	// Alias FlagSet.Changed so options are still updated when fields are empty.
	changed := func(flagName string) bool {
		return f.flagSet.Changed(flagName)
	}
	if f.flagSet == nil {
		//nolint:golint
		changed = func(_ string) bool { return false }
	}

	// TODO(2.0.0): remove metrics-addr
	if changed("metrics-bind-address") || changed("metrics-addr") || options.Metrics.BindAddress == "" {
		options.Metrics.BindAddress = f.MetricsBindAddress
	}
	if changed("health-probe-bind-address") || options.HealthProbeBindAddress == "" {
		options.HealthProbeBindAddress = f.ProbeAddr
	}
	// TODO(2.0.0): remove enable-leader-election
	if changed("leader-elect") || changed("enable-leader-election") || !options.LeaderElection {
		options.LeaderElection = f.LeaderElection
	}
	if changed("leader-election-id") || options.LeaderElectionID == "" {
		options.LeaderElectionID = f.LeaderElectionID
	}
	if changed("leader-election-namespace") || options.LeaderElectionNamespace == "" {
		options.LeaderElectionNamespace = f.LeaderElectionNamespace
	}
	if options.LeaderElectionResourceLock == "" {
		options.LeaderElectionResourceLock = resourcelock.LeasesResourceLock
	}

	disableHTTP2 := func(c *tls.Config) {
		c.NextProtos = []string{"http/1.1"}
	}
	if !f.EnableHTTP2 {
		options.WebhookServer = webhook.NewServer(webhook.Options{
			TLSOpts: []func(*tls.Config){disableHTTP2},
		})
		options.Metrics.TLSOpts = append(options.Metrics.TLSOpts, disableHTTP2)
	}
	options.Metrics.SecureServing = f.SecureMetrics

	if f.MetricsRequireRBAC {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/metrics/filters#WithAuthenticationAndAuthorization
		options.Metrics.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	return options
}
