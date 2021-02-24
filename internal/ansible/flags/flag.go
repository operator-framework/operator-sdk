// Copyright 2018 The Operator-SDK Authors
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
	"runtime"
	"time"

	"github.com/spf13/pflag"
)

// Flags - Options to be used by an ansible operator
type Flags struct {
	ReconcilePeriod         time.Duration
	WatchesFile             string
	InjectOwnerRef          bool
	EnableLeaderElection    bool
	MaxConcurrentReconciles int
	AnsibleVerbosity        int
	AnsibleRolesPath        string
	AnsibleCollectionsPath  string
	MetricsAddress          string
	ProbeAddr               string
	LeaderElectionID        string
	LeaderElectionNamespace string
	GracefulShutdownTimeout time.Duration
	AnsibleArgs             string
}

const AnsibleRolesPathEnvVar = "ANSIBLE_ROLES_PATH"
const AnsibleCollectionsPathEnvVar = "ANSIBLE_COLLECTIONS_PATH"

// AddTo - Add the ansible operator flags to the the flagset
func (f *Flags) AddTo(flagSet *pflag.FlagSet) {
	flagSet.DurationVar(&f.ReconcilePeriod,
		"reconcile-period",
		time.Minute,
		"Default reconcile period for controllers",
	)
	flagSet.StringVar(&f.WatchesFile,
		"watches-file",
		"./watches.yaml",
		"Path to the watches file to use",
	)
	flagSet.BoolVar(&f.InjectOwnerRef,
		"inject-owner-ref",
		true,
		"The ansible operator will inject owner references unless this flag is false",
	)
	flagSet.IntVar(&f.MaxConcurrentReconciles,
		"max-concurrent-reconciles",
		runtime.NumCPU(),
		"Maximum number of concurrent reconciles for controllers. Overridden by environment variable.",
	)
	flagSet.IntVar(&f.AnsibleVerbosity,
		"ansible-verbosity",
		2,
		"Ansible verbosity. Overridden by environment variable.",
	)
	flagSet.StringVar(&f.AnsibleRolesPath,
		"ansible-roles-path",
		"",
		"Ansible Roles Path. If unset, roles are assumed to be in {{CWD}}/roles.",
	)
	flagSet.StringVar(&f.AnsibleCollectionsPath,
		"ansible-collections-path",
		"",
		"Path to installed Ansible Collections. If set, collections should be located in {{value}}/ansible_collections/. If unset, collections are assumed to be in ~/.ansible/collections or /usr/share/ansible/collections.",
	)
	flagSet.StringVar(&f.MetricsAddress,
		"metrics-addr",
		":8080",
		"The address the metric endpoint binds to",
	)
	// todo: for Go/Helm the port used is: 8081
	// update it to keep the project aligned to the other
	// types for 2.0
	flagSet.StringVar(&f.ProbeAddr,
		"health-probe-bind-address",
		":6789",
		"The address the probe endpoint binds to.",
	)
	flagSet.BoolVar(&f.EnableLeaderElection,
		"enable-leader-election",
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
	flagSet.DurationVar(&f.GracefulShutdownTimeout,
		"graceful-shutdown-timeout",
		30*time.Second,
		"The amount of time that will be spent waiting"+
			" for runners to gracefully exit.",
	)
	flagSet.StringVar(&f.AnsibleArgs,
		"ansible-args",
		"",
		"Ansible args. Allows user to specify arbitrary arguments for ansible-based operators.",
	)
}
