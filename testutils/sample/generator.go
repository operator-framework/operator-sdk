package sample

import "fmt"

// Generator is a utility that can be used to generate multiple samples at a time.
// A Generator can be configured to run only specific subcommands for the samples via GeneratorOptions functions
type Generator struct {
	init    bool
	api     bool
	webhook bool
}

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

// NewGenerator creates a new Generator. The returned Generator can be configured with GeneratorOptions functions.
// By default the Generator that is returned will be set to run all Generate functions of a Sample (GenerateInit, GenerateApi, GenerateWebhook)
func NewGenerator(opts ...GeneratorOptions) *Generator {
	g := &Generator{
		init:    true,
		api:     true,
		webhook: true,
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
			err := sample.GenerateInit()
			if err != nil {
				return fmt.Errorf("error in init generation for sample %s: %w", sample.Name(), err)
			}
		}

		if g.api {
			err := sample.GenerateApi()
			if err != nil {
				return fmt.Errorf("error in api generation for sample %s: %w", sample.Name(), err)
			}
		}

		if g.webhook {
			err := sample.GenerateWebhook()
			if err != nil {
				return fmt.Errorf("error in webhook generation for sample %s: %w", sample.Name(), err)
			}
		}
	}

	return nil
}
