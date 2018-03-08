package generator

// handlerTmpl is the template for stub/handler.go.
const handlerTmpl = `package stub

import (
	"{{.OperatorSDKImport}}/handler"
	"{{.OperatorSDKImport}}/types"
	"github.com/sirupsen/logrus"
	apps_v1 "k8s.io/api/apps/v1"
)

func NewHandler() handler.Handler {
	return &Handler{}
}

type Handler struct {
	// Fill me
}

func (h *Handler) Handle(ctx types.Context, event types.Event) []types.Action {
	// Change me
	switch o := event.Object.(type) {
	case *apps_v1.Deployment:
		logrus.Printf("Received Deployment: %v", o.Name)
	}
	return nil
}
`
