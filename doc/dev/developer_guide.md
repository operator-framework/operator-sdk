# Operator SDK Developer guide

This document explains how to setup your dev environment.

## Prerequisites
- [git][git-tool]
- [go][go-tool] version v1.13+

## Download Operator SDK

Go to the [Operator SDK repo][repo-sdk] and follow the [fork guide][fork-guide] to fork, clone, and setup the local operator-sdk repository.

## Build the Operator SDK CLI

Build the Operator SDK CLI `operator-sdk` binary:

```sh
$ make install
```

Then, now you are able to test and use the operator-sdk build using the source code.

## Testing

The SDK includes many tests that are run as part of CI.
To build the binary and run all tests (assuming you have a correctly configured environment),
you can simple run:

```sh
$ make test-ci
```

If you simply want to run the unit tests, you can run:

```sh
$ make test
```

For more information on running testing and correctly configuring your environment,
refer to the [`Running the Tests Locally`][running-the-tests] document.

To run the lint checks done in the CI locally, run:

```sh
$ make lint
```

**NOTE** Note that for it is required to install `golangci-lint` locally. For more info see its [doc](https://github.com/golangci/golangci-lint#install)

## How to test the changes done for Ansible/Helm based-operator projects?

If the change performed is NOT in the scaffold files or sdk commands then, is required to build an new image with the changes done to test it locally. In this way, by using this dev image in an operator project locally we will be able to check the changes made for the Ansible/Helm based-operator. 

### For Ansible

- Update the `ANSIBLE_BASE_IMAGE` var in the [Makefile][makefile] to generate an image for your repository (quay.io or docker.hub.io). See:

Replace:
  
```
quay.io/operator-framework/ansible-operator
```

With (eg):

```
quay.io/my-repo-user/ansible-operator
```

- Build the image locally by running `make image-build-ansible`
- Push your new image. (E.g `docker push quay.io/my-repo-user/ansible-operator:dev`)

**NOTE** Ensure that you configured the repo, `quay.io/my-repo-user/ansible-operator`, to be public.

- Update the `Dockerfile` of your POC project to test your changes with the new image as follows. 

```
FROM quay.io/my-repo-user/ansible-operator:dev

COPY watches.yaml ${HOME}/watches.yaml
COPY roles/ ${HOME}/roles/
```  

### For Helm


- Update the `HELM_BASE_IMAGE` var in the [Makefile][makefile] to generate an image for your repository (quay.io or docker.hub.io). See:

Replace:
  
```
quay.io/operator-framework/helm-operator
```

With (eg):

```
quay.io/my-repo-user/helm-operator
```

- Build the image locally by running `make image-build-helm`
- Push your new image. (E.g `docker push quay.io/my-repo-user/helm-operator:dev`)

**NOTE** Ensure that you configured the repo, `quay.io/my-repo-user/helm-operator`, to be public.

- Update the `Dockerfile` of your POC project to test your changes with the new image as follows. 

```
FROM quay.io/my-repo-user/helm-operator:dev

COPY watches.yaml ${HOME}/watches.yaml
COPY helm-charts/ ${HOME}/helm-charts/
```  

See the project [README][sdk-readme] for more details.

[git-tool]:https://git-scm.com/downloads
[go-tool]:https://golang.org/dl/
[repo-sdk]:https://github.com/operator-framework/operator-sdk
[fork-guide]:https://help.github.com/en/articles/fork-a-repo
[docker-tool]:https://docs.docker.com/install/
[kubectl-tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[sdk-readme]:../../README.md
[running-the-tests]: ./testing/running-the-tests.md
[makefile]:../../Makefile 
