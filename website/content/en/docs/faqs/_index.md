---
title: Operator SDK FAQ
linkTitle: FAQ
weight: 12
---

## What are the the differences between Kubebuilder and Operator-SDK?

Kubebuilder and Operator SDK are both projects that allow you to quickly create and manage an operator project. Operator SDK uses Kubebuilder under the hood to do so for Go projects, such that the `operator-sdk` CLI tool will work with a project created by `kubebuilder`. Therefore each project makes use of [controller-runtime][controller-runtime] and will have the same [basic layout][kb-doc-what-is-a-basic-project]. For further information also check the [SDK Project Layout][project-doc].

Operator SDK offers additional features on top of the basic project scaffolding that Kubebuilder provides. By default, `operator-sdk init` generates a project integrated with:
- [Operator Lifecycle Manager][olm], an installation and runtime management system for operators
- [OperatorHub][operatorhub.io], a community hub for publishing operators
- Operator SDK [scorecard][scorecard-doc], a tool for ensuring operator best-practices and developing cluster tests

Operator SDK supports operator types other than Go as well, such as Ansible and Helm.

For further context about the relationship between Kubebuilder and Operator SDK, see [this blog post][operator-sdk-reaches-v1.0].

## Can I use the Kubebuilder docs?

Yes, you can use [https://book.kubebuilder.io/](https://book.kubebuilder.io/). Just keep in mind that when you see an instruction such as:
`$ kubebuilder <command>` you will use `$ operator-sdk <command>`.

## Controller Runtime FAQ

Please see the upstream [Controller Runtime FAQ][cr-faq] first for any questions related to runtime mechanics or controller-runtime APIs.

## Can I customize the projects initialized with `operator-sdk`?

After using the CLI to create your project, you are free to customize based on how you see fit. Please note that it is not recommended to deviate from the proposed layout unless you know what you are doing.

For example, you should refrain from moving the scaffolded files, doing so will make it difficult to upgrade your project in the future. You may also lose the ability to use some of the CLI features and helpers. For further information on the project layout, see the doc [Project Layout][project-doc]

## How can I have separate logic for Create, Update, and Delete events? When reconciling an object can I access its previous state?

You should not have separate logic. Instead design your reconciler to be idempotent. See the [controller-runtime FAQ][cr-faq] for more details.

## How do I wait for some specific cluster state such as a resource being deleted?

You don't. Instead, design your reconciler to be idempotent by  taking the next step based on the current state, and then returning and requeuing. For example, waiting for an object to be deleted might look something like this:

```
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
    ...
    if !r.IfPodWasDeleted(ctx, pod) {
        if err := r.Delete(ctx, pod); err != nil {
            return ctrl.Result{}, err
        }
        return ctrl.Result{Requeue: true}, nil
    }
    // This code will be invoked only after pod deletion    
    r.DeployBiggerPod(ctx)
    ...
}
```


## When my Custom Resource is deleted, I need to know its contents or perform cleanup tasks. How can I do that?

Use a [finalizer].

## I see the warning in my Operator's logs: `The resourceVersion for the provided watch is too old.` What's wrong?

This is completely normal and expected behavior.

The `kube-apiserver` watch request handler is designed to periodically close a watch to spread out load among controller node instances. Once disconnected, your Operator's informer will automatically reconnect and re-establish the watch. If an event is missed during re-establishment, the watch will fail with the above warning message. The Operator's informer then does a list request and uses the new `resourceVersion` from that list to restablish the watch and replace the cache with the latest objects.

This warning should not be stifled. It ensures that the informer is not stuck or wedged.

Never seeing this warning may suggest that your watch or cache is not healthy. If the message is repeating every few seconds, this may signal a network connection problem or issue with etcd.

For more information on `kube-apiserver` request timeout options, see the [Kubernetes API Server Command Line Tool Reference][kube-apiserver_options]


## My Ansible module is missing a dependency. How do I add it to the image?

Unfortunately, adding the entire dependency tree for all Ansible modules would be excessive. Fortunately, you can add it easily. Simply edit your build/Dockerfile. You'll want to change to root for the install command, just be sure to swap back using a series of commands like the following right after the `FROM` line.

```docker
USER 0
RUN yum -y install my-dependency
RUN pip3 install my-python-dependency
USER 1001
```

If you aren't sure what dependencies are required, start up a container using the image in the `FROM` line as root. That will look something like this:
```sh
docker run -u 0 -it --rm --entrypoint /bin/bash quay.io/operator-framework/ansible-operator:<sdk-tag-version>
```

## After deploying my operator, I see errors like "Failed to watch <external type>"

If you run into the following error message, it means that your operator is unable to watch the resource:

```
E0320 15:42:17.676888       1 reflector.go:280] pkg/mod/k8s.io/client-go@v0.0.0-20191016111102-bec269661e48/tools/cache/reflector.go:96: Failed to watch *v1.ImageStreamTag: unknown (get imagestreamtags.image.openshift.io)
{"level":"info","ts":1584718937.766342,"logger":"controller_memcached","msg":"ImageStreamTag resource not found.
```

Using controller-runtime's split client means that read operations (gets and lists) are read from a cache, and write operations are written directly to the API server. To populate the cache for reads, controller-runtime initiates a `list` and then a `watch` even when your operator is only attempting to `get` a single resource. The above scenario occurs when the operator does not have an [RBAC][rbac] permission to `watch` the resource. The solution is to add an RBAC directive to generate a `config/rbac/role.yaml` with `watch` privileges:

```go
//+kubebuilder:rbac:groups=some.group.com,resources=myresources,verbs=watch
```

Alternatively, if the resource you're attempting to cannot be watched (like `v1.ImageStreamTag` above), you can specify that objects of this type should not be cached by adding the following to `main.go`:

```go
import (
	...
	imagev1 "github.com/openshift/api/image/v1"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	// Add imagev1's scheme.
	utilruntime.Must(imagev1.AddToScheme(scheme))
}

func main() {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:        scheme,
		// Specify that ImageStreamTag's should not be cached.
		ClientBuilder: manager.NewClientBuilder().WithUncached(&imagev1.ImageStreamTag{}),
	})
}
```

Then in your controller file, add an RBAC directive to generate a `config/rbac/role.yaml` with `get` privileges:

```go
//+kubebuilder:rbac:groups=image.openshift.io,resources=imagestreamtags,verbs=get
```

Now run `make manifests` to update your `role.yaml`.


## After deploying my operator, why do I see errors like "is forbidden: cannot set blockOwnerDeletion if an ownerReference refers to a resource you can't set finalizers on: ..."?

If you are facing this issue, it means that the operator is missing the required RBAC permissions to update finalizers on the APIs it manages. This permission is necessary if the [OwnerReferencesPermissionEnforcement][owner-references-permission-enforcement] plugin is enabled in your cluster.

For Helm and Ansible operators, this permission is configured by default. However for Go operators, it may be necessary to add this permission yourself
by adding an RBAC directive to generate a `config/rbac/role.yaml` with `update` privileges on your CR's finalizers:

```go
//+kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/finalizers,verbs=update
```

Now run `make manifests` to update your `role.yaml`.

## When invoking `make` targets, why do I see errors like `fork/exec /usr/local/kubebuilder/bin/etcd: no such file or directory occurred`?

If using an OS or distro that does not point `sh` to the `bash` shell (Ubuntu for example), add the following line to the `Makefile`:

```make
SHELL := /bin/bash
```

## How do I make my Operator proxy-friendly?
---

Administrators can configure proxy-friendly Operators to support network proxies by
specifiying `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment
variables in the Operator deployment. (These variables can be handled by OLM.)

Proxy-friendly Operators are responsible for inspecting the Operator
environment and passing these variables along to the rquired operands.
For more information and examples, please see the type-specific docs:
- [Ansible][ansible-proxy-vars]
- [Golang][go-proxy-vars]
- [Helm][helm-proxy-vars]

[ansible-proxy-vars]: /docs/building-operators/ansible/reference/proxy-vars/
[client.Reader]:https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client#Reader
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[cr-faq]:https://github.com/kubernetes-sigs/controller-runtime/blob/master/FAQ.md
[finalizer]:/docs/building-operators/golang/advanced-topics/#handle-cleanup-on-deletion
[go-proxy-vars]: /docs/building-operators/golang/references/proxy-vars/
[helm-proxy-vars]: /docs/building-operators/helm/reference/proxy-vars/
[kb-doc-what-is-a-basic-project]: https://book.kubebuilder.io/cronjob-tutorial/basic-project.html
[kube-apiserver_options]: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/#options
[olm]:  https://github.com/operator-framework/operator-lifecycle-manager
[operator-sdk-reaches-v1.0]: https://www.openshift.com/blog/operator-sdk-reaches-v1.0
[operatorhub.io]: https://operatorhub.io/
[owner-references-permission-enforcement]: https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#ownerreferencespermissionenforcement
[rbac-markers]: https://book.kubebuilder.io/reference/markers/rbac.html
[rbac]:https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[scorecard-doc]: https://sdk.operatorframework.io/docs/testing-operators/scorecard/
[project-doc]: /docs/overview/project-layout

## Preserve the `preserveUnknownFields` in your CRDs

The [`preserveUnknownFields`](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#field-pruning) will be removed if set to false when running `make bundle`. Because of some underlying data structure changes and how yaml is unmarshalled, it is best to add them back in after they have been written.

You can use this script to post process the files to add the `preserveUnknownFields` back in.

```sh
function generate_preserveUnknownFieldsdata() {
    for j in config/crd/patches/*.yaml ; do
        if grep -qF "preserveUnknownFields" "$j";then
            variable=`awk '/metadata/{flag=1} flag && /name:/{print $NF;flag=""}' "$j"`
            for k in config/crd/bases/*.yaml ; do
                if grep -qF "$variable" "$j";then
                    filename=`awk 'END{ var=FILENAME; split (var,a,/\//); print a[4]}' "$k"`
                    awk '/^spec:/{print;print "  preserveUnknownFields: false";next}1' "bundle/manifests/$filename" > testfile.tmp && mv testfile.tmp "bundle/manifests/$filename"
                fi
            done
        fi
    done
}

generate_preserveUnknownFieldsdata
```

You can then modify the `bundle` target in your `Makefile` by adding a call to the script at the end of the target. See the example below:

```
.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle | ./preserve_script.sh
```

Note:
Though this is a bug with controller-gen which is used by Operator SDK to generate CRD, this is a workaround from our end to enable users to preserve the field after controller-gen has run.

## What is the bundle limit size? Was this amount increased?

Bundles have a size limitation because their manifests are used to create a configMap, and the Kubernetes API does not 
allow configMaps larger than `~1MB`. Beginning with [OLM](https://github.com/operator-framework/operator-lifecycle-manager) version `v0.19.0` 
and [OPM](https://github.com/operator-framework/operator-registry) `1.17.5`, 
these values are now compressed accommodating larger bundles. ([More info](https://github.com/operator-framework/operator-registry/pull/685)).

The change to allow bigger bundles from [OLM](https://github.com/operator-framework/operator-lifecycle-manager) version `v0.19.0` only impacts the full bundle size amount. 
Any single manifest within the bundle such as the CRD will still make the bundle uninstallable if it exceeds the default file size limit on clusters (`~1MB`).

## The size of my Operator bundle is too big. What can I do?

If your bundle is too large, there are a few things you can try:

  * Reducing the number of [CRD versions][k8s-crd-versions] supported in your Operator by deprecating and then removing older API versions. It is a good idea to have a clear plan for deprecation and removal of old CRDs versions when new ones get added, see [Kubernetes API change practices][k8s-api-change]. Also, refer to the [Kubernetes API conventions][k8s-api-convention].
  * Reduce the verbosity of your API documentation. (We do not recommend eliminating documenting the APIs)


[k8s-crd-versions]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#specify-multiple-versions
[k8s-api-change]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md
[k8s-api-convention]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md