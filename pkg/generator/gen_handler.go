package generator

import (
	"io"
	"text/template"
)

// Handler contains all the customized data needed to generate stub/handler.go for a new operator
// when pairing with handlerTmpl template.
type Handler struct {
	// imports
	OperatorSDKImport string
}

// renderHandler generates the stub/handler.go file.
func renderHandler(w io.Writer) error {
	t := template.New("stub/handler.go")
	t, err := t.Parse(handlerTmpl)
	if err != nil {
		return err
	}

	h := Handler{
		OperatorSDKImport: sdkImport,
	}
	return t.Execute(w, h)
}
