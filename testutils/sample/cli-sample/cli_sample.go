package clisample

import (
	"fmt"
	"os"
	"strings"

	manifestsv2 "github.com/operator-framework/operator-sdk/internal/plugins/manifests/v2"
	scorecardv2 "github.com/operator-framework/operator-sdk/internal/plugins/scorecard/v2"
	"github.com/operator-framework/operator-sdk/testutils/command"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kubebuilder/v3/pkg/cli"
	cfgv3 "sigs.k8s.io/kubebuilder/v3/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	kustomizev1 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v1"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang"
	golangv3 "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3"
)

// CliSample is a generalized object that implements the Sample interface. It is meant
// to be as simple and versatile to make the process of generating test samples easier
// TODO: Consider making it easier to create custom cli flags on a per GVK basis for the `create api` and `create webhook` subcommands
type CliSample struct {
	domain         string
	repo           string
	gvks           []schema.GroupVersionKind
	commandContext command.CommandContext
	name           string
	plugins        []string
	cli            *cli.CLI

	initOptions    []string
	apiOptions     []string
	webhookOptions []string
}

// CliSampleOption is a function that modifies the values in a CliSample to be used when creating a new CliSample
type CliSampleOption func(gs *CliSample)

// WithDomain sets the domain to be used during scaffold execution
func WithDomain(domain string) CliSampleOption {
	return func(gs *CliSample) {
		gs.domain = domain
	}
}

// WithRepository sets the repository to be used during scaffold execution
func WithRepository(repo string) CliSampleOption {
	return func(gs *CliSample) {
		gs.repo = repo
	}
}

// WithGvk sets the GroupVersionKind to be used during scaffold execution
func WithGvk(gvks ...schema.GroupVersionKind) CliSampleOption {
	return func(gs *CliSample) {
		gs.gvks = make([]schema.GroupVersionKind, len(gvks))
		copy(gs.gvks, gvks)
	}
}

// WithName sets the name of the sample that is scaffolded
func WithName(name string) CliSampleOption {
	return func(gs *CliSample) {
		gs.name = name
	}
}

// WithCommandContext sets the CommandContext that is used to execute scaffold commands
func WithCommandContext(commandContext command.CommandContext) CliSampleOption {
	return func(gs *CliSample) {
		gs.commandContext = commandContext
	}
}

// WithPlugins sets the plugins that should be used during scaffolding
func WithPlugins(plugins ...string) CliSampleOption {
	return func(gs *CliSample) {
		gs.plugins = make([]string, len(plugins))
		copy(gs.plugins, plugins)
	}
}

// WithExtraInitOptions sets any additional options that should be passed into an init subcommand
func WithExtraInitOptions(options ...string) CliSampleOption {
	return func(gs *CliSample) {
		gs.initOptions = make([]string, len(options))
		copy(gs.initOptions, options)
	}
}

// WithExtraApiOptions sets any additional options that should be passed into a create api subcommand
func WithExtraApiOptions(options ...string) CliSampleOption {
	return func(gs *CliSample) {
		gs.apiOptions = make([]string, len(options))
		copy(gs.apiOptions, options)
	}
}

// WithExtraWebhookOptions sets any additional options that should be passed into a create webhook subcommand
func WithExtraWebhookOptions(options ...string) CliSampleOption {
	return func(gs *CliSample) {
		gs.webhookOptions = make([]string, len(options))
		copy(gs.webhookOptions, options)
	}
}

// WithCLI sets the CLI tool that this sample should use for generation
func WithCLI(c *cli.CLI) CliSampleOption {
	return func(gs *CliSample) {
		gs.cli = c
	}
}

// NewCliSample will return a new CliSample object. The values used in the CliSample can be modified using CliSampleOption functions
func NewCliSample(opts ...CliSampleOption) (*CliSample, error) {
	// Create a very basic default CLI with a go/v3 plugin
	gov3Bundle, _ := plugin.NewBundle(golang.DefaultNameQualifier, golangv3.Plugin{}.Version(),
		kustomizev1.Plugin{},
		golangv3.Plugin{},
		manifestsv2.Plugin{},
		scorecardv2.Plugin{},
	)

	c, err := cli.New(
		cli.WithCommandName("cli"),
		cli.WithVersion("v0.0.0"),
		cli.WithPlugins(
			gov3Bundle,
		),
		cli.WithDefaultPlugins(cfgv3.Version, gov3Bundle),
		cli.WithDefaultProjectVersion(cfgv3.Version),
		cli.WithCompletion(),
	)
	if err != nil {
		return nil, fmt.Errorf("encountered an error creating a new CliSample: %w", err)
	}

	gs := &CliSample{
		domain: "example.com",
		name:   "cli-sample",
		gvks: []schema.GroupVersionKind{
			{
				Group:   "sample",
				Version: "v1",
				Kind:    "Cli",
			},
		},
		repo:           "",
		commandContext: command.NewGenericCommandContext(),
		plugins:        []string{"go/v3"},
		cli:            c,
	}

	for _, opt := range opts {
		opt(gs)
	}

	return gs, nil
}

// CommandContext returns the CommandContext that the CliSample is using
func (gs *CliSample) CommandContext() command.CommandContext {
	return gs.commandContext
}

// Name returns the name of the CliSample
func (gs *CliSample) Name() string {
	return gs.name
}

// GVKs returns the list of GVKs that is used for creating apis and webhooks
func (gs *CliSample) GVKs() []schema.GroupVersionKind {
	return gs.gvks
}

// Dir returns the directory the sample is created in
func (gs *CliSample) Dir() string {
	return gs.commandContext.Dir() + "/" + gs.name
}

// Binary returns the binary used when creating the sample
func (gs *CliSample) Binary() string {
	return "custom-cli"
}

// Domain returns the domain of the CliSample
func (gs *CliSample) Domain() string {
	return gs.domain
}

// GenerateInit runs the `init` subcommand of the binary provided
func (gs *CliSample) GenerateInit() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("encountered an error getting the current working directory: %w", err)
	}
	// defer returning to the original working directory
	defer func() { os.Chdir(cwd) }()
	// Change directory to the context specified
	err = os.Chdir(gs.commandContext.Dir())
	if err != nil {
		return fmt.Errorf("encountered an error switching to context directory: %w", err)
	}

	// Cobra's Execute command by default uses os.Args[1:] so we need to add an extra
	// arg to take place of os.Args[0]
	args := []string{
		"cli",
		"init",
		"--plugins",
		strings.TrimRight(strings.Join(gs.plugins, ","), ","),
		"--domain",
		gs.domain,
	}

	if gs.repo != "" {
		args = append(args, "--repo", gs.repo)
	}

	args = append(args, gs.initOptions...)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = args

	err = gs.cli.Run()
	if err != nil {
		return fmt.Errorf("encountered an error when running `init` subcommand for the cli sample: %w", err)
	}

	return nil
}

// GenerateApi runs the `create api` subcommand of the binary provided
func (gs *CliSample) GenerateApi() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("encountered an error getting the current working directory: %w", err)
	}
	// defer returning to the original working directory
	defer func() { os.Chdir(cwd) }()
	// Change directory to the context specified
	err = os.Chdir(gs.commandContext.Dir())
	if err != nil {
		return fmt.Errorf("encountered an error switching to context directory: %w", err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	for _, gvk := range gs.gvks {
		args := []string{
			"cli",
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

		args = append(args, gs.apiOptions...)

		os.Args = args

		err := gs.cli.Run()
		if err != nil {
			return fmt.Errorf("encountered an error when running `create api` subcommand for the cli sample: %w", err)
		}
	}

	return nil
}

// GenerateWebhook runs the `create webhook` subcommand of the binary provided
func (gs *CliSample) GenerateWebhook() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("encountered an error getting the current working directory: %w", err)
	}
	// defer returning to the original working directory
	defer func() { os.Chdir(cwd) }()
	// Change directory to the context specified
	err = os.Chdir(gs.commandContext.Dir())
	if err != nil {
		return fmt.Errorf("encountered an error switching to context directory: %w", err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	for _, gvk := range gs.gvks {
		args := []string{
			"cli",
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

		args = append(args, gs.webhookOptions...)

		os.Args = args

		err := gs.cli.Run()
		if err != nil {
			return fmt.Errorf("encountered an error when running `create api` subcommand for the cli sample: %w", err)
		}
	}

	return nil
}
