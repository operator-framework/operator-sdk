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

package v3

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	kbutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/internal/pkg"
)

// Memcached defines the Memcached Sample in GO using webhooks
type Memcached struct {
	ctx *pkg.SampleContext
}

var generateWithMonitoring bool

// GenerateSample will call all actions to create the directory and generate the sample
// Note that it should NOT be called in the e2e tests.
func GenerateSample(binaryPath, samplesPath string) {
	log.Infof("starting to generate Go memcached sample with webhooks")
	ctx, err := pkg.NewSampleContext(binaryPath, filepath.Join(samplesPath, "memcached-operator"), "GO111MODULE=on")
	pkg.CheckError("generating Go memcached with webhooks context", err)

	generateWithMonitoring = false
	if strings.HasSuffix(samplesPath, "monitoring") {
		generateWithMonitoring = true
	}

	memcached := Memcached{&ctx}
	memcached.Prepare()
	memcached.Run()
}

// Prepare the Context for the Memcached with WebHooks Go Sample
// Note that sample directory will be re-created and the context data for the sample
// will be set such as the domain and GVK.
func (mh *Memcached) Prepare() {
	log.Infof("destroying directory for Memcached with Webhooks Go samples")
	mh.ctx.Destroy()

	log.Infof("creating directory")
	err := mh.ctx.Prepare()
	pkg.CheckError("creating directory for Go Sample", err)

	log.Infof("setting domain and GVK")
	mh.ctx.Domain = "example.com"
	mh.ctx.Version = "v1alpha1"
	mh.ctx.Group = "cache"
	mh.ctx.Kind = "Memcached"
}

// Run the steps to create the Memcached with Webhooks Go Sample
func (mh *Memcached) Run() {

	if strings.Contains(mh.ctx.Dir, "v4-alpha") {
		log.Infof("creating the v4-alpha project")
		err := mh.ctx.Init(
			"--plugins", "go/v4-alpha",
			"--project-version", "3",
			"--repo", "github.com/example/memcached-operator",
			"--domain", mh.ctx.Domain)
		pkg.CheckError("creating the project", err)
	} else {
		log.Infof("creating the go/v3 project")
		err := mh.ctx.Init(
			"--plugins", "go/v3",
			"--project-version", "3",
			"--repo", "github.com/example/memcached-operator",
			"--domain", mh.ctx.Domain)
		pkg.CheckError("creating the project", err)
	}

	err := mh.ctx.CreateAPI(
		"--group", mh.ctx.Group,
		"--version", mh.ctx.Version,
		"--kind", mh.ctx.Kind,
		"--plugins", "deploy-image/v1-alpha",
		"--image", "memcached:1.4.36-alpine",
		"--image-container-command", "memcached,-m=64,-o,modern,-v",
		"--image-container-port", "11211",
		"--run-as-user", "1001",
		"--make=false",
		"--manifests=false")
	pkg.CheckError("scaffolding apis", err)

	err = mh.ctx.UncommentRestrictivePodStandards()
	pkg.CheckError("creating the bundle", err)

	log.Infof("implementing the API markers")
	mh.implementingAPIMarkers()

	log.Infof("implementing the Controller")
	mh.implementingController()

	if generateWithMonitoring {
		log.Infof("implementing Monitoring")
		mh.implementingMonitoring()

		log.Infof("customizing the Controller to include monitoring")
		mh.customizingController()

		log.Infof("customizing Main to include monitoring")
		mh.customizingMain()

		log.Infof("customizing Dockerfile to include monitoring")
		mh.customizingDockerfile()
	}

	log.Infof("scaffolding webhook")
	err = mh.ctx.CreateWebhook(
		"--group", mh.ctx.Group,
		"--version", mh.ctx.Version,
		"--kind", mh.ctx.Kind,
		"--defaulting",
		"--defaulting")
	pkg.CheckError("scaffolding webhook", err)

	mh.implementingWebhooks()

	if strings.Contains(mh.ctx.Dir, "v4-alpha") {
		mh.uncommentDefaultKustomizationV4()
		mh.uncommentManifestsKustomizationv4()
	} else {
		mh.uncommentDefaultKustomizationV3()
		mh.uncommentManifestsKustomizationv3()
	}

	mh.implementingE2ETests()

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = mh.ctx.Dir
	_, err = mh.ctx.Run(cmd)
	pkg.CheckError("Running go mod tidy", err)

	log.Infof("creating the bundle")
	err = mh.ctx.GenerateBundle()
	pkg.CheckError("creating the bundle", err)

	log.Infof("striping bundle annotations")
	err = mh.ctx.StripBundleAnnotations()
	pkg.CheckError("striping bundle annotations", err)

	pkg.CheckError("formatting project", mh.ctx.Make("fmt"))

	// Clean up built binaries, if any.
	pkg.CheckError("cleaning up", os.RemoveAll(filepath.Join(mh.ctx.Dir, "bin")))
}

