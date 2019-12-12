# Helm Developer guide for Operator SDK

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

### Testing a Helm chart locally

Sometimes it is beneficial for a developer to run the Helm chart installation
from their local machine as opposed to running/rebuilding the operator each
time. To do this, initialize a new project:

```sh
$ operator-sdk new --type helm --kind Foo --api-version foo.example.com/v1alpha1 foo-operator
INFO[0000] Creating new Helm operator 'foo-operator'.
INFO[0000] Created helm-charts/foo
INFO[0000] Generating RBAC rules
WARN[0000] The RBAC rules generated in deploy/role.yaml are based on the chart's default manifest. Some rules may be missing for resources that are only enabled with custom values, and some existing rules may be overly broad. Double check the rules generated in deploy/role.yaml to ensure they meet the operator's permission requirements.
INFO[0000] Created build/Dockerfile
INFO[0000] Created watches.yaml
INFO[0000] Created deploy/service_account.yaml
INFO[0000] Created deploy/role.yaml
INFO[0000] Created deploy/role_binding.yaml
INFO[0000] Created deploy/operator.yaml
INFO[0000] Created deploy/crds/foo.example.com_foos_crd.yaml
INFO[0000] Created deploy/crds/foo.example.com_v1alpha1_foo_cr.yaml
INFO[0000] Project creation complete.

$ cd foo-operator
```

For this example we will use the default Nginx helm chart scaffolded by
`operator-sdk new`. Without making any changes, we can see what the default
release manifests are:

```sh
$ helm install --dry-run test-release helm-charts/foo
NAME: test-release
LAST DEPLOYED: Wed Nov 27 15:41:10 2019
NAMESPACE: default
STATUS: pending-install
REVISION: 1
HOOKS:
---
# Source: foo/templates/tests/test-connection.yaml
apiVersion: v1
kind: Pod
metadata:
  name: "test-release-foo-test-connection"
  labels:

    helm.sh/chart: foo-0.1.0
    app.kubernetes.io/name: foo
    app.kubernetes.io/instance: test-release
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
  annotations:
    "helm.sh/hook": test-success
spec:
  containers:
    - name: wget
      image: busybox
      command: ['wget']
      args:  ['test-release-foo:80']
  restartPolicy: Never
MANIFEST:
---
# Source: foo/templates/serviceaccount.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-release-foo
  labels:

    helm.sh/chart: foo-0.1.0
    app.kubernetes.io/name: foo
    app.kubernetes.io/instance: test-release
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
---
# Source: foo/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: test-release-foo
  labels:
    helm.sh/chart: foo-0.1.0
    app.kubernetes.io/name: foo
    app.kubernetes.io/instance: test-release
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
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
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-release-foo
  labels:
    helm.sh/chart: foo-0.1.0
    app.kubernetes.io/name: foo
    app.kubernetes.io/instance: test-release
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
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
      serviceAccountName: test-release-foo
      securityContext:
        {}
      containers:
        - name: foo
          securityContext:
            {}
          image: "nginx:1.16.0"
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

NOTES:
1. Get the application URL by running these commands:
  export POD_NAME=$(kubectl get pods --namespace default -l "app.kubernetes.io/name=foo,app.kubernetes.io/instance=test-release" -o jsonpath="{.items[0].metadata.name}")
  echo "Visit http://127.0.0.1:8080 to use your application"
  kubectl --namespace default port-forward $POD_NAME 8080:80
```

Next, let's go ahead and install the release:

```sh
$ helm install test-release helm-charts/foo
NAME: test-release
LAST DEPLOYED: Wed Nov 27 15:43:04 2019
NAMESPACE: default
STATUS: deployed
REVISION: 1
NOTES:
1. Get the application URL by running these commands:
  export POD_NAME=$(kubectl get pods --namespace default -l "app.kubernetes.io/name=foo,app.kubernetes.io/instance=test-release" -o jsonpath="{.items[0].metadata.name}")
  echo "Visit http://127.0.0.1:8080 to use your application"
  kubectl --namespace default port-forward $POD_NAME 8080:80
```

