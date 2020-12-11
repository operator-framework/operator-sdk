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

package registrypod

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
)

// CreateOwnedPod creates a pod owned by owner and verifies that the pod is running.
func CreateOwnedPod(ctx context.Context, cfg *operator.Configuration, pod *corev1.Pod, owner client.Object) error {

	pod.SetNamespace(cfg.Namespace)
	podKey := client.ObjectKeyFromObject(pod)
	log.Infof("Creating registry pod %q", podKey)

	// make catalog source the owner of registry pod object
	if err := controllerutil.SetOwnerReference(owner, pod, cfg.Scheme); err != nil {
		return fmt.Errorf("error setting registry pod owner to %q: %v", client.ObjectKeyFromObject(owner), err)
	}

	if err := cfg.Client.Create(ctx, pod); err != nil {
		return fmt.Errorf("error creating registry pod: %v", err)
	}

	// poll and verify that pod is running
	podCheck := wait.ConditionFunc(func() (done bool, err error) {
		err = cfg.Client.Get(ctx, podKey, pod)
		if err != nil {
			return false, err
		}
		return pod.Status.Phase == corev1.PodRunning, nil
	})

	// check pod status to be `Running`
	// poll every 200 ms until podCheck is true or context is done
	if err := wait.PollImmediateUntil(200*time.Millisecond, podCheck, ctx.Done()); err != nil {
		return fmt.Errorf("error waiting for registry pod to run: %v", err)
	}
	log.Infof("Registry pod %q is running", podKey)

	return nil
}

func GetHostName(ipStr string, port int32) string {
	return fmt.Sprintf("%s:%d", ipStr, port)
}
