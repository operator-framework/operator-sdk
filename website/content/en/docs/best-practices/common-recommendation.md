---
title: "Common recommendations and suggestions"
linkTitle: "Common suggestions"
weight: 2
description: Common recommendations and suggestions to built solutions with Operator SDK
---

## Overview

Any recommendations or best practices suggested by the Kubernetes community, such as how to [develop Operator pattern solutions][operator-pattern] or how to [use controller-runtime][controller-runtime] are good recommendations for those who are looking to build operator projects with operator-sdk. Also, see [Operator Best Practices][operator-best-practices]. However, here are some common recommendations.

## Common Recommendations

### Develop idempotent reconciliation solutions

When developing operators, it is essential for the controller’s reconciliation loop to be idempotent. By following the [Operator pattern][operator-pattern] you will create [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster. Breaking this recommendation goes against  the design principles of [controller-runtime][controller-runtime] and may lead to unforeseen consequences such as resources becoming stuck and requiring manual intervention.

### Understanding Kubernetes APIs

Building your own operator commonly involves extending the Kubernetes API itself. It is helpful to understand exactly how [Custom Resource Definitions interact with the Kubernetes API][k8s-crd-doc]. Also, the Kubebuilder documentation on [Groups and Versions and Kinds][kb-gkv] may be helpful to better understand these concepts as they relate to operators.

### Avoid a design solution where more than one Kind is reconciled by the same controller

Having many Kinds (such as CRDs) which are all managed by the same controller usually goes against the design proposed by [controller-runtime][controller-runtime]. Furthermore this might hurt concepts such as encapsulation, the Single Responsibility Principle, and Cohesion. Damaging these concepts may cause unexpected side effects, and increase the difficulty of extending, reusing, or maintaining the operator.

### Ideally Operators does not manage other Operators

From [best practices][best practices]: 

- _"Operators should own a CRD and only one Operator should control a CRD on a cluster.
Two Operators managing the same CRD is not a recommended best practice. In the case where an API exists but 
with multiple implementations, this is typically an example of a no-op Operator because it doesn't 
have any deployment or reconciliation loop to define the shared API and other 
Operators depend on this Operator to provide one implementation of the 
API, e.g. similar to PVCs or Ingress."_

- _"An Operator shouldn't deploy or manage other operators (such patterns are known as meta or super operators 
or include CRDs in its Operands). It's the Operator Lifecycle Manager's job to manage the deployment and 
lifecycle of operators. For further information check [Dependency Resolution][Dependency Resolution]."_

#### What does it mainly mean:

- If you want to define that your Operator depends on APIs which are owned by another Operator or on 
another whole Operator itself you should use Operator Lifecycle Manager's [Dependency Resolution][Dependency Resolution]
- If you want to reconcile core APIs (_defined by Kubernetes_) or External APIs (_defined from other operators_)
you should not re-define the API as owned by your project. Therefore, you can create the controller in this 
cases by using the flag `--resource=false`. (i.e. `$ operator-sdk create api --group ship --version v1beta1 --kind External --resource=false --controller=true`). 
**Attention:** If you are using Golang-based language Operator then, you will need to update the markers and imports 
manually until it become officially supported by the tool. For further information check the issue [#1999](https://github.com/kubernetes-sigs/kubebuilder/issues/1999).

**WARNING:** if you create CRD's via the reconciliations or via the Operands then, OLM cannot handle CRDs migration and update, validation.

**NOTE:** By not following this guidance you might probably to be hurting concepts like as single responsibility principle
and damaging these concepts could cause unexpected side effects, such as; difficulty extending, reuse, or maintenance, only to mention a few. 

### Other common suggestions

- Provide the images and tags used by the operator solution via environment variables in the `config/manager/manager.yaml`: 

```yaml
...
spec:
  ...
    spec:
      ...
      containers:
      - command:
        - /manager
        ...
        env:
        - name: MY_IMAGE
          value: "quay.io/example.com/image:0.0.1"
```

- Manage your solutions using [Status Conditionals][status-conditionals] 
- Use [finalizers][finalizers] when/if required 
- Cover the project with tests/CI to ensure its quality:
    - For any language-based operator, you can use [Scorecard][scorecard] to implement functional tests
    - For Go-based operators, you can also use [envtest][envtest] to cover the controllers. For further information see [Testing with EnvTest][testing-with-envtest]. Also, see the `test` directory for the Memcached sample under the [testdata/go/v3/memcached-operator][sample] to know how can you build e2e tests.
    - For Ansible-based operators, you can also use [Molecule][molecule], an Ansible testing framework. For further information see [Testing with Molecule][molecule-tests]
    - For Helm-based operators, you can also use [Chart tests][helm-chart-tests]
- Ensure that you checked the [Can I customize the projects initialized with operator-sdk?][faq] and understand the [Project Layout][project-layout] before starting to do your customizations as please you on top.
- Optimize manager resource values in `config/manager/manager.yaml` according to project requirements. It is recommended to define resources limits in order to follow good practices and for security reasons. More info: [Managing Resources for Containers][k8s-manage-resources] and [Docker Security Cheat Sheet][docker-cheats].
- Look for `TODO(user)` in the source code generated by the CLI to ensure that you follow all suggested customizations.
- If you wish to integrate your project with OLM, you can also check its [Best Practices][olm-best-practices] section.
 
[env-test]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest
[scorecard]: /docs/testing-operators/scorecard/
[testing-with-envtest]: /docs/building-operators/golang/testing
[olm-best-practices]: https://olm.operatorframework.io/docs/best-practices/
[finalizers]: /docs/building-operators/golang/advanced-topics/#handle-cleanup-on-deletion
[status-conditionals]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
[faq]: /docs/faqs/#can-i-customize-the-projects-initialized-with-operator-sdk
[project-layout]: /docs/overview/project-layout
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[k8s-crd-doc]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/
[operator-best-practices]: /docs/best-practices/best-practices
[kb-gkv]: https://book.kubebuilder.io/cronjob-tutorial/gvks.html
[operator-pattern]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
[molecule]: https://molecule.readthedocs.io/
[molecule-tests]: /docs/building-operators/ansible/testing-guide
[helm-chart-tests]: https://helm.sh/docs/topics/chart_tests/
[envtest]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest
[docker-cheats]: https://cheatsheetseries.owasp.org/cheatsheets/Docker_Security_Cheat_Sheet.html#rule-7-limit-resources-memory-cpu-file-descriptors-processes-restarts
[k8s-manage-resources]: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
[best practices]: https://olm.operatorframework.io/docs/concepts/olm-architecture/dependency-resolution/
[Dependency Resolution]:  /docs/best-practices/best-practices
[sample]: https://github.com/operator-framework/operator-sdk/tree/master/testdata/go/v3/memcached-operator
