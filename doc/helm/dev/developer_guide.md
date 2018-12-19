# Developer guide

This document provides some useful information and tips for a developer
creating an operator powered by Helm.

## Getting started with Helm Charts

Since we are interested in using Helm for the lifecycle management of our
application on Kubernetes, it is beneficial for a developer to get a good grasp
of [Helm charts][helm_charts]. Helm charts allow a developer to leverage their
existing Kubernetes resource files (written in YAML). One of the biggest
benefits of using Helm in conjunction with existing Kubernetes resource files
is the ability to use templating so that you can customize Kubernetes resources
with the simplicity of a few [Helm values][helm_values].

### Installing Helm

If you are unfamiliar with Helm, the easiest way to get started is to
[install Helm][helm_install] and test your charts using the `helm` command line
tool.

**NOTE:** Installing Helm's Tiller component in your cluster is not required,
because the Helm operator runs a Tiller component internally.

### Testing a Helm chart locally

Sometimes it is beneficial for a developer to run the Helm chart installation
from their local machine as opposed to running/rebuilding the operator each
time. To do this, initialize a new project:

```sh
$ operator-sdk new --type helm --kind Foo --api-version foo.example.com/v1alpha1 foo-operator
INFO[0000] Creating new Helm operator 'foo-operator'.
INFO[0000] Create build/Dockerfile
INFO[0000] Create watches.yaml
INFO[0000] Create deploy/service_account.yaml
INFO[0000] Create deploy/role.yaml
INFO[0000] Create deploy/role_binding.yaml
INFO[0000] Create deploy/operator.yaml
INFO[0000] Create deploy/crds/foo_v1alpha1_foo_crd.yaml
INFO[0000] Create deploy/crds/foo_v1alpha1_foo_cr.yaml
INFO[0000] Create helm-charts/foo/
INFO[0000] Run git init ...
Initialized empty Git repository in /home/joe/go/src/github.com/operator-framework/foo-operator/.git/
INFO[0000] Run git init done
INFO[0000] Project creation complete.

$ cd foo-operator
```

For this example we will use the default Nginx helm chart scaffolded by
`operator-sdk new`. Without making any changes, we can see what the default
release manifests are:

```sh
$ helm template --name test-release helm-charts/foo
---
# Source: foo/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: test-release-foo
  labels:
    app.kubernetes.io/name: foo
    helm.sh/chart: foo-0.1.0
    app.kubernetes.io/instance: test-release
    app.kubernetes.io/managed-by: Tiller
spec:
  type: ClusterIP
  ports:
    - port: 80
      targetPort: http
      protocol: TCP
      name: http
  selector:
    app.kubernetes.io/name: foo
    app.kubernetes.io/instance: test-release

---
# Source: foo/templates/deployment.yaml
apiVersion: apps/v1beta2
kind: Deployment
metadata:
  name: test-release-foo
  labels:
    app.kubernetes.io/name: foo
    helm.sh/chart: foo-0.1.0
    app.kubernetes.io/instance: test-release
    app.kubernetes.io/managed-by: Tiller
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: foo
      app.kubernetes.io/instance: test-release
  template:
    metadata:
      labels:
        app.kubernetes.io/name: foo
        app.kubernetes.io/instance: test-release
    spec:
      containers:
        - name: foo
          image: "nginx:stable"
          imagePullPolicy: IfNotPresent
          ports:
            - name: http
              containerPort: 80
              protocol: TCP
          livenessProbe:
            httpGet:
              path: /
              port: http
          readinessProbe:
            httpGet:
              path: /
              port: http
          resources:
            {}


---
# Source: foo/templates/ingress.yaml

```

Next, deploy these resource manifests to your cluster without using
Tiller:

```sh
$ helm template --name test-release helm-charts/foo | kubectl apply -f -
service/test-release-foo created
deployment.apps/test-release-foo created
```

Check that the release resources were created:

```sh
$ kubectl get all -l app.kubernetes.io/instance=test-release
NAME                                    READY   STATUS    RESTARTS   AGE
pod/test-release-foo-5554d49986-47676   1/1     Running   0          2m

NAME                       TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)   AGE
service/test-release-foo   ClusterIP   10.100.136.126   <none>        80/TCP    2m

NAME                               DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/test-release-foo   1         1         1            1           2m

NAME                                          DESIRED   CURRENT   READY   AGE
replicaset.apps/test-release-foo-5554d49986   1         1         1       2m
```

Next, let's create a simple values file that we can use to override the Helm
chart's defaults:

```sh
cat << EOF >> overrides.yaml
replicaCount: 2
service:
  port: 8080
EOF
```

Re-run the templates and re-apply them to the cluster, this time using the
`overrides.yaml` file we just created:

```sh
$ helm template -f overrides.yaml --name test-release helm-charts/foo | kubectl apply -f -
service/test-release-foo configured
deployment.apps/test-release-foo configured
```

