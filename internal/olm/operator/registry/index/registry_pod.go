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
	"path"
	"text/template"
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	pointer "k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

const (
	// defaultGRPCPort is the default grpc container port that the registry pod exposes
	defaultGRPCPort = 50051
	defaultDBPath   = "/database/index.db"

	defaultContainerName     = "registry-grpc"
	defaultContainerPortName = "grpc"
)

// BundleItem contains the metadata of a bundle image relevant to the registry pod.
type BundleItem struct {
	// ImageTag is the bundle image's tag.
	ImageTag string `json:"imageTag"`
	// AddMode describes how the bundle should be added to an index image.
	AddMode BundleAddMode `json:"mode"`
}

// SQLiteRegistryPod holds resources necessary for creation of a registry server
type SQLiteRegistryPod struct { //nolint:maligned
	// BundleItems contains all bundles to be added to a registry pod.
	BundleItems []BundleItem

	// Index image contains a database of pointers to operator manifest content that is queriable via an API.
	// new version of an operator bundle when published can be added to an index image
	IndexImage string

	// DBPath refers to the registry DB;
	// if an index image is provided, the existing registry DB is located at /database/index.db
	DBPath string

	// GRPCPort is the container grpc port
	GRPCPort int32

	// SecretName holds the name of an image pull secret to mount into the Pod so `opm registry add`
	// can pull bundle images from a private registry.
	SecretName string

	// SecretName holds the name of a secret for a CA cert file containing root certificates.
	// This file is transiently added to the registry Pod's cert pool via `opm registry add --ca-file`.
	// The secret's key for this file must be "cert.pem".
	CASecretName string

	// SkipTLSVerify represents skip TLS certificate verification for container image registries while pulling bundles.
	SkipTLSVerify bool `json:"SkipTLSVerify"`

	// UseHTTP uses plain HTTP for container image registries while pulling bundles.
	UseHTTP bool `json:"UseHTTP"`

	// SecurityContext defines the security context which will enable the
	// SecurityContext on the Pod
	SecurityContext string

	// pod represents a kubernetes *corev1.pod that will be created on a cluster using an index image
	pod *corev1.Pod

	cfg *operator.Configuration
}

// init initializes the SQLiteRegistryPod struct and sets defaults for empty fields
func (rp *SQLiteRegistryPod) init(cfg *operator.Configuration) error {
	if rp.GRPCPort == 0 {
		rp.GRPCPort = defaultGRPCPort
	}
	if rp.DBPath == "" {
		rp.DBPath = defaultDBPath
	}
	rp.cfg = cfg

	// validate the SQLiteRegistryPod struct and ensure required fields are set
	if err := rp.validate(); err != nil {
		return fmt.Errorf("invalid registry pod: %v", err)
	}

	// podForBundleRegistry() to make the pod definition
	pod, err := rp.podForBundleRegistry()
	if err != nil {
		return fmt.Errorf("error building registry pod definition: %v", err)
	}
	rp.pod = pod

	return nil
}

