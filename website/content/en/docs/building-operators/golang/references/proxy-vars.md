---
title: Proxy Friendly Operators
linkTitle: Proxy Vars
weight: 20
---

Proxy-friendly Operators should inspect their environment for the
standard proxy variables (`HTTPS_PROXY`, `HTTP_PROXY`, and `NO_PROXY`)
and pass the values to Operands.

Operator-lib provides a helper function `proxy.ReadProxyVarsFromEnv`
that does this inspection, all you need to do is append the
results to the Operand environments.

Using the memcached tutorial as an example, add the following to the
Reconcile loop in `controllers/memcached_controller.go`:


```go
import (
  ...
   "github.com/operator-framework/operator-lib/proxy"
)


for i, container := range dep.Spec.Template.Spec.Containers {
		dep.Spec.Template.Spec.Containers[i].Env = append(container.Env, proxy.ReadProxyVarsFromEnv()...)
}
```

You can set the environment variable on the Operator deployment. Using the memcached tutorial, edit config/manager/manager.yaml:

```yaml
containers:
 - args:
   - --leader-elect
   - --leader-election-id=go-proxy-demo
   image: controller:latest
   name: manager
   env:
     - name: "HTTP_PROXY"
       value: "http_proxy_test"
```
