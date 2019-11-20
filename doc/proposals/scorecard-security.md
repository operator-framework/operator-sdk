---
title: Testing the Security of Operators by tooling
authors:
  - "@camilamacedo"
reviewers:
  - TBD
approvers:
  - TBD
creation-date: 2019-10-31
last-updated: 2019-10-31
status: provisional
see-also:
  - "doc/proposals/openshift-4.3/operator-testing-tool.md" 
---

# Testing the Security of Operators by tooling

## Release Signoff Checklist

- \[ X\] Enhancement is `provisional`
- \[ X\] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Graduation criteria for dev preview, tech preview, GA
- \[ \] User-facing documentation is created in [operator-sdk/doc][operator-sdk-doc]

## Open Questions (optional)

- Would have more security checks which may are important to the Operators and are not mapped here? 

## Summary

Currently, the scorecard will perform some tests. For further information see its [doc](https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/scorecard.md#tests-performed).  
This proposal is regarded to add a new suite of tests in order to ensure the quality of the operators by checking if they are complying with few common/general security practices. 
In order to clarifies and exemplify this need as solution see the project https://kubesec.io/ and its demo which verifies and check if the resource is complying with the security common practices.   

Note that the checks of this proposal are relevant to any solution which will be deployed in K8s platforms, however, it could be improved in the future in order to accommodate specific implementations and/or checks per type of platform or as suggested to perform further checks in the images used.     

## Motivation

- Ensure the quality of the operators projects by checking their security.   
- Ensure the quality by checking the security of the projects which are distributed by OperatorHub.io and OpenShift catalogue

## Goals

* The test should fail when 1 or more resources created by the project are not complaining with the security rules adopted

## Proposal

### User Stories (optional)

#### Story 1

I as an operator dev user, I'd like to check the quality of my operator by verifying via tooling if it is complying with the minimal/common requirements. 

**Acceptance Criteria**

- The tooling/check should fail if the operator project creates a resource which not complain with the roles defined
- The tooling/check should fail if the operator project do not label all resources created by it
- The tooling/check should show what/where/why the operator did not pass in the tests for each failure rule

### Implementation Details/Notes/Constraints

#### Details 

For the following checks is possible to verify the Operator.YAML file:

- Check if Operator has resources limits defined (memory/CPU)
- Check if Operator has requests limits setup (memory/CPU)

```yaml
          resources:
            limits:
              cpu: 60m
              memory: 128Mi
            requests:
              cpu: 30m
              memory: 64Mi
```
NOTE: we can parse the YAML file to JSON and just check if the fields are populated to check t. 
For the following checks would be required to have the operator and its CR/CRD's applied in the cluster in order to get all pods/deployments created by it. 

- Check if Pods/Deployments containers are using root as user
- Check if Pods/Deployments has resources limits setup (memory/cpu)
- Check if Pods/Deployments has requests limits setup (memory/cpu)
- Check if Pods/Deployments containers are using privileged capability as SYS_ADMIN and CAP_SYS_ADMIN
- Check if Pods/Deployments containers are using user ID > 10000 (should fail when < 10000)

To ensure that all resources created by the operator will have a label: 
- (suggestion/possible solution) Then, make mandatory the `labels` field be filled in the CSV file.   E.g [cvs](https://github.com/dev4devs-com/postgresql-operator/blob/master/deploy/olm-catalog/postgresql-operator/0.1.1/postgresql-operator.v0.1.1.clusterserviceversion.yaml#L661)

[operator-sdk-doc]:  ../../doc

#### Notes

Following some explanations and references over these checks. 

**Resources limits defined (memory/CPU)**

A good explanation can be found [here](https://kubesec.io/basics/containers-resources-limits-cpu/) as some reference links. 

**SYS_ADMIN and CAP_SYS_ADMIN**

Note that malicious users with this privileged capability as `CAP_SYS_ADMIN` can have access to the `/proc/` which allows access to the host and, in this way, compromise not only the container.

See:

> Don't choose CAP_SYS_ADMIN if you can possibly avoid it! A vast
proportion of existing capability checks are associated with this
capability (see the partial list above). It can plausibly be
called "the new root", since on the one hand, it confers a wide
range of powers, and on the other hand, its broad scope means that
this is the capability that is required by many privileged
programs. Don't make the problem worse. The only new features
that should be associated with CAP_SYS_ADMIN are ones that closely
match existing uses in that silo.

Reference: http://man7.org/linux/man-pages/man7/capabilities.7.html

**Containers are using user ID > 10000**

A good explanation can be found [here](https://kubesec.io/basics/containers-securitycontext-runasuser/) as some reference links. 

> NOTE: Regards OCP the users attribute to the containers will be by default > 10000. The OCP installation will use `"openshift.io/sa.scc.uid-range": "1000000000/10000"`. See [here](https://github.com/openshift/openshift-tools/search?q=openshift.io%2Fsa.scc.uid-range&unscoped_q=openshift.io%2Fsa.scc.uid-range). 

**Containers are using root as user** 

Malicious users with full access as root can compromise not just the container but the host as well. 
