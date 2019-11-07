---
title: Images Sub-specification
authors:
  - '@zachncst'
reviewers:
  - TBD
approvers:
  - TBD
creation-date: 2019-11-04
last-updated: 2019-11-05
status: provisional
help-from:
  - '@camilamacedo86'
  - '@kevinrizza'
  - '@shawn-hurley'
see-also:
  - 'https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/contributors/design-proposals/related-images.md'
---

# Images Sub Spec

## Release Signoff Checklist

- \[ \] Enhancement is `implementable`
- \[ \] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Graduation criteria for dev preview, tech preview, GA
- \[ \] User-facing documentation is created in
  [operator-sdk/doc][operator-sdk-doc]

## Tag Line

> Who likes hard coding image locations anyways?

## Summary

When an operator is installed, the images that the operator installs as operands in the
form of pods, deployments or other kubernetes resources is not always clearly
defined and editable by a user of an operator. This proposal would define
an operator-sdk ImageSpec and ImagesListSpec object structs that operator developers
could use to enhance the creation of operator CRDs or improve existing
operators. The spec objects would let operator users override image locations
for that custom resource.

Additionally, the proposal would provide tooling to generate scaffold code for
creating new operators. When creating a new operator, an operator developer
would be able to pass flags that would create a base spec object that includes
the ImagesListSpec.

## Motivation

This proposal is driven by a few motivations:

- Makes operators more transparent with what is being installed on the cluster.
- Formalizes a best practice of operator development.
- The depencency between operator version and operator resource version is broken.
- Allows for out of band updates of the underlying images.
- Registry information for the images can easily be overridden to point at
  private registries.
- Provides some (potential) standardization to the CRD.

## Goals

- Provide an easy mechanism of defining and overriding images and related image
  information (like pull secrets).
- Increase transparency with what operators are installing.
- Simple, reusable specs to build operators; similar to PodSpec in Kubernetes.

### Non-Goals

- OLM management of the images spec is not in this proposal.
- Mapping/re-mapping other container values (ports, etc.).
- Additional pod information, like env vars.

## Proposal

Implementing the proposal would include:

- Add an optional section to the CRD spec named images that contains definitions of
  all images used by the operator. An example:

  ```yaml
  apiVersion: app.example.com/v1alpha1
  kind: AppService
  metadata:
    name: example-appservice
  spec:
    size: 3
    images:
      - name: nginx
        image: nginx:1.7.9
      - name: example
        iamge: my-registry:5000/example:1.0.0
  ```

- The images spec property is required to be read by the operator to deploy resources
  it creates (pods, deployments, statefulsets) where additional images are used.
- Add scorecard tests to verify the images spec is being used.
- Modify any operator-sdk clients to generate the images spec.

### User Stories

#### Create a common ImageListSpec and ImageSpec Interface.

Follow a kubernetes api pattern and provide an ImageSpec and ImageListSpec structs that
operator CRDs can reuse. I imagine operator-sdk would publish these like the
[PodSpec](https://github.com/kubernetes/api/blob/master/core/v1/types.go#L2831) found for kubernetes.

OpenAPIv3 Def for ImageSpec:

```yaml
type: Object
properties:
  name:
    type: string
    description: Name of image used for lookup.
  image:
    type: string
    description: Location of the image.
  pullPolicy:
    type: string
    description: Pull Policy for the image.
  pullSecrets:
    type: array
    description: Array of pull secret names to use.
    items:
      types: string
required: ['name', 'image']
```

OpenAPIv3 Def for ImageListSpec:

```yaml
images:
  type: array
  items:
    type: object
    properties:
      name:
        type: string
        description: Name of image used for lookup.
      image:
        type: string
        description: Location of the image.
      pullPolicy:
        type: string
        description: Pull Policy for the image.
      pullSecrets:
        type: array
        description: Array of pull secret names to use.
        items:
          types: string
  required: ['name', 'image']
```

#### Add an image flag to sdk add api

Add a flag to preset images for operator-sdk add api command.

```bash
operator-sdk add api --api-version=app.example.com/v1alpha1 \
  --kind=AppService --image=ngnix:latest
operator-sdk add api --api-version=app.example.com/v1alpha1 \
  --kind=AppService \
  --image=ngnix:latest \
  --image=redis:latest
```

This flag will create an AppServiceSpec struct with prefilled data.

```golang
type AppServiceSpec struct {

  // Object map of image to name to an image location, pull
  // secret names, and pull policy for the image.
  Images operatorsdk.ImagesSpec

	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

}
```

#### Add a generate flag for operator-sdk specs

Generate flag added to operator-sdk add controller.

**Blanket generate:**
```bash
 operator-sdk add controller --api-version=app.example.com/v1alpha1 \
   --kind=AppService  \
   --generate
```

The generate flag will create the controller with scaffolding using the Images
property to create resources to be deployed. The generate command could be a
global flag that enables any type of operatorsdk features that generate
controller code. And we can include an option to subselect a generation.

**Subselecting a generation:**
```bash
 operator-sdk add controller --api-version=app.example.com/v1alpha1 \
   --kind=AppService  \
   --generate=images
```

### Future Work

#### Validation

Operator score card and linters could perform validation of the use of
ImageSpecs in the controllers.


### Risks and Mitigations

- A user could override the images in an operator with images
  that will not work with the operator. This would just result in a broken
  operator. The status would have to reflect a failure because of wrong software
  version.
- Providing default values for the images can be tricky. The default keyword in
  the openapi v3 schema is in beta. Using enum is another method of providing a
  default value but requires input by the creator of the CR. Mutating webhooks
  is another method.

### Upgrade / Downgrade Strategy

NA.

### Version Skew Strategy

NA.

## Implementation History

Major milestones in the life cycle of a proposal should be tracked in `Implementation History`.

## Drawbacks

- Requires changes from the operator developers.
- Different operators may already solve this problem in different ways.
- Not all CRDs require an image (adding a user to a database for example).
- Adds a requried data to an operator CRD that is otherwise freeform.

## Alternatives

There is a similar proposal for the CSV definition [related
images](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/contributors/design-proposals/related-images.md)
that OLM will use. This approach has a few problems for operands:

- Directly couples the image versions with the operator.
- Requires OLM support, or custom kubernetes runtimes (CRI-O), to override the image
  locations at runtime without operator support.
- Images cannot be overriden by a user.

[operator-sdk-doc]: ../../doc
