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
	kbutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/internal/pkg"
)

// monitoringString is appended to logs and error messages to signify the inclusion of monitoring
const monitoringString = " and monitoring"

// Memcached defines the Memcached Sample in GO using webhooks or webhooks and monitoring code
type Memcached struct {
	ctx *pkg.SampleContext
}

var generateWithMonitoring bool
var goFilesHeader string
var prometheusAPIVersion = "v0.59.0"

// GenerateSample will call all actions to create the directory and generate the sample
// Note that it should NOT be called in the e2e tests.
func GenerateSample(binaryPath, samplesPath string) {
	generateWithMonitoring = strings.HasSuffix(samplesPath, "monitoring")

	logInfo := "starting to generate Go memcached sample with webhooks"
	errorInfo := "generating Go memcached with context: webhooks"

	if generateWithMonitoring {
		logInfo += monitoringString
		errorInfo += monitoringString
	}

	log.Info(logInfo)
	ctx, err := pkg.NewSampleContext(binaryPath, filepath.Join(samplesPath, "memcached-operator"), "GO111MODULE=on")
	pkg.CheckError(errorInfo, err)

	memcached := Memcached{&ctx}
	memcached.Prepare()
	memcached.Run()
}

// Prepare the Context for the Memcached with webhooks or with webhooks and monitoring Go Sample
// Note that sample directory will be re-created and the context data for the sample
// will be set such as the domain and GVK.
func (mh *Memcached) Prepare() {
	logInfo := "destroying directory for Go Memcached sample with webhooks"

	if generateWithMonitoring {
		logInfo += monitoringString
	}

	log.Info(logInfo)
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

// Run the steps to create the Memcached with webhooks or with webhooks and monitoring Go Sample
func (mh *Memcached) Run() {

	log.Infof("creating the go/v3 project")
	err := mh.ctx.Init(
		"--plugins", "go/v4",
		"--project-version", "3",
		"--repo", "github.com/example/memcached-operator",
		"--domain", mh.ctx.Domain)
	pkg.CheckError("creating the project", err)

	err = mh.ctx.CreateAPI(
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
		goGetcmd := exec.Command("go", "get", fmt.Sprintf("%s@%s", "github.com/prometheus-operator/prometheus-operator", prometheusAPIVersion))
		goGetcmd.Dir = mh.ctx.Dir
		if _, err := mh.ctx.Run(goGetcmd); err != nil {
			pkg.CheckError("error getting prometheus dependency", err)
		}

		log.Infof("implementing Monitoring")
		mh.implementingMonitoring()
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

	mh.uncommentDefaultKustomizationV4()
	mh.uncommentManifestsKustomizationv4()

	mh.customizingMain()

	mh.implementingE2ETests()

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = mh.ctx.Dir
	_, err = mh.ctx.Run(cmd)
	pkg.CheckError("Running go mod tidy", err)

	if generateWithMonitoring {
		cmd := exec.Command("make", "generate-metricsdocs")
		cmd.Dir = mh.ctx.Dir
		_, err = mh.ctx.Run(cmd)
		pkg.CheckError("Running make generate-metricsdocs", err)
	}

	log.Infof("creating the bundle")
	err = mh.ctx.GenerateBundle()
	pkg.CheckError("creating the bundle", err)

	log.Infof("setting createdAt annotation")
	csv := filepath.Join(mh.ctx.Dir, "bundle", "manifests", mh.ctx.ProjectName+".clusterserviceversion.yaml")
	err = kbutil.ReplaceRegexInFile(csv, "createdAt:.*", createdAt)
	pkg.CheckError("setting createdAt annotation", err)

	log.Infof("stripping bundle annotations")
	err = mh.ctx.StripBundleAnnotations()
	pkg.CheckError("stripping bundle annotations", err)

	pkg.CheckError("formatting project", mh.ctx.Make("fmt"))

	// Clean up built binaries, if any.
	pkg.CheckError("cleaning up", os.RemoveAll(filepath.Join(mh.ctx.Dir, "bin")))

	// TODO: remove when this is fixed
	// Update the test make target to properly shell out
	// to the go list command
	pkg.CheckError("fixing \"test\" make target", kbutil.ReplaceInFile(filepath.Join(mh.ctx.Dir, "Makefile"),
		"KUBEBUILDER_ASSETS=\"$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)\"  go test $(go list ./... | grep -v /test/) -coverprofile cover.out",
		"KUBEBUILDER_ASSETS=\"$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)\"  go test $(shell go list ./... | grep -v /test/) -coverprofile cover.out"))

}

// uncommentDefaultKustomizationV4 will uncomment code in config/default/kustomization.yaml
func (mh *Memcached) uncommentDefaultKustomizationV4() {
	var err error
	kustomization := filepath.Join(mh.ctx.Dir, "config", "default", "kustomization.yaml")
	log.Info("uncommenting config/default/kustomization.yaml to enable webhooks and ca injection")

	err = kbutil.UncommentCode(kustomization, "#- ../certmanager", "#")
	pkg.CheckError("uncomment certmanager", err)

	err = kbutil.UncommentCode(kustomization, "#- ../prometheus", "#")
	pkg.CheckError("uncomment prometheus", err)

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
func (mh *Memcached) uncommentManifestsKustomizationv4() {
	var err error
	kustomization := filepath.Join(mh.ctx.Dir, "config", "manifests", "kustomization.yaml")
	log.Info("uncommenting config/manifests/kustomization.yaml to enable webhooks in OLM")

	err = kbutil.UncommentCode(kustomization,
		`#patches:
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
		"\"errors\"\n\n\"k8s.io/apimachinery/pkg/runtime\"\n\n\"sigs.k8s.io/controller-runtime/pkg/webhook/admission\"")
	pkg.CheckError("adding imports", err)
}

// implementingController will customizations in the Controller
func (mh *Memcached) implementingController() {
	controllerPath := filepath.Join(mh.ctx.Dir, "internal", "controller", fmt.Sprintf("%s_controller.go",
		strings.ToLower(mh.ctx.Kind)))

	err := kbutil.InsertCode(controllerPath,
		`						SecurityContext: &corev1.SecurityContext{`, userIDWarningFragment)
	pkg.CheckError("adding warning comment for UserID", err)

	err = kbutil.InsertCode(controllerPath,
		`							RunAsNonRoot:             &[]bool{true}[0],`, runAsUserCommentFragment)
	pkg.CheckError("adding comment regarding RunAsUser field in Security Context", err)
}

// implementingMonitoring will customize monitoring
func (mh *Memcached) implementingMonitoring() {
	err := os.Mkdir(filepath.Join(mh.ctx.Dir, "monitoring"), os.ModePerm)
	pkg.CheckError("creating monitoring directory", err)

	header, err := os.ReadFile((filepath.Join(mh.ctx.Dir, "hack", "boilerplate.go.txt")))
	goFilesHeader = string(header)
	pkg.CheckError("reading go files header", err)

	log.Infof("implementing metrics")
	mh.implementingMetrics()

	log.Infof("implementing alerts")
	mh.implementingAlerts()

	log.Infof("implementing prom-rule-ci")
	mh.implementingPromRuleCi()

	log.Infof("implementing runbooks")
	mh.implementingRunbooks()

	log.Infof("implementing prometheus RBAC")
	mh.implementingPrometheusRBAC()

	log.Infof("customizing the Controller")
	mh.customizingController()

	log.Infof("customizing Main")
	mh.customizingMainMonitoring()

	log.Infof("customizing Dockerfile")
	mh.customizingDockerfile()

	log.Infof("customizing Makefile")
	mh.customizingMakefile()
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
		`// +kubebuilder:subresource:status`,
		`
// +operator-sdk:csv:customresourcedefinitions:resources={{Deployment,v1,memcached-deployment}}`)
	pkg.CheckError("inserting CRD owned resources CSV marker", err)
}

func (mh *Memcached) implementingMetrics() {
	// Create metrics file
	metricsPath := filepath.Join(mh.ctx.Dir, "monitoring/metrics.go")
	_, err := os.Create(metricsPath)
	pkg.CheckError("creating metrics file", err)

	// Add go files header
	err = kbutil.InsertCode(metricsPath,
		"",
		goFilesHeader)
	pkg.CheckError("adding go files header", err)

	// Add metrics implementation
	err = kbutil.InsertCode(metricsPath,
		goFilesHeader,
		metricsFragment)
	pkg.CheckError("adding metrics content", err)

	// Add metricsdocs directory
	err = os.Mkdir(filepath.Join(mh.ctx.Dir, "monitoring/metricsdocs"), os.ModePerm)
	pkg.CheckError("creating metricsdocs directory", err)

	// Create metricsdocs file
	metricsdocsPath := filepath.Join(mh.ctx.Dir, "monitoring/metricsdocs/metricsdocs.go")
	_, err = os.Create(metricsdocsPath)
	pkg.CheckError("creating metricsdocs file", err)

	// Add go files header
	err = kbutil.InsertCode(metricsdocsPath,
		"",
		goFilesHeader)
	pkg.CheckError("adding go files header", err)

	// Create metricsdocs generator tool
	err = kbutil.InsertCode(metricsdocsPath,
		goFilesHeader,
		metricsdocsFragment)
	pkg.CheckError("creating metricsdocs generator tool", err)

	// Create docs directory
	err = os.Mkdir(filepath.Join(mh.ctx.Dir, "docs"), os.ModePerm)
	pkg.CheckError("creating docs directory", err)

	// Create monitoring directory
	err = os.Mkdir(filepath.Join(mh.ctx.Dir, "docs/monitoring"), os.ModePerm)
	pkg.CheckError("creating monitoring directory", err)
}

func (mh *Memcached) implementingAlerts() {
	// Create alerts file
	alertsPath := filepath.Join(mh.ctx.Dir, "monitoring/alerts.go")
	_, err := os.Create(alertsPath)
	pkg.CheckError("creating alerts file", err)

	// Add go files header
	err = kbutil.InsertCode(alertsPath,
		"",
		goFilesHeader)
	pkg.CheckError("adding go files header", err)

	// Add alerts implementation
	err = kbutil.InsertCode(alertsPath,
		goFilesHeader,
		alertsFragment)
	pkg.CheckError("adding alerts content", err)
}

func (mh *Memcached) implementingPromRuleCi() {
	// Create prom-rule-ci directory
	err := os.Mkdir(filepath.Join(mh.ctx.Dir, "monitoring/prom-rule-ci"), os.ModePerm)
	pkg.CheckError("creating prom-rule-ci directory", err)

	// Create prom-rules-tests file
	promRuleTestsPath := filepath.Join(mh.ctx.Dir, "monitoring/prom-rule-ci/prom-rules-tests.yaml")
	_, err = os.Create(promRuleTestsPath)
	pkg.CheckError("creating prom-rules-tests file", err)

	// Add prom-rules-tests implementation
	err = kbutil.InsertCode(promRuleTestsPath,
		"",
		promRuleTestsFragment)
	pkg.CheckError("adding prom-rules-tests content", err)

	// Create rule-spec-dumper file
	ruleSpecDumperPath := filepath.Join(mh.ctx.Dir, "monitoring/prom-rule-ci/rule-spec-dumper.go")
	_, err = os.Create(ruleSpecDumperPath)
	pkg.CheckError("creating rule-spec-dumper file", err)

	// Add go files header to rule-spec-dumper
	err = kbutil.InsertCode(ruleSpecDumperPath,
		"",
		goFilesHeader)
	pkg.CheckError("adding go files header", err)

	// Add rule-spec-dumper implementation
	err = kbutil.InsertCode(ruleSpecDumperPath,
		goFilesHeader,
		ruleSpecDumperFragment)
	pkg.CheckError("adding rule-spec-dumper content", err)

	// Create verify-rules file
	verifyRulesPath := filepath.Join(mh.ctx.Dir, "monitoring/prom-rule-ci/verify-rules.sh")
	_, err = os.Create(verifyRulesPath)
	pkg.CheckError("creating verify-rules file", err)

	err = os.Chmod(filepath.Join(mh.ctx.Dir, "monitoring/prom-rule-ci/verify-rules.sh"), 0777)
	pkg.CheckError("changing verify-rules file permissions ", err)

	// Add verify-rules implementation
	err = kbutil.InsertCode(verifyRulesPath,
		"",
		verifyRulesFragment)
	pkg.CheckError("adding verify-rules content", err)
}

func (mh *Memcached) implementingRunbooks() {
	runbooksPath := "docs/monitoring/runbooks/"

	// Create runbooks directory
	err := os.Mkdir(filepath.Join(mh.ctx.Dir, runbooksPath), os.ModePerm)
	pkg.CheckError("creating runbooks directory", err)

	// Create MemcachedDeploymentSizeUndesired runbook file
	memcachedDeploymentSizeUndesiredRunbookPath := filepath.Join(mh.ctx.Dir, runbooksPath, "memcachedDeploymentSizeUndesired.md")
	_, err = os.Create(memcachedDeploymentSizeUndesiredRunbookPath)
	pkg.CheckError("creating MemcachedDeploymentSizeUndesired runbook file", err)

	// Add MemcachedDeploymentSizeUndesired runbook content
	err = kbutil.InsertCode(memcachedDeploymentSizeUndesiredRunbookPath,
		"",
		memcachedDeploymentSizeUndesiredRunbookFragment)
	pkg.CheckError("adding MemcachedDeploymentSizeUndesired runbook content", err)

	// Create MemcachedOperatorDown runbook file
	memcachedOperatorDownRunbookPath := filepath.Join(mh.ctx.Dir, runbooksPath, "memcachedOperatorDown.md")
	_, err = os.Create(memcachedOperatorDownRunbookPath)
	pkg.CheckError("creating MemcachedOperatorDown runbook file", err)

	// Add MemcachedOperatorDown runbook content
	err = kbutil.InsertCode(memcachedOperatorDownRunbookPath,
		"",
		memcachedOperatorDownRunbookFragment)
	pkg.CheckError("adding MemcachedDeploymentSizeUndesired runbook content", err)
}

func (mh *Memcached) implementingPrometheusRBAC() {
	// Create prometheus role file
	prometheusRolePath := filepath.Join(mh.ctx.Dir, "config/rbac/prometheus_role.yaml")
	_, err := os.Create(prometheusRolePath)
	pkg.CheckError("creating prometheus role file", err)

	// Add prometheus role content
	err = kbutil.InsertCode(prometheusRolePath,
		"",
		prometheusRoleFragment)
	pkg.CheckError("adding prometheus role content", err)

	// Create prometheus role binding file
	prometheusRoleBindingPath := filepath.Join(mh.ctx.Dir, "config/rbac/prometheus_role_binding.yaml")
	_, err = os.Create(prometheusRoleBindingPath)
	pkg.CheckError("creating prometheus role binding file", err)

	// Add prometheus role binding content
	err = kbutil.InsertCode(prometheusRoleBindingPath,
		"",
		prometheusRoleBindingFragment)
	pkg.CheckError("adding prometheus role binding content", err)

	// Add prometheus rbac files to kustomization
	kustomizationPath := filepath.Join(mh.ctx.Dir, "config/rbac/kustomization.yaml")
	err = kbutil.InsertCode(kustomizationPath,
		`- leader_election_role.yaml
- leader_election_role_binding.yaml`,
		`
- prometheus_role.yaml
- prometheus_role_binding.yaml`)
	pkg.CheckError("adding prometheus role binding content", err)
}

// customizingController will customize the Controller to include monitoring
func (mh *Memcached) customizingController() {
	controllerPath := filepath.Join(mh.ctx.Dir, "internal", "controller", fmt.Sprintf("%s_controller.go",
		strings.ToLower(mh.ctx.Kind)))

	// Add monitoring imports
	err := kbutil.InsertCode(controllerPath,
		`"os"`,
		`
	"reflect"`)
	pkg.CheckError("adding reflect import", err)

	err = kbutil.InsertCode(controllerPath,
		`"sigs.k8s.io/controller-runtime/pkg/log"`,
		monitoringv1ImportFragment)
	pkg.CheckError("adding monitoringv1 import", err)

	err = kbutil.InsertCode(controllerPath,
		`cachev1alpha1 "github.com/example/memcached-operator/api/v1alpha1"`,
		monitoringImportFragment)
	pkg.CheckError("adding monitoring import", err)

	// Add monitoring parts
	err = kbutil.InsertCode(controllerPath,
		`const memcachedFinalizer = "cache.example.com/finalizer"`,
		`
		const ruleName = "memcached-operator-rules"
		const namespace = "memcached-operator-system"`)
	pkg.CheckError("adding monitoring constants", err)

	err = kbutil.InsertCode(controllerPath,
		`// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch`,
		`
		// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch;create;update;delete`)
	pkg.CheckError("adding monitoring.coreos.com rbac", err)

	err = kbutil.ReplaceInFile(controllerPath,
		controllerMemcachedInstanceFragment,
		controllerPrometheusRuleFragment)
	pkg.CheckError("adding prometheus rule reconciliation", err)

	err = kbutil.InsertCode(controllerPath,
		`	if *found.Spec.Replicas != size {`,
		`
		// Increment MemcachedDeploymentSizeUndesiredCountTotal metric by 1
		monitoring.MemcachedDeploymentSizeUndesiredCountTotal.Inc()`)
	pkg.CheckError("adding metric incrementation", err)
}

// customizingMain will add comments to main
func (mh *Memcached) customizingMain() {
	mainPath := filepath.Join(mh.ctx.Dir, "cmd", "main.go")

	err := kbutil.InsertCode(mainPath,
		"Scheme:   mgr.GetScheme(),",
		mainRecorderFragment)
	pkg.CheckError("adding recorder fragment", err)
}

// customizingMainMonitoring will customize main.go to register metrics
func (mh *Memcached) customizingMainMonitoring() {
	mainPath := filepath.Join(mh.ctx.Dir, "cmd", "main.go")
	marker := "\"github.com/example/memcached-operator/internal/controller\""

	err := kbutil.InsertCode(mainPath,
		marker,
		monitoringImportFragment)
	pkg.CheckError("adding monitoringv1 import", err)

	// Add monitoring imports
	err = kbutil.InsertCode(mainPath,
		`"sigs.k8s.io/controller-runtime/pkg/log/zap"`,
		monitoringv1ImportFragment)
	pkg.CheckError("adding monitoringv1 import", err)

	// Add monitoring parts
	err = kbutil.InsertCode(mainPath,
		"utilruntime.Must(cachev1alpha1.AddToScheme(scheme))",
		mainMonitoringFragment)
	pkg.CheckError("adding monitoring parts", err)
}

// customizingDockerfile will customize the Dockerfile to include monitoring
func (mh *Memcached) customizingDockerfile() {
	dockerfilePath := filepath.Join(mh.ctx.Dir, "Dockerfile")

	// Copy monitoring
	ctrlCopy := "internal/controller/"

	err := kbutil.InsertCode(dockerfilePath,
		fmt.Sprintf("COPY %s %s", ctrlCopy, ctrlCopy),
		"\nCOPY monitoring/ monitoring/")
	pkg.CheckError("adding COPY monitoring/", err)
}

const createdAt = `createdAt: "2022-11-08T17:26:37Z"`

// customizingMakefile will customize the Makefile to include monitoring
func (mh *Memcached) customizingMakefile() {
	makefilePath := filepath.Join(mh.ctx.Dir, "Makefile")

	// TODO: update this to be different based on go plugin version
	// Add prom-rule-ci target to the makefile
	err := kbutil.InsertCode(makefilePath,
		`$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -`,
		makefileFragment)
	pkg.CheckError("adding prom-rule-ci target to the makefile", err)

	// Add metrics documentation
	err = kbutil.InsertCode(makefilePath,
		`$(MAKE) docker-push IMG=$(CATALOG_IMG)`,
		metricsdocsMakefileFragment)
	pkg.CheckError("adding metrics documentation", err)
}

const metricsFragment = `

package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// MetricDescription is an exported struct that defines the metric description (Name, Help)
// as a new type named MetricDescription.
type MetricDescription struct {
	Name string
	Help string
	Type string
}

// metricsDescription is a map of string keys (metrics) to MetricDescription values (Name, Help).
var metricDescription = map[string]MetricDescription{
	"MemcachedDeploymentSizeUndesiredCountTotal": {
		Name: "memcached_deployment_size_undesired_count_total",
		Help: "Total number of times the deployment size was not as desired.",
		Type: "Counter",
	},
}

var (
	// MemcachedDeploymentSizeUndesiredCountTotal will count how many times was required
	// to perform the operation to ensure that the number of replicas on the cluster
	// is the same as the quantity desired and specified via the custom resource size spec.
	MemcachedDeploymentSizeUndesiredCountTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: metricDescription["MemcachedDeploymentSizeUndesiredCountTotal"].Name,
			Help: metricDescription["MemcachedDeploymentSizeUndesiredCountTotal"].Help,
		},
	)
)

// RegisterMetrics will register metrics with the global prometheus registry
func RegisterMetrics() {
	metrics.Registry.MustRegister(MemcachedDeploymentSizeUndesiredCountTotal)
}

// ListMetrics will create a slice with the metrics available in metricDescription
func ListMetrics() []MetricDescription {
	v := make([]MetricDescription, 0, len(metricDescription))
	// Insert value (Name, Help) for each metric
	for _, value := range metricDescription {
		v = append(v, value)
	}

	return v
}
`

const metricsdocsFragment = `

package main

import (
	"bytes"
	"fmt"
	"sort"
	"text/template"

	"github.com/example/memcached-operator/monitoring"
)

func main() {
	metricDescriptions := monitoring.ListMetrics()
	sort.Slice(metricDescriptions, func(i, j int) bool {
		return metricDescriptions[i].Name < metricDescriptions[j].Name
	})

	tmpl, err := template.New("Operator metrics").Parse("# Operator Metrics\n" +
		"This document aims to help users that are not familiar with metrics exposed by this operator.\n" +
		"The metrics documentation is auto-generated by the utility tool \"monitoring/metricsdocs\" and reflects all of the metrics that are exposed by the operator.\n\n" +
		"## Operator Metrics List" +
		"{{range .}}\n" +
		"### {{.Name}}\n" +
		"{{.Help}} " +
		"Type: {{.Type}}.\n" +
		"{{end}}" +
		"## Developing new metrics\n" +
		"After developing new metrics or changing old ones, please run \"make generate-metricsdocs\" to regenerate this document.\n\n" +
		"If you feel that the new metric doesn't follow these rules, please change \"monitoring/metricsdocs\" according to your needs.")

	if err != nil {
		panic(err)
	}

	// generate the template using the sorted list of metrics
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, metricDescriptions); err != nil {
		panic(err)
	}

	// print the generated metrics documentation
	fmt.Println(buf.String())
}
`

const alertsFragment = `

package monitoring

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	ruleName                     = "memcached-operator-rules"
	alertRuleGroup               = "memcached.rules"
	deploymentSizeUndesiredAlert = "MemcachedDeploymentSizeUndesired"
	operatorDownAlert            = "MemcachedOperatorDown"
	operatorUpTotalRecordingRule = "memcached_operator_up_total"
	runbookURLBasePath           = "https://github.com/operator-framework/operator-sdk/tree/master/testdata/go/v4/monitoring/memcached-operator/docs/monitoring/runbooks/"
)

// NewPrometheusRule creates new PrometheusRule(CR) for the operator to have alerts and recording rules
func NewPrometheusRule(namespace string) *monitoringv1.PrometheusRule {
	return &monitoringv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monitoringv1.SchemeGroupVersion.String(),
			Kind:       "PrometheusRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleName,
			Namespace: namespace,
		},
		Spec: *NewPrometheusRuleSpec(),
	}
}

// NewPrometheusRuleSpec creates PrometheusRuleSpec for alerts and recording rules
func NewPrometheusRuleSpec() *monitoringv1.PrometheusRuleSpec {
	return &monitoringv1.PrometheusRuleSpec{
		Groups: []monitoringv1.RuleGroup{{
			Name: alertRuleGroup,
			Rules: []monitoringv1.Rule{
				createDeploymentSizeUndesiredAlertRule(),
				createOperatorDownAlertRule(),
				createOperatorUpTotalRecordingRule(),
			},
		}},
	}
}

// createDeploymentSizeUndesiredAlertRule creates MemcachedDeploymentSizeUndesired alert rule
func createDeploymentSizeUndesiredAlertRule() monitoringv1.Rule {
	return monitoringv1.Rule{
		Alert: deploymentSizeUndesiredAlert,
		Expr:  intstr.FromString("increase(memcached_deployment_size_undesired_count_total[5m]) >= 3"),
		Annotations: map[string]string{
			"description": "Memcached-sample deployment size was not as desired more than 3 times in the last 5 minutes.",
		},
		Labels: map[string]string{
			"severity":    "warning",
			"runbook_url": runbookURLBasePath + "MemcachedDeploymentSizeUndesired.md",
		},
	}
}

// createOperatorDownAlertRule creates MemcachedOperatorDown alert rule
func createOperatorDownAlertRule() monitoringv1.Rule {
	return monitoringv1.Rule{
		Alert: operatorDownAlert,
		Expr:  intstr.FromString("memcached_operator_up_total == 0"),
		Annotations: map[string]string{
			"description": "No running memcached-operator pods were detected in the last 5 min.",
		},
		For: "5m",
		Labels: map[string]string{
			"severity":    "critical",
			"runbook_url": runbookURLBasePath + "MemcachedOperatorDown.md",
		},
	}
}

// createOperatorUpTotalRecordingRule creates memcached_operator_up_total recording rule
func createOperatorUpTotalRecordingRule() monitoringv1.Rule {
	return monitoringv1.Rule{
		Record: operatorUpTotalRecordingRule,
		Expr:   intstr.FromString("sum(up{pod=~'memcached-operator-controller-manager-.*'} or vector(0))"),
	}
}
`

const promRuleTestsFragment = `---
# Prometheus official unit-testing documentation - https://prometheus.io/docs/prometheus/latest/configuration/unit_testing_rules/
# rule_files contains the list of files to be tested
rule_files:
  - /tmp/rules.verify

# group_eval_order contains the list of groups to be tested
group_eval_order:
  - memcached.rules

tests:
# for each time frame based on the interval, we define the metrics values
  - interval: 1m
    input_series:
      - series: 'memcached_deployment_size_undesired_count_total'
        # time:  0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15
        values: "0 0 0 1 2 3 3 3 3 3  3  3  3  4  5  6"
      - series: 'memcached_operator_up_total'
        # time:  0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15
        values: "0 0 0 0 0 0 1 1 1 1  0  0  0  0  0  0"

# then, we evaluate the alerts behaviour in the eval_time we choose
    alert_rule_test:
      # it must not trigger before 5m
      - eval_time: 4m
        alertname: MemcachedDeploymentSizeUndesired
        exp_alerts: []
      - eval_time: 4m
        alertname: MemcachedOperatorDown
        exp_alerts: []
      # it must trigger after 5m
      - eval_time: 5m
        alertname: MemcachedDeploymentSizeUndesired
        exp_alerts:
          - exp_annotations:
              description: "Memcached-sample deployment size was not as desired more than 3 times in the last 5 minutes."
            exp_labels:
              severity: "warning"
              runbook_url: "https://github.com/operator-framework/operator-sdk/tree/master/testdata/go/v4/monitoring/memcached-operator/docs/monitoring/runbooks/MemcachedDeploymentSizeUndesired.md"
      - eval_time: 5m
        alertname: MemcachedOperatorDown
        exp_alerts:
          - exp_annotations:
              description: "No running memcached-operator pods were detected in the last 5 min."
            exp_labels:
              severity: "critical"
              runbook_url: "https://github.com/operator-framework/operator-sdk/tree/master/testdata/go/v4/monitoring/memcached-operator/docs/monitoring/runbooks/MemcachedOperatorDown.md"
      # it must not trigger before 15m
      - eval_time: 14m
        alertname: MemcachedDeploymentSizeUndesired
        exp_alerts: [ ]
      - eval_time: 14m
        alertname: MemcachedOperatorDown
        exp_alerts: [ ]
      # it must trigger after 15m
      - eval_time: 15m
        alertname: MemcachedDeploymentSizeUndesired
        exp_alerts:
          - exp_annotations:
              description: "Memcached-sample deployment size was not as desired more than 3 times in the last 5 minutes."
            exp_labels:
              severity: "warning"
              runbook_url: "https://github.com/operator-framework/operator-sdk/tree/master/testdata/go/v4/monitoring/memcached-operator/docs/monitoring/runbooks/MemcachedDeploymentSizeUndesired.md"
      - eval_time: 15m
        alertname: MemcachedOperatorDown
        exp_alerts:
          - exp_annotations:
              description: "No running memcached-operator pods were detected in the last 5 min."
            exp_labels:
              severity: "critical"
              runbook_url: "https://github.com/operator-framework/operator-sdk/tree/master/testdata/go/v4/monitoring/memcached-operator/docs/monitoring/runbooks/MemcachedOperatorDown.md"
`

const ruleSpecDumperFragment = `

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/example/memcached-operator/monitoring"
)

func verifyArgs(args []string) error {
	numOfArgs := len(os.Args[1:])
	if numOfArgs != 1 {
		return fmt.Errorf("expected exactly 1 argument, got: %d", numOfArgs)
	}
	return nil
}

func main() {
	if err := verifyArgs(os.Args); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}

	targetFile := os.Args[1]

	promRuleSpec := monitoring.NewPrometheusRuleSpec()
	b, err := json.Marshal(promRuleSpec)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(targetFile, b, 0644)
	if err != nil {
		panic(err)
	}
}
`

const verifyRulesFragment = `#!/bin/bash -e

readonly PROM_IMAGE="docker.io/prom/prometheus:v2.15.2"

function cleanup() {
    local cleanup_files=("${@:?}")
    for file in "${cleanup_files[@]}"; do
        rm -f "$file"
    done
}

function lint() {
    local target_file="${1:?}"
    docker run --rm --entrypoint=/bin/promtool \
        -v "$target_file":/tmp/rules.verify:ro "$PROM_IMAGE" \
        check rules /tmp/rules.verify
}

function unit_test() {
    local target_file="${1:?}"
    local tests_file="${2:?}"
    docker run --rm --entrypoint=/bin/promtool \
        -v "$tests_file":/tmp/rules.test:ro \
        -v "$target_file":/tmp/rules.verify:ro \
        "$PROM_IMAGE" \
        test rules /tmp/rules.test
}

function main() {
    local prom_spec_dumper="${1:?}"
    local tests_file="${2:?}"
    local target_file
    target_file="$(mktemp --tmpdir -u tmp.prom_rules.XXXXX)"
    trap "cleanup $target_file" RETURN EXIT INT
    "$prom_spec_dumper" "$target_file"
    echo "INFO: Rules file content:"
    cat "$target_file"
    echo
    lint "$target_file"
    unit_test "$target_file" "$tests_file"
}

main "$@"
`
const memcachedDeploymentSizeUndesiredRunbookFragment = `# MemcachedDeploymentSizeUndesired

## Meaning
MemcachedDeploymentSizeUndesired is triggered when the number of available
<code>memcached-sample</code> replicas doesn't match the requested configuration.

## Impact
Unavailability of distributed memory object caching system in the cluster.

## Diagnosis
- Check memcached-sample's pod namespace:

  <code>export NAMESPACE="$(kubectl get deployment -A | grep memcached-sample | awk '{print $1}')"</code>

- Observe the status of the memcached-sample deployment:

  <code>kubectl get deploy memcached-sample -n $NAMESPACE -o yaml</code>

- Observe the logs of the memcached manager pod, to see why it cannot create the memcached-sample pods.

   <code>kubectl get logs <memcached-operator-controller-manager-pod> -n memcached-operator-system</code>

## Mitigation
There can be several reasons. Like:
- Node resource exhaustion
- Not enough memory on the cluster
- Nodes are down

Try to identify the root cause and fix it.`

const memcachedOperatorDownRunbookFragment = `# MemcachedOperatorDown

## Meaning
No running memcached-operator-controller-manager pods were detected in the last 5 min.

## Impact
Complete failure in the <code>Memcached</code> CR lifecycle management.
i.e. launching a new <code>Memcached</code> instance or shutting down an existing one.
## Diagnosis
- Observe the status of the memcached-operator-controller-manager deployment:

  <code>kubectl get deploy memcached-operator-controller-manager -n mecmached-operator-system -o yaml</code>

## Mitigation
There can be several reasons for the memcached-operator-controller-manager pod to be down, identify the root cause and fix it.

- Check the status of the memcached-operator-controller-manager deployment to
find out more information. The following command will provide the associated events and show if there are any issues with pulling an image, crashing pod, etc.

<code>kubectl describe deploy memcached-operator-controller-manager -n memcached-operator-system</code>

- Check if there are issues with the nodes. For example, if they are in a NotReady state.

  </code>kubectl get nodes</code>
`

const prometheusRoleFragment = `---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: prometheus-role
  namespace: system
rules:
  - apiGroups: [""]
    resources:
      - services
      - endpoints
      - pods
    verbs: ["get", "list"]
`
const prometheusRoleBindingFragment = `---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: prometheus-role-binding
  namespace: system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: prometheus-role
subjects:
- kind: ServiceAccount
  name: prometheus-k8s
  namespace: monitoring
`

const controllerMemcachedInstanceFragment = `
	// Fetch the Memcached instance
	// The purpose is check if the Custom Resource for the Kind Memcached
	// is applied on the cluster if not we return nil to stop the reconciliation
	memcached := &cachev1alpha1.Memcached{}
	err := r.Get(ctx, req.NamespacedName, memcached)`

const controllerPrometheusRuleFragment = `
	// Check if prometheus rule already exists, if not create a new one
	foundRule := &monitoringv1.PrometheusRule{}
	err := r.Get(ctx, types.NamespacedName{Name: ruleName, Namespace: namespace}, foundRule)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new prometheus rule
		prometheusRule := monitoring.NewPrometheusRule(namespace)
		if err := r.Create(ctx, prometheusRule); err != nil {
			log.Error(err, "Failed to create prometheus rule")
			return ctrl.Result{}, nil
		}
	}

	if err == nil {
		// Check if prometheus rule spec was changed, if so set as desired
		desiredRuleSpec := monitoring.NewPrometheusRuleSpec()
		if !reflect.DeepEqual(foundRule.Spec.DeepCopy(), desiredRuleSpec) {
			desiredRuleSpec.DeepCopyInto(&foundRule.Spec)
			if r.Update(ctx, foundRule); err != nil {
				log.Error(err, "Failed to update prometheus rule")
				return ctrl.Result{}, nil
			}
		}
	}

	// Fetch the Memcached instance
	// The purpose is check if the Custom Resource for the Kind Memcached
	// is applied on the cluster if not we return nil to stop the reconciliation
	memcached := &cachev1alpha1.Memcached{}
	err = r.Get(ctx, req.NamespacedName, memcached)`

const mainRecorderFragment = `
// Add a Recorder to the reconciler.
// This allows the operator author to emit events during reconcilliation.`

const monitoringv1ImportFragment = `

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
`

const monitoringImportFragment = `
	"github.com/example/memcached-operator/monitoring"
`

const mainMonitoringFragment = `

	utilruntime.Must(monitoringv1.AddToScheme(scheme))

	monitoring.RegisterMetrics()`

const makefileFragment = `
LDFLAGS            ?= -w -s

build-prom-spec-dumper: ## Build binary from source
	go build -ldflags="${LDFLAGS}" -o _out/rule-spec-dumper ./monitoring/prom-rule-ci/rule-spec-dumper.go

current-dir := $(realpath .)

# Unit testing for the operator alerts and recording rules
# rule-spec-dumper dumps the prometheus rule spec to a temp _out/rule-spec-dumper file which prom-rules-tests runs against
prom-rules-verify: build-prom-spec-dumper
	./monitoring/prom-rule-ci/verify-rules.sh \
		"${current-dir}/_out/rule-spec-dumper" \
		"${current-dir}/monitoring/prom-rule-ci/prom-rules-tests.yaml"

`

const metricsdocsMakefileFragment = `

##@ Generate the metrics documentation
.PHONY: generate-metricsdocs
generate-metricsdocs:
	mkdir -p $(shell pwd)/docs/monitoring
	go run -ldflags="${LDFLAGS}" ./monitoring/metricsdocs > docs/monitoring/metrics.md
`

const webhooksFragment = `
// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-cache-example-com-v1alpha1-memcached,mutating=false,failurePolicy=fail,sideEffects=None,groups=cache.example.com,resources=memcacheds,verbs=create;update,versions=v1alpha1,name=vmemcached.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Memcached{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Memcached) ValidateCreate() (admission.Warnings, error) {
	memcachedlog.Info("validate create", "name", r.Name)

	return nil, validateOdd(r.Spec.Size)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Memcached) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	memcachedlog.Info("validate update", "name", r.Name)

	return nil, validateOdd(r.Spec.Size)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Memcached) ValidateDelete() (admission.Warnings, error) {
	memcachedlog.Info("validate delete", "name", r.Name)

	return nil, nil
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
