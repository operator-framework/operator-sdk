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
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	kbtestutils "sigs.k8s.io/kubebuilder/v2/test/e2e/utils"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/internal/pkg"
	"github.com/operator-framework/operator-sdk/internal/testutils"
)

// MemcachedGoWithWebhooks defines the Memcached Sample in GO using webhooks
type MemcachedGoWithWebhooks struct {
	ctx *pkg.SampleContext
}

// MemcachedGoWithWebhooks return a MemcachedGoWithWebhooks
func NewMemcachedGoWithWebhooks(ctx *pkg.SampleContext) MemcachedGoWithWebhooks {
	return MemcachedGoWithWebhooks{ctx}
}

// Prepare the Context for the Memcached with WebHooks Go Sample
// Note that sample directory will be re-created and the context data for the sample
// will be set such as the domain and GVK.
func (mh *MemcachedGoWithWebhooks) Prepare() {
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
func (mh *MemcachedGoWithWebhooks) Run() {
	log.Infof("creating the project")
	err := mh.ctx.Init(
		// TODO(estroz): change this to 3 when stabilized.
		"--project-version", "3-alpha",
		// TODO(estroz): change this to go/v3 when stabilized.
		"--plugins", "go/v3-alpha",
		"--repo", "github.com/example/memcached-operator",
		"--domain",
		mh.ctx.Domain)
	pkg.CheckError("creating the project", err)

	err = mh.ctx.CreateAPI(
		"--group", mh.ctx.Group,
		"--version", mh.ctx.Version,
		"--kind", mh.ctx.Kind,
		"--controller", "true",
		"--resource", "true")
	pkg.CheckError("scaffolding apis", err)

	log.Infof("implementing the API")
	mh.implementingAPI()

	log.Infof("implementing the Controller")
	mh.implementingController()

	log.Infof("scaffolding webhook")
	err = mh.ctx.CreateWebhook(
		"--group", mh.ctx.Group,
		"--version", mh.ctx.Version,
		"--kind", mh.ctx.Kind,
		"--defaulting",
		"--defaulting")
	pkg.CheckError("scaffolding webhook", err)

	mh.implementingWebhooks()
	mh.uncommentKustomizationFile()

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

// uncommentKustomizationFile will uncomment the file kustomization.yaml
func (mh *MemcachedGoWithWebhooks) uncommentKustomizationFile() {
	log.Infof("uncomment kustomization.yaml to enable webhook and ca injection")
	err := testutils.UncommentCode(
		filepath.Join(mh.ctx.Dir, "config", "default", "kustomization.yaml"),
		"#- ../webhook", "#")
	pkg.CheckError("uncomment webhook", err)

	err = testutils.UncommentCode(
		filepath.Join(mh.ctx.Dir, "config", "default", "kustomization.yaml"),
		"#- ../certmanager", "#")
	pkg.CheckError("uncomment certmanager", err)

	err = testutils.UncommentCode(
		filepath.Join(mh.ctx.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")
	pkg.CheckError("uncomment prometheus", err)

	err = testutils.UncommentCode(
		filepath.Join(mh.ctx.Dir, "config", "default", "kustomization.yaml"),
		"#- manager_webhook_patch.yaml", "#")
	pkg.CheckError("uncomment manager_webhook_patch.yaml", err)

	err = testutils.UncommentCode(
		filepath.Join(mh.ctx.Dir, "config", "default", "kustomization.yaml"),
		"#- webhookcainjection_patch.yaml", "#")
	pkg.CheckError("uncomment webhookcainjection_patch.yaml", err)

	err = testutils.UncommentCode(filepath.Join(mh.ctx.Dir, "config", "default", "kustomization.yaml"),
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

// implementingWebhooks will customize the kind wekbhok file
func (mh *MemcachedGoWithWebhooks) implementingWebhooks() {
	log.Infof("implementing webhooks")
	webhookPath := filepath.Join(mh.ctx.Dir, "api", mh.ctx.Version, fmt.Sprintf("%s_webhook.go",
		strings.ToLower(mh.ctx.Kind)))

	// Add webhook methods
	err := kbtestutils.InsertCode(webhookPath,
		"// TODO(user): fill in your defaulting logic.\n}",
		webhooksFragment)
	pkg.CheckError("replacing webhook validate implementation", err)

	err = testutils.ReplaceInFile(webhookPath,
		"// TODO(user): fill in your defaulting logic.", "if r.Spec.Size == 0 {\n\t\tr.Spec.Size = 3\n\t}")
	pkg.CheckError("replacing webhook default implementation", err)

	// Add imports
	err = kbtestutils.InsertCode(webhookPath,
		"import (",
		// TODO(estroz): remove runtime dep when --programmatic-validation is added to `ccreate webhook` above.
		"\"errors\"\n\n\"k8s.io/apimachinery/pkg/runtime\"")
	pkg.CheckError("adding imports", err)
}

// implementingController will customize the Controller
func (mh *MemcachedGoWithWebhooks) implementingController() {
	controllerPath := filepath.Join(mh.ctx.Dir, "controllers", fmt.Sprintf("%s_controller.go",
		strings.ToLower(mh.ctx.Kind)))

	// Add imports
	err := kbtestutils.InsertCode(controllerPath,
		"import (",
		importsFragment)
	pkg.CheckError("adding imports", err)

	// Add RBAC permissions on top of reconcile
	err = kbtestutils.InsertCode(controllerPath,
		"/finalizers,verbs=update",
		rbacFragment)
	pkg.CheckError("adding rbac", err)

	// Replace reconcile content
	err = testutils.ReplaceInFile(controllerPath,
		fmt.Sprintf("_ = r.Log.WithValues(\"%s\", req.NamespacedName)", strings.ToLower(mh.ctx.Kind)),
		fmt.Sprintf("log := r.Log.WithValues(\"%s\", req.NamespacedName)", strings.ToLower(mh.ctx.Kind)))
	pkg.CheckError("replacing reconcile content", err)

	// Add reconcile implementation
	err = testutils.ReplaceInFile(controllerPath,
		"// your logic here", reconcileFragment)
	pkg.CheckError("replacing reconcile", err)

	// Add helpers funcs to the controller
	err = kbtestutils.InsertCode(controllerPath,
		"return ctrl.Result{}, nil\n}", controllerFuncsFragment)
	pkg.CheckError("adding helpers methods in the controller", err)

	// Add watch for the Kind
	err = testutils.ReplaceInFile(controllerPath,
		fmt.Sprintf(watchOriginalFragment, mh.ctx.Group, mh.ctx.Version, mh.ctx.Kind),
		fmt.Sprintf(watchCustomizedFragment, mh.ctx.Group, mh.ctx.Version, mh.ctx.Kind))
	pkg.CheckError("replacing reconcile", err)
}

// implementingAPI will customize the API
func (mh *MemcachedGoWithWebhooks) implementingAPI() {
	err := kbtestutils.InsertCode(
		filepath.Join(mh.ctx.Dir, "api", mh.ctx.Version, fmt.Sprintf("%s_types.go", strings.ToLower(mh.ctx.Kind))),
		fmt.Sprintf("type %sSpec struct {\n\t// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster\n\t// Important: Run \"make\" to regenerate code after modifying this file", mh.ctx.Kind),
		`

	// Size defines the number of Memcached instances
	Size int32 `+"`"+`json:"size,omitempty"`+"`"+`
`)
	pkg.CheckError("inserting spec Status", err)

	log.Infof("implementing MemcachedStatus")
	err = kbtestutils.InsertCode(
		filepath.Join(mh.ctx.Dir, "api", mh.ctx.Version, fmt.Sprintf("%s_types.go", strings.ToLower(mh.ctx.Kind))),
		fmt.Sprintf("type %sStatus struct {\n\t// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster\n\t// Important: Run \"make\" to regenerate code after modifying this file", mh.ctx.Kind),
		`

	// Nodes store the name of the pods which are running Memcached instances
	Nodes []string `+"`"+`json:"nodes,omitempty"`+"`"+`
`)
	pkg.CheckError("inserting Node Status", err)

	sampleFile := filepath.Join("config", "samples",
		fmt.Sprintf("%s_%s_%s.yaml", mh.ctx.Group, mh.ctx.Version, strings.ToLower(mh.ctx.Kind)))

	log.Infof("updating sample to have size attribute")
	err = testutils.ReplaceInFile(filepath.Join(mh.ctx.Dir, sampleFile), "foo: bar", "size: 1")
	pkg.CheckError("updating sample", err)
}

// GenerateMemcachedGoWithWebhooksSample will call all actions to create the directory and generate the sample
// Note that it should NOT be called in the e2e tests.
func GenerateMemcachedGoWithWebhooksSample(samplesPath string) {
	log.Infof("starting to generate Go memcached sample with webhooks")
	ctx, err := pkg.NewSampleContext(testutils.BinaryName, filepath.Join(samplesPath, "memcached-operator"), "GO111MODULE=on")
	pkg.CheckError("generating Go memcached with webhooks context", err)

	memcached := NewMemcachedGoWithWebhooks(&ctx)
	memcached.Prepare()
	memcached.Run()
}

const rbacFragment = `
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;`

const reconcileFragment = `// Fetch the Memcached instance
	memcached := &cachev1alpha1.Memcached{}
	err := r.Get(ctx, req.NamespacedName, memcached)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Memcached resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Memcached")
		return ctrl.Result{}, err
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: memcached.Name, Namespace: memcached.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForMemcached(memcached)
		log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// Ensure the deployment size is the same as the spec
	size := memcached.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Spec updated - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}

	// Update the Memcached status with the pod names
	// List the pods for this memcached's deployment
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(memcached.Namespace),
		client.MatchingLabels(labelsForMemcached(memcached.Name)),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "Memcached.Namespace", memcached.Namespace, "Memcached.Name", memcached.Name)
		return ctrl.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, memcached.Status.Nodes) {
		memcached.Status.Nodes = podNames
		err := r.Status().Update(ctx, memcached)
		if err != nil {
			log.Error(err, "Failed to update Memcached status")
			return ctrl.Result{}, err
		}
	}
`

const controllerFuncsFragment = `

// deploymentForMemcached returns a memcached Deployment object
func (r *MemcachedReconciler) deploymentForMemcached(m *cachev1alpha1.Memcached) *appsv1.Deployment {
	ls := labelsForMemcached(m.Name)
	replicas := m.Spec.Size

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:   "memcached:1.4.36-alpine",
						Name:    "memcached",
						Command: []string{"memcached", "-m=64", "-o", "modern", "-v"},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 11211,
							Name:          "memcached",
						}},
					}},
				},
			},
		},
	}
	// Set Memcached instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// labelsForMemcached returns the labels for selecting the resources
// belonging to the given memcached CR name.
func labelsForMemcached(name string) map[string]string {
	return map[string]string{"app": "memcached", "memcached_cr": name}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}
`

const importsFragment = `
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"

`

const watchOriginalFragment = `return ctrl.NewControllerManagedBy(mgr).
		For(&%s%s.%s{}).
		Complete(r)
`

const watchCustomizedFragment = `return ctrl.NewControllerManagedBy(mgr).
		For(&%s%s.%s{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
`

const webhooksFragment = `
// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:path=/validate-cache-example-com-v1alpha1-memcached,mutating=false,failurePolicy=fail,sideEffects=None,groups=cache.example.com,resources=memcacheds,verbs=create;update,versions=v1alpha1,name=vmemcached.kb.io,admissionReviewVersions={v1,v1beta1}

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
