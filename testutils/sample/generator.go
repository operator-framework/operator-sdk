package sample

import "fmt"

// Generator is a utility that can be used to generate multiple samples at a time.
// A Generator can be configured to run only specific subcommands for the samples via GeneratorOptions functions
type Generator struct {
	init        bool
	api         bool
	webhook     bool
	preInit     GeneratorHook
	postInit    GeneratorHook
	preApi      GeneratorHook
	postApi     GeneratorHook
	preWebhook  GeneratorHook
	postWebhook GeneratorHook
}

// GeneratorHook is a function that takes in a sample and represents a function that gets called in a Pre/Post hook for different stages of sample generation
type GeneratorHook func(Sample)

// GeneratorOptions is a type of function that is used to configure a Generator
type GeneratorOptions func(g *Generator)

// WithNoInit will configure a Generator to not run the GenerateInit function of a Sample
func WithNoInit() GeneratorOptions {
	return func(g *Generator) {
		g.init = false
	}
}

// WithNoApi will configure a Generator to not run the GenerateApi function of a Sample
func WithNoApi() GeneratorOptions {
	return func(g *Generator) {
		g.api = false
	}
}

// WithNoWebhook will configure a Generator to not run the GenerateWebhook function of a Sample
func WithNoWebhook() GeneratorOptions {
	return func(g *Generator) {
		g.webhook = false
	}
}

// WithPreInitHook will configure a Generator to run the given GeneratorHook before executing the GenerateInit function of a Sample
func WithPreInitHook(hook GeneratorHook) GeneratorOptions {
	return func(g *Generator) {
		g.preInit = hook
	}
}

// WithPostInitHook will configure a Generator to run the given GeneratorHook after executing the GenerateInit function of a Sample
func WithPostInitHook(hook GeneratorHook) GeneratorOptions {
	return func(g *Generator) {
		g.postInit = hook
	}
}

// WithPreApiHook will configure a Generator to run the given GeneratorHook before executing the GenerateApi function of a Sample
func WithPreApiHook(hook GeneratorHook) GeneratorOptions {
	return func(g *Generator) {
		g.preApi = hook
	}
}

// WithPostApiHook will configure a Generator to run the given GeneratorHook after executing the GenerateApi function of a Sample
func WithPostApiHook(hook GeneratorHook) GeneratorOptions {
	return func(g *Generator) {
		g.postApi = hook
	}
}

// WithPreWebhookHook will configure a Generator to run the given GeneratorHook before executing the GenerateWebhook function of a Sample
func WithPreWebhookHook(hook GeneratorHook) GeneratorOptions {
	return func(g *Generator) {
		g.preWebhook = hook
	}
}

// WithPostWebhookHook will configure a Generator to run the given GeneratorHook after executing the GenerateWebhook function of a Sample
func WithPostWebhookHook(hook GeneratorHook) GeneratorOptions {
	return func(g *Generator) {
		g.postWebhook = hook
	}
}

// NewGenerator creates a new Generator. The returned Generator can be configured with GeneratorOptions functions.
// By default the Generator that is returned will be set to run all Generate functions of a Sample (GenerateInit, GenerateApi, GenerateWebhook)
func NewGenerator(opts ...GeneratorOptions) *Generator {
	// create a default GeneratorHook that does nothing so we dont get nil pointer errors
	defaultHook := func(s Sample) {}
	g := &Generator{
		init:        true,
		api:         true,
		webhook:     true,
		preInit:     defaultHook,
		postInit:    defaultHook,
		preApi:      defaultHook,
		postApi:     defaultHook,
		preWebhook:  defaultHook,
		postWebhook: defaultHook,
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// GenerateSamples will perform the generation logic for a list of Samples based on the Generator configuration
func (g *Generator) GenerateSamples(samples ...Sample) error {
	for _, sample := range samples {
		fmt.Println("scaffolding sample: ", sample.Name())
		if g.init {
			g.preInit(sample)
			err := sample.GenerateInit()
			if err != nil {
				return fmt.Errorf("error in init generation for sample %s: %w", sample.Name(), err)
			}
			g.postInit(sample)
		}

		if g.api {
			g.preApi(sample)
			err := sample.GenerateApi()
			if err != nil {
				return fmt.Errorf("error in api generation for sample %s: %w", sample.Name(), err)
			}
			g.postApi(sample)
		}

		if g.webhook {
			g.preWebhook(sample)
			err := sample.GenerateWebhook()
			if err != nil {
				return fmt.Errorf("error in webhook generation for sample %s: %w", sample.Name(), err)
			}
			g.postWebhook(sample)
		}
	}

	return nil
}
