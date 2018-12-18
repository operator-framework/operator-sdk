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

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/helm/pkg/proto/hapi/release"
)

const (
	testNamespaceName = "helm-test"
	testReleaseName   = "helm-test-dory"
)

var now = metav1.Now()

func TestSetCondition(t *testing.T) {
	message := "uninstall was successful"
	releaseName := "TestRelease"
	newStatus, err := newTestStatus().SetCondition(HelmAppCondition{
		Type:    ConditionDeployed,
		Status:  StatusFalse,
		Reason:  ReasonUninstallSuccessful,
		Message: message,
		Release: &release.Release{Name: releaseName},
	}).ToMap()
	assert.NoError(t, err)

	resource := newTestResource()
	resource.Object["status"] = newStatus
	actual := StatusFor(resource)

	assert.Equal(t, ConditionDeployed, actual.Conditions[0].Type)
	assert.Equal(t, StatusFalse, actual.Conditions[0].Status)
	assert.Equal(t, ReasonUninstallSuccessful, actual.Conditions[0].Reason)
	assert.Equal(t, message, actual.Conditions[0].Message)
	assert.Equal(t, releaseName, actual.Conditions[0].Release.Name)
	assert.NotEqual(t, metav1.Now(), actual.Conditions[0].LastTransitionTime)
}
func TestRemoveCondition(t *testing.T) {
	newStatus, err := newTestStatus().RemoveCondition(ConditionDeployed).ToMap()
	assert.NoError(t, err)

	resource := newTestResource()
	resource.Object["status"] = newStatus
	actual := StatusFor(resource)

	assert.Empty(t, actual.Conditions)
}

func TestStatusForEmpty(t *testing.T) {
	status := StatusFor(newTestResource())

	assert.Equal(t, &HelmAppStatus{}, status)
}

func TestStatusForFilled(t *testing.T) {
	expectedResource := newTestResource()
	expectedResource.Object["status"] = newTestStatus()
	status := StatusFor(expectedResource)

	assert.EqualValues(t, newTestStatus().Conditions, status.Conditions)
}

func TestStatusForFilledRaw(t *testing.T) {
	expectedResource := newTestResource()
	expectedResource.Object["status"] = newTestStatusRaw()
	status := StatusFor(expectedResource)

	assert.Equal(t, ConditionDeployed, status.Conditions[0].Type)
	assert.Equal(t, StatusTrue, status.Conditions[0].Status)
	assert.Equal(t, ReasonInstallSuccessful, status.Conditions[0].Reason)
	assert.Equal(t, "some message", status.Conditions[0].Message)
	assert.Equal(t, "SomeRelease", status.Conditions[0].Release.Name)
	assert.NotEqual(t, metav1.Now(), status.Conditions[0].LastTransitionTime)
}

func newTestResource() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "Character",
			"apiVersion": "stable.nicolerenee.io",
			"metadata": map[string]interface{}{
				"name":      "dory",
				"namespace": testNamespaceName,
			},
			"spec": map[string]interface{}{
				"Name": "Dory",
				"From": "Finding Nemo",
				"By":   "Disney",
			},
		},
	}
}

func newTestStatus() *HelmAppStatus {
	return &HelmAppStatus{
		Conditions: []HelmAppCondition{
			{
				Type:               ConditionDeployed,
				Status:             StatusTrue,
				Reason:             ReasonInstallSuccessful,
				Message:            "some message",
				Release:            &release.Release{Name: "SomeRelease"},
				LastTransitionTime: now,
			},
		},
	}
}

func newTestStatusRaw() map[string]interface{} {
	return map[string]interface{}{
		"conditions": []map[string]interface{}{
			{
				"type":               "Deployed",
				"status":             "True",
				"reason":             "InstallSuccessful",
				"message":            "some message",
				"release":            map[string]interface{}{"name": "SomeRelease"},
				"lastTransitionTime": now.UTC(),
			},
		},
	}
}
