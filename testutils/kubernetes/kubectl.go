package kubernetes

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/operator-framework/operator-sdk/testutils/command"
)

// Kubectl represents a command line utility that is used for interacting with a Kubernetes cluster
type Kubectl interface {
	// CommandContext returns the CommandContext that is being used by a Kubectl implementation
	CommandContext() command.CommandContext
	// Namespace returns the namespace that a Kubectl implementation is configured to use
	Namespace() string
	// ServiceAccount returns the service account that a Kubectl implementation is configured to use
	ServiceAccount() string

	// Command is used to run any command prefaced by the tool name. i.e `kubectl ...`
	Command(options ...string) (string, error)
	// CommandInNamespace is used to run any command in a namespace, prefaced by the tool name. i.e `kubectl -n namespace ...`
	CommandInNamespace(options ...string) (string, error)
	// Apply is used to run the `apply` subcommand, and can be run in the specified namespace. i.e `kubectl apply ...` or `kubectl apply -n namespace ...`.
	Apply(inNamespace bool, options ...string) (string, error)
	// Get is used to run the `get` subcommand, and can be run in the specified namespace. i.e `kubectl get ...` or `kubectl get -n namespace ...`.
	Get(inNamespace bool, options ...string) (string, error)
	// Delete is used to run the `delete` subcommand, and can be run in the specified namespace. i.e `kubectl delete ...` or `kubectl delete -n namespace ...`.
	Delete(inNamespace bool, options ...string) (string, error)
	// Logs is used to run the `logs` subcommand, and can be run in the specified namespace. i.e `kubectl logs ...` or `kubectl logs -n namespace ...`.
	Logs(inNamespace bool, options ...string) (string, error)
	// Wait is used to run the `wait` subcommand, and can be run in the specified namespace. i.e `kubectl wait ...` or `kubectl wait -n namespace ...`.
	Wait(inNamespace bool, options ...string) (string, error)
	// Version will return the KubernetesVersion that can be retrieved from running `kubectl version`
	Version() (KubernetesVersion, error)
}

// KubectlUtil is an implementation of the Kubectl interface that uses the `kubectl` command line utility
type KubectlUtil struct {
	commandContext command.CommandContext
	namespace      string
	serviceAccount string
}

// KubectlUtilOptions are functions used to configure a KubectlUtil
type KubectlUtilOptions func(ku *KubectlUtil)

// WithCommandContext configures the CommandContext used by a KubectlUtil
func WithCommandContext(cc command.CommandContext) KubectlUtilOptions {
	return func(ku *KubectlUtil) {
		ku.commandContext = cc
	}
}

// WithNamespace configures the namespace used by a KubectlUtil when commands are run namespaced
func WithNamespace(ns string) KubectlUtilOptions {
	return func(ku *KubectlUtil) {
		ku.namespace = ns
	}
}

// WithServiceAccount configures the service account used by a KubectlUtil
func WithServiceAccount(sa string) KubectlUtilOptions {
	return func(ku *KubectlUtil) {
		ku.serviceAccount = sa
	}
}

// NewKubectlUtil creates a new KubectlUtil that can be configured with KubectlUtilOptions functions
func NewKubectlUtil(opts ...KubectlUtilOptions) *KubectlUtil {
	ku := &KubectlUtil{
		commandContext: command.NewGenericCommandContext(),
		namespace:      "test-ns",
		serviceAccount: "test-sa",
	}

	for _, opt := range opts {
		opt(ku)
	}

	return ku
}

// CommandContext returns the CommandContext that is being used by a KubectlUtil
func (ku *KubectlUtil) CommandContext() command.CommandContext {
	return ku.commandContext
}

// Namespace returns the namespace that a KubectlUtil is configured to use
func (ku *KubectlUtil) Namespace() string {
	return ku.namespace
}

// ServiceAccount returns the service account that a KubectlUtil is configured to use
func (ku *KubectlUtil) ServiceAccount() string {
	return ku.serviceAccount
}

// Command is used to run any command prefaced by `kubectl`. i.e `kubectl ...`
func (ku *KubectlUtil) Command(options ...string) (string, error) {
	cmd := exec.Command("kubectl", options...)
	output, err := ku.commandContext.Run(cmd)
	return string(output), err
}

// CommandInNamespace is used to run any command in a namespace, prefaced by `kubectl`. i.e `kubectl -n namespace ...`
func (ku *KubectlUtil) CommandInNamespace(options ...string) (string, error) {
	opts := append([]string{"-n", ku.namespace}, options...)
	return ku.Command(opts...)
}

// Apply is used to run the `kubectl apply` subcommand, and can be run in the specified namespace. i.e `kubectl apply ...` or `kubectl apply -n namespace ...`.
func (ku *KubectlUtil) Apply(inNamespace bool, options ...string) (string, error) {
	return ku.prefixCommand("apply", inNamespace, options...)
}

// Get is used to run the `kubectl get` subcommand, and can be run in the specified namespace. i.e `kubectl get ...` or `kubectl get -n namespace ...`.
func (ku *KubectlUtil) Get(inNamespace bool, options ...string) (string, error) {
	return ku.prefixCommand("get", inNamespace, options...)
}

// Delete is used to run the `kubectl delete` subcommand, and can be run in the specified namespace. i.e `kubectl delete ...` or `kubectl delete -n namespace ...`.
func (ku *KubectlUtil) Delete(inNamespace bool, options ...string) (string, error) {
	return ku.prefixCommand("delete", inNamespace, options...)
}

// Logs is used to run the `kubectl logs` subcommand, and can be run in the specified namespace. i.e `kubectl logs ...` or `kubectl logs -n namespace ...`.
func (ku *KubectlUtil) Logs(inNamespace bool, options ...string) (string, error) {
	return ku.prefixCommand("logs", inNamespace, options...)
}

// Wait is used to run the `kubectl wait` subcommand, and can be run in the specified namespace. i.e `kubectl wait ...` or `kubectl wait -n namespace ...`.
func (ku *KubectlUtil) Wait(inNamespace bool, options ...string) (string, error) {
	return ku.prefixCommand("wait", inNamespace, options...)
}

// Version is used to run the `kubectl version` subcommand, and returns the KubernetesVersion that is parsed from its output
func (ku *KubectlUtil) Version() (KubernetesVersion, error) {
	out, err := ku.Command("version", "-o", "json")
	if err != nil {
		return nil, err
	}

	var versions map[string]json.RawMessage

	err = json.Unmarshal([]byte(out), &versions)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling json: %w", err)
	}

	clientVersion, err := NewKubeVersionInfo(string(versions["clientVersion"]))
	if err != nil {
		return nil, fmt.Errorf("error getting client version: %w", err)
	}

	serverVersion, err := NewKubeVersionInfo(string(versions["serverVersion"]))
	if err != nil {
		return nil, fmt.Errorf("error getting server version: %w", err)
	}

	return NewKubeVersion(WithClientVersion(clientVersion), WithServerVersion(serverVersion)), nil
}

// prefixCommand is a helper function that will prefix the subcommand and its options with `kubectl` or `kubectl -n namespace`.
func (ku *KubectlUtil) prefixCommand(subcommand string, inNamespace bool, options ...string) (string, error) {
	opts := append([]string{subcommand}, options...)

	if inNamespace {
		return ku.CommandInNamespace(opts...)
	}

	return ku.Command(opts...)
}
