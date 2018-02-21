package generator

// handlerTmpl is the template for stub/handler.go.
const handlerTmpl = `package stub

import (
	"{{.OperatorSDKImport}}/types"
)

type Handler struct {
	// Fill me
}

func (h *Handler) Handle(ctx types.Context, event types.Event) {
	// Fill me
}
`
