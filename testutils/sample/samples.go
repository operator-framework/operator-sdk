package sample

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/operator-framework/operator-sdk/testutils/command"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Sample represents a sample project that can be created and used for testing
type Sample interface {
	// CommandContext returns the CommandContext that the Sample is using
	CommandContext() command.CommandContext
	// Name returns the name of the Sample
	Name() string
	// GVKs return an array of GVKs that are used when generating the apis and webhooks for the Sample
	GVKs() []schema.GroupVersionKind
	// Domain returs the domain of the sample
	Domain() string
	// Dir returns the directory the sample is created in
	Dir() string
	// Binary returns the binary that is used when creating a sample
	Binary() string
	// GenerateInit scaffolds using the `init` subcommand
	GenerateInit() error
	// GenerateApi scaffolds using the `create api` subcommand
	GenerateApi() error
	// GenerateWebhook scaffolds using the `create webhook` subcommand
	GenerateWebhook() error
}

// GenericSample is a generalized object that implements the Sample interface. It is meant
// to be as simple and versatile to make the process of generating test samples easier
// TODO: Consider making it easier to create custom cli flags on a per GVK basis for the `create api` and `create webhook` subcommands
type GenericSample struct {
	domain         string
	repo           string
	gvks           []schema.GroupVersionKind
	commandContext command.CommandContext
	name           string
	binary         string
	plugins        []string

	initOptions    []string
	apiOptions     []string
	webhookOptions []string
}

// GenericSampleOption is a function that modifies the values in a GenericSample to be used when creating a new GenericSample
type GenericSampleOption func(gs *GenericSample)

// WithDomain sets the domain to be used during scaffold execution
func WithDomain(domain string) GenericSampleOption {
	return func(gs *GenericSample) {
		gs.domain = domain
	}
}

// WithRepository sets the repository to be used during scaffold execution
func WithRepository(repo string) GenericSampleOption {
	return func(gs *GenericSample) {
		gs.repo = repo
	}
}

// WithGvk sets the GroupVersionKind to be used during scaffold execution
func WithGvk(gvks ...schema.GroupVersionKind) GenericSampleOption {
	return func(gs *GenericSample) {
		gs.gvks = make([]schema.GroupVersionKind, len(gvks))
		copy(gs.gvks, gvks)
	}
}

// WithName sets the name of the sample that is scaffolded
func WithName(name string) GenericSampleOption {
	return func(gs *GenericSample) {
		gs.name = name
	}
}

// WithCommandContext sets the CommandContext that is used to execute scaffold commands
func WithCommandContext(commandContext command.CommandContext) GenericSampleOption {
	return func(gs *GenericSample) {
		gs.commandContext = commandContext
	}
}

// WithBinary sets the binary that should be used to run scaffold commands
func WithBinary(binary string) GenericSampleOption {
	return func(gs *GenericSample) {
		gs.binary = binary
	}
}

// WithPlugins sets the plugins that should be used during scaffolding
func WithPlugins(plugins ...string) GenericSampleOption {
	return func(gs *GenericSample) {
		gs.plugins = make([]string, len(plugins))
		copy(gs.plugins, plugins)
	}
}

// WithExtraInitOptions sets any additional options that should be passed into an init subcommand
func WithExtraInitOptions(options ...string) GenericSampleOption {
	return func(gs *GenericSample) {
		gs.initOptions = make([]string, len(options))
		copy(gs.initOptions, options)
	}
}

// WithExtraApiOptions sets any additional options that should be passed into a create api subcommand
func WithExtraApiOptions(options ...string) GenericSampleOption {
	return func(gs *GenericSample) {
		gs.apiOptions = make([]string, len(options))
		copy(gs.apiOptions, options)
	}
}

// WithExtraWebhookOptions sets any additional options that should be passed into a create webhook subcommand
func WithExtraWebhookOptions(options ...string) GenericSampleOption {
	return func(gs *GenericSample) {
		gs.webhookOptions = make([]string, len(options))
		copy(gs.webhookOptions, options)
	}
}

// NewGenericSample will return a new GenericSample object. The values used in the GenericSample can be modified using GenericSampleOption functions
func NewGenericSample(opts ...GenericSampleOption) *GenericSample {
	gs := &GenericSample{
		domain: "example.com",
		name:   "generic-sample",
		gvks: []schema.GroupVersionKind{
			{
				Group:   "sample",
				Version: "v1",
				Kind:    "Generic",
			},
		},
		// by default use kubebuilder unless otherwise specified
		binary:         "kubebuilder",
		repo:           "",
		commandContext: command.NewGenericCommandContext(),
		plugins:        []string{"go/v3"},
	}

	for _, opt := range opts {
		opt(gs)
	}

	return gs
}

// CommandContext returns the CommandContext that the GenericSample is using
func (gs *GenericSample) CommandContext() command.CommandContext {
	return gs.commandContext
}

// Name returns the name of the GenericSample
func (gs *GenericSample) Name() string {
	return gs.name
}

// GVKs returns the list of GVKs that is used for creating apis and webhooks
func (gs *GenericSample) GVKs() []schema.GroupVersionKind {
	return gs.gvks
}

// Dir returns the directory the sample is created in
func (gs *GenericSample) Dir() string {
	return gs.commandContext.Dir() + "/" + gs.name
}

// Binary returns the binary used when creating the sample
func (gs *GenericSample) Binary() string {
	return gs.binary
}

// Domain returns the domain of the GenericSample
func (gs *GenericSample) Domain() string {
	return gs.domain
}

// GenerateInit runs the `init` subcommand of the binary provided
func (gs *GenericSample) GenerateInit() error {
	options := []string{
		"init",
		"--plugins",
		strings.TrimRight(strings.Join(gs.plugins, ","), ","),
		"--domain",
		gs.domain,
	}

	if gs.repo != "" {
		options = append(options, "--repo", gs.repo)
	}

	options = append(options, gs.initOptions...)

	ex := exec.Command(gs.binary, options...)

	output, err := gs.commandContext.Run(ex, gs.name)
	if err != nil {
		return fmt.Errorf("error running command: %w | output: %s", err, string(output))
	}

	return nil
}

// GenerateApi runs the `create api` subcommand of the binary provided
func (gs *GenericSample) GenerateApi() error {
	for _, gvk := range gs.gvks {
		options := []string{
			"create",
			"api",
			"--plugins",
			strings.TrimRight(strings.Join(gs.plugins, ","), ","),
			"--group",
			gvk.Group,
			"--version",
			gvk.Version,
			"--kind",
			gvk.Kind,
		}

		options = append(options, gs.apiOptions...)

		ex := exec.Command(gs.binary, options...)

		output, err := gs.commandContext.Run(ex, gs.name)
		if err != nil {
			return fmt.Errorf("error running command: %w | output: %s", err, string(output))
		}
	}

	return nil
}

// GenerateWebhook runs the `create webhook` subcommand of the binary provided
func (gs *GenericSample) GenerateWebhook() error {
	for _, gvk := range gs.gvks {
		options := []string{
			"create",
			"webhook",
			"--plugins",
			strings.TrimRight(strings.Join(gs.plugins, ","), ","),
			"--group",
			gvk.Group,
			"--version",
			gvk.Version,
			"--kind",
			gvk.Kind,
		}

		options = append(options, gs.webhookOptions...)

		ex := exec.Command(gs.binary, options...)

		output, err := gs.commandContext.Run(ex, gs.name)
		if err != nil {
			return fmt.Errorf("error running command: %w | output: %s", err, string(output))
		}
	}

	return nil
}
