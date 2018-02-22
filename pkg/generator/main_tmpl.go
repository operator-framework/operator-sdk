package generator

// mainTmpl is the template for cmd/main.go.
const mainTmpl = `package main

import (
	"context"

	sdk "{{.OperatorSDKImport}}"
	stub "{{.StubImport}}"
)

func main() {
	namespace := "default"
	sdk.Watch("apps/v1", "Deployment", namespace)
 	sdk.Handle(&stub.Handler{})
 	sdk.Run(context.TODO())
}
`
