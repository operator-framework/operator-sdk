---
title: Testing Changes Locally
linkTitle: Local Changes
---

## Testing Changes Locally

If your changes are in the SDK commands then you just need to run `make install` to be able to use an SDK binary built from the source code and then test locally it. Also, see that you can run `operator-sdk version` to check what is the commit used to built it. 

However, If the change performed is NOT in the scaffold files or sdk commands then, is required to build an new image with the changes done to test it locally in a POC operator project. In this way, by using this dev image in an operator project locally we will be able to check the changes made for the Ansible/Helm based-operator. And then, for the GO based-operators, will be required ensure that you are import your version of code implementation to be used in the POC operator project were it will be checked.  

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

### For Go

Following an example over how to test the changes made from a source code of a fork PR. 

- Update the `go.mod` file of the POC operator project with a replace for the fork. See:

```
require (
	...
	github.com/operator-framework/operator-sdk v0.0.0
	...
)

// # Add a replace to the fork and branch with the changes
replace github.com/operator-framework/operator-sdk => github.com/<fork>/operator-sdk <branch>
```

[makefile]: https://github.com/operator-framework/operator-sdk/blob/master/Makefile