Now you'll see that there are 2 deployment replicas and the service port
has been updated to `8080`.

```sh
$ kubectl get deployment -l app.kubernetes.io/instance=test-release
NAME               DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
test-release-foo   2         2         2            2           3m

$ kubectl get service -l app.kubernetes.io/instance=test-release
NAME               TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)     AGE
test-release-foo   ClusterIP   10.100.136.126   <none>        8080/TCP    3m
```

Lastly, delete the release:

```sh
$ helm template -f overrides.yaml --name test-release helm-charts/foo | kubectl delete -f -
service "test-release-foo" deleted
deployment.apps "test-release-foo" deleted
```

Check that the resources were deleted:

```sh
$ kubectl get all -l app.kubernetes.io/instance=test-release
No resources found.
```

## Using Helm inside of an Operator
Now that we have demonstrated using the Helm CLI, we want to trigger this Helm
chart release process when a custom resource changes. We want to map our `foo`
Helm chart to a specific Kubernetes resource that the operator will watch. This
mapping is done in a file called `watches.yaml`.

### Watches file

The Operator expects a mapping file, which lists each GVK to watch and the
corresponding path to a Helm chart, to be copied into the
container at a predefined location: `/opt/helm/watches.yaml`

Dockerfile example:

```Dockerfile
COPY watches.yaml /opt/helm/watches.yaml
```

The Watches file format is yaml and is an array of objects. The object has
mandatory fields:

**version**:  The version of the Custom Resource that you will be watching.

**group**:  The group of the Custom Resource that you will be watching.

**kind**:  The kind of the Custom Resource that you will be watching.

**chart**:  This is the path to the Helm chart that you have added to the
container. For example, if your Helm charts directory is at
`/opt/helm/helm-charts/` and your Helm chart is named `busybox`, this value
will be `/opt/helm/helm-charts/busybox`.

Example specifying a Helm chart watch:

```yaml
---
- version: v1alpha1
  group: foo.example.com
  kind: Foo
  chart: /opt/helm/helm-charts/foo
```

### Custom Resource file

The Custom Resource file format is Kubernetes resource file. The object has
mandatory fields:

**apiVersion**:  The version of the Custom Resource that will be created.

**kind**:  The kind of the Custom Resource that will be created

**metadata**:  Kubernetes specific metadata to be created

**spec**:  The spec contains the YAML for values that override the Helm chart's
defaults. This corresponds to the `overrides.yaml` file we created above. This
field is optional and can be empty, which results in the default Helm chart
being released by the operator.

### Testing a Helm operator locally

Once a developer is comfortable working with the above workflow, it will be
beneficial to test the logic inside of an operator. To accomplish this, we can
use `operator-sdk up local` from the top-level directory of our project. The
`up local` command reads from `./watches.yaml` and uses `~/.kube/config` to
communicate with a kubernetes cluster just as the `kubectl apply` commands did
when we were testing our Helm chart locally. This section assumes the developer
has read the [Helm Operator user guide][helm_operator_user_guide] and has the
proper dependencies installed.

Since `up local` reads from `./watches.yaml`, there are a couple options
available to the developer. If `chart` is left alone (by default
`/opt/helm/helm-charts/<name>`) the Helm chart must exist at that location in
the filesystem. It is recommended that the developer create a symlink at this
location, pointed to the Helm chart in the project directory, so that changes
to the Helm chart are reflected where necessary.

```sh
sudo mkdir -p /opt/helm/helm-charts
sudo ln -s $PWD/helm-charts/<name> /opt/helm/helm-charts/<name>
```

Create a Custom Resource Definition (CRD) and proper Role-Based Access Control
(RBAC) definitions for resource Foo. `operator-sdk` autogenerates these files
inside of the `deploy` folder:

```sh
kubectl create -f deploy/crds/foo_v1alpha1_foo_crd.yaml
kubectl create -f deploy/service_account.yaml
kubectl create -f deploy/role.yaml
kubectl create -f deploy/role_binding.yaml
```

Run the `up local` command:

```sh
$ operator-sdk up local
INFO[0000] Running the operator locally.
INFO[0000] Go Version: go1.10.3
INFO[0000] Go OS/Arch: linux/amd64
INFO[0000] operator-sdk Version: v0.2.0+git
{"level":"info","ts":1543357618.0081263,"logger":"kubebuilder.controller","caller":"controller/controller.go:120","msg":"Starting EventSource","Controller":"foo-controller","Source":{"Type":{"apiVersion":"foo.example.com/v1alpha1","kind":"Foo"}}}
{"level":"info","ts":1543357618.008322,"logger":"helm.controller","caller":"controller/controller.go:73","msg":"Watching resource","apiVersion":"foo.example.com/v1alpha1","kind":"Foo","namespace":"default","resyncPeriod":"5s"}
```

