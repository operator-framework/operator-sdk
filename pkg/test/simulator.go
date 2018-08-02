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

package test

import (
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (ctx *TestCtx) SimulatePodFailure(deploymentName string) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}
	dep, err := Global.KubeClient.AppsV1().Deployments(namespace).Get(deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	labelSelector := labels.SelectorFromSet(dep.Spec.Selector.MatchLabels).String()
	rep, err := Global.KubeClient.AppsV1().ReplicaSets(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return err
	}
	if len(rep.Items) == 0 {
		return fmt.Errorf("no replica set for deployment '%s'", deploymentName)
	} else if len(rep.Items) != 1 {
		return fmt.Errorf("deployment '%s' has more than 1 replica set", deploymentName)
	}
	labelSelector = labels.SelectorFromSet(rep.Items[0].Spec.Selector.MatchLabels).String()
	pods, err := Global.KubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
	for _, pod := range pods.Items {
		err := Global.KubeClient.CoreV1().Pods(namespace).Delete(pod.Name, metav1.NewDeleteOptions(0))
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
