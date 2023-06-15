---
title: Proxy Friendly Operators
linkTitle: Proxy Vars
weight: 20
---

Proxy-friendly Operators should inspect their environment for the
standard proxy variables (`HTTPS_PROXY`, `HTTP_PROXY`, and `NO_PROXY`)
and pass the values to Operands.

Example:

```yaml
- name: start proxy-test job
  kubernetes.core.k8s:
    definition:
      apiVersion: batch/v1
      kind: Job
      metadata:
        name: env-job
        namespace: "{{ ansible_operator_meta.namespace }}"
      spec:
        template:
          spec:
            containers:
              - name: curl-example
                image: registry.access.redhat.com/ubi8/ubi:8.8
                command: ["curl"]
                args: ["http://example.com/job-request"]
                env:
                  - name: HTTP_PROXY
                    value: '{{ lookup("env", "HTTP_PROXY") | default("", True) }}'
                  - name: http_proxy
                    value: '{{ lookup("env", "HTTP_PROXY") | default("", True) }}'
            restartPolicy: Never
        backoffLimit: 4
```

You can set the environment variable on the Operator deployment. Using the memcached tutorial, edit config/manager/manager.yaml:

```yaml
containers:
 - args:
   - --leader-elect
   - --leader-election-id=ansible-proxy-demo
   image: controller:latest
   name: manager
   env:
     - name: "HTTP_PROXY"
       value: "http_proxy_test"
```