Now that the operator is watching resource `Foo` for events, the creation of a
Custom Resource will trigger our Helm chart to be executed. Take a look at
`deploy/crds/foo_v1alpha1_foo_cr.yaml`. Our chart does not have a `size` value,
so let's remove it. Your CR file should look like the following:

```yaml
apiVersion: "foo.example.com/v1alpha1"
kind: "Foo"
metadata:
  name: "example-foo"
spec:
  # Add fields here
```

Since `spec` is not set, Helm is invoked with no extra variables. The next
section covers how extra variables are passed from a Custom Resource to
Helm. This is why it is important to set sane defaults for the operator.

Create a Custom Resource instance of Foo with default var `state` set to
`present`:

```sh
$ kubectl apply -f deploy/crds/foo_v1alpha1_foo_cr.yaml
foo.foo.example.com/example-foo created
```

The custom resource status will be updated with the release information if the
installation succeeds. Let's get the release name:

```sh
$ export RELEASE_NAME=$(kubectl get foos example-foo -o jsonpath={..status.release.name})
$ echo $RELEASE_NAME
example-foo-4f8ay4vfr99ulx905hax3j6x1
```

Check that the release resources were created:

```sh
$ kubectl get all -l app.kubernetes.io/instance=${RELEASE_NAME}
NAME                                                        READY   STATUS    RESTARTS   AGE
pod/example-foo-4f8ay4vfr99ulx905hax3j6x1-9dfd67fc6-s6krb   1/1     Running   0          4m

NAME                                            TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)   AGE
service/example-foo-4f8ay4vfr99ulx905hax3j6x1   ClusterIP   10.102.91.83   <none>        80/TCP    4m

NAME                                                    DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/example-foo-4f8ay4vfr99ulx905hax3j6x1   1         1         1            1           4m

NAME                                                              DESIRED   CURRENT   READY   AGE
replicaset.apps/example-foo-4f8ay4vfr99ulx905hax3j6x1-9dfd67fc6   1         1         1       4m

```

Modify `deploy/crds/foo_v1alpha1_foo_cr.yaml` to set `replicaCount` to `2`:

```yaml
apiVersion: "foo.example.com/v1alpha1"
kind: "Foo"
metadata:
  name: "example-foo"
spec:
  # Add fields here
  replicaCount: 2
```

Apply the changes to Kubernetes and confirm that the deployment has 2 replicas:

```sh
$ kubectl apply -f deploy/crds/foo_v1alpha1_foo_cr.yaml
foo.foo.example.com/example-foo configured

$ kubectl get deployment -l app.kubernetes.io/instance=${RELEASE_NAME}
NAME                                    DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
example-foo-4f8ay4vfr99ulx905hax3j6x1   2         2         2            2           6m
```

### Testing a Helm operator on a cluster

Now that a developer is confident in the operator logic, testing the operator
inside of a pod on a Kubernetes cluster is desired. Running as a pod inside a
Kubernetes cluster is preferred for production use.

Build the `foo-operator` image and push it to a registry:

```sh
operator-sdk build quay.io/example/foo-operator:v0.0.1
docker push quay.io/example/foo-operator:v0.0.1
```

Kubernetes deployment manifests are generated in `deploy/operator.yaml`. The
deployment image in this file needs to be modified from the placeholder
`REPLACE_IMAGE` to the previous built image. To do this run:

```sh
sed -i 's|REPLACE_IMAGE|quay.io/example/foo-operator:v0.0.1|g' deploy/operator.yaml
```

**Note**
If you are performing these steps on OSX, use the following command:

```sh
sed -i "" 's|REPLACE_IMAGE|quay.io/example/foo-operator:v0.0.1|g' deploy/operator.yaml
```

Deploy the foo-operator:

```sh
kubectl create -f deploy/crds/foo_v1alpha1_foo_crd.yaml # if CRD doesn't exist already
kubectl create -f deploy/service_account.yaml
kubectl create -f deploy/role.yaml
kubectl create -f deploy/role_binding.yaml
kubectl create -f deploy/operator.yaml
```

Verify that the foo-operator is up and running:

```sh
$ kubectl get deployment
NAME               DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
foo-operator       1         1         1            1           1m
```

## Override values sent to Helm

The override values that are sent to Helm are managed by the operator. The
contents of the `spec` section are passed along verbatim to Helm and treated
like a value file would be if the Helm CLI were used (e.g.
`helm install -f overrides.yaml ./my-chart`)

For the CR example:

```yaml
apiVersion: "app.example.com/v1alpha1"
kind: "Database"
metadata:
  name: "example"
spec:
  message: "Hello world 2"
  newParameter: "newParam"
```

The structure passed to Helm as values is:

```yaml
message: "Hello world 2"
newParameter: "newParam"
```

[helm_charts]:https://docs.helm.sh/developing_charts
[helm_values]:https://docs.helm.sh/using_helm/#customizing-the-chart-before-installing
[helm_install]:https://docs.helm.sh/using_helm/
[helm_operator_user_guide]:../user-guide.md
