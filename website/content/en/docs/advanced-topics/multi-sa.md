---
title: Multiple Service Accounts
linkTitle: Multiple Service Accounts
weight: 80
---

### Using Multiple Service Accounts

There may be a need to have multiple service accounts to provide only the necessary permissions to various objects that the operator creates on a Kubernetes cluster.

This can be accomplished by using the `--extra-service-accounts` flag when generating the bundle with `make bundle`.

#### Updating the `Makefile` to use `--extra-service-accounts`

Update the `bundle` target in the `Makefile` to add the `--extra-service-accounts` flag with the name of the desired service account. This ensures that the permissions and configurations do not get overwritten by `make bundle`.
For example, modify the line that contains `operator-sdk generate bundle` similar to below replacing `myOperator-name-additional-service-account` to the desired service account name appended to the operator name.

```
bundle: manifests kustomize operator-sdk ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --extra-service-accounts myOperator-name-additional-service-account --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle
```

The `--extra-service-accounts` flag takes a comma-separated list of strings, so you can add more than a single service account name if desired.

#### Add RBAC configurations for `--extra-service-accounts`

These steps will need to be followed for every additional service account.

1. Create a new service account file. For example:
   ```
   cat << EOF > config/rbac/additional_service_account.yaml
   apiVersion: v1
   kind: ServiceAccount
   metadata:
     name: additional-service-account
     namespace: system
   EOF
   ```

2. Create a role binding. In this example, it is a `ClusterRoleBinding`:
   ```
   cat << EOF > config/rbac/additional_role_binding.yaml
   apiVersion: rbac.authorization.k8s.io/v1
   kind: ClusterRoleBinding
   metadata:
     name: additional-service-account-rolebinding
   roleRef:
     apiGroup: rbac.authorization.k8s.io
     kind: ClusterRole
     name: additional-service-account-role
   subjects:
   - kind: ServiceAccount
     name: additional-service-account
     namespace: system
   EOF
   ```

3. Create a role with desired permissions. In this example, it is a `ClusterRole` that provides permission to the `privileged` `SecurityContextConstraint` (`SCC`).
   ```
   cat << EOF > config/rbac/additional_role.yaml
   apiVersion: rbac.authorization.k8s.io/v1
   kind: ClusterRole
   metadata:
     creationTimestamp: null
     name: additional-service-account-role
   rules:
   - apiGroups:
     - security.openshift.io
     resourceNames:
     - privileged
     resources:
     - securitycontextconstraints
     verbs:
     - use
   EOF
   ```


#### Update the RBAC `kustomization.yaml`

Make sure to update the RBAC configuration `kustomization.yaml` file with the previously created RBAC `yaml` files.
For example:

```
cat << EOF >> config/rbac/kustomization.yaml

# Add MyCustomObject service account
- additional_service_account.yaml
- additional_role.yaml
- additional_role_binding.yaml
EOF
```

