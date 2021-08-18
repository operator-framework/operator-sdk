---
title: Proxy Friendly Operators
linkTitle: Proxy Vars
weight: 20
---

Proxy-friendly Operators should inspect their environment for the
standard proxy variables (`HTTPS_PROXY`, `HTTP_PROXY`, and `NO_PROXY`)
and pass the values to Operands.

This can be accomplished by modifying the `watches.yaml` to include the
overrides based on an environment variable:

```yaml
- group: demo.example.com
  version: v1alpha1
  kind: Nginx
  chart: helm-charts/nginx
  overrideValues:
    proxy.http: $HTTP_PROXY
#+kubebuilder:scaffold:watch
```

Note: This example assumes that `proxy.http` is included in your chart's
`Values.yaml`. The nginx tutorial does not have this value, but you can
add to the `helmcharts/nginx/Values.yaml`:

```yaml
proxy:
  http: ""
  https: ""
  no_proxy: ""
```

You will also need to make sure the chart template supports the usage of
these values. Using the nginx tutorial, edit
`helm-charts/nginx/templates/deployment.yaml`

```yaml
containers:                                                                                                                                                                                                                             
  - name: {{ .Chart.Name }}                                                                                                                                                                                                             
    securityContext:                                                                                                                                                                                                                    
      {{- toYaml .Values.securityContext | nindent 12 }}                                                                                                                                                                                
    image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"                                                                                                                                         
    imagePullPolicy: {{ .Values.image.pullPolicy }}                                                                                                                                                                                     
    env:                                                                                                                                                                                                                                
      - name: http_proxy                                                                                                                                                                                                                
        value: "{{ .Values.proxy.http }}"  
```



You can set the environment variable on the Operator deployment. Using
the nginx tutorial, edit `config/manager/manager.yaml`:

```yaml
containers:
 - args:
   - --leader-elect
   - --leader-election-id=helm-proxy-demo
   image: controller:latest
   name: manager
   env:
     - name: "HTTP_PROXY"
       value: "http_proxy_test"
```