// uncommentDefaultKustomizationV3 will uncomment code in config/default/kustomization.yaml
func (mh *Memcached) uncommentDefaultKustomizationV3() {
	var err error
	kustomization := filepath.Join(mh.ctx.Dir, "config", "default", "kustomization.yaml")
	log.Info("uncommenting config/default/kustomization.yaml to enable webhooks and ca injection")

	err = kbutil.UncommentCode(kustomization, "#- ../webhook", "#")
	pkg.CheckError("uncomment webhook", err)

	err = kbutil.UncommentCode(kustomization, "#- ../certmanager", "#")
	pkg.CheckError("uncomment certmanager", err)

	err = kbutil.UncommentCode(kustomization, "#- ../prometheus", "#")
	pkg.CheckError("uncomment prometheus", err)

	err = kbutil.UncommentCode(kustomization, "#- manager_webhook_patch.yaml", "#")
	pkg.CheckError("uncomment manager_webhook_patch.yaml", err)

	err = kbutil.UncommentCode(kustomization, "#- webhookcainjection_patch.yaml", "#")
	pkg.CheckError("uncomment webhookcainjection_patch.yaml", err)

	err = kbutil.UncommentCode(kustomization,
		`#- name: CERTIFICATE_NAMESPACE # namespace of the certificate CR
#  objref:
#    kind: Certificate
#    group: cert-manager.io
#    version: v1
#    name: serving-cert # this name should match the one in certificate.yaml
#  fieldref:
#    fieldpath: metadata.namespace
#- name: CERTIFICATE_NAME
#  objref:
#    kind: Certificate
#    group: cert-manager.io
#    version: v1
#    name: serving-cert # this name should match the one in certificate.yaml
#- name: SERVICE_NAMESPACE # namespace of the service
#  objref:
#    kind: Service
#    version: v1
#    name: webhook-service
#  fieldref:
#    fieldpath: metadata.namespace
#- name: SERVICE_NAME
#  objref:
#    kind: Service
#    version: v1
#    name: webhook-service`, "#")
	pkg.CheckError("uncommented certificate CR", err)
}

// uncommentDefaultKustomizationV3 will uncomment code in config/default/kustomization.yaml
func (mh *Memcached) uncommentDefaultKustomizationV4() {
	var err error
	kustomization := filepath.Join(mh.ctx.Dir, "config", "default", "kustomization.yaml")
	log.Info("uncommenting config/default/kustomization.yaml to enable webhooks and ca injection")

	err = kbutil.UncommentCode(kustomization, "#- ../webhook", "#")
	pkg.CheckError("uncomment webhook", err)

	err = kbutil.UncommentCode(kustomization, "#- ../certmanager", "#")
	pkg.CheckError("uncomment certmanager", err)

	err = kbutil.UncommentCode(kustomization, "#- ../prometheus", "#")
	pkg.CheckError("uncomment prometheus", err)

	err = kbutil.UncommentCode(kustomization, "#- manager_webhook_patch.yaml", "#")
	pkg.CheckError("uncomment manager_webhook_patch.yaml", err)

	err = kbutil.UncommentCode(kustomization, "#- webhookcainjection_patch.yaml", "#")
	pkg.CheckError("uncomment webhookcainjection_patch.yaml", err)

	err = kbutil.UncommentCode(kustomization,
		`#replacements:
#  - source: # Add cert-manager annotation to ValidatingWebhookConfiguration, MutatingWebhookConfiguration and CRDs
#      kind: Certificate
#      group: cert-manager.io
#      version: v1
#      name: serving-cert # this name should match the one in certificate.yaml
#      fieldPath: .metadata.namespace # namespace of the certificate CR
#    targets:
#      - select:
#          kind: ValidatingWebhookConfiguration
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 0
#          create: true
#      - select:
#          kind: MutatingWebhookConfiguration
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 0
#          create: true
#      - select:
#          kind: CustomResourceDefinition
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 0
#          create: true
#  - source:
#      kind: Certificate
#      group: cert-manager.io
#      version: v1
#      name: serving-cert # this name should match the one in certificate.yaml
#      fieldPath: .metadata.name
#    targets:
#      - select:
#          kind: ValidatingWebhookConfiguration
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 1
#          create: true
#      - select:
#          kind: MutatingWebhookConfiguration
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 1
#          create: true
#      - select:
#          kind: CustomResourceDefinition
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 1
#          create: true
#  - source: # Add cert-manager annotation to the webhook Service
#      kind: Service
#      version: v1
#      name: webhook-service
#      fieldPath: .metadata.name # namespace of the service
#    targets:
#      - select:
#          kind: Certificate
#          group: cert-manager.io
#          version: v1
#        fieldPaths:
#          - .spec.dnsNames.0
#          - .spec.dnsNames.1
#        options:
#          delimiter: '.'
#          index: 0
#          create: true
#  - source:
#      kind: Service
#      version: v1
#      name: webhook-service
#      fieldPath: .metadata.namespace # namespace of the service
#    targets:
#      - select:
#          kind: Certificate
#          group: cert-manager.io
#          version: v1
#        fieldPaths:
#          - .spec.dnsNames.0
#          - .spec.dnsNames.1
#        options:
#          delimiter: '.'
#          index: 1
#          create: true`, "#")
	pkg.CheckError("uncommented certificate CR", err)
}

