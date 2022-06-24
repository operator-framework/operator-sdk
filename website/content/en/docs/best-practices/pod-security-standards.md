---
title: "Pod Security Standards"
linkTitle: "Pod Security Standards"
weight: 5
description: This guide describes best practices for security standards for Operators and Operands. This guide will also cover how to configure the Pods/Containers that will be created when your Operator is installed with OLM.
---

## Overview

The [PodSecurityPolicy][pod-security] API is deprecated and will be removed from Kubernetes in version 1.25. 
This API is replaced by a new built-in admission controller ([KEP-2579: Pod Security Admission Control][2579-psp-replacement]) which allows cluster admins to [enforce 
 Pod Security Standards Labels][enforce-standards-namespace-labels].

### What does that mean?

Namespace and Pod/Container can be defined with three different policies which are; **Privileged, Baseline and Restricted.** 
([More info](https://kubernetes.io/docs/concepts/security/pod-security-standards/)). Therefore, Pod(s)/Container(s) that 
are **not** configured according to the enforced security standards defined globally or
on the namespace level will **not** be admitted and it will **not** be possible to run them.

**As a best practice, you must ensure that workloads (Operators and Operands) are defined to run under 
restricted permissions unless they need further privileges. For the cases where Pod/Container(s) requires 
escalating permissions, the recommendation is to use the label as described below**

### How should I configure my Operators and Operands to comply with the criteria?

- **For common cases that do not require escalating privileges:** configure all containers to comply with the [restricted][restricted] policy as shown in the following the examples:

**IMPORTANT NOTE** The `seccompProfile` field to define that a container is [restricted][restricted] was introduced with K8s `1.19` and might **not** be supported on some vendors by default.
Please, do **not** use this field if you are looking to build Operators that work on K8s versions < `1.19` or on vendors that do **not** support this field. Having this field when it is not supported can result in your Pods/Containers **not** being allowed to run (i.e. On Openshift versions < `4.11` with its default configuration the deployments will fail with errors like `Forbidden: seccomp`.)

**In Kubernetes manifests:**

```yaml
    spec:
      securityContext:
        runAsNonRoot: true
        # Please ensure that you can use SeccompProfile and do not use
        # if your project must work on old Kubernetes
        # versions < 1.19 or on vendors versions which
        # do NOT support this field by default (i.e. Openshift < 4.11 )
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

**Note:** *if you are setting the `RunAsNonRoot` value to `true` in the `SecurityContext` you will also need to specify the `RunAsUser` value to set the user to a numeric user. In this implementation it sets the user uid to `1000`.*
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
           // Ensure restricted context for the Pod    
           SecurityContext: &corev1.PodSecurityContext{
              RunAsNonRoot: &[]bool{true}[0],
			  // Please ensure that you can use SeccompProfile and do NOT use
			  // this filed if your project must work on old Kubernetes
			  // versions < 1.19 or on vendors versions which 
			  // do NOT support this field by default (i.e. Openshift < 4.11)
              SeccompProfile: &corev1.SeccompProfile{
                 Type: corev1.SeccompProfileTypeRuntimeDefault,
              },
           },
           Containers: []corev1.Container{{
              Image:   "memcached:1.4.36-alpine",
              Name:    "memcached",
              // Ensure restricted context for the container  
              SecurityContext: &corev1.SecurityContext{
                 RunAsNonRoot:  &[]bool{true}[0],
                 AllowPrivilegeEscalation:  &[]bool{false}[0],
                 Capabilities: &corev1.Capabilities{
                    Drop: []corev1.Capability{
                       "ALL",
                    },
                 },
                 RunAsUser: &[]int64{1000}[0],
              },
           }},
        },
     },
  },
}
```

**For Ansible and Helm language based Operators:** Ansible playbooks or Helm charts MUST create manifests that comply 
with the requirements in the same way. You can find some examples by looking at the samples under the 
[testdata](https://github.com/operator-framework/operator-sdk/tree/master/testdata) directory.

- **For workloads that need elevated permissions:** it is recommended that you ensure the namespace containing your 
solution is labeled accordingly. You can either update your operator to manage the namespace labels or include 
the namespace labeling as part of the manual install instructions. Following the labels for this case scenario:

```yaml
  labels:
    ...
    pod-security.kubernetes.io/enforce: privileged
    pod-security.kubernetes.io/audit: privileged
    pod-security.kubernetes.io/warn: privileged
```

### How the Operator bundle (CSV) must be configured to apply the standards to the Pod/Containers which are installed by OLM (Operator itself)?

For Operators integrated with OLM, there is an Operator bundle with a CSV where the `spec.install.spec.deployments` has a Deployment 
which defines the Pod/Container(s) that will be installed by OLM to get your Operator running on the cluster. 
In order for the security standards to be followed you will need to ensure the configurations are set correctly.

**Note: Ensure the configuration is carried to the Pod/Containers on the bundle CSV after running make bundle**

Note that the Operator bundle generated with the target is built from
the manifests under the `config` directory. 

To check an example of a CSV which complies with the [restricted][restricted] policy, see the Golang sample
under the [testdata/go/v3/memcached-operator/bundle/manifests/memcached-operator.clusterserviceversion.yaml](https://github.com/operator-framework/operator-sdk/blob/master/testdata/go/v3/memcached-operator/bundle/manifests/memcached-operator.clusterserviceversion.yaml)

#### How can I verify my manifest? How do I check what policy is configured?

To verify the policy of your Pod/Container(s) you might to use a [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/)
cluster as described in the [K8s documentation](https://kubernetes.io/docs/tutorials/security/cluster-level-pss/).

**However, an easy way might be using the tool:** [psachecker](https://github.com/stlaz/psachecker). This tool is only 
able to be used to check locally the `Deployments/Pods` manifests and not the CSV. To check your CSV you can follow these steps:

- Create a new `test.yaml` file
- Add Deployment schema to the file:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-csv-deployment-strategy
spec:
...
```

- Now, add the CSV deployment to this test (`spec.install.spec.deployments`) 
- Then, you can run the tool and check if its result will be `restricted` as expected (i.e.):

```sh
$ ./kubectl-psachecker inspect-workloads -f test.yaml
system: restricted
```

[pod-security]: https://kubernetes.io/blog/2021/04/06/podsecuritypolicy-deprecation-past-present-and-future/#what-is-podsecuritypolicy
[2579-psp-replacement]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-auth/2579-psp-replacement
[enforce-standards-namespace-labels]: https://kubernetes.io/docs/tasks/configure-pod-container/enforce-standards-namespace-labels/
[restricted]: https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted
