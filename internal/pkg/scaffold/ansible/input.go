package ansible

import (
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	"github.com/spf13/afero"
)

type StaticInput struct {
	input.Input
}

func (s *StaticInput) CustomRender() ([]byte, error) {
	return []byte(s.TemplateBody), nil
}

func (s StaticInput) SetFS(_ afero.Fs) {}
