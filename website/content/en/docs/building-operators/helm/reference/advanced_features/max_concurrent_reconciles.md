---
title: Maximum Concurrent Reconciles in Helm-based Operators
linkTitle: Maximum Concurrent Reconciles
weight: 200
description: Increase concurrency of custom reconciliation to scale your operator to large clusters.
---

Depending on the number of CRs your operator is managing, it might be necessary to tune the maximum number of concurrent reconciles to ensure timely reconciliations. The `--max-concurrent-reconciles` flag can be used to override the default max concurrent reconciles, which by default is the number of CPUs on the node on which the operator is running. For example:

```sh
$ cat config/manager/manager.yaml
...
    spec:
      containers:
      - args:
        - manager
        - --max-concurrent-reconciles=10
...
```

While running locally, this flag can also be added to the helm binary. For example, running `helm-operator` binary with the above mentioned flag would give us a similar result:
```
helm-operator --max-concurrent-reconciles=10
```

**NOTE**: If you're using the default scaffolding, it is necessary to also apply this change to the `config/default/manager_auth_proxy_patch.yaml` file. This file is a `kustomize` patch to the operator deployment that configures [kube-rbac-proxy][kube-rbac-proxy] to require authorization for accessing your operator metrics. When `kustomize` applies this patch, it overrides the args defined in `config/manager/manager.yaml`

[kube-rbac-proxy]: https://github.com/brancz/kube-rbac-proxy
