---
title: Writing Kuttl Scorecard Tests
weight: 50
---

This guide outlines the steps which can be followed to implement scorecard
tests using a [kuttl][kuttl] test suite.

## Defining kuttl Test Cases

On running `operator-sdk create api` for the first time, a kuttl config
and a set of kuttl [steps][steps] and [asserts][asserts] are scaffolded in `test/kuttl`
that run a simple install-update-delete test cycle for your new API (and are named as such).
Let's say you've created an API kind `Memcached`, group `cache`, and version `v1alpha1`.
The kuttl test files would look like:

```console
$ tree test/
test/
└── kuttl
    ├── kuttl-test.yaml
    └── v1alpha1
        ├── 00-install-memcached.yaml
        ├── 01-assert-memcached.yaml
        └── 01-modify-memcached.yaml
```

Here's a quick overview of what the files do:
- `test/kuttl/kuttl-test.yaml` - contains a kuttl [`TestSuite` configuration][suite].
This particular configuration is geared towards running your test cases via scorecard so is quite minimal.
- `test/kuttl/v1alpha1/00-install-memcached.yaml` - installs a Memcached CR with a minimal spec.
- `test/kuttl/v1alpha1/01-modify-memcached.yaml` - updates the installed Memcached's spec.
- `test/kuttl/v1alpha1/01-assert-memcached.yaml` - asserts that the spec update occurred.

Lets take a look at a few of these files, first being `kuttl-test.yaml`:

```yaml
apiVersion: kudo.dev/v1beta1
kind: TestSuite
testDirs:
- v1alpha1
# +kubebuilder:scaffold:kuttl:testDirs
timeout: 120
```

First, you should be aware that while this file looks like a `kubectl apply`-able manifest,
it is in fact a component configuration file only meant to be read by kuttl commands. It uses
Kubernetes versioning semantics to convey a defined degree of compatibility. This particular
configuration includes a single test directory `v1alpha1`, the path to our test steps relative
to the config file. The `# +kubebuilder:scaffold:kuttl:testDirs` comment helps `operator-sdk`
commands update this file when new APIs are added, so you don't need to worry (or remove) it.

Next, if you peek inside `01-assert-memcached.yaml` you'll see something like the following:

```yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  name: memcached-test
spec:
  foo: bar
```

Yep, it's a plain Memcached CR manifest, and it _is_ `kubectl apply`-able! Most if not all kuttl
test case files are just Kubernetes YAML manifests that are either merge-patched against in-cluster
objects or compared in assert or error scenarios. Hopefully you can see how powerful this
declarative testing pattern can be.

For further reading take a look through the kuttl [testing docs][kuttl-testing].

## Running kuttl Test Suites with `scorecard`

