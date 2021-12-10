---
title: "Multi-Tenancy"
linkTitle: "Multi-Tenancy"
weight: 4
description: This guide describes the best practices concepts to write Operators for Multi-Tenancy solutions.
---

## NetworkPolicy 

If your Operator creates or manages [NetworkPolicy][k8s-network-policy] configurations ensure that your solution:

* applies fine-grained network policies to the extent that is required for your managed application to function properly
* applies fine-grained network policies to enable your managed application internal components to communicate among each other
* allows users to configure your operator so it does not create or manage [NetworkPolicy][k8s-network-policy] instances
* does not create `allow traffic from everywhere in the cluster` type policies

[NetworkPolicies][k8s-network-policy] are popular in multi-tenant cluster to provide an extra layer
of segregation among tenants within SDN solutions. Users typically customize this extensively with goal of disallowing network traffic among unrelated
tenants. Operators that deploy an `accept all traffic from anywhere in the cluster` style policy, are creating an obstacle in
pursuing this goal, especially in instances where these policies cannot be disabled. In security-conscious environments policies like these are not allowed
in production. In such cases your operator should minimally **have the option to prevent [NetworkPolicy][k8s-network-policy] objects from being
created** and leave this responsibility to the user. In more advanced cases, your operator should create [NetworkPolicy][k8s-network-policy] 
configurations that follow the least-privilege principle, i.e. denying access to everything and from everywhere by default and only allowing access to specific authorized resources from specific authorized components.

## Traffic sharding

The goal is to split or to isolated ingress traffic from certain environments, e.g. production and development environments, ending up on
different routers and in this way, being managed by a different Ingress controller. This is a popular configuration 
option for heavily populated multi-tenant clusters, with several [IngressController][k8s-ingress-controllers] deployed. 

If your Operator creates ingress resources the recommendation is to allow the users to customize them,
through the use of a CRD. The required 
[IngressClass][ingress-class] needs then to be 
propagated to the ingress resources created so that they get picked up by the desired [IngressController][k8s-ingress-controllers].
Annotations are deprecated in favour of [IngressClass][ingress-class]. 

### Route resources

To run on the OpenShift distribution of Kubernetes you probably will use 
the [Route][ocp-route] API. When sharding these routes may be configured with a label selector. 
Based on this label selector they will amend their configuration when a route (having the label) 
is created or not (if the route does not have the label). The label is applied at the `Route` level
and there is no pre-defined convention here, so users set these custom labels in
accordance to how they configured their [IngressController from operator.openshift.io/v1][k8s-ingress-controllers] instances. 

In this way, **your operator should allow the user to specify custom labels for any
Route that it manages**. Check the [doc][ocp-ingress-doc] and this [blog][ocp-blog] for an end-to-end examples 
and further information.

[olm-docs]: /docs/olm-integration/
[ocp-route]: https://docs.openshift.com/container-platform/4.9/rest_api/network_apis/route-route-openshift-io-v1.html#route-route-openshift-io-v1
[ocp-ingress-doc]: https://docs.openshift.com/container-platform/4.9/networking/ingress-operator.html#nw-ingress-sharding-route-labels_configuring-ingress
[k8s-ingress-controllers]: https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/
[ocp-blog]: https://rcarrata.com/openshift/ocp4_route_sharding/
[k8s-network-policy]: https://kubernetes.io/docs/concepts/services-networking/network-policies/
[ingress-class]: https://kubernetes.io/docs/concepts/services-networking/ingress/#ingress-class