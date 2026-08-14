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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// stubTLSPolicy is a minimal ClusterTLSPolicy used to verify the
// registration/hook contract without depending on any real implementation
// (e.g. the downstream-only openshifttls package).
type stubTLSPolicy struct {
	applyCalled, watchCalled bool
	applyErr, watchErr       error
}

func (s *stubTLSPolicy) Apply(_ context.Context, _ *rest.Config, options manager.Options) (manager.Options, error) {
	s.applyCalled = true
	return options, s.applyErr
}

func (s *stubTLSPolicy) Watch(_ context.Context, _ manager.Manager, _ context.CancelFunc) error {
	s.watchCalled = true
	return s.watchErr
}

// withRegisteredTLSPolicy registers p for the duration of the test and
// restores the previous value on cleanup, so tests don't leak global state
// into each other or into non-TLS-policy tests in this package.
func withRegisteredTLSPolicy(t *testing.T, p ClusterTLSPolicy) {
	t.Helper()
	previous := registeredTLSPolicy
	t.Cleanup(func() { registeredTLSPolicy = previous })
	registeredTLSPolicy = p
}

func TestNoClusterTLSPolicyRegisteredByDefault(t *testing.T) {
	// Guards the upstream operator-sdk default: without a distribution
	// registering an implementation (e.g. via a blank import of an
	// OpenShift-specific package), the hook must be a no-op.
	assert.Nil(t, registeredTLSPolicy)
}

func TestRegisterClusterTLSPolicy(t *testing.T) {
	require.Nil(t, registeredTLSPolicy)

	policy := &stubTLSPolicy{}
	withRegisteredTLSPolicy(t, policy)

	assert.Same(t, ClusterTLSPolicy(policy), registeredTLSPolicy)
}

func TestRegisterClusterTLSPolicy_Overwrites(t *testing.T) {
	first := &stubTLSPolicy{}
	withRegisteredTLSPolicy(t, first)

	second := &stubTLSPolicy{}
	RegisterClusterTLSPolicy(second)

	assert.Same(t, ClusterTLSPolicy(second), registeredTLSPolicy)
}

func TestClusterTLSPolicy_ApplyFailsOpen(t *testing.T) {
	// Documents the interface contract: Apply may return an error, and
	// callers (cmd.go's run()) are expected to log and continue with the
	// input options unchanged rather than fail startup.
	policy := &stubTLSPolicy{applyErr: errors.New("profile source unreachable")}
	withRegisteredTLSPolicy(t, policy)

	in := manager.Options{}
	out, err := registeredTLSPolicy.Apply(context.Background(), &rest.Config{}, in)

	require.Error(t, err)
	assert.True(t, policy.applyCalled)
	assert.Equal(t, in, out)
}
