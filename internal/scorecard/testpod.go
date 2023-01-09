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

package scorecard

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
)

const (
	// PodBundleRoot is the directory containing all bundle data within a test pod.
	PodBundleRoot = "/bundle"
)

// getPodDefinition fills out a Pod definition based on
// information from the test
func getPodDefinition(configMapName string, test v1alpha3.TestConfiguration, r PodTestRunner) *v1.Pod {

	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("scorecard-test-%s", rand.String(4)),
			Namespace: r.Namespace,
			Labels: map[string]string{
				"app":     "scorecard-test",
				"testrun": configMapName,
			},
		},
		Spec: v1.PodSpec{
			ServiceAccountName: r.ServiceAccount,
			RestartPolicy:      v1.RestartPolicyNever,
			Containers: []v1.Container{
				{
					Name:            "scorecard-test",
					Image:           test.Image,
					ImagePullPolicy: v1.PullIfNotPresent,
					Command:         test.Entrypoint,
					VolumeMounts: []v1.VolumeMount{
						{
							MountPath: PodBundleRoot,
							Name:      "scorecard-untar",
							ReadOnly:  true,
						},
					},
					Env: []v1.EnvVar{
						{
							Name: "SCORECARD_NAMESPACE",
							ValueFrom: &v1.EnvVarSource{
								FieldRef: &v1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
						},
					},
				},
			},
			InitContainers: []v1.Container{
				{
					Name:            "scorecard-untar",
					Image:           r.UntarImage,
					ImagePullPolicy: v1.PullIfNotPresent,
					Args: []string{
						"tar",
						"xvzf",
						"/scorecard/bundle.tar.gz",
						"-C",
						"/scorecard-bundle",
					},
					VolumeMounts: []v1.VolumeMount{
						{
							MountPath: "/scorecard",
							Name:      "scorecard-bundle",
							ReadOnly:  true,
						},
						{
							MountPath: "/scorecard-bundle",
							Name:      "scorecard-untar",
							ReadOnly:  false,
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
								Name: configMapName,
							},
						},
					},
				},
				{
					Name: "scorecard-untar",
					VolumeSource: v1.VolumeSource{
						EmptyDir: &v1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
}

// getPodLog fetches the test results which are found in the pod log
func getPodLog(ctx context.Context, client kubernetes.Interface, pod *v1.Pod) ([]byte, error) {
	podLogOptions := v1.PodLogOptions{
		Container: "scorecard-test",
	}

	req := client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOptions)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	return buf.Bytes(), err
}

// deletePods deletes a collection of pods that match a predefined selector value
func (r PodTestRunner) deletePods(ctx context.Context, configMapName string) error {
	do := metav1.DeleteOptions{}
	selector := fmt.Sprintf("testrun=%s", configMapName)
	lo := metav1.ListOptions{LabelSelector: selector}
	err := r.Client.CoreV1().Pods(r.Namespace).DeleteCollection(ctx, do, lo)
	if err != nil {
		return fmt.Errorf("error deleting pods (label selector %q): %w", selector, err)
	}
	return nil
}