// uncommentManifestsKustomization will uncomment code in config/manifests/kustomization.yaml
func (mh *Memcached) uncommentManifestsKustomizationv3() {
	var err error
	kustomization := filepath.Join(mh.ctx.Dir, "config", "manifests", "kustomization.yaml")
	log.Info("uncommenting config/manifests/kustomization.yaml to enable webhooks in OLM")

	err = kbutil.UncommentCode(kustomization,
		`#patchesJson6902:
#- target:
#    group: apps
#    version: v1
#    kind: Deployment
#    name: controller-manager
#    namespace: system
#  patch: |-
#    # Remove the manager container's "cert" volumeMount, since OLM will create and mount a set of certs.
#    # Update the indices in this path if adding or removing containers/volumeMounts in the manager's Deployment.
#    - op: remove
#      path: /spec/template/spec/containers/1/volumeMounts/0
#    # Remove the "cert" volume, since OLM will create and mount a set of certs.
#    # Update the indices in this path if adding or removing volumes in the manager's Deployment.
#    - op: remove
#      path: /spec/template/spec/volumes/0`, "#")
	pkg.CheckError("uncommented webhook volume removal patch", err)
}

// uncommentManifestsKustomization will uncomment code in config/manifests/kustomization.yaml
func (mh *Memcached) uncommentManifestsKustomizationv4() {
	var err error
	kustomization := filepath.Join(mh.ctx.Dir, "config", "manifests", "kustomization.yaml")
	log.Info("uncommenting config/manifests/kustomization.yaml to enable webhooks in OLM")

	err = kbutil.UncommentCode(kustomization,
		`#patchesJson6902:
#- target:
#    group: apps
#    version: v1
#    kind: Deployment
#    name: controller-manager
#    namespace: system
#  patch: |-
#    # Remove the manager container's "cert" volumeMount, since OLM will create and mount a set of certs.
#    # Update the indices in this path if adding or removing containers/volumeMounts in the manager's Deployment.
#    - op: remove

#      path: /spec/template/spec/containers/0/volumeMounts/0
#    # Remove the "cert" volume, since OLM will create and mount a set of certs.
#    # Update the indices in this path if adding or removing volumes in the manager's Deployment.
#    - op: remove
#      path: /spec/template/spec/volumes/0`, "#")
	pkg.CheckError("uncommented webhook volume removal patch", err)
}

// implementingWebhooks will customize the kind wekbhok file
func (mh *Memcached) implementingWebhooks() {
	log.Infof("implementing webhooks")
	webhookPath := filepath.Join(mh.ctx.Dir, "api", mh.ctx.Version, fmt.Sprintf("%s_webhook.go",
		strings.ToLower(mh.ctx.Kind)))

	// Add webhook methods
	err := kbutil.InsertCode(webhookPath,
		"// TODO(user): fill in your defaulting logic.\n}",
		webhooksFragment)
	pkg.CheckError("replacing webhook validate implementation", err)

	err = kbutil.ReplaceInFile(webhookPath,
		"// TODO(user): fill in your defaulting logic.", "if r.Spec.Size == 0 {\n\t\tr.Spec.Size = 3\n\t}")
	pkg.CheckError("replacing webhook default implementation", err)

	// Add imports
	err = kbutil.InsertCode(webhookPath,
		"import (",
		// TODO(estroz): remove runtime dep when --programmatic-validation is added to `ccreate webhook` above.
		"\"errors\"\n\n\"k8s.io/apimachinery/pkg/runtime\"")
	pkg.CheckError("adding imports", err)
}

// implementingController will customize the Controller
func (mh *Memcached) implementingController() {
	controllerPath := filepath.Join(mh.ctx.Dir, "controllers", fmt.Sprintf("%s_controller.go",
		strings.ToLower(mh.ctx.Kind)))

	err := kbutil.InsertCode(controllerPath,
		`						SecurityContext: &corev1.SecurityContext{`, userIDWarningFragment)
	pkg.CheckError("adding warning comment for UserID", err)

	err = kbutil.InsertCode(controllerPath,
		`							RunAsNonRoot:             &[]bool{true}[0],`, runAsUserCommentFragment)
	pkg.CheckError("adding comment regarding RunAsUser field in Security Context", err)
}

// customizingController will customize the Controller to include monitoring
func (mh *Memcached) customizingController() {
	controllerPath := filepath.Join(mh.ctx.Dir, "controllers", fmt.Sprintf("%s_controller.go",
		strings.ToLower(mh.ctx.Kind)))

	err := kbutil.InsertCode(controllerPath,
		`	if *found.Spec.Replicas != size {`,
		`
		// Increment MemcachedDeploymentSizeUndesiredCountTotal metric by 1
		monitoring.MemcachedDeploymentSizeUndesiredCountTotal.Inc()`)
	pkg.CheckError("adding metric incrementation", err)

	err = kbutil.InsertCode(controllerPath,
		`cachev1alpha1 "github.com/example/memcached-operator/api/v1alpha1"`,
		monitoringImportFragment)
	pkg.CheckError("adding monitoring import", err)
}

// nolint:gosec
// implementingAPI will customize the API
func (mh *Memcached) implementingAPIMarkers() {
	err := kbutil.InsertCode(
		filepath.Join(mh.ctx.Dir, "api", mh.ctx.Version, fmt.Sprintf("%s_types.go", strings.ToLower(mh.ctx.Kind))),
		"// Port defines the port that will be used to init the container with the image",
		`
	// +operator-sdk:csv:customresourcedefinitions:type=spec`)
	pkg.CheckError("inserting Port spec marker", err)

	err = kbutil.InsertCode(
		filepath.Join(mh.ctx.Dir, "api", mh.ctx.Version, fmt.Sprintf("%s_types.go", strings.ToLower(mh.ctx.Kind))),
		"// +kubebuilder:validation:ExclusiveMaximum=false",
		`
	// +operator-sdk:csv:customresourcedefinitions:type=spec`)
	pkg.CheckError("inserting spec Status", err)

	log.Infof("implementing MemcachedStatus marker")
	err = kbutil.ReplaceInFile(
		filepath.Join(mh.ctx.Dir, "api", mh.ctx.Version, fmt.Sprintf("%s_types.go", strings.ToLower(mh.ctx.Kind))),
		`	// For further information see: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
`,
		`	// For further information see: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// Conditions store the status conditions of the Memcached instances
	// +operator-sdk:csv:customresourcedefinitions:type=status`,
	)
	pkg.CheckError("inserting Status marker", err)

	err = kbutil.ReplaceInFile(
		filepath.Join(mh.ctx.Dir, "api", mh.ctx.Version, fmt.Sprintf("%s_types.go", strings.ToLower(mh.ctx.Kind))),
		`
	// Size defines the number of Memcached instances
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3
	// +kubebuilder:validation:ExclusiveMaximum=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec`,
		`
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3
	// +kubebuilder:validation:ExclusiveMaximum=false
	
	// Size defines the number of Memcached instances
	// +operator-sdk:csv:customresourcedefinitions:type=spec`,
	)
	pkg.CheckError("updating Size spec marker", err)

	// Add CSV marker that shows CRD owned resources
	err = kbutil.InsertCode(
		filepath.Join(mh.ctx.Dir, "api", mh.ctx.Version, fmt.Sprintf("%s_types.go", strings.ToLower(mh.ctx.Kind))),
		`//+kubebuilder:subresource:status`,
		`
// +operator-sdk:csv:customresourcedefinitions:resources={{Deployment,v1,memcached-deployment}}`)
	pkg.CheckError("inserting CRD owned resources CSV marker", err)
}

