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

package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rpb "helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	helmtypes "github.com/operator-framework/operator-sdk/internal/helm/internal/types"
	"github.com/operator-framework/operator-sdk/internal/helm/release"
)

func TestDetermineReconcilePeriod(t *testing.T) {
	testPeriod1, _ := time.ParseDuration("10s")
	obj1 := &unstructured.Unstructured{
		Object: map[string]any{
			"metadata": map[string]any{
				"annotations": map[string]any{
					"name":                        "test-obj-1",
					helmReconcilePeriodAnnotation: "3s",
				},
			},
		},
	}
	expected1, _ := time.ParseDuration("3s")
	finalPeriod1, err := determineReconcilePeriod(testPeriod1, obj1)
	assert.Equal(t, nil, err, "Verify that no error is returned on parsing the time period")
	assert.Equal(t, expected1, finalPeriod1, "Verify that the annotations period takes precedence")

	testPeriod2, _ := time.ParseDuration("1h3m4s")
	obj2 := &unstructured.Unstructured{
		Object: map[string]any{
			"metadata": map[string]any{
				"annotations": map[string]any{
					"name": "test-obj-2",
				},
			},
		},
	}
	expected2, _ := time.ParseDuration("1h3m4s")
	finalPeriod2, err := determineReconcilePeriod(testPeriod2, obj2)
	assert.Equal(t, nil, err, "Verify that no error is returned on parsing the time period")
	assert.Equal(t, expected2, finalPeriod2, "Verify that when no time period is present under the CR's annotations, the original time period value gets used")

	testPeriod3, _ := time.ParseDuration("5m15s")
	obj3 := &unstructured.Unstructured{
		Object: map[string]any{
			"metadata": map[string]any{
				"annotations": map[string]any{
					"name":                        "test-obj-3",
					helmReconcilePeriodAnnotation: "4x",
				},
			},
		},
	}
	finalPeriod3, err := determineReconcilePeriod(testPeriod3, obj3)
	expected3, _ := time.ParseDuration("5m15s")
	assert.NotEqual(t, nil, err, "Verify that error is thrown when invalid time period is passed in the CR annotations")
	assert.Equal(t, expected3, finalPeriod3, "Verify that when a faulty time period is present under the CR's annotations, the original time period value gets used")
}

func TestHasAnnotation(t *testing.T) {
	upgradeForceTests := []struct {
		input       map[string]any
		expectedVal bool
		expectedOut string
		name        string
	}{
		{
			input: map[string]any{
				"helm.sdk.operatorframework.io/upgrade-force": "True",
			},
			expectedVal: true,
			name:        "upgrade force base case true",
		},
		{
			input: map[string]any{
				"helm.sdk.operatorframework.io/upgrade-force": "False",
			},
			expectedVal: false,
			name:        "upgrade force base case false",
		},
		{
			input: map[string]any{
				"helm.sdk.operatorframework.io/upgrade-force": "1",
			},
			expectedVal: true,
			name:        "upgrade force true as int",
		},
		{
			input: map[string]any{
				"helm.sdk.operatorframework.io/upgrade-force": "0",
			},
			expectedVal: false,
			name:        "upgrade force false as int",
		},
		{
			input: map[string]any{
				"helm.sdk.operatorframework.io/wrong-annotation": "true",
			},
			expectedVal: false,
			name:        "upgrade force annotation not set",
		},
		{
			input: map[string]any{
				"helm.sdk.operatorframework.io/upgrade-force": "invalid",
			},
			expectedVal: false,
			name:        "upgrade force invalid value",
		},
	}

	for _, test := range upgradeForceTests {
		assert.Equal(t, test.expectedVal, hasAnnotation(helmUpgradeForceAnnotation, annotations(test.input)), test.name)
	}

	uninstallWaitTests := []struct {
		input       map[string]any
		expectedVal bool
		expectedOut string
		name        string
	}{
		{
			input: map[string]any{
				"helm.sdk.operatorframework.io/uninstall-wait": "True",
			},
			expectedVal: true,
			name:        "uninstall wait base case true",
		},
		{
			input: map[string]any{
				"helm.sdk.operatorframework.io/uninstall-wait": "False",
			},
			expectedVal: false,
			name:        "uninstall wait base case false",
		},
		{
			input: map[string]any{
				"helm.sdk.operatorframework.io/uninstall-wait": "1",
			},
			expectedVal: true,
			name:        "uninstall wait true as int",
		},
		{
			input: map[string]any{
				"helm.sdk.operatorframework.io/uninstall-wait": "0",
			},
			expectedVal: false,
			name:        "uninstall wait false as int",
		},
		{
			input: map[string]any{
				"helm.sdk.operatorframework.io/wrong-annotation": "true",
			},
			expectedVal: false,
			name:        "uninstall wait annotation not set",
		},
		{
			input: map[string]any{
				"helm.sdk.operatorframework.io/uninstall-wait": "invalid",
			},
			expectedVal: false,
			name:        "uninstall wait invalid value",
		},
	}

	for _, test := range uninstallWaitTests {
		assert.Equal(t, test.expectedVal, hasAnnotation(helmUninstallWaitAnnotation, annotations(test.input)), test.name)
	}
}

