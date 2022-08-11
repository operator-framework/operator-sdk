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
	"text/template"
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	pointer "k8s.io/utils/pointer"
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

	defaultConfigMapName = "operator-sdk-run-bundle-config"
	defaultConfigMapKey  = "extraFBC"
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

	// FBCIndexRootDir is the FBC directory that exists under root of an FBC container image.
	// This directory has the File-Based Catalog representation of a catalog index.
	FBCIndexRootDir string

	cfg *operator.Configuration
}

// init initializes the FBCRegistryPod struct.
func (f *FBCRegistryPod) init(cfg *operator.Configuration, cs *v1alpha1.CatalogSource) error {
	if f.GRPCPort == 0 {
		f.GRPCPort = defaultGRPCPort
	}

	if f.FBCIndexRootDir == "" {
		f.FBCIndexRootDir = fmt.Sprintf("/%s-configs", cs.Name)
	}

	f.cfg = cfg

	// validate the FBCRegistryPod struct and ensure required fields are set
	if err := f.validate(); err != nil {
		return fmt.Errorf("invalid FBC registry pod: %v", err)
	}

	// podForBundleRegistry() to make the pod definition
	pod, err := f.podForBundleRegistry(cs)
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
	if err := f.init(cfg, cs); err != nil {
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
func (f *FBCRegistryPod) podForBundleRegistry(cs *v1alpha1.CatalogSource) (*corev1.Pod, error) {
	// rp was already validated so len(f.BundleItems) must be greater than 0.
	bundleImage := f.BundleItems[len(f.BundleItems)-1].ImageTag

	// construct the container command for pod spec
	containerCmd, err := f.getContainerCmd()
	if err != nil {
		return nil, err
	}

	// create ConfigMap if it does not exist,
	// if it exists, then update it with new content.
	cm, err := f.createConfigMap(cs)
	if err != nil {
		return nil, fmt.Errorf("configMap error: %w", err)
	}

	// make the pod definition
	f.pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getPodName(bundleImage),
			Namespace: f.cfg.Namespace,
		},
		Spec: corev1.PodSpec{
			// DO NOT set RunAsUser and RunAsNonRoot, we must leave this empty to allow
			// those that want to use this command against Openshift vendor do not face issues.
			//
			// Why not set RunAsUser?
			// RunAsUser cannot be set because in OpenShift each namespace has a valid range like
			// [1000680000, 1000689999]. Therefore, values like 1001 will not work. Also, in OCP each namespace
			// has a valid range allocate. Therefore, by leaving it empty the OCP will adopt RunAsUser strategy
			// of MustRunAsRange. The PSA will look for the openshift.io/sa.scc.uid-range annotation
			// in the namespace to populate RunAsUser fields when the pod be admitted. Note that
			// is NOT possible to know a valid value that could be accepeted beforehand.
			//
			// Why not set RunAsNonRoot?
			// If we set RunAsNonRoot = true and the image informed does not define the UserID
			// (i.e. in the Dockerfile we have not `USER 11211:11211 `) then, the Pod will fail to run with the
			// error `"container has runAsNonRoot and image will run as root …` in ANY Kubernetes cluster.
			// (vanilla or OCP). Therefore, by leaving it empty this field will be set by OCP if/when the Pod be
			// qualified for restricted-v2 SCC policy.

			// TODO: remove when OpenShift 4.10 and Kubernetes 1.19 be no longer supported
			// Why not set SeccompProfile?
			// This option can only work in OCP versions >= 4.11 and Kubernetes versions >= 19.
			//SecurityContext: &corev1.PodSecurityContext{
			//	SeccompProfile: &corev1.SeccompProfile{
			//		Type: corev1.SeccompProfileTypeRuntimeDefault,
			//	},
			//},
			Volumes: []corev1.Volume{
				{
					Name: k8sutil.TrimDNS1123Label(cm.Name + "-volume"),
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							Items: []corev1.KeyToPath{
								{
									Key:  defaultConfigMapKey,
									Path: path.Join(defaultConfigMapName, fmt.Sprintf("%s.yaml", defaultConfigMapKey)),
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
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      k8sutil.TrimDNS1123Label(cm.Name + "-volume"),
							MountPath: path.Join(f.FBCIndexRootDir, cm.Name),
							SubPath:   cm.Name,
						},
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged:               pointer.Bool(false),
						ReadOnlyRootFilesystem:   pointer.Bool(false),
						AllowPrivilegeEscalation: pointer.Bool(false),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{"ALL"},
						},
					},
				},
			},
			ServiceAccountName: f.cfg.ServiceAccount,
		},
	}

	return f.pod, nil
}

// container creation command for FBC type images.
const fbcCmdTemplate = `opm serve {{ .FBCIndexRootDir}} -p {{ .GRPCPort }}`

// createConfigMap creates a ConfigMap if it does not exist and if it does, then update it with new content.
// Also, sets the owner reference by making CatalogSource the owner of ConfigMap object for cleanup purposes.
func (f *FBCRegistryPod) createConfigMap(cs *v1alpha1.CatalogSource) (*corev1.ConfigMap, error) {
	// new ConfigMap
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultConfigMapName,
			Namespace: f.cfg.Namespace,
		},
		Data: map[string]string{
			defaultConfigMapKey: f.FBCContent,
		},
	}

	// set owner reference by making catalog source the owner of ConfigMap object
	if err := controllerutil.SetOwnerReference(cs, cm, f.cfg.Scheme); err != nil {
		return nil, fmt.Errorf("set configmap %q owner reference: %v", cm.GetName(), err)
	}

	cmKey := types.NamespacedName{
		Namespace: f.cfg.Namespace,
		Name:      defaultConfigMapName,
	}

	// create a ConfigMap if it does not exist;
	// update it with new data if it already exists.
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		err := f.cfg.Client.Get(context.TODO(), cmKey, cm)
		if apierrors.IsNotFound(err) {
			if err := f.cfg.Client.Create(context.TODO(), cm); err != nil {
				return fmt.Errorf("error creating ConfigMap: %w", err)
			}
		}
		// update ConfigMap with new FBCContent
		cm.Data = map[string]string{defaultConfigMapKey: f.FBCContent}
		return f.cfg.Client.Update(context.TODO(), cm)
	}); err != nil {
		return nil, fmt.Errorf("error updating ConfigMap: %w", err)
	}

	return cm, nil

}

// getContainerCmd uses templating to construct the container command
// and throws error if unable to parse and execute the container command
func (f *FBCRegistryPod) getContainerCmd() (string, error) {
	// add the custom dirname template function to the
	// template's FuncMap and parse the cmdTemplate
	t := template.Must(template.New("cmd").Parse(fbcCmdTemplate))

	// execute the command by applying the parsed template to command
	// and write command output to out
	out := &bytes.Buffer{}
	if err := t.Execute(out, f); err != nil {
		return "", fmt.Errorf("parse container command: %w", err)
	}

	return out.String(), nil
}