`operator-sdk scorecard` runs your kuttl suite by passing it to `kubectl-kuttl`, the kuttl testing CLI tool,
from within the `quay.io/operator-framework/scorecard-test-kuttl` image. Since `scorecard` runs tests against
a running operator using its [bundle][scorecard-bundle], your kuttl tests are expected to exist within either a
[local bundle directory][doc-bundle] generated using `make bundle` or a bundle image. However, you
should not bundle test dependencies with your production manifests. This problem is solved by
the [`test-kuttl` Makefile rule](#test-kuttl-rule), which to deploys the operator,
combines on-disk bundle with kuttl manifests, and runs `scorecard` with the kuttl suite selected.

Before discussing `make test-kuttl` usage, lets walk through configuring kuttl to run with scorecard,
assuming your operator is running in-cluster already. Initially your bundle may look like the following:

```console
$ tree bundle/
bundle/
├── manifests
│   |   ...
│   └── memcached-operator.clusterserviceversion.yaml
├── metadata
│   └── annotations.yaml
└── tests
    └── scorecard
        └── config.yaml
```

If you look at scorecard's [configuration file][scorecard-config] `bundle/tests/scorecard/config.yaml`,
you'll notice nothing that refers to the scorecard-test-kuttl image described above. This is by design:
kuttl tests should be enabled only once your operator is ready for in-cluster testing. Therefore the
following patch specified in `config/scorecard/kustomization.yaml` was commented out at first:

```yaml
patchesJson6902:
...
#- path: patches/kuttl.config.yaml
#  target:
#    group: scorecard.operatorframework.io
#    version: v1alpha3
#    kind: Configuration
#    name: config
```

Now that you've written a robust test suite and your operator is ready for testing, uncomment the above
patch, regenerate your bundle, and run your operator:

```sh
make bundle deploy IMG=<operator image>
```

If you inspect `bundle/tests/scorecard/config.yaml` you should now see a kuttl test stage:

```yaml
apiVersion: scorecard.operatorframework.io/v1alpha3
kind: Configuration
metadata:
  name: config
stages:
- parallel: true
  tests:
    ...
- parallel: true
  tests:
  - image: quay.io/operator-framework/scorecard-test-kuttl:v1.2.0
    labels:
      suite: kuttl
```

All that is left to do is to copy your tests to `bundle/tests/scorecard` and run the `scorecard` command
(assuming the `memcached-operator-system` namespace has Memcached CRUD permissions):

```console
$ cp -r test/kuttl bundle/tests/scorecard
$ operator-sdk scorecard ./bundle --selector suite=kuttl --namespace memcached-operator-system
--------------------------------------------------------------------------------
Image:      quay.io/operator-framework/scorecard-test-kuttl:v1.2.0
Labels:
	"suite":"kuttl"
Results:
	Name: cache
	State: pass
$ rm -rf bundle/tests/scorecard/kuttl # Ensure kuttl manifests are not packaged into your production image.
```

#### `test-kuttl` rule

The above steps have been conveniently wrapped for you in the `test-kuttl` Makefile rule, such that
you can run the above like so:

```sh
make test-kuttl IMG=<operator image>
```

Running the above command creates a `testbundle/` directory containing all your bundle manifests, metadata,
and test configuration (including kuttl test manifests) for `scorecard` to run.

### kuttl Test Privileges

The kuttl tests a user might write can vary widely in functionality
and in particular require special Kubernetes RBAC privileges outside
of what your default service account might have. It is therefore very likely
you will be required to run scorecard with an existing custom service account
that holds the required RBAC permissions.

You can specify an existing custom service account in scorecard as follows:
```sh
operator-sdk scorecard ./bundle --service-account=test-sa
```

You can instruct `scorecard` to run tests in an existing non-default namespace
as well:
```sh
operator-sdk scorecard ./bundle --namespace=test-ns
```

If you do not specify either of these flags, the default namespace
and service account for that namespace will be used by the scorecard to run test pods.
Typically you will supply a non-default namespace due to the default namespace lacking
CR CRUD permissions; if `scorecard` is run in the default namespace, you may see errors like:

```
memcacheds.cache.example.com "memcached-test" is forbidden:
  User "system:serviceaccount:default:default" cannot get resource "memcacheds"
  in API group "cache.example.com" in the namespace "default"
```

It is worth noting that the namespace specified to scorecard (or the default if none is set)
is passed to `kubectl-kuttl`, which is responsible for kicking off kuttl tests.
This program will not clean up existing namespaces passed via the CLI (a desirable property),
so you may need to perform additional resource cleanup in your tests. This can typically
be avoided by using [case `delete` configs][step-delete].

Also of note is the `suppress: ["events"]` config setting, which means that kuttl will add
cluster events that occurred over the course of a test suite to kuttl logs. If you wish
to have events included in your scorecard output, bind the following `Role` to your
test service account:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: kuttl-testing-role
rules:
- apiGroups:
  - events.k8s.io
  resources:
  - events
  verbs:
  - list
```

[kuttl]: https://kuttl.dev
[steps]:https://kudo.dev/docs/testing/reference.html#teststep
[suite]:https://kudo.dev/docs/testing/reference.html#testsuite
[assert]:https://kudo.dev/docs/testing/reference.html#testassert
[kuttl-testing]:https://kudo.dev/docs/testing.html#writing-your-first-test
[doc-bundle]:/docs/olm-integration/quickstart-bundle/#creating-a-bundle
[scorcard-bundle]:/docs/advanced-topics/scorecard/scorecard/#running-the-scorecard
[scorcard-config]:/docs/advanced-topics/scorecard/scorecard/#configuration
[steps-delete]:https://kudo.dev/docs/testing/steps.html#deleting-objects
