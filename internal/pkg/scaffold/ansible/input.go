package ansible

import (
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	"github.com/spf13/afero"
)

// StaticInput is the input for scaffolding a static file with
// no parameteres
type StaticInput struct {
	input.Input
}

// CustomRender return the template body unmodified
func (s *StaticInput) CustomRender() ([]byte, error) {
	return []byte(s.TemplateBody), nil
}

func (s StaticInput) SetFS(_ afero.Fs) {}
