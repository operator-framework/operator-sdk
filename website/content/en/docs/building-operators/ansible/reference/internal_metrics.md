---
title: Internal Operator Metrics
linkTitle: Metrics
weight: 20
---

The Ansible Operator comes with three internal metrics that provide an insight to the frequency and time of operator reconciliations. These metrics can be
scraped by a Prometheus instance or any other openmetrics system. To publish operator metrics and scrape them with an openmetrics system such as Prometheus, view 
[Kubebuilder documentation](https://book.kubebuilder.io/reference/metrics.html) on publishing metrics.

The default metrics recorded in Operator SDK are collected in a [histogram](https://prometheus.io/docs/practices/histograms/).

The following three metrics are derived from the histogram:
1. `ansible_operator_reconciles_bucket` - Each bucket in the histogram counts the number of reconciliations that have a period (in seconds) less than or equal
to the upper limit of the bucket.
3. `ansible_operator_reconciles_count` - The total number of reconciliations that have occurred up to that instance of time while running an Ansible operator.
4. `ansible_operator_reconciles_sum` - The cumulative amount of time (in seconds) of all reconciliations that have occurred up to that instance of time while 
running an Ansible operator.

These metrics can be queried in the Prometheus UI.

![Screen Shot 2021-06-24 at 2 10 28 PM](https://user-images.githubusercontent.com/37827279/123332879-f0fb2900-d4f5-11eb-87ea-7afd04f35b1c.png)
