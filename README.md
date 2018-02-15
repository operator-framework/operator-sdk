# Getting Start with Operator SDK

## What’s Operator SDK?

## Guide assumptions

## Installation

## Creating a new project

Operator SDK comes with a number of code generators that are designed to facilitate development lifecycle. It helps create the project scaffolding, preprocess custom resource API to generate Kubernetes related code, generate deployment scripts -- just everything that is necessary to build an operator.

Navigate to `$GOPATH/src/github.com/example.com/`.

To start a project, we use the `new` generator to provide the foundation of a fresh operator project. Run the following command:

```
operator-sdk new play-operator
```

This will create the `play-operator` project with scaffolding with dependency code ready. It generates Kubernetes custom resource API of APIGroup `play.example.com` and Kind `PlayService` by default. APIGroups and Kinds can be overridden and added by flags.

Navigate to the project root folder:

```
cd play-operator
```

More details about the structure of the project can be found in [this doc][scaffold_doc].

## Up and running

At this point we are ready to build and deploy a functional operator. First build the binary and container:

```
operator-sdk build $image
docker push $image
```

Kubernetes deployment manifests will be generated in `deploy/play-operator/operator.yaml`. Deploy play-operator:

```
kubectl create -f deploy/play-operator/operator.yaml
```

The play-operator would be up and running:

```
# kubectl get deploy
NAME                DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
play-operator       1         1         1            1           1m
```

A `PlayService` CR with the following spec can be created to demonstrate the use of operator:

```
apiVersion: "play.coreos.com/v1"
kind: "PlayService"
metadata:
  name: "example"
spec:
  replica: 1
```

Once the CR is created, a new pod `example-box` will be created:

```
# kubectl get pod
NAME              READY     STATUS    RESTARTS   AGE
example-box       2/2       Running   0          1m
```

This is a basic test that verifies everything works correctly. Next we are going to write the business logic and do something more interesting.


## Customizing operator logic


[scaffold_doc]:./doc/project_layout.md
