<img src="website/static/operator_logo_sdk_color.svg" height="125px"></img>

> ⚠️ **IMPORTANT NOTICE:** Images under `gcr.io/kubebuilder/` Will Be Unavailable Soon
>
> **If your project uses `gcr.io/kubebuilder/kube-rbac-proxy`** it will be affected.
> Your project may fail to work if the image cannot be pulled. **You must move as soon as possible**, sometime from early 2025, the GCR will go away.
>
> The usage of the project [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) was discontinued from Kubebuilder and Operator-SDK.
> It was replaced for similar protection using `authn/authz` via Controller-Runtime's feature [WithAuthenticationAndAuthorization](https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/metrics/filters#WithAuthenticationAndAuthorization).
>
> For more information and guidance see the discussion https://github.com/kubernetes-sigs/kubebuilder/discussions/3907

[![Build Status](https://github.com/operator-framework/operator-sdk/workflows/deploy/badge.svg)](https://github.com/operator-framework/operator-sdk/actions)
[![License](http://img.shields.io/:license-apache-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0.html)

## Documentation

Docs can be found on the [Operator SDK website][sdk-docs].

## Overview

This project is a component of the [Operator Framework][of-home], an
open source toolkit to manage Kubernetes native applications, called
Operators, in an effective, automated, and scalable way. Read more in
the [introduction blog post][of-blog].

[Operators][operator-link] make it easy to manage complex stateful
applications on top of Kubernetes. However writing an Operator today can
be difficult because of challenges such as using low level APIs, writing
boilerplate, and a lack of modularity which leads to duplication.

The Operator SDK is a framework that uses the
[controller-runtime][controller-runtime] library to make writing
operators easier by providing:

- High level APIs and abstractions to write the operational logic more intuitively
- Tools for scaffolding and code generation to bootstrap a new project fast
- Extensions to cover common Operator use cases

## Dependency and platform support

### Go version

Release binaries will be built with the Go compiler version specified in the [developer guide][dev-guide-prereqs].
A Go Operator project's Go version can be found in its `go.mod` file.

[dev-guide-prereqs]:https://sdk.operatorframework.io/docs/contribution-guidelines/developer-guide#prerequisites

### Kubernetes versions

Supported Kubernetes versions for your Operator project or relevant binary can be determined
by following this [compatibility guide][k8s-compat].

[k8s-compat]:https://sdk.operatorframework.io/docs/overview#kubernetes-version-compatibility

### Platforms

The set of supported platforms for all binaries and images can be found in [these tables][platforms].

[platforms]:https://sdk.operatorframework.io/docs/overview#platform-support

## Community and how to get involved

- [Operator framework community][operator-framework-community]
- [Communication channels][operator-framework-communication]
- [Project meetings][operator-framework-meetings]

## How to contribute

Check out the [contributor documentation][contribution-docs].

## License

Operator SDK is under Apache 2.0 license. See the [LICENSE][license_file] file for details.

[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[license_file]:./LICENSE
[of-home]: https://github.com/operator-framework
[of-blog]: https://www.openshift.com/blog/introducing-the-operator-framework
[operator-link]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
[sdk-docs]: https://sdk.operatorframework.io
[operator-framework-community]: https://github.com/operator-framework/community
[operator-framework-communication]: https://github.com/operator-framework/community#get-involved
[operator-framework-meetings]: https://github.com/operator-framework/community#meetings
[contribution-docs]: https://sdk.operatorframework.io/docs/contribution-guidelines/
