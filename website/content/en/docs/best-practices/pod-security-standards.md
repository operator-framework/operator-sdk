---
title: "Pod Security Standards"
linkTitle: "Pod Security Standards"
weight: 5
description: This guide describes good practices for security standards in Pods and containers.
---

## Overview

The [PodSecurityPolicy][pod-security] API is deprecated and will be removed from Kubernetes in version 1.25. 
This API is replaced by a new built-in admission controller ([KEP-2579: Pod Security Admission Control][2579-psp-replacement]) which allows cluster admins to [enforce 
 Pod Security Standards][enforce-standards-namespace-labels].

#### What does that mean?

That means that Pod/containers that are not configured according to the enforced security standards defined globally or 
on the namespace level will not be admitted and in this way, it will not be possible to run them.

**As a best practice, you must ensure that workloads (Operators and Operands) are defined to run under restricted permissions.**

#### How should I configure my Operators and Operands to comply with the criteria?

- **For common cases that do not require escalating privileges:** configure all containers to comply with the [restrictive][restricted] policy as shown in the following the examples:

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

**On Reconciliations, such as code implementation in Go:**

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

**Note:** For Ansible- and Helm-based Operator projects, your Ansible playbooks or Helm charts must create manifests that comply with the requirements.

**OR**

- B) **For workloads that need elevated permissions:** Ensure the namespace has the appropriate enforcement level label as shown in the following example.
You might need include this in the installation documentation for your Operator.  While the label syncer should handle this for you in most cases, it is a good practice for Operators to explicitly state its requirements.

```yaml
  labels:
    ...
    pod-security.kubernetes.io/enforce: privileged
    pod-security.kubernetes.io/audit: privileged
    pod-security.kubernetes.io/warn: privileged
```

**You should ensure the configuration is carried to the Pod/Containers on the bundle CSV (install.spec.deployments.containers).**
To check an example of CSV which complies with the [restrictive][restricted] policy, see the Golang sample
under the [testdata/go/v3/memcached-operator/bundle/manifests/memcached-operator.clusterserviceversion.yaml](https://github.com/kubernetes-sigs/kubebuilder/blob/master/testdata/go/v3/memcached-operator/bundle/manifests/memcached-operator.clusterserviceversion.yaml)

- [pod-security]: https://kubernetes.io/blog/2021/04/06/podsecuritypolicy-deprecation-past-present-and-future/#what-is-podsecuritypolicy
- [2579-psp-replacement]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-auth/2579-psp-replacement
- [enforce-standards-namespace-labels]: https://kubernetes.io/docs/tasks/configure-pod-container/enforce-standards-namespace-labels/
- [restricted]: https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted