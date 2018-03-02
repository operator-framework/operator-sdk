package generator

// handlerTmpl is the template for stub/handler.go.
const handlerTmpl = `package stub

import (
	"{{.OperatorSDKImport}}/handler"
	"{{.OperatorSDKImport}}/types"
)

func NewHandler() handler.Handler {
	return &Handler{}
}

type Handler struct {
	// Fill me
}

func (h *Handler) Handle(ctx types.Context, event types.Event) []types.Action {
	// Fill me
	return nil
}
`
