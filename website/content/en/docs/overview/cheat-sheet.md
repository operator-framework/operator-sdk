---
title: "Cheat Sheet"
linkTitle: "Cheat Sheet"
weight: 3
description: >
    Operator-SDK Cheat Sheet commands and operations
---

Below you will find a cheat sheet with options and helpers for projects, which are built with the SDK and are respecting its proposed layout.

## Common commands and options

| Command   | Description  |
|-------|-----------|
| `operator-sdk init`          | To initialize an operator project in the current directory. |
| `operator-sdk init --plugins=<plugin-key>`          | To initialize an operator project in the current directory using a specific plugin. To check the available plugins you can run `operator-sdk --help`. E.g (`operator-sdk init --plugins=helm`).|
| `operator-sdk create api [flags]`          | Lets you create your own APIs with its [GKV][gkvs] by [Extending the Kubernetes API with CustomResourceDefinitions][extend-k8s-api], or lets you use external/core-types. Also generates their respective [controllers][controllers-k8s-doc].|
| `operator-sdk create webhook [flags]`          | To scaffold [Webhooks][webhooks-k8s-doc] for the APIs declared in the project. Currently, only the Go-based project supports this option. |
| `make docker-build IMG=<some-registry>/<project-name>:<tag>`          | Build the operator image.      |
| `make docker-build docker-push IMG=<some-registry>/<project-name>:<tag>`      | Build and push the operator image for your registry.  |
| `make install`         | Install the CRDs into the cluster. |
| `make uninstall`         | Uninstall the CRDs into the cluster. |
| `make run`         | Run your controller locally and outside of the cluster. Note that this will run in the foreground, so switch to a new terminal if you want to leave it running. |
| `make deploy`         | Deploy your project on the cluster. |
| `make undeploy`         | Undeploy your project on the cluster. |

## To create bundles, catalogs, and develop for OLM

For further information check [Operator SDK Integration with Operator Lifecycle Manager][olm-integration].

| Command   | Description  |
|-------|-----------|
| `make bundle`          | Create/update the [bundle][bundle] based on the project manifests in the `bundle/` directory. For more info see [Create a bundle][creating-a-bundle].      |
| `operator-sdk bundle validate ./bundle`          | To validate your [bundle][bundle] spec definition.      |
| `operator-sdk bundle validate ./bundle --select-optional suite=operatorframework` | Validate your bundle against [OperatorHub.io][operatorhub-io] criteria. For further information use the flag `--help`. |
| `operator-sdk olm install` | To install OLM on your cluster for development purposes. |
| `operator-sdk olm uninstall` | To uninstall OLM from your cluster. |
| `make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>` | To build your bundle operator image. |
| `make bundle-build bundle-push BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>` | To build and push your bundle operator image. |
| `operator-sdk run bundle <some-registry>/<project-name-bundle>:<tag>` | To deploy your bundle operator using OLM on your cluster for development purposes. |
| `operator-sdk run bundle private-registry.org/bundle:v1.2.3 --service-account sa-with-secret --pull-secret-name regcred --ca-secret-name cert-sec` | Configure `run bundle` (and `run bundle-upgrade`) to use an image pull secret, non-default service account configured with that secret, and custom CA certificate secret |
<!-- TODO(estroz): remove the service account requirement once OLM releases a patch or new
minor release containing https://github.com/operator-framework/operator-lifecycle-manager/pull/1941 -->

### Updating bundle channels
 
The following examples let you update the [bundle][bundle] with data-informed. For further information also check [Upgrade your Operator][upgrade-project] and see [Channel Naming][channel-namming-doc].  
 
**NOTE:** Note that it will carry over any customizations you have made and ensure a rolling update to the next version of your Operator. 

```sh
make bundle CHANNELS=fast,preview DEFAULT_CHANNEL=stable VERSION=1.0.0 IMG=<some-registry>/<project-name-bundle>:<tag>
```

**NOTE** You can use environment variables to pass the values such as `export CHANNELS=fast,candidate`. Note that, their values will be used by `make bundle` command.

## To test your projects

| Command   | Description  |
|-------|-----------|
| `operator-sdk scorecard ./bundle`          |  Run the [Scorecard][scorcard] tests for your bundle.  |
| `make test`          |  Run Go tests. It is valid only for Go-based operators.    |
| `molecule test`          |  Run [Molecule][molecule-tests] tests.  It is valid only for Ansible-based operators. |
| `helm test`          |  Run [Helm chart tests][helm-chart-tests].  It is valid only for Helm-based operators. |

**NOTE:** This is not a comprehensive list of make targets or commands. Please see the scaffolded Makefile and `make help` for the full list of targets. Note that you can use `operator-sdk <command> --help` and check the [CLI][cli] section to check all options.
 
[olm-integration]: /docs/olm-integration/
[creating-a-bundle]: /docs/olm-integration/tutorial-bundle/#creating-a-bundle
[bundle]:https://github.com/operator-framework/operator-registry/blob/v1.16.1/docs/design/operator-bundle.md
[operatorhub-io]: https://operatorhub.io/
[upgrade-project]: /docs/olm-integration/generation/#upgrade-your-operator
[channel-namming-doc]: https://olm.operatorframework.io/docs/best-practices/channel-naming/
[controllers-k8s-doc]: https://kubernetes.io/docs/concepts/architecture/controller
[gkvs]: https://book.kubebuilder.io/cronjob-tutorial/gvks.html
[extend-k8s-api]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/
[webhooks-k8s-doc]: https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/
[scorcard]: /docs/testing-operators/scorecard/
[molecule-tests]: /docs/building-operators/ansible/testing-guide
[helm-chart-tests]: https://helm.sh/docs/topics/chart_tests/
[cli]: /docs/cli/
