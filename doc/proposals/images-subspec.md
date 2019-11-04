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
see-also:
  - 'https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/contributors/design-proposals/related-images.md'
---

# Images Sub Spec

## Release Signoff Checklist

- \[ \] Enhancement is `implementable`
- \[ \] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Graduation criteria for dev preview, tech preview, GA
- \[ \] User-facing documentation is created in [operator-sdk/doc][operator-sdk-doc]

## Summary

When an operator is installed, the images that the operator installs as operands in the
form of pods, deployments or other kubernetes resources is not always clearly
defined. This proposal would require that images would be exposed and
overridable on the operator custom resource definitions (CRDs). This allows a user of a deployed
operator to override the images needed at deployment of the custom resource
(CR).

## Motivation

This proposal is driven by a few motivations:

- Makes operators more transparent with what is being installed on the cluster.
- Formalizes a best practice of operator development.
- The depencency between operator version and operator resource version is broken.
- Allows for out of band updates of the underlying images.
- Registry information for the images can easily be overridden to point at
  private registries.
- Provides some standardization to the operator CRD pattern.

## Goals

- Provide an easy mechanism of defining and overriding images and related image
  information (like pull secrets).
- Ensure the operators respect overridden images.
- Increase transparency with what operators are installing.

### Non-Goals

- OLM management of the images spec is not in this proposal.
- Mapping/re-mapping other container values (ports, etc.).

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

#### Add a spec section named images under CRD properties.

Create a custom spec interface that the generated spec objects will extend.
The spec interface includes a new list of objects called `images`. These images follows the
OpenAPIV3 spec listed below. Additionally add APIs that transform images rows to
Kubernetes API resources to help developers onboard.

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

#### Add a scorecard test to verify changes to image locations.

Scorecard would have a default check for images similar to the writing into CRs
check. Ideally the scorecard test would create a new version of the image in the
internal docker registry that points to the old images and updates the CR. The
newly deployed images would be using the local paths.

#### Create an operator migration tool for clients.

A tool could reasonably look through code and generate the images subspec for
the operator developers. May also be able to modify code in place. Goal would be
to ease onboarding.

#### Add Images Spec support for operator-sdk golang client.

Newly generated golang operators should use the new Images Spec and provide
documentation for the new feature.

Modify the `operator-sdk add crd` command to add a type of crd that has images.

#### Add Images Spec support for operator-sdk ansible client.

Newly generated ansible operators should use the new Images Spec and provide
documentation for the new feature.

#### Add Images Spec support for operator-sdk helm client.

Newly generated helm operators should use the new Images Spec and provide
documentation for the new feature.

### Risks and Mitigations

- A user could override the images in an operator with images
  that will not work with the operator. This would just result in a broken
  operator. The status would have to reflect a failure because of wrong software
  version.
- Providing default values for the images can be tricky. The default keyword in
  the openapi v3 schema is in beta. Using enum is another method of providing a
  default value but requires input by the creator of the CR. Mutating webhooks
  is another method.
- If a scorecard check is not practical, then validation becomes a concern.
- Separating a CRD for deployable resource like a mongo db vs a mongo user.

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