func annotations(m map[string]any) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"metadata": map[string]any{
				"annotations": m,
			},
		},
	}
}

func Test_readBoolAnnotationWithDefault(t *testing.T) {
	objBuilder := func(anno map[string]string) *unstructured.Unstructured {
		object := &unstructured.Unstructured{}
		object.SetAnnotations(anno)
		return object
	}

	type args struct {
		obj        *unstructured.Unstructured
		annotation string
		fallback   bool
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Should return value of annotation read",
			args: args{
				obj: objBuilder(map[string]string{
					"helm.sdk.operatorframework.io/rollback-force": "false",
				}),
				annotation: "helm.sdk.operatorframework.io/rollback-force",
				fallback:   true,
			},
			want: false,
		},
		{
			name: "Should return fallback when annotation is not present",
			args: args{
				obj: objBuilder(map[string]string{
					"helm.sdk.operatorframework.io/upgrade-force": "true",
				}),
				annotation: "helm.sdk.operatorframework.io/rollback-force",
				fallback:   false,
			},
			want: false,
		},
		{
			name: "Should return fallback when errors while parsing bool value",
			args: args{
				obj: objBuilder(map[string]string{
					"helm.sdk.operatorframework.io/rollback-force": "force",
				}),
				annotation: "helm.sdk.operatorframework.io/rollback-force",
				fallback:   true,
			},
			want: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := readBoolAnnotationWithDefault(tc.args.obj, tc.args.annotation, tc.args.fallback); got != tc.want {
				assert.Equal(t, tc.want, got, "readBoolAnnotationWithDefault() function")
			}
		})
	}
}

var testGVK = schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "App"}

func TestReconcileSetsObservedGeneration(t *testing.T) {
	deployed := &rpb.Release{Name: "app", Version: 2, Manifest: "kind: ConfigMap\n"}

	tests := []struct {
		name          string
		status        map[string]any
		manager       *fakeManager
		wantErr       bool
		wantCondition helmtypes.HelmAppConditionType
	}{
		{
			name:          "Should record the generation on install",
			manager:       &fakeManager{installed: &rpb.Release{Name: "app", Version: 1, Manifest: "kind: ConfigMap\n"}},
			wantCondition: helmtypes.ConditionDeployed,
		},
		{
			name:          "Should record the generation on upgrade",
			status:        deployedStatus(t, 1, deployed),
			manager:       &fakeManager{isInstalled: true, upgradeRequired: true, upgraded: deployed},
			wantCondition: helmtypes.ConditionDeployed,
		},
		{
			// The release is unchanged, so only the generation differs from
			// the stored status.
			name:          "Should record the generation when the release is unchanged",
			status:        deployedStatus(t, 1, deployed),
			manager:       &fakeManager{isInstalled: true, reconciled: deployed},
			wantCondition: helmtypes.ConditionDeployed,
		},
		{
			name:          "Should record the generation when sync fails",
			status:        deployedStatus(t, 1, deployed),
			manager:       &fakeManager{isInstalled: true, syncErr: errors.New("sync failed")},
			wantErr:       true,
			wantCondition: helmtypes.ConditionIrreconcilable,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			obj := newApp(2, tc.status)
			c := fake.NewClientBuilder().WithObjects(obj).WithStatusSubresource(obj).
				WithInterceptorFuncs(interceptor.Funcs{SubResourceUpdate: typedStatusToMap}).Build()
			r := HelmOperatorReconciler{
				Client:         c,
				EventRecorder:  record.NewFakeRecorder(10),
				GVK:            testGVK,
				ManagerFactory: fakeManagerFactory{tc.manager},
			}

			_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "ns", Name: "app"}})
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			got := &unstructured.Unstructured{}
			got.SetGroupVersionKind(testGVK)
			require.NoError(t, c.Get(context.Background(), client.ObjectKeyFromObject(obj), got))
			status := helmtypes.StatusFor(got)
			assert.Equal(t, got.GetGeneration(), status.ObservedGeneration)
			assert.True(t, hasCondition(status, tc.wantCondition), "missing condition %s", tc.wantCondition)
		})
	}
}