Check that the release resources were created:

```sh
$ kubectl get all -l app.kubernetes.io/instance=test-release
NAME                                    READY   STATUS    RESTARTS   AGE
pod/test-release-foo-76bd6c5f58-vqj7m   1/1     Running   0          36s

NAME                       TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)   AGE
service/test-release-foo   ClusterIP   10.106.222.153   <none>        80/TCP    36s

NAME                               READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/test-release-foo   1/1     1            1           36s

NAME                                          DESIRED   CURRENT   READY   AGE
replicaset.apps/test-release-foo-76bd6c5f58   1         1         1       36s
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

Now let's upgrade the release to use these new values from `overrides.yaml`:

```sh
$ helm upgrade -f overrides.yaml test-release helm-charts/foo
Release "test-release" has been upgraded. Happy Helming!
NAME: test-release
LAST DEPLOYED: Wed Nov 27 15:45:15 2019
NAMESPACE: default
STATUS: deployed
REVISION: 2
NOTES:
1. Get the application URL by running these commands:
  export POD_NAME=$(kubectl get pods --namespace default -l "app.kubernetes.io/name=foo,app.kubernetes.io/instance=test-release" -o jsonpath="{.items[0].metadata.name}")
  echo "Visit http://127.0.0.1:8080 to use your application"
  kubectl --namespace default port-forward $POD_NAME 8080:80
```

Now you'll see that there are 2 deployment replicas and the service port
has been updated to `8080`.

```sh
$ kubectl get deployment -l app.kubernetes.io/instance=test-release
NAME               READY   UP-TO-DATE   AVAILABLE   AGE
test-release-foo   2/2     2            2           2m58s

$ kubectl get service -l app.kubernetes.io/instance=test-release
NAME               TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)    AGE
test-release-foo   ClusterIP   10.106.222.153   <none>        8080/TCP   2m59s
```

Lastly, uninstall the release:

```sh
$ helm uninstall test-release
release "test-release" uninstalled
```

Check that the resources were deleted:

```sh
$ kubectl get all -l app.kubernetes.io/instance=test-release
No resources found in default namespace.
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
will be `/opt/helm/helm-charts/busybox`. If the path is relative, it is
relative to the current working directory.

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
communicate with a Kubernetes cluster just as the `helm` CLI commands did
when we were testing our Helm chart locally. This section assumes the developer
has read the [Helm Operator user guide][helm_operator_user_guide] and has the
proper dependencies installed.

Create a Custom Resource Definition (CRD) for resource Foo. `operator-sdk` autogenerates this file
inside of the `deploy` folder:

```sh
kubectl create -f deploy/crds/foo.example.com_foos_crd.yaml
```

**NOTE:** When running the Helm operator locally, the `up local` command will default to using the kubeconfig file specified by `$KUBECONFIG` with a fallback to `$HOME/.kube/config` if not set. In this case, the autogenerated RBAC definitions do not need to be applied to the cluster.

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
`deploy/crds/foo.example.com_v1alpha1_foo_cr.yaml`. The `spec` was generated
from the chart's default `values.yaml` file, so creating this CR will result
in the chart's default manifest being installed.

```yaml
apiVersion: foo.example.com/v1alpha1
kind: Foo
metadata:
  name: example-foo
spec:
  # Default values copied from <project_dir>/helm-charts/foo/values.yaml

  affinity: {}
  fullnameOverride: ""
  image:
    pullPolicy: IfNotPresent
    repository: nginx
  imagePullSecrets: []
  ingress:
    annotations: {}
    enabled: false
    hosts:
    - host: chart-example.local
      paths: []
    tls: []
  nameOverride: ""
  nodeSelector: {}
  podSecurityContext: {}
  replicaCount: 1
  resources: {}
  securityContext: {}
  service:
    port: 80
    type: ClusterIP
  serviceAccount:
    create: true
    name: null
  tolerations: []

```


