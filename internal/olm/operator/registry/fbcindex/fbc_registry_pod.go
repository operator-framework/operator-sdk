// Copyright 2022 The Operator-SDK Authors
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

package fbcindex

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/registry/index"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

const (
	// defaultGRPCPort is the default grpc container port that the registry pod exposes
	defaultGRPCPort = 50051

	defaultContainerName     = "registry-grpc"
	defaultContainerPortName = "grpc"

	// The FBC directory that exists under root of an FBC container image.
	// This directory has the File-Based Catalog representation of a catalog index.
	defaultFBCIndexRootDir = "/configs"
)

// FBCRegistryPod holds resources necessary for creation of a registry pod in FBC scenarios.
type FBCRegistryPod struct { //nolint:maligned
	// BundleItems contains all bundles to be added to a registry pod.
	BundleItems []index.BundleItem

	// Index image contains a database of pointers to operator manifest content that is queriable via an API.
	// new version of an operator bundle when published can be added to an index image
	IndexImage string

	// GRPCPort is the container grpc port
	GRPCPort int32

	// pod represents a kubernetes *corev1.pod that will be created on a cluster using an index image
	pod *corev1.Pod

	// FBCContent represents the contents of the FBC file (string YAML).
	FBCContent string

	// FBCDir is the name of the FBC directory name where the FBC resides in.
	FBCDir string

	// FBCFile represents the FBC filename that has all the contents to be served through the registry pod.
	FBCFile string

	cfg *operator.Configuration
}

// init initializes the FBCRegistryPod struct.
func (f *FBCRegistryPod) init(cfg *operator.Configuration) error {
	if f.GRPCPort == 0 {
		f.GRPCPort = defaultGRPCPort
	}

	f.cfg = cfg

	// validate the FBCRegistryPod struct and ensure required fields are set
	if err := f.validate(); err != nil {
		return fmt.Errorf("invalid FBC registry pod: %v", err)
	}

	bundleImage := f.BundleItems[len(f.BundleItems)-1].ImageTag
	trimmedbundleImage := strings.Split(bundleImage, ":")[0]
	f.FBCDir = fmt.Sprintf("%s-index", filepath.Join("/tmp", strings.Split(trimmedbundleImage, "/")[2]))
	f.FBCFile = filepath.Join(f.FBCDir, strings.Split(bundleImage, ":")[1])

	// podForBundleRegistry() to make the pod definition
	pod, err := f.podForBundleRegistry()
	if err != nil {
		return fmt.Errorf("error building registry pod definition: %v", err)
	}
	f.pod = pod

	return nil
}

// Create creates a bundle registry pod built from an fbc index image,
// sets the catalog source as the owner for the pod and verifies that
// the pod is running
func (f *FBCRegistryPod) Create(ctx context.Context, cfg *operator.Configuration, cs *v1alpha1.CatalogSource) (*corev1.Pod, error) {
	if err := f.init(cfg); err != nil {
		return nil, err
	}

	// make catalog source the owner of registry pod object
	if err := controllerutil.SetOwnerReference(cs, f.pod, f.cfg.Scheme); err != nil {
		return nil, fmt.Errorf("error setting owner reference: %w", err)
	}

	if err := f.cfg.Client.Create(ctx, f.pod); err != nil {
		return nil, fmt.Errorf("error creating pod: %w", err)
	}

	// get registry pod key
	podKey := types.NamespacedName{
		Namespace: f.cfg.Namespace,
		Name:      f.pod.GetName(),
	}

	// poll and verify that pod is running
	podCheck := wait.ConditionFunc(func() (done bool, err error) {
		err = f.cfg.Client.Get(ctx, podKey, f.pod)
		if err != nil {
			return false, fmt.Errorf("error getting pod %s: %w", f.pod.Name, err)
		}
		return f.pod.Status.Phase == corev1.PodRunning, nil
	})

	// check pod status to be `Running`
	if err := f.checkPodStatus(ctx, podCheck); err != nil {
		return nil, fmt.Errorf("registry pod did not become ready: %w", err)
	}
	log.Infof("Created registry pod: %s", f.pod.Name)
	return f.pod, nil
}

// checkPodStatus polls and verifies that the pod status is running
func (f *FBCRegistryPod) checkPodStatus(ctx context.Context, podCheck wait.ConditionFunc) error {
	// poll every 200 ms until podCheck is true or context is done
	err := wait.PollImmediateUntil(200*time.Millisecond, podCheck, ctx.Done())
	if err != nil {
		return fmt.Errorf("error waiting for registry pod %s to run: %v", f.pod.Name, err)
	}

	return err
}

