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
([More info][security-standards]). Therefore, Pod(s)/Container(s) that 
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
        # WARNING: Ensure that the image used defines an UserID in the Dockerfile
        # otherwise the Pod will not run and will fail with `container has runAsNonRoot and image has non-numeric user`.
        # If you want your workloads admitted in namespaces enforced with the restricted mode in OpenShift/OKD vendors
        # then, you MUST ensure that the Dockerfile defines a User ID OR you MUST leave the `RunAsNonRoot` and
        # RunAsUser fields empty.
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

**Note:** *if you are setting the `RunAsNonRoot` value to `true` in the `SecurityContext` you will need to verify that the Pod or Container(s) are running with a numeric user that is not 0 (root). If the Pod or Container(s) do not use a non-zero numeric user, you can use the `RunAsUser` value to set the user to a non-zero numeric user. In this example, the memcached container does not use a non-zero numeric user and therefore the `RunAsUser` value is set to use a uid of `1000`.*
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
				 // WARNING: Ensure that the image used defines an UserID in the Dockerfile
				 // otherwise the Pod will not run and will fail with `container has runAsNonRoot and image has non-numeric user`.
				 // If you want your workloads admitted in namespaces enforced with the restricted mode in OpenShift/OKD vendors 
				 // then, you MUST ensure that the Dockerfile defines a User ID OR you MUST leave the `RunAsNonRoot` and
				 // RunAsUser fields empty. 
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