Create a Custom Resource instance of Foo using these default values:

```sh
$ kubectl apply -f deploy/crds/foo.example.com_v1alpha1_foo_cr.yaml
foo.foo.example.com/example-foo created
```

The custom resource status will be updated with the release information if the
installation succeeds. Let's get the release name:

```sh
$ export RELEASE_NAME=$(kubectl get foos example-foo -o jsonpath={..status.deployedRelease.name})
$ echo $RELEASE_NAME
example-foo
```

Note that the release name matches the name of the CR. Next, check that the release resources were created:

```sh
$ kubectl get all -l app.kubernetes.io/instance=${RELEASE_NAME}
NAME                               READY   STATUS    RESTARTS   AGE
pod/example-foo-69594454bc-mvz2l   1/1     Running   0          2m39s

NAME                  TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
service/example-foo   ClusterIP   10.96.236.249   <none>        80/TCP    2m39s

NAME                          READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/example-foo   1/1     1            1           2m39s

NAME                                     DESIRED   CURRENT   READY   AGE
replicaset.apps/example-foo-69594454bc   1         1         1       2m39s
```

Modify `deploy/crds/foo.example.com_v1alpha1_foo_cr.yaml` to set `replicaCount` to `2`:

```yaml
apiVersion: "foo.example.com/v1alpha1"
kind: "Foo"
metadata:
  name: "example-foo"
spec:
  # ...
  replicaCount: 2
  # ...
```

Apply the changes to Kubernetes and confirm that the deployment has 2 replicas:

```sh
$ kubectl apply -f deploy/crds/foo.example.com_v1alpha1_foo_cr.yaml
foo.foo.example.com/example-foo configured

$ kubectl get deployment -l app.kubernetes.io/instance=${RELEASE_NAME}
NAME          READY   UP-TO-DATE   AVAILABLE   AGE
example-foo   2/2     2            2           3m35s
```

Lastly, to uninstall the release, simply delete the CR and verify that the
release resources have been deleted:

```sh
$ kubectl delete -f deploy/crds/foo.example.com_v1alpha1_foo_cr.yaml
foo.foo.example.com "example-foo" deleted

$ kubectl get all -l app.kubernetes.io/instance=${RELEASE_NAME}
No resources found in default namespace.
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
kubectl create -f deploy/crds/foo.example.com_foos_crd.yaml # if CRD doesn't exist already
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

Apply the CR to Kubernetes and confirm that the release resources have been
created:

```sh
$ kubectl apply -f deploy/crds/foo.example.com_v1alpha1_foo_cr.yaml
foo.foo.example.com/example-foo configured

$ kubectl get all -l app.kubernetes.io/instance=${RELEASE_NAME}
NAME                               READY   STATUS    RESTARTS   AGE
pod/example-foo-69594454bc-4z92w   1/1     Running   0          10s
pod/example-foo-69594454bc-wp4sl   1/1     Running   0          10s

NAME                  TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)   AGE
service/example-foo   ClusterIP   10.107.177.143   <none>        80/TCP    10s

NAME                          READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/example-foo   2/2     2            2           10s

NAME                                     DESIRED   CURRENT   READY   AGE
replicaset.apps/example-foo-69594454bc   2         2         2       10s
```

Lastly, to uninstall the release, simply delete the CR and verify that the
release resources have been deleted:

```sh
$ kubectl delete -f deploy/crds/foo.example.com_v1alpha1_foo_cr.yaml
foo.foo.example.com "example-foo" deleted

$ kubectl get all -l app.kubernetes.io/instance=${RELEASE_NAME}
No resources found in default namespace.
```

[helm_charts]:https://helm.sh/docs/topics/charts/
[helm_values]:https://helm.sh/docs/intro/using_helm/#customizing-the-chart-before-installing
[helm_install]:https://helm.sh/docs/intro/install/
[helm_operator_user_guide]:../user-guide.md
