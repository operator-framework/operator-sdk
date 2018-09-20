package scaffold

import "io"

type version struct {
}

func NewVersionCoden() Codegen {
	return &version{}
}

func (v *version) Render(w io.Writer) error {
	_, err := w.Write([]byte(versionTemplate))
	return err
}

const versionTemplate = `package version

var (
	Version = "0.0.1"
)
`
