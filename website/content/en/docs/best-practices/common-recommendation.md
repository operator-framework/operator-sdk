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
    - For Go-based operators, you can also use [envtest][envtest] to cover the controllers. For further information see [Testing with EnvTest][testing-with-envtest]
    - For Ansible-based operators, you can also use [Molecule][molecule], an Ansible testing framework. For further information see [Testing with Molecule][molecule-tests]
    - For Helm-based operators, you can also use [Chart tests][helm-chart-tests]
- Ensure that you checked the [Can I customize the projects initialized with operator-sdk?][faq] and understand the [Project Layout][project-layout] before starting to do your customizations as please you on top.
- If you will integrate your project with OLM then, also check its [Best Practices][olm-best-practices] section.
 
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
[molecule]: https://molecule.readthedocs.io/en/latest/
[molecule-tests]: /docs/building-operators/ansible/testing-guide
[helm-chart-tests]: https://helm.sh/docs/topics/chart_tests/
[envtest]: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest
