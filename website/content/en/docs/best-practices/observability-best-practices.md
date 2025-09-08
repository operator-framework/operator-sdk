---
title: "Operator Observability Best Practices"
linkTitle: "Observability Best Practices"
weight: 6
description: This guide describes the best practices concepts for adding Observability to operators.
---

## Operator Observability Best Practices

In this document, we provide best practices and examples for creating metrics, [recording rules](https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/#recording-rules) and alerts. It is based on the general guidelines in [Operator Capability Levels](https://sdk.operatorframework.io/docs/overview/operator-capabilities/).

**Note:** For technical documentation of how to add metrics to your operator, please read the [Metrics](https://book.kubebuilder.io/reference/metrics.html) section of the Kubebuilder documentation.

### Operator Observability Recommended Components

1. **Health and Performance metrics** - for all of the operator components  
    1.1. Metrics should be implemented based on the guidelines below.  
    1.2. **Metrics Documentation** - All metrics should have documentation.  
    1.3. **Metrics Tests** - Metrics should include tests that verify that they exist and that their value is correct.  
2. **Alerts** for when things are not working as expected for each of the operator's components  
    2.1  Alerts should be implemented based on the guidelines below.  
    2.2. **Alerts Runbooks** - Each alert can include a `runbook_url` annotation and an alert runbook that describes it. See additional details below.  
    2.3. **Alerts Tests** - Alerts should include E2E Testing and unit tests.  
3. **Events** - Custom Resources should emit custom events for the operations taking place.

Additional components would be `Dashboards`, `Logs` and `Traces`, which are not addressed in this document at this point.  

### Operators Observability General Guidelines
**Important:** It is highly recommended to separate your monitoring code from your core operator code.  
We recommend to create a dedicated `/monitoring` subfolder that will include all the code of the [Operator Observability Recommended Components](#operator-observability-recommended-components), that are outlined above. For example, in the [memcached-operator](https://github.com/operator-framework/operator-sdk/tree/master/testdata/go/v4/monitoring/memcached-operator/monitoring).

In your core operator code only call the functions that will update the metrics value from your desired location. For example, in the [memcached-operator](https://github.com/operator-framework/operator-sdk/blob/367bd3597c30607099aa73637f5286f7120b847a/testdata/go/v3/monitoring/memcached-operator/controllers/memcached_controller.go#L242).

All operators start small. This separation will help you, as a developer, with easier maintenance of both your operator core code and the monitoring code and for other stakeholders to understand your monitoring code better.

### Metrics Guidelines

#### Metrics Naming
Kubernetes components emit metrics in [Prometheus format](https://prometheus.io/docs/instrumenting/exposition_formats/). This format is structured plain text, designed so that people and machines can both read it.

Your operator users should get the same experience when searching for a metric across Kubernetes operators, resources and custom resources.
1. Check if a similar Kubernetes metric, for node, container or pod, exists and try to align to it.
2. The metrics search list, in the Prometheus, Grafana UI and even in the /metrics end point, is sorted in alphabetical order.
When searching for a metric, it should be easy to identify metrics that are related to a specific operator.
That is why we recommend that your operator metrics name will follow this format:
`operator name` prefix + the `sub-operator name` or `entity` + `metric name` based on the [Prometheus naming conventions](https://prometheus.io/docs/practices/naming/). For example, in the [memcached-operator](https://github.com/operator-framework/operator-sdk/blob/0d2fa86f0d3cc92c4672cb9e1d246efaefcf7ced/testdata/go/v4-alpha/monitoring/memcached-operator/monitoring/metrics.go#L14).

**Note:** In [Prometheus Node Exporter](https://github.com/prometheus/node_exporter) metrics are separated like this:
- node_network_**receive**_packets_total
- node_network_**transmit**_packets_total  
 
In this example, based on `receive` and `transmit`.

Please follow the same principle and don't put similar metrics details as labels, so the user experience would be fluent.  
Example for this in an operator:
- kubevirt_vmi_network_**receive**_errors_total
- kubevirt_vmi_network_**transmit**_bytes_total
- kubevirt_migrate_vmi_**data_processed**_bytes
- kubevirt_migrate_vmi_**data_remaining**_bytes

3. Your metric suffix should indicate the metric unit. For better compatibility, [Prometheus base units](https://prometheus.io/docs/practices/naming/#base-units) should be used.
4. Prometheus supports four [metric types](https://prometheus.io/docs/concepts/metric_types/#metric-types). `Gauge`,`Counter`,`Histogram` and `Summary`. You can read more about the different types here, [Understanding metrics types](https://prometheus.io/docs/tutorials/understanding_metric_types/#types-of-metrics).  
The most common types are:
 - `Counter` - Value can only increase or reset.
 - `Gauge` Value can be increased and decreased as needed.
5. `_total` suffix should be used for accumulating count. If your metrics has labels with high cardinality, like `pod`/`container` it usually means that you can aggregate it more, thus it will not require `_total` suffix.

### Prometheus Labels
[Prometheus labels](https://prometheus.io/docs/practices/naming/#labels) are used to differentiate the characteristics of the thing that is being measured.

1. **Important** - Be cautious when adding labels to metrics. Labels can dramatically increase the amount of data stored. Do not use labels to store dimensions with high cardinality (many different label values), such as user IDs, email addresses, or other unbounded sets of values. Note: There are still cases when we will still need to have a high cardinality label like the `pod name`, but try to keep this to the minimum.
2. When creating a new metric, recording rule or alert, that reports a resource like a `pod` or a `container`, make sure that the `namespace` is included, in order to be able to uniquely identify it.

#### Metrics `Help` message

Your operator metrics `help` message should include the following details:
- What does this metric measure?
- What does the output mean?
- What important labels does the metric use? (Optional. If applicable).

The `Help` message can be used to create auto-generated documentation, like it's done in [KubeVirt](https://github.com/kubevirt/kubevirt/blob/main/docs/observability/metrics.md) and generated by the [KubeVirt metrics doc generator](https://github.com/kubevirt/kubevirt/blob/main/tools/doc-generator/doc-generator.go).

We recommend to auto-generated metrics documentation and save it in your operator repository, to a location like `/docs/monitoring/`, so that the users can find the information about your operator metrics easily. 

See [Alerts, Metrics and Recording Rules Tests](#alerts-metrics-and-recording-rules-tests) section for metrics testing recommendations. 

#### Prometheus Recording Rules Naming
As per [Prometheus](https://prometheus.io/docs/prometheus) documentation, [Recording rules](https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/#recording-rules) allow you to pre-compute frequently needed or computationally expensive expressions and save their result as a new set of time series.

**Note:** The Prometheus recording rules appear in Prometheus UI as metrics.
Recording rule names should follow the `level:metric:operations` format as specified in the [Prometheus recording rules best practices](https://prometheus.io/docs/practices/rules/). This naming convention makes it clear that the metric is a recording rule and helps consumers understand they need to examine the underlying query to fully understand what the metric provides.

- **level:** represents the aggregation level and labels of the rule output
- **metric:** is the metric name  
- **operations:** is a list of operations that were applied to the metric, newest operation first

For example: `job:up:avg_over_time` or `instance:node_cpu_utilisation:rate5m`

In addition to this format, your operator recording rules should also follow the same naming guidelines as metrics for consistency within your operator's observability stack.

See [Alerts, Metrics and Recording Rules Tests](#alerts-metrics-and-recording-rules-tests) section for recording rules testing recommendations. 

### Prometheus Alerts Guidelines
Clear and actionable alerts are a key component of a smooth operational experience and will result in a better experience for the end users.

The following guidances aim to align alert naming, severities, labels, etc., in order to avoid alerts fatigue for administrators.

#### Recommended Reading

A list of references on good alerting practices:

* [Google SRE Book - Monitoring Distributed Systems](https://sre.google/sre-book/monitoring-distributed-systems/)
* [Prometheus Alerting Documentation](https://prometheus.io/docs/practices/alerting/)
* [Alerting for Distributed Systems](https://www.usenix.org/sites/default/files/conference/protected-files/srecon16europe_slides_rabenstein.pdf)

#### Alert Ownership

Individual operator authors are responsible for writing and maintaining alerting rules for
their components, i.e. their operators and operands.

Operator authors should also take into consideration how their components interact with
existing monitoring and alerting.

As an example, if your operator deploys a service which creates one or more `PersistentVolume` resources,
and these volumes are expected to be mostly full as part of normal operation, it's likely
that this will cause unnecessary `KubePersistentVolumeFillingUp` alerts to fire.

You should work to find a solution to avoid triggering these alerts if they are not actionable.

#### Alerts Style Guide

* Alert names MUST be CamelCase, e.g.: `PrometheusRuleFailures`
* Alert names SHOULD be prefixed with a component, e.g.: `AlertmanagerFailedReload`
  * There may be exceptions for some broadly scoped alerts, e.g.: `TargetDown`
* Alerts MUST include a `severity` label indicating the alert's urgency.
  * Valid severities are: `critical`, `warning`, or `info` — see below for
    guidelines on writing alerts of each severity.
* Alerts MUST include `summary` and `description` annotations.
  * Think of `summary` as the first line of a commit message, or an email
    subject line.  It should be brief but informative.  The `description` is the
    longer, more detailed explanation of the alert.
* Alerts SHOULD include a `namespace` label indicating the source of the alert.
  * Many alerts will include this by virtue of the fact that their PromQL
    expressions result in a namespace label.  Others may require a static
    namespace label — see for example, the [KubeCPUOvercommit](https://github.com/openshift/cluster-monitoring-operator/blob/79cdf68/assets/control-plane/prometheus-rule.yaml#L235-L247) alert.


**Optional Alerts Labels and Annotations**
* `priority` label indicating the alert's level of importance and the order in which it should be fixed.
  * Valid priorities are: `high`, `medium`, or `low`.
    Higher the priority the sooner the alert should be resolved.
  * If the alert doesn't include a `priority` label, we can assume it is a `medium` priority alert.
This label will usually be used for alerts with `warning` severity, to indicate the order in which the alert should be addressed by, even though it doesn't require immediate action.
* `runbook_url` annotation is a link to an alert runbook which is intended to guide a cluster owner and/or operator through the steps of fixing problems on clusters, which are surfaced by alerts.
  * An example for [Runbook style documentation](https://github.com/openshift/runbooks/blob/master/example.md).
  * Your operator alert runbooks can be saved at your operator repository, to `/docs/monitoring/runbooks/` for example,
    at [OpenShift Runbooks](https://github.com/openshift/runbooks) if your operator is shipped with OpenShift
    or another location that fits your operator.
  * If you are using Github, you can use [Github Pages](https://pages.github.com/) for a better view of the runbooks.
* `kubernetes_operator_part_of` label indicating the operator name. Label name is based on the  [Kubernetes Recommended Labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/#labels).

#### Alerts Severity
##### Critical Alerts

For alerting current and impending disaster situations. These alerts
page an SRE. The situation should warrant waking someone in the middle of the
night.

Timeline:  ~5 minutes.

Reserve critical level alerts only for reporting conditions that may lead to loss of data or inability to deliver service for the cluster as a whole.  
Failures of most individual components should not trigger critical level alerts, unless they would result in either of those conditions.  
Configure critical level alerts so they fire before the situation becomes irrecoverable.  
Expect users to be notified of a critical alert within a short period of time after it fires so
they can respond with corrective action quickly.

Example critical alert: [KubeAPIDown](https://github.com/openshift/cluster-monitoring-operator/blob/79cdf68/assets/control-plane/prometheus-rule.yaml#L412-L421)

```yaml
- alert: KubeAPIDown
  annotations:
    summary: Target disappeared from Prometheus target discovery.
    description: KubeAPI has disappeared from Prometheus target discovery.
    runbook_url: https://github.com/openshift/runbooks/blob/master/alerts/cluster-monitoring-operator/KubeAPIDown.md
  expr: |
    absent(up{job="apiserver"} == 1)
  for: 15m
  labels:
    severity: critical
```

This alert fires if no Kubernetes API server instance has reported metrics successfully in the last 15 minutes.  
This is a clear example of a critical control-plane issue that represents a threat to the operability of the cluster as a whole, and likely warrants paging someone.  
The alert has clear summary and description annotations, and it links to a runbook with information on investigating and resolving the issue.

The group of critical alerts should be small, very well defined, highly documented, polished and with a high bar set for entry.

##### Warning Alerts

The vast majority of alerts should use this severity.  
Issues at the warning level should be addressed in a timely manner, but don't pose an immediate threat to the operation of the cluster as a whole.

Timeline:  ~60 minutes

If your alert does not meet the criteria in "Critical Alerts" above, it belongs to the warning level or lower.

Use warning level alerts for reporting conditions that may lead to inability to deliver individual features of the cluster, but not service for the cluster as a
whole. Most alerts are likely to be warnings.  
Configure warning level alerts so that they do not fire until components have sufficient time to try to recover from the interruption automatically.  
Expect users to be notified of a warning, but for them not to respond with corrective action immediately.

Example warning alert: [ClusterNotUpgradeable](https://github.com/openshift/cluster-version-operator/blob/513a2fc/install/0000_90_cluster-version-operator_02_servicemonitor.yaml#L68-L76)

```yaml
- alert: ClusterNotUpgradeable
  annotations:
    summary: One or more cluster operators have been blocking minor version cluster upgrades for at least an hour.
    description: In most cases, you will still be able to apply patch releases.
      Reason {{ "{{ with $cluster_operator_conditions := \"cluster_operator_conditions\" | query}}{{range $value := .}}{{if and (eq (label \"name\" $value) \"version\") (eq (label \"condition\" $value) \"Upgradeable\") (eq (label \"endpoint\" $value) \"metrics\") (eq (value $value) 0.0) (ne (len (label \"reason\" $value)) 0) }}{{label \"reason\" $value}}.{{end}}{{end}}{{end}}"}}
      For more information refer to 'oc adm upgrade'{{ "{{ with $console_url := \"console_url\" | query }}{{ if ne (len (label \"url\" (first $console_url ) ) ) 0}} or {{ label \"url\" (first $console_url ) }}/settings/cluster/{{ end }}{{ end }}" }}.
    expr: |
      max by (name, condition, endpoint) (cluster_operator_conditions{name="version", condition="Upgradeable", endpoint="metrics"} == 0)
    for: 60m
    labels:
      severity: warning
```

This alert fires if one or more operators have not reported their `Upgradeable` condition as true in more than an hour.  
The alert has a clear name and informative summary and description annotations.  
The timeline is appropriate for allowing the operator a chance to resolve the issue automatically, avoiding the need to alert an administrator.

##### Info Alerts

Info level alerts represent situations an administrator should be aware of, but they don't necessarily require any action.  
Use these sparingly, and consider instead reporting this information via Kubernetes events.

Example info alert: [MultipleContainersOOMKilled](https://github.com/openshift/cluster-monitoring-operator/blob/79cdf68/assets/cluster-monitoring-operator/prometheus-rule.yaml#L326-L338)

```yaml
- alert: MultipleContainersOOMKilled
  annotations:
    description: Multiple containers were out of memory killed within the past
      15 minutes. There are many potential causes of OOM errors, however issues
      on a specific node or containers breaching their limits is common.
      summary: Containers are being killed due to OOM
  expr: sum(max by(namespace, container, pod) (increase(kube_pod_container_status_restarts_total[12m]))
    and max by(namespace, container, pod) (kube_pod_container_status_last_terminated_reason{reason="OOMKilled"}) == 1) > 5
  for: 15m
  labels:
    namespace: kube-system
    severity: info
```

This alert fires if multiple containers have been terminated due to out of memory conditions in the last 15 minutes.  
This is something the administrator should be aware of, but may not require immediate action.

### Alerts, Metrics and Recording Rules Tests

1. Add tests for alerts that validate that:
   - Each alert includes all mandatory fields.
   - Each `runbook_url` link is valid.
   - Each alert that includes a `pod` or a `container` also includes the `namespace`.
2. Add e2e tests that inspect the alerts during upgrade and make sure that the alerts don’t fire when they shouldn’t (Zero noise).
3. Add tests for metrics and recording rules that validate that:
   - Metric / Recording rule exists.
   - Metric / Recording rule value is as expected.
   - Metric / Recording rule name follows the best practices guideline.