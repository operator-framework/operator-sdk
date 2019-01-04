# Ansible Operator Testing

## Background
All operators should fit into the e2e testing framework used by operator-sdk, including Ansible Operator.

## Goals

- Add support for the `test {local|cluster}` e2e testing subcommand with Ansible Operators
- Integrate with existing Ansible test frameworks to provide a uniform experience across the Ansible ecosystem
- Maintain the existing interface for the `test` subcommands

## Non-Goals

- Create a new testing framework for Ansible that plays nicely with operators
- Allow Ansible users to write non-Ansible tests

## Solution
1. Update scaffolding to (optionally?) include [molecule](https://molecule.readthedocs.io/en/latest/) initialization
    - Set up a molecule scenario for the e2e environment (may have different behavior between `local` and `cluster` scenarios)
1. Create a [delegated driver](https://molecule.readthedocs.io/en/latest/configuration.html#delegated) for molecule that:
    - In the `test local` case handles the creation of the necessary resources (namespace, CRDs, roles, rolebindings, operator deployment) in the Kubernetes cluster
    - In the `test cluster` case does nothing
1. Add a custom entrypoint for testing that will spin up the operator and then run the proper molecule scenario, which can be included in the
   Ansible Operator image when it is built with the `--enable-tests` option
1. Update the `test local` subcommand so that when it is run in the context of an Ansible Operator, it will trigger a molecule run of the proper scenario
1. Update the `test cluster` subcommand so that when it is run in the context of an Ansible Operator, a deployment of the operator with the custom testing entrypoint 
   is created. The behavior here should approximate the Golang operator equivalent, in terms of reporting/termination

## Discussion / Further Investigation
- Should test scaffolding be optional, or should we always initialize tests?
- Can we easily get access to the `deploy/` resources from our molecule test?
- How do we distribute the custom molecule driver?
- Should `test local` and `test cluster` be two different molecule scenarios?
    - `test local` can run molecule at the permissions required to create CRDs, roles, SAs, etc
        - What if it isn't? Maybe we should do a best effort here, and not crash the run if we hit permissions issues.
    - `test cluster` will be run at normal operator permissions, and will require that the prerequisites be satisfied before invocation
