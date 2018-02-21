package generator

// mainTmpl is the template for cmd/main.go.
const mainTmpl = `package main

import (
	"context"

	sdk "{{.OperatorSDKImport}}"
	api "{{.APIImport}}"
	stub "{{.StubImport}}"
)

func main() {
	namespace := "default"
	sdk.Watch(api.{{.ServicePlural}}, namespace, api.{{.Service}})
 	sdk.Handle(&stub.Handler{})
 	sdk.Run(context.TODO())
}
`