// Create creates a bundle registry pod built from an index image,
// sets the catalog source as the owner for the pod and verifies that
// the pod is running
func (rp *SQLiteRegistryPod) Create(ctx context.Context, cfg *operator.Configuration, cs *v1alpha1.CatalogSource) (*corev1.Pod, error) {
	if err := rp.init(cfg); err != nil {
		return nil, err
	}

	// make catalog source the owner of registry pod object
	if err := controllerutil.SetOwnerReference(cs, rp.pod, rp.cfg.Scheme); err != nil {
		return nil, fmt.Errorf("error setting owner reference: %w", err)
	}

	// Add security context if the user passed in the --security-context-config flag
	if rp.SecurityContext == "restricted" {
		rp.pod.Spec.SecurityContext = &corev1.PodSecurityContext{
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		}

		// Update the Registry Pod container security context to be restrictive
		rp.pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{
			Privileged:               pointer.To(false),
			ReadOnlyRootFilesystem:   pointer.To(false),
			AllowPrivilegeEscalation: pointer.To(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		}
	}

	if err := rp.cfg.Client.Create(ctx, rp.pod); err != nil {
		return nil, fmt.Errorf("error creating pod: %w", err)
	}

	// get registry pod key
	podKey := types.NamespacedName{
		Namespace: rp.cfg.Namespace,
		Name:      rp.pod.GetName(),
	}

	// poll and verify that pod is running
	podCheck := wait.ConditionWithContextFunc(func(pctx context.Context) (done bool, err error) {
		err = rp.cfg.Client.Get(pctx, podKey, rp.pod)
		if err != nil {
			return false, fmt.Errorf("error getting pod %s: %w", rp.pod.Name, err)
		}
		return rp.pod.Status.Phase == corev1.PodRunning, nil
	})

	// check pod status to be `Running`
	if err := rp.checkPodStatus(ctx, podCheck); err != nil {
		return nil, fmt.Errorf("registry pod did not become ready: %w", err)
	}
	log.Infof("Created registry pod: %s", rp.pod.Name)
	return rp.pod, nil
}

// checkPodStatus polls and verifies that the pod status is running
func (rp *SQLiteRegistryPod) checkPodStatus(ctx context.Context, podCheck wait.ConditionWithContextFunc) error {
	// poll every 200 ms until podCheck is true or context is done
	err := wait.PollUntilContextCancel(ctx, 200*time.Millisecond, false, podCheck)
	if err != nil {
		return fmt.Errorf("error waiting for registry pod %s to run: %v", rp.pod.Name, err)
	}

	return err
}

// validate will ensure that SQLiteRegistryPod required fields are set
// and throws error if not set
func (rp *SQLiteRegistryPod) validate() error {
	if len(rp.BundleItems) == 0 {
		return errors.New("bundle image set cannot be empty")
	}
	for _, item := range rp.BundleItems {
		if item.ImageTag == "" {
			return errors.New("bundle image cannot be empty")
		}
		if err := item.AddMode.Validate(); err != nil {
			return fmt.Errorf("invalid bundle add mode: %v", err)
		}
	}

	if rp.IndexImage == "" {
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
func (rp *SQLiteRegistryPod) podForBundleRegistry() (*corev1.Pod, error) {
	// rp was already validated so len(rp.BundleItems) must be greater than 0.
	bundleImage := rp.BundleItems[len(rp.BundleItems)-1].ImageTag

	// construct the container command for pod spec
	containerCmd, err := rp.getContainerCmd()
	if err != nil {
		return nil, err
	}

	// make the pod definition
	rp.pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getPodName(bundleImage),
			Namespace: rp.cfg.Namespace,
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
			//
			// TODO: remove when OpenShift 4.10 and Kubernetes 1.19 be no longer supported
			// Why not set SeccompProfile?
			// This option can only work in OCP versions >= 4.11 and Kubernetes versions >= 19.
			//
			// 2022-09-27 (jesusr): We added a --security-context-config flag to run bundle
			// that will add the following stanza to the pod. This will allow
			// users to selectively enable this stanza. Once this context
			// becomes the default, we should uncomment this code and remove the
			// --security-context-config flag.
			// ---- end of update comment
			//
			// SecurityContext: &corev1.PodSecurityContext{
			//     SeccompProfile: &corev1.SeccompProfile{
			//         Type: corev1.SeccompProfileTypeRuntimeDefault,
			//     },
			// },
			Containers: []corev1.Container{
				{
					Name:  defaultContainerName,
					Image: rp.IndexImage,
					Command: []string{
						"sh",
						"-c",
						containerCmd,
					},
					Ports: []corev1.ContainerPort{
						{Name: defaultContainerPortName, ContainerPort: rp.GRPCPort},
					},
					WorkingDir: "/tmp",
				},
			},
			ServiceAccountName: rp.cfg.ServiceAccount,
		},
	}

	addImagePullSecret(rp.pod, rp.SecretName)
	addCertSecret(rp.pod, rp.CASecretName)

	return rp.pod, nil
}

// addImagePullSecret creates a docker config volume for secretName
// and a volumeMount for that secret in each container in pod.
func addImagePullSecret(pod *corev1.Pod, secretName string) {
	if secretName == "" {
		return
	}

	// Require a non-legacy docker config secret.
	volume := makeSecretVolume(secretName, corev1.KeyToPath{Key: ".dockerconfigjson", Path: ".docker/config.json"})
	pod.Spec.Volumes = append(pod.Spec.Volumes, volume)

	addVolumeMountForSecret(pod, volume.Name, "/root")
}

// addCertSecret creates and mounts a volume containing a CA root certificate
// to pass to `opm registry add`.
func addCertSecret(pod *corev1.Pod, secretName string) {
	if secretName == "" {
		return
	}

	// Ensure the secret contains a key "cert.pem".
	volume := makeSecretVolume(secretName, corev1.KeyToPath{Key: "cert.pem", Path: "cert.pem"})
	pod.Spec.Volumes = append(pod.Spec.Volumes, volume)

	addVolumeMountForSecret(pod, volume.Name, "/certs")
}

func makeSecretVolume(secretName string, items ...corev1.KeyToPath) corev1.Volume {
	return corev1.Volume{
		Name: secretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  secretName,
				DefaultMode: newInt32(0400),
				Optional:    newBool(false),
				Items:       items,
			},
		},
	}
}

func addVolumeMountForSecret(pod *corev1.Pod, secretName, mountPath string) {
	volumeMount := corev1.VolumeMount{
		Name:      secretName,
		ReadOnly:  true,
		MountPath: mountPath,
	}
	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, volumeMount)
	}
}

func newInt32(i int32) *int32 {
	ip := new(int32)
	*ip = i
	return ip
}

func newBool(b bool) *bool {
	bp := new(bool)
	*bp = b
	return bp
}

const cmdTemplate = `[[ -f {{ .DBPath }} ]] && cp {{ .DBPath }} /tmp/tmp.db; \
{{- range $i, $item := .BundleItems }}
opm registry add -d /tmp/tmp.db -b {{ $item.ImageTag }} --mode={{ $item.AddMode }}{{ if $.CASecretName }} --ca-file=/certs/cert.pem{{ end }} --skip-tls-verify={{ $.SkipTLSVerify }} --use-http={{ $.UseHTTP }} && \
{{- end }}
opm registry serve -d /tmp/tmp.db -p {{ .GRPCPort }}
`

// getContainerCmd uses templating to construct the container command
// and throws error if unable to parse and execute the container command
func (rp *SQLiteRegistryPod) getContainerCmd() (string, error) {
	// create a custom dirname template function
	funcMap := template.FuncMap{
		"dirname": path.Dir,
	}

	// add the custom dirname template function to the
	// template's FuncMap and parse the cmdTemplate
	t := template.Must(template.New("cmd").Funcs(funcMap).Parse(cmdTemplate))

	// execute the command by applying the parsed template to command
	// and write command output to out
	out := &bytes.Buffer{}
	if err := t.Execute(out, rp); err != nil {
		return "", fmt.Errorf("parse container command: %w", err)
	}

	return out.String(), nil
}
