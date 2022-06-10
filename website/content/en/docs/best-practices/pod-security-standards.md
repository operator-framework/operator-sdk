---
title: "Pod Security Standards"
linkTitle: "Pod Security Standards"
weight: 5
description: This guide describes good practices concepts to ensure the Pod/containers Security Standards.
---

## Overview

Kubernetes API has been changing, and the [PodSecurityPolicy][pod-security] API is deprecated and will no longer be served from k8s `1.25`. 
This API is replaced by a new built-in admission controller ([KEP-2579: Pod Security Admission Control][2579-psp-replacement]) which allows cluster admins to [enforce 
the Pod Security Standards][enforce-standards-namespace-labels].

#### What does that mean?

That means that Pod/containers that are not configured according to the enforced security standards defined globally or 
on the namespace level will not be admitted and in this way, it will not be possible to run them.

**In this way, the best approach is to ensure that workloads (Operators, Operands) are defined such that they can run under restricted permissions.**

#### How should I configure my Operators and Operands to comply with the criteria?

- A) **(Recommend for common cases, when it does not require escalating privileges)** ensure all containers are configured 
to comply with the [restrictive][restricted] policy. Following the examples:

**On Kubernetes manifests:**

```yaml
    spec:
      securityContext:
        runAsNonRoot: true
        seccompProfile:
          type: RuntimeDefault
      ...
      containers:
      - name: controller-manager
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
        ...
```

**On Reconciliations (i.e of code implementation in Go):**

```go
dep:= &appsv1.Deployment{
  ObjectMeta: metav1.ObjectMeta{
  ….
  },
  Spec: appsv1.DeploymentSpec{
    …
     Template: corev1.PodTemplateSpec{
       ….
        Spec: corev1.PodSpec{
           // Ensure restrictive context for the Pod    
           SecurityContext: &corev1.PodSecurityContext{
              RunAsNonRoot: &[]bool{true}[0],
              SeccompProfile: &corev1.SeccompProfile{
                 Type: corev1.SeccompProfileTypeRuntimeDefault,
              },
           },
           Containers: []corev1.Container{{
              Image:   "memcached:1.4.36-alpine",
              Name:    "memcached",
              // Ensure restrictive context for the container  
              SecurityContext: &corev1.SecurityContext{
                 RunAsNonRoot:  &[]bool{true}[0],
                 AllowPrivilegeEscalation:  &[]bool{false}[0],
                 Capabilities: &corev1.Capabilities{
                    Drop: []corev1.Capability{
                       "ALL",
                    },
                 },
              },
           }},
        },
     },
  },
}
```

**Note:** For Ansible/Helm based-language operators, you need to ensure that your ansible playbooks or charts respectively 
are creating the manifests complying with the requirements.

**OR**

- B) Ensure the namespace has the appropriate enforcement level label if the workload needs more than restricted permissions ( check the following example ). 
This may need to be part of your operator install instructions.  While the label syncer should handle this for you in most cases, it is preferred that the Operator be explicit about its requirements.

```yaml
  labels:
    ...
    pod-security.kubernetes.io/enforce: privileged
    pod-security.kubernetes.io/audit: privileged
    pod-security.kubernetes.io/warn: privileged
```

**Note that you should ensure the above configuration is carried to the Pod/Containers on the bundle CSV (install.spec.deployments.containers).**
To check an example of CSV which comply with the [restrictive][restricted] policy, see the Golang sample
under the [testdata/go/v3/memcached-operator/bundle/manifests/memcached-operator.clusterserviceversion.yaml](./../../../../../testdata/go/v3/memcached-operator/bundle/manifests/memcached-operator.clusterserviceversion.yaml)

[pod-security]: https://kubernetes.io/blog/2021/04/06/podsecuritypolicy-deprecation-past-present-and-future/#what-is-podsecuritypolicy
[2579-psp-replacement]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-auth/2579-psp-replacement
[enforce-standards-namespace-labels]: https://kubernetes.io/docs/tasks/configure-pod-container/enforce-standards-namespace-labels/
[restricted]: https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted