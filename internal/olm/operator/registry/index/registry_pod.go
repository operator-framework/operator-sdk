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

package index

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

// BundleAddModeType - type of BundleAddMode in RegistryPod struct
type BundleAddModeType = string

const (
	// SemverBundleAddMode - bundle add mode for semver
	SemverBundleAddMode BundleAddModeType = "semver"
	// ReplacesBundleAddMode - bundle add mode for replaces
	ReplacesBundleAddMode BundleAddModeType = "replaces"
)
const (
	// defaultGRPCPort is the default grpc container port that the registry pod exposes
	defaultGRPCPort          = 50051
	defaultIndexImage        = "quay.io/operator-framework/upstream-opm-builder:latest"
	defaultContainerName     = "registry-grpc"
	defaultContainerPortName = "grpc"
)

var (
	// Internal error
	errPodNotInit = errors.New("internal error: RegistryPod not initialized")
)

// RegistryPod holds resources necessary for creation of a registry server
type RegistryPod struct {
	// BundleAddMode specifies the graph update mode that defines how channel graphs are updated
	// It is of the type BundleAddModeType
	BundleAddMode BundleAddModeType

	// BundleImage specifies the container image that opm uses to generate and incrementally update the database
	BundleImage string

	// Index image contains a database of pointers to operator manifest content that is queriable via an API.
	// new version of an operator bundle when published can be added to an index image
	IndexImage string

	// DBPath refers to the registry DB;
	// if an index image is provided, the existing registry DB is located at /database/index.db
	DBPath string

	// Namespace refers to the specific namespace in which the registry pod will be created and scoped to
	Namespace string

	// GRPCPort is the container grpc port
	GRPCPort int32

	// client refers to a controller runtime client
	client client.Client

	// pod represents a kubernetes *corev1.pod that will be created on a cluster using an index image
	pod *corev1.Pod
}

// NewRegistryPod initializes the RegistryPod struct and sets defaults for empty fields
func NewRegistryPod(client client.Client, dbPath, bundleImage, namespace string) (*RegistryPod, error) {
	rp := &RegistryPod{}

	if rp.GRPCPort == 0 {
		rp.GRPCPort = defaultGRPCPort
	}

	if len(strings.TrimSpace(rp.IndexImage)) < 1 {
		rp.IndexImage = defaultIndexImage
	}

	if len(strings.TrimSpace(rp.BundleAddMode)) < 1 {
		if rp.IndexImage == defaultIndexImage {
			rp.BundleAddMode = SemverBundleAddMode
		} else {
			rp.BundleAddMode = ReplacesBundleAddMode
		}
	}

	rp.client = client
	rp.DBPath = dbPath
	rp.BundleImage = bundleImage
	rp.Namespace = namespace

	// validate the RegistryPod struct and ensure required fields are set
	if err := rp.validate(); err != nil {
		return nil, fmt.Errorf("error in validating registry pod struct: %v", err)
	}

	// call podForBundleRegistry() to make the pod definition
	pod, err := rp.podForBundleRegistry()
	if err != nil {
		return nil, fmt.Errorf("error in building registry pod definition: %v", err)
	}
	rp.pod = pod

	return rp, nil
}

// Create creates a bundle registry pod built from an index image
// and returns error
func (rp *RegistryPod) Create(ctx context.Context) error {
	if rp.pod == nil {
		return errPodNotInit
	}

	podKey, err := client.ObjectKeyFromObject(rp.pod)
	if err != nil {
		return fmt.Errorf("error in getting object key from the registry pod name %s: %v", rp.pod.Name, err)
	}

	if err := rp.client.Get(ctx, podKey, rp.pod); err != nil {
		if k8serrors.IsNotFound(err) {
			if err = rp.client.Create(ctx, rp.pod); err != nil {
				return fmt.Errorf("error creating registry pod: %v", err)
			}
		} else {
			return fmt.Errorf("registry pod name %s already exists: %v", rp.pod.Name, err)
		}
	}
	return nil
}

// VerifyPodRunning calls checkPodStatus to verify pod status
// and returns error if pod is not running
func (rp *RegistryPod) VerifyPodRunning(ctx context.Context) error {
	if rp.pod == nil {
		return errPodNotInit
	}

	podKey, err := client.ObjectKeyFromObject(rp.pod)
	if err != nil {
		return fmt.Errorf("error in getting object key from the registry pod name %s: %v", rp.pod.Name, err)
	}

	// upon creation of new pod, poll and verify that pod status is running
	podCheck := wait.ConditionFunc(func() (done bool, err error) {
		err = rp.client.Get(ctx, podKey, rp.pod)
		if err != nil {
			return false, fmt.Errorf("error getting pod %s: %w", rp.pod.Name, err)
		}
		return rp.pod.Status.Phase == corev1.PodRunning, nil
	})

	// check pod status to be Running
	if err := rp.checkPodStatus(ctx, podCheck); err != nil {
		return fmt.Errorf("registry pod did not become ready: %w", err)
	}
	return nil
}

// checkPodStatus polls and verifies that the pod status is running
func (rp *RegistryPod) checkPodStatus(ctx context.Context, podCheck wait.ConditionFunc) error {
	// poll every 200 ms until podCheck is true or context is done
	err := wait.PollImmediateUntil(200*time.Millisecond, podCheck, ctx.Done())
	if err != nil {
		return fmt.Errorf("error waiting for registry pod %s to run: %v", rp.pod.Name, err)
	}

	return err
}

// validate will ensure that RegistryPod required fields are set
// and throws error if not set
func (rp *RegistryPod) validate() error {
	if len(strings.TrimSpace(rp.BundleImage)) < 1 {
		return errors.New("bundle image cannot be empty")
	}
	if len(strings.TrimSpace(rp.DBPath)) < 1 {
		return errors.New("registry database path cannot be empty")
	}

	if len(strings.TrimSpace(rp.Namespace)) < 1 {
		return errors.New("pod namespace cannot be empty")
	}

	if len(strings.TrimSpace(rp.BundleAddMode)) < 1 {
		return errors.New("bundle add mode cannot be empty")
	}

	if rp.BundleAddMode != SemverBundleAddMode && rp.BundleAddMode != ReplacesBundleAddMode {
		return fmt.Errorf("invalid bundle mode %q: must be one of [%q, %q]",
			rp.BundleAddMode, ReplacesBundleAddMode, SemverBundleAddMode)
	}

	return nil
}

func GetRegistryPodHost(ipStr string) string {
	return fmt.Sprintf("%s:%d", ipStr, defaultGRPCPort)
}

// getPodName will return a string constructed from the bundle Image name
func getPodName(bundleImage string) string {
	// todo(rashmigottipati): need to come up with human-readable references
	// to be able to handle SHA references in the bundle images
	return k8sutil.TrimDNS1123Label(k8sutil.FormatOperatorNameDNS1123(bundleImage))
}

// podForBundleRegistry constructs and returns the registry pod definition
// and throws error when unable to build the pod definition successfully
func (rp *RegistryPod) podForBundleRegistry() (*corev1.Pod, error) {
	// construct the container command for pod spec
	containerCmd, err := rp.getContainerCmd()
	if err != nil {
		return nil, fmt.Errorf("error in parsing container command: %v", err)
	}

	// make the pod definition
	rp.pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getPodName(rp.BundleImage),
			Namespace: rp.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  defaultContainerName,
					Image: rp.IndexImage,
					Command: []string{
						"/bin/sh",
						"-c",
						containerCmd,
					},
					Ports: []corev1.ContainerPort{
						{Name: defaultContainerPortName, ContainerPort: rp.GRPCPort},
					},
				},
			},
		},
	}

	return rp.pod, nil
}

// getContainerCmd uses templating to construct the container command
// and throws error if unable to parse and execute the container command
func (rp *RegistryPod) getContainerCmd() (string, error) {
	const containerCommand = "/bin/mkdir -p {{ .DBPath }} &&" +
		"/bin/opm registry add -d {{ .DBPath }} -b {{.BundleImage}} --mode={{.BundleAddMode}} &&" +
		"/bin/opm registry serve -d {{ .DBPath }} -p {{.GRPCPort}}"
	type bundleCmd struct {
		BundleImage, DBPath, BundleAddMode string
		GRPCPort                           int32
	}

	var command = bundleCmd{rp.BundleImage, rp.DBPath,
		rp.BundleAddMode, rp.GRPCPort}

	out := &bytes.Buffer{}

	// add the custom basename template function to the
	// template's FuncMap and parse the containerCommand
	tmp := template.Must(template.New("containerCommand").Parse(containerCommand))

	// execute the command by applying the parsed tmp to command
	// and write command output to out
	if err := tmp.Execute(out, command); err != nil {
		return "", fmt.Errorf("error in parsing container command: %w", err)
	}

	return out.String(), nil
}

// GetPod returns the registry pod
func (rp *RegistryPod) GetPod() (*corev1.Pod, error) {
	if rp == nil {
		return nil, errPodNotInit
	}
	return rp.pod, nil
}