// implementingMonitoring will customize monitoring
func (mh *Memcached) implementingMonitoring() {
	// Create monitoring directory
	err := os.Mkdir(filepath.Join(mh.ctx.Dir, "monitoring"), os.ModePerm)
	pkg.CheckError("creating monitoring directory", err)

	// Create metrics file
	_, err = os.Create(filepath.Join(mh.ctx.Dir, "monitoring/metrics.go"))
	pkg.CheckError("creating metrics file", err)

	metricsPath := filepath.Join(mh.ctx.Dir, "monitoring/metrics.go")

	// Add imports
	err = kbutil.InsertCode(metricsPath,
		"",
		metricsImportsFragment)
	pkg.CheckError("adding imports", err)

	// Add metrics implementation
	err = kbutil.InsertCode(metricsPath,
		")",
		metricsFragment)
	pkg.CheckError("adding metrics content", err)
}

// customizingMain will customize main.go to register metrics
func (mh *Memcached) customizingMain() {
	mainPath := filepath.Join(mh.ctx.Dir, "main.go")

	// Add monitoring import
	err := kbutil.InsertCode(mainPath,
		`"github.com/example/memcached-operator/controllers"`,
		monitoringImportFragment)
	pkg.CheckError("adding monitoring import", err)

	// Add metrics registry
	err = kbutil.InsertCode(mainPath,
		"utilruntime.Must(cachev1alpha1.AddToScheme(scheme))",
		mainMonitoringRegisterMetricsFragment)
	pkg.CheckError("adding metrics registry", err)
}

// customizingDockerfile will customize the Dockerfile to include monitoring
func (mh *Memcached) customizingDockerfile() {
	dockerfilePath := filepath.Join(mh.ctx.Dir, "Dockerfile")

	// Copy monitoring
	err := kbutil.InsertCode(dockerfilePath,
		"COPY controllers/ controllers/",
		"\nCOPY monitoring/ monitoring/")
	pkg.CheckError("adding COPY monitoring/", err)
}

const metricsFragment = `
var (
	// MemcachedDeploymentSizeUndesiredCountTotal will count how many times was required
	// to perform the operation to ensure that the number of replicas on the cluster
	// is the same as the quantity desired and specified via the custom resource size spec.
	MemcachedDeploymentSizeUndesiredCountTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "memcached_deployment_size_undesired_count_total",
			Help: "Total number of times the deployment size was not as desired.",
		},
	)
)
// Register metrics with the global prometheus registry
func RegisterMetrics() {
	metrics.Registry.MustRegister(MemcachedDeploymentSizeUndesiredCountTotal)
}
`

const metricsImportsFragment = `
package monitoring
import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)
`

const monitoringImportFragment = `
	"github.com/example/memcached-operator/monitoring"
`

const mainMonitoringRegisterMetricsFragment = `

	monitoring.RegisterMetrics()`

const webhooksFragment = `
// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-cache-example-com-v1alpha1-memcached,mutating=false,failurePolicy=fail,sideEffects=None,groups=cache.example.com,resources=memcacheds,verbs=create;update,versions=v1alpha1,name=vmemcached.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Memcached{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Memcached) ValidateCreate() error {
	memcachedlog.Info("validate create", "name", r.Name)

	return validateOdd(r.Spec.Size)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Memcached) ValidateUpdate(old runtime.Object) error {
	memcachedlog.Info("validate update", "name", r.Name)

	return validateOdd(r.Spec.Size)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Memcached) ValidateDelete() error {
	memcachedlog.Info("validate delete", "name", r.Name)

	return nil
}
func validateOdd(n int32) error {
	if n%2 == 0 {
		return errors.New("Cluster size must be an odd number")
	}
	return nil
}
`

const userIDWarningFragment = `
							// WARNING: Ensure that the image used defines an UserID in the Dockerfile
							// otherwise the Pod will not run and will fail with "container has runAsNonRoot and image has non-numeric user"".
							// If you want your workloads admitted in namespaces enforced with the restricted mode in OpenShift/OKD vendors
							// then, you MUST ensure that the Dockerfile defines a User ID OR you MUST leave the "RunAsNonRoot" and
							// "RunAsUser" fields empty.`

const runAsUserCommentFragment = `
							// The memcached image does not use a non-zero numeric user as the default user.
							// Due to RunAsNonRoot field being set to true, we need to force the user in the
							// container to a non-zero numeric user. We do this using the RunAsUser field.
							// However, if you are looking to provide solution for K8s vendors like OpenShift
							// be aware that you cannot run under its restricted-v2 SCC if you set this value.`