// validate will ensure that RegistryPod required fields are set
// and throws error if not set
func (f *FBCRegistryPod) validate() error {
	if len(f.BundleItems) == 0 {
		return errors.New("bundle image set cannot be empty")
	}
	for _, item := range f.BundleItems {
		if item.ImageTag == "" {
			return errors.New("bundle image cannot be empty")
		}
	}

	if f.IndexImage == "" {
		return errors.New("index image cannot be empty")
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
func (f *FBCRegistryPod) podForBundleRegistry() (*corev1.Pod, error) {
	// rp was already validated so len(f.BundleItems) must be greater than 0.
	bundleImage := f.BundleItems[len(f.BundleItems)-1].ImageTag

	// construct the container command for pod spec
	containerCmd, err := f.getContainerCmd()
	if err != nil {
		return nil, err
	}

	// (todo) remove comment: ConfigMap related
	// create a ConfigMap
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "operator-sdk-run-bundle-config",
			Namespace: f.cfg.Namespace,
		},
		// (todo) remove comment (rashmi/venkat):
		// can we have key as something random, and value to be the extra FBC string
		// but how do we specifically access the value in CM?
		Data: map[string]string{
			"test": "runbundle",
		},
	}

	// make the pod definition
	f.pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getPodName(bundleImage),
			Namespace: f.cfg.Namespace,
		},
		Spec: corev1.PodSpec{
			// (todo) remove comment: ConfigMap related
			Volumes: []corev1.Volume{
				{
					Name: k8sutil.TrimDNS1123Label(cm.Name + "-volume"),
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							// todo: do we have to add items like this or can we just reference the CM below?
							Items: []corev1.KeyToPath{
								{
									Key:  cm.Name,
									Path: cm.Name,
								},
							},
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cm.Name,
							},
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:  defaultContainerName,
					Image: f.IndexImage,
					Command: []string{
						"sh",
						"-c",
						containerCmd,
					},
					Ports: []corev1.ContainerPort{
						{Name: defaultContainerPortName, ContainerPort: f.GRPCPort},
					},
					// (todo) remove comment: ConfigMap related
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      k8sutil.TrimDNS1123Label(cm.Name + "-volume"),
							MountPath: path.Join(defaultFBCIndexRootDir, cm.Name),
							SubPath:   cm.Name,
						},
					},
				},
			},
			// (todo) remove comment (not configmap related).
			// InitContainer related to doing untar of the extra fbc tar.
			// InitContainers: []corev1.Container{
			// 	{
			// 		Name:            "extra-FBC-untar",
			// 		Image:           f.IndexImage, // should this be the same image as regular container?
			// 		ImagePullPolicy: corev1.PullIfNotPresent,
			// 		Args: []string{
			// 			"tar",
			// 			"xvzf",
			// 			"/configs/extrafbc.tar.gz",
			// 			"-C",
			// 			path.Join(defaultFBCIndexRootDir, cm.Name),
			// 		},
			// 		VolumeMounts: []corev1.VolumeMount{
			// 			{
			// 				MountPath: path.Join(defaultFBCIndexRootDir, cm.Name),
			// 				Name:      k8sutil.TrimDNS1123Label(cm.Name + "-volume"),
			// 			},
			// 		},
			// 	},
			// },
		},
	}

	return f.pod, nil
}

const fbcCmdTemplate = `mkdir -p {{ .FBCDir }} && \
echo '{{ .FBCContent }}' >> {{ .FBCFile  }} && \
opm serve {{ .FBCDir }} -p {{ .GRPCPort }}
`

// (todo) remove comment.
// This maybe the new container creation command for handling the extra FBC for large indexes for both ConfigMap
// InitContainer.
// const extraFBCCmdTemplate = `
// opm serve {{ .ExtraFBCDir }} -p {{ .GRPCPort }}
// `

// getContainerCmd uses templating to construct the container command
// and throws error if unable to parse and execute the container command
func (f *FBCRegistryPod) getContainerCmd() (string, error) {
	var t *template.Template
	// create a custom dirname template function
	funcMap := template.FuncMap{
		"dirname": path.Dir,
	}

	// add the custom dirname template function to the
	// template's FuncMap and parse the cmdTemplate
	t = template.Must(template.New("cmd").Funcs(funcMap).Parse(fbcCmdTemplate))

	// execute the command by applying the parsed template to command
	// and write command output to out
	out := &bytes.Buffer{}
	if err := t.Execute(out, f); err != nil {
		return "", fmt.Errorf("parse container command: %w", err)
	}

	return out.String(), nil
}