**For Ansible and Helm language based Operators:** Ansible playbooks or Helm charts MUST create manifests that comply 
with the requirements in the same way. You can find some examples by looking at the samples under the 
[testdata](https://github.com/operator-framework/operator-sdk/tree/master/testdata) directory.

- **For workloads that need elevated permissions:** it is recommended that you ensure the namespace containing your 
solution is labeled accordingly. You can either update your operator to manage the namespace labels or include 
the namespace labeling as part of the manual install instructions. 

It is recommended that you provide a description to help cluster admins understand why elevated permissions are required.
You can add this information and the prerequisites to the description of your 
Operator Bundle (CSV).

Following you will find a detailed description of how to configure and test your solutions. 
The most straightforward way to ensure if your workloads will work in a restricted namespace is verifying if your solution can run in namespaces enforced as restricted. 

**NOTE**: It is recommended that you test the desired behavior as part of an e2e test suite. Examples of an e2e test for this will be shown in a later section.

### How the Operator bundle (CSV) must be configured to apply the standards to the Pod/Containers which are installed by OLM (Operator itself)?

For Operators integrated with OLM, there is an Operator bundle with a CSV where the `spec.install.spec.deployments` has a Deployment 
which defines the Pod/Container(s) that will be installed by OLM to get your Operator running on the cluster. 
In order for the security standards to be followed you will need to ensure the configurations are set correctly.

**Note: Ensure the configuration is carried to the Pod/Containers on the bundle CSV after running make bundle**. See that 
the Operator bundle generated with the target is built from the manifests under the `config` directory. To know more about
the layout of your operator built with Operator-SDK see [Project Layout][project-layout].

To check an example of a CSV which complies with the [restricted][restricted] policy, see the Golang sample
under the [testdata/go/v4/memcached-operator/bundle/manifests/memcached-operator.clusterserviceversion.yaml](https://github.com/operator-framework/operator-sdk/blob/master/testdata/go/v4/memcached-operator/bundle/manifests/memcached-operator.clusterserviceversion.yaml)

### How can I verify my manifest? 

#### Using Kind

To verify the policy of your Pod/Container(s) you might to use a [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/)
cluster as described in the [K8s documentation](https://kubernetes.io/docs/tutorials/security/cluster-level-pss/).

**Example**

1. First lets create a namespace where we will test the files:

```sh
kubectl create ns mytest
```

2. Now, let's label the namespace so that we can check our manifest

```sh
 kubectl label --overwrite ns mytest \
   pod-security.kubernetes.io/enforce=restricted \
   pod-security.kubernetes.io/enforce-version=v1.24 \
   pod-security.kubernetes.io/audit=restricted
```

3. Now, apply the following Pod which is **not** restricted on the namespace:

```yaml
apiVersion: v1
kind: Pod
metadata:
 name: example
 namespace: mytest
spec:
 containers:
  - name: test
    securityContext:
     # see that we are allowing privilege escalation  
     allowPrivilegeEscalation: true 
    image: 'busybox:1.28'
    ports:
     - containerPort: 8080

```

4. Then, when we try to apply the Pod manifest above we should see an error:

```sh
$ kubectl apply -f mypodtest.yaml 
Error from server (Forbidden): error when creating "mypodtest.yaml": pods "example" is forbidden: violates PodSecurity "restricted:v1.24": allowPrivilegeEscalation != false (container "test" must set securityContext.allowPrivilegeEscalation=false), unrestricted capabilities (container "test" must set securityContext.capabilities.drop=["ALL"]), runAsNonRoot != true (pod or container "test" must set securityContext.runAsNonRoot=true), seccompProfile (pod or container "test" must set securityContext.seccompProfile.type to "RuntimeDefault" or "Localhost")
```

#### How do I check what policy (Privileged, Baseline and Restricted) is configured for my Operator and Operand(s)?

An easy way might be using the tool: [psachecker][psachecker]. This tool is only
able to be used to check locally the `Deployments/Pods` manifests and not the CSV.

Alternatively, you could test the policy enforcement by labeling the namespaces and running your operator or applying the manifests. [More info][enforce-standards-namespace-labels].

##### How do I install psachecker?

The following steps will install [psachecker][psachecker] in a Golang environment.

```sh
git clone git@github.com:stlaz/psachecker.git $GOPATH/src/github.com/stlaz/psachecker
cd $GOPATH/src/github.com/stlaz/psachecker
make build
cp kubectl-psachecker $GOPATH/bin/
```

#### How do I use psachecker to verify the policy of a manifests?

The following steps will demonstrate how we can use [psachecker][psachecker] to verify that our Operator and deployments are properly configured in the CSV.

- Create a new `test.yaml` file
- Add Deployment schema to the file:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-manifest
  namespace: test
spec:
...
```

- Now, add the CSV deployment to this test (`spec.install.spec.deployments`) 
- Then, you can run the tool and check if its result will be `restricted` as expected (i.e.):

```sh
$ kubectl-psachecker inspect-workloads -f test.yaml
system: restricted
```

### Can I use the metrics to check if my Pod/Containers are violating the PodSecurity policies?

Yes, you can. You need to label the namespaces with `pod-security.kubernetes.io/audit: restricted` 
(i.e. `kubectl label --overwrite ns --all pod-security.kubernetes.io/enforce-version=v1.24 pod-security.kubernetes.io/audit=restricted`).
 It is important to note that the results may include metrics that do not come from your Operator and Operand(s). 
If you are looking to use the metrics to do the checks, ensure that you check the
results before and after performing the tests, for example: 

```sh
kubectl get --raw /metrics | prom2json | jq '[.[] | select(.name=="pod_security_evaluations_total") ]'
```
```json
[
  {
    "name": "pod_security_evaluations_total",
    "help": "[ALPHA] Number of policy evaluations that occurred, not counting ignored or exempt requests.",
    "type": "COUNTER",
    "metrics": [
      {
        "labels": {
          "decision": "allow",
          "mode": "enforce",
          "policy_level": "privileged",
          "policy_version": "latest",
          "request_operation": "create",
          "resource": "pod",
          "subresource": ""
        },
        "value": "29"
      },
      {
        "labels": {
          "decision": "allow",
          "mode": "enforce",
          "policy_level": "privileged",
          "policy_version": "latest",
          "request_operation": "update",
          "resource": "pod",
          "subresource": ""
        },
        "value": "0"
      },
      {
        "labels": {
          "decision": "deny",
          "mode": "audit",
          "policy_level": "restricted",
          "policy_version": "latest",
          "request_operation": "create",
          "resource": "controller",
          "subresource": ""
        },
        "value": "6"
      },
      {
        "labels": {
          "decision": "deny",
          "mode": "audit",
          "policy_level": "restricted",
          "policy_version": "latest",
          "request_operation": "create",
          "resource": "pod",
          "subresource": ""
        },
        "value": "7"
      },
      {
        "labels": {
          "decision": "deny",
          "mode": "warn",
          "policy_level": "restricted",
          "policy_version": "latest",
          "request_operation": "create",
          "resource": "controller",
          "subresource": ""
        },
        "value": "6"
      },
      {
        "labels": {
          "decision": "deny",
          "mode": "warn",
          "policy_level": "restricted",
          "policy_version": "latest",
          "request_operation": "create",
          "resource": "pod",
          "subresource": ""
        },
        "value": "7"
      }
    ]
  }
]
```
Also, you might be able to create Prometheus alerts such as:

```sh
sum (increase(pod_security_evaluations_total{decision="deny",mode="audit"}[1h])) by (policy_level)
```

### (Recommended) How can I automate this check using e2e testing to ensure that my solutions can run under the policies?

In the tests, you can create and label a namespace with the desired policy (i.e. restricted). This would allow you to verify
if any warnings/errors occur and if the Pod(s)/Container(s) are in the `Running` state. It can be something like:

**NOTE** See the `test` directory for the Memcached sample under the [testdata/go/v3/memcached-operator][sample] to see a full example.

```go
// namespace store the ns where the Operator and Operand(s) will be executed
const namespace = "my-operator-system"

var _ = Describe("my operator test", func() {
    BeforeEach(func() {
        ...
        
        // Where we will create the namespace for to install the Operator and Operands
        By("creating namespace")
        cmd := exec.Command("kubectl", "create", "ns", namespace)
        _, _ = utils.Run(cmd)
        
        // We will label as follows all namespaces so that we can check if warnings will be raised when 
		// a manifest be applied. (Not that some namespaces might not acced the label, i.e. if it
		// has a container running with a less restrictive policy )
        By("labeling all namespaces to warn against what can violate the restricted policy")
        cmd = exec.Command("kubectl", "label", "--overwrite", "ns", "--all",
            "pod-security.kubernetes.io/audit=restricted",
            "pod-security.kubernetes.io/enforce-version=v1.24",
            "pod-security.kubernetes.io/warn=restricted")
        _, err := utils.Run(cmd)
        ExpectWithOffset(1, err).NotTo(HaveOccurred())
        
		// We will enforce the restricted policy so that if our Operator
		// or Operand be unable to run as restricted we will be able to check
		// it by validating their status
        By("enforcing restricted to the ns where the Operator/Operand(s) will be checked")
        cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
            "pod-security.kubernetes.io/audit=restricted",
            "pod-security.kubernetes.io/enforce-version=v1.24",
            "pod-security.kubernetes.io/enforce=restricted")
        _, err = utils.Run(cmd)
        Expect(err).To(Not(HaveOccurred()))
        })
    })
    
    AfterEach(func() {
        ...
    })

    It("should successfully run the Operator and Operand(s)", func() {
        // Then, here we build the operator and deploy the
        // manager as the operand in the namespaces were 
        // the policy restricted was enforced.
		
        // Therefore, we can check the Operator and Operand
        // status to ensure that all is Running.
		
        // Note that we can also verify if warns like with
        // the message Warning: would violate PodSecurity were 
        // returned when the manifest were applied on the cluster
    })
})
```

### After following the recommendations to be restricted my workload is not running (CreateContainerConfigError). What should I do?

If you are encountering errors similar to `Error: container has runAsNonRoot and image has non-numeric user` 
or `container has runAsNonRoot and image will run as root` that means that the image used does not have a non-zero numeric user defined, i.e.:

```shell
USER 65532:65532 
OR
USER 1001
```

Due to the `RunAsNonRoot` field being set to `true`, we need to force the user in the
container to a non-zero numeric user.
It is recommended that the images used by your operator have a non-zero numeric user set in the image itself (similar to the example above). For further information check the note [Consider an explicit UID/GID][docker-good-practices-doc] in the Dockerfile best practices guide.
If your Operator will be distributed and used in vanilla Kubernetes clusters you can also fix the issue by defining the user via the security context configuration). (i.e. `RunAsUser: &[]int64{1000}[0],`). 

**NOTE** If your Operator should work with specific vendors please ensure that you check if they have specific rules for Pod Security Admission.  For example, we know that if you use `RunAsUser` on OpenShift it will disqualify the Pod from their restricted-v2 SCC. 
Therefore, if you want your workloads running in namespaces labeled to enforce restricted you must leave `RunAsUser` and `RunAsNonRoot` fields empty or if you want set `RunAsNonRoot` then, you MUST ensure that the image itself properly defines the UserID.

[project-layout]: /docs/overview/project-layout
[pod-security]: https://kubernetes.io/blog/2021/04/06/podsecuritypolicy-deprecation-past-present-and-future/#what-is-podsecuritypolicy
[2579-psp-replacement]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-auth/2579-psp-replacement
[enforce-standards-namespace-labels]: https://kubernetes.io/docs/tasks/configure-pod-container/enforce-standards-namespace-labels/
[restricted]: https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted
[security-standards]: https://kubernetes.io/docs/concepts/security/pod-security-standards/
[psachecker]: https://github.com/stlaz/psachecker
[sample]: https://github.com/operator-framework/operator-sdk/tree/master/testdata/go/v4/memcached-operator
[docker-good-practices-doc]: https://docs.docker.com/develop/develop-images/dockerfile_best-practices/#user
