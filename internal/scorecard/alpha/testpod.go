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
	"bytes"
	"context"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
)

// getPodDefinition fills out a Pod definition based on
// information from the test
func getPodDefinition(test Test, o Scorecard) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("scorecard-test-%s", rand.String(4)),
			Namespace: o.Namespace,
			Labels: map[string]string{
				"app": "scorecard-test",
			},
		},
		Spec: v1.PodSpec{
			ServiceAccountName: o.ServiceAccount,
			RestartPolicy:      v1.RestartPolicyNever,
			Containers: []v1.Container{
				{
					Name:            "scorecard-test",
					Image:           "quay.io/operator-framework/scorecard-test:dev",
					ImagePullPolicy: v1.PullIfNotPresent,
					Command:         test.Entrypoint,
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
								Name: o.bundleConfigMap.Name,
							},
						},
					},
				},
			},
		},
	}
}

func getPodLog(client kubernetes.Interface, pod *v1.Pod) (logOutput []byte, err error) {

	req := client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{})
	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		return logOutput, err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return logOutput, err
	}
	return buf.Bytes(), err
}

func deletePods(client kubernetes.Interface, tests []Test) {
	for _, test := range tests {
		p := test.TestPod
		err := client.CoreV1().Pods(p.Namespace).Delete(context.TODO(), p.Name, metav1.DeleteOptions{})
		if err != nil {
			log.Errorf("Error deleting pod %s %s\n", p.Name, err.Error())
		}

	}
}
