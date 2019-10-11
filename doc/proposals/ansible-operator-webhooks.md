---
title: Ansible Operator Webhooks
authors:
  - "@fabianvf"
reviewers:
  - TBD
approvers:
  - TBD
creation-date: 2019-10-14
last-updated: 2019-10-14
status: provisional
---

# Ansible Operator Webhooks

## Release Signoff Checklist

- \[ \] Enhancement is `implementable`
- \[ \] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Graduation criteria for dev preview, tech preview, GA
- \[ \] User-facing documentation is created in [operator-sdk/doc][operator-sdk-doc]

## Open Questions (optional)

- For non-trivial playbooks, will the performance be acceptable?
- How much webhook configuration should we support?
- Should we support side-effects?
    Side-effects will require adding reconciliation mechanisms to the webhook,
    and would likely impact the performance even more extremely. My instinct is to not support it as it adds
    a lot of complexity and seems dangerous, but then we're obviously restricting the power of the webhooks.
    If we don't allow side-effects, we could enforce that by making the proxy reject requests from the webhooks
    that do anything but `GET`.
- Need to determine exactly what information we should send to the webhooks, and how it will be structured.
    Passing this information to the Runner will require some minor refactors.
- Where should artifacts for the webhook runs end up?

## Summary

This proposal defines a way for users to implement webhooks using Ansible playbooks/roles.

## Motivation

- [Conversion webhooks](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definition-versioning/#webhook-conversion)
    are necessary to support CRD versioning
- [Mutating webhooks](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook)
    are necessary to support defaulting, though the addition of the default field to the OpenAPI spec may make this one 
    less important in the long-term.
- [Validating webhooks](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#validatingadmissionwebhook) 
    are necessary for higher level validation of incoming resources, for example SAR checks. There is an open community 
    issue explicitly requesting this feature: https://github.com/operator-framework/operator-sdk/issues/1658 .

Currently Ansible-based operators have no webhook support at all, a user would have to write their own webhook server 
and deploy it alongside the operator to get that functionality.

## Goals
- Allow the creation and use of conversion/validating/mutating webhooks implemented with Ansible

### Non-Goals
- Handle xWebhookConfiguration objects or associated Services
- Handle cert management

## Proposal

Add a webhooks section to the `watches.yaml`, that allows the user to specify lists of validating/mutating/conversion 
webhooks that should be created. 

The user should only need to provide a playbook or a role, and the url path for the webhook. 

The rest of the webhook configuration, ie the creation of the Service that points to the webhook server, the Webhook 
Configuration resources that lay out the rules and point to the proper Service, and the management of the certificates 
for the webhook server, will be the responsiblity of the user. 

The webhook configuration will look roughly as follows, with a big asterisk on the conversion webhooks which need some
further investigation.

```yaml
---
- version: v1alpha1
  group: apps.example.com
  kind: MyApp
  role: /opt/ansible/roles/myapp
  webhooks:
    validating:
    - playbook: /opt/ansible/validate.yml
      path: /validate
    mutating:
    - role: /opt/ansible/roles/mutate-myapp
      path: /mutate
    conversion:
    - playbook: /opt/ansible/conversion.yml
      path: /convert
```

The justification for allowing multiple webhooks per type is that webhooks can select objects by a variety of
attributes, so you could have a different webhook which is selected based on `metadata.labels`.

Based on the webhook configuration, during initialization we will create a webhook server and add it to manager,
and then register each of the webhooks to the webhook server, at the given path. We will create handlers that
respond to the requests by running the associated Ansible roles or playbooks.

- For validating webhooks, playbook success => Allowed, and playbook failure => Denied. We will report the
    failureMessages in the denial response.
- For mutating webhooks, we will create a custom module that will be used to report back the final desired
    object. We may need to write additional logic here (either in the module or on the Go-side) that generates
    a proper patch object from this.
- Conversion webhooks should work roughly the same as mutating webhooks.

### Risks and Mitigations


## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything
that would count as tricky in the implementation and anything particularly
challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations).

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. Initial proposal
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this
enhancement:
- Maturity levels - `Dev Preview`, `Tech Preview`, `GA`
- Deprecation

Clearly define what graduation means.

#### Examples

These are generalized examples to consider, in addition to the aforementioned maturity levels(`Dev Preview`, `Tech Preview`, `GA`).

##### Dev Preview -> Tech Preview

- Ability to utilize the enhancement end to end
- End user documentation, relative API stability
- Sufficient test coverage
- Gather feedback from users rather than just developers

##### Tech Preview -> GA 

- More testing (upgrade, downgrade, scale)
- Sufficient time for feedback
- Available by default

**For non-optional features moving to GA, the graduation criteria must include
end to end tests.**

##### Removing a deprecated feature

- Announce deprecation and support policy of the existing feature
- Deprecate the feature

### Upgrade / Downgrade Strategy

If applicable, how will the component be upgraded and downgraded? Make sure this
is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to make use of the enhancement?

### Version Skew Strategy

How will the component handle version skew with other components?
What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- During an upgrade, we will always have skew among components, how will this impact your work?
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI, CRI
  or CNI may require updating that component before the kubelet.

## Implementation History

Major milestones in the life cycle of a proposal should be tracked in `Implementation
History`.

## Drawbacks

The idea is to find the best form of an argument why this enhancement should _not_ be implemented.

## Alternatives

Similar to the `Drawbacks` section the `Alternatives` section is used to
highlight and record other possible approaches to delivering the value proposed
by an enhancement.
