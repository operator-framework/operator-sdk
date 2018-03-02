package generator

// mainTmpl is the template for cmd/main.go.
const mainTmpl = `package main

import (
	"context"

	stub "{{.StubImport}}"
	sdk "{{.OperatorSDKImport}}"
)

func main() {
	sdk.Watch("apps/v1", "Deployment", "default")
	sdk.Handle(stub.NewHandler())
 	sdk.Run(context.TODO())
}
`
