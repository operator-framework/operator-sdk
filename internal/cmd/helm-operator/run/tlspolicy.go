// Copyright 2026 The Operator-SDK Authors
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

package run

import (
	"context"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// ClusterTLSPolicy is an extension point that lets a distribution or
// deployment of the helm-operator enforce a centralized TLS policy (for
// example, one derived from cluster-wide configuration) on top of the
// manager's default TLS settings. operator-sdk ships no implementation;
// it is nil unless a build registers one via RegisterClusterTLSPolicy.
type ClusterTLSPolicy interface {
	// Apply augments manager options (e.g. options.Metrics.TLSOpts) with
	// any policy-driven TLS configuration. Implementations must return
	// promptly and should fail open (return the input options unchanged
	// on error) so a missing/unreachable policy source never blocks
	// operator startup.
	Apply(ctx context.Context, cfg *rest.Config, options manager.Options) (manager.Options, error)

	// Watch is invoked once after the manager is constructed, for
	// implementations that need to react to policy changes at runtime
	// (e.g. triggering a graceful restart via cancel).
	Watch(ctx context.Context, mgr manager.Manager, cancel context.CancelFunc) error
}

var registeredTLSPolicy ClusterTLSPolicy

// RegisterClusterTLSPolicy installs a ClusterTLSPolicy implementation.
// Intended to be called from an init() in a distribution-specific package,
// wired in via a blank import in cmd/helm-operator/main.go.
func RegisterClusterTLSPolicy(p ClusterTLSPolicy) {
	registeredTLSPolicy = p
}