// typedStatusToMap converts the typed status that updateResourceStatus stores
// in the unstructured object into a map, as JSON serialization does for a real
// API server. The fake client cannot deep copy the typed value.
func typedStatusToMap(ctx context.Context, c client.Client, subResource string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	if u, ok := obj.(*unstructured.Unstructured); ok {
		if s, ok := u.Object["status"].(*helmtypes.HelmAppStatus); ok {
			m, err := s.ToMap()
			if err != nil {
				return err
			}
			u.Object["status"] = m
		}
	}
	return c.SubResource(subResource).Update(ctx, obj, opts...)
}

func newApp(generation int64, status map[string]any) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]any{"spec": map[string]any{"replicas": int64(1)}}}
	obj.SetGroupVersionKind(testGVK)
	obj.SetNamespace("ns")
	obj.SetName("app")
	obj.SetGeneration(generation)
	obj.SetFinalizers([]string{uninstallFinalizer})
	if status != nil {
		obj.Object["status"] = status
	}
	return obj
}

func deployedStatus(t *testing.T, generation int64, rel *rpb.Release) map[string]any {
	t.Helper()
	status := &helmtypes.HelmAppStatus{
		Conditions: []helmtypes.HelmAppCondition{
			{Type: helmtypes.ConditionInitialized, Status: helmtypes.StatusTrue},
			{Type: helmtypes.ConditionDeployed, Status: helmtypes.StatusTrue, Reason: helmtypes.ReasonUpgradeSuccessful},
		},
		DeployedRelease:    &helmtypes.HelmAppRelease{Name: rel.Name, Manifest: rel.Manifest},
		ObservedGeneration: generation,
	}
	m, err := status.ToMap()
	require.NoError(t, err)
	return m
}

func hasCondition(status *helmtypes.HelmAppStatus, ct helmtypes.HelmAppConditionType) bool {
	for _, c := range status.Conditions {
		if c.Type == ct && c.Status == helmtypes.StatusTrue {
			return true
		}
	}
	return false
}

type fakeManagerFactory struct {
	manager *fakeManager
}

func (f fakeManagerFactory) NewManager(*unstructured.Unstructured, map[string]string, string) (release.Manager, error) {
	return f.manager, nil
}

type fakeManager struct {
	isInstalled     bool
	upgradeRequired bool
	syncErr         error
	installed       *rpb.Release
	upgraded        *rpb.Release
	reconciled      *rpb.Release
}

func (m *fakeManager) ReleaseName() string                      { return "app" }
func (m *fakeManager) IsInstalled() bool                        { return m.isInstalled }
func (m *fakeManager) IsUpgradeRequired() bool                  { return m.upgradeRequired }
func (m *fakeManager) Sync() error                              { return m.syncErr }
func (m *fakeManager) RollBack(...release.RollBackOption) error { return nil }
func (m *fakeManager) CleanupRelease(string) (bool, error)      { return true, nil }

func (m *fakeManager) InstallRelease(...release.InstallOption) (*rpb.Release, error) {
	return m.installed, nil
}

func (m *fakeManager) UpgradeRelease(...release.UpgradeOption) (*rpb.Release, *rpb.Release, error) {
	return m.upgraded, m.upgraded, nil
}

func (m *fakeManager) ReconcileRelease(context.Context) (*rpb.Release, error) {
	return m.reconciled, nil
}

func (m *fakeManager) UninstallRelease(...release.UninstallOption) (*rpb.Release, error) {
	return nil, nil
}
