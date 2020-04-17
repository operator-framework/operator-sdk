// Copyright 2020 The Operator-SDK Authors
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

package alpha

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getPodDefinition fills out a Pod definition based on
// information from the test
func getPodDefinition(test ScorecardTest, namespace, serviceAccount string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scorecard-test",
			Namespace: namespace,
			Labels: map[string]string{
				"name": "scorecard-test",
			},
		},
		Spec: v1.PodSpec{
			ServiceAccountName: serviceAccount,
			RestartPolicy:      v1.RestartPolicyNever,
			Containers: []v1.Container{
				{
					Name:            "scorecard-test",
					Image:           "quay.io/operator-framework/scorecard-test:dev",
					ImagePullPolicy: v1.PullIfNotPresent,
					Command: []string{
						"/usr/local/bin/scorecard-test",
					},
					Args: []string{
						test.Entrypoint,
					},
					VolumeMounts: []v1.VolumeMount{
						{
							MountPath: "/scorecard",
							Name:      "scorecard-bundle",
							ReadOnly:  true,
						},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: "scorecard-bundle",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "scorecard-bundle",
							},
						},
					},
				},
			},
		},
	}
}
