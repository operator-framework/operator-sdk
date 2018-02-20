package generator

import (
	"io"
	"path/filepath"

	. "github.com/dave/jennifer/jen"
)

// sdkImport is the operator-sdk import path.
const (
	sdkImport = "github.com/coreos/operator-sdk/pkg/sdk"
	main      = "main"
)

// renderMain generates the cmd/main.go with a repo path ("github.com/coreos/play"), apiVersion ("v1alpha1"),
// projectName ("play"), service ("PlayService"), and servicePlural ("PlayServicePlural").
//
// for example:
//
// renderMain(w, "github.com/coreos/play", "v1alpha1", "play", "PlayService", "PlayServicePlural" )
//
// Output:
//
// package main
// import (
// 	context "context"
// 	sdk "github.com/coreos/operator-sdk/pkg/sdk"
// 	v1alpha1 "github.com/coreos/play/pkg/apis/play/v1alpha1"
// 	stub "github.com/coreos/play/pkg/stub"
// )

// func main() {
// 	namespace := "default"
// 	sdk.Watch(v1alpha1.PlayServicePlural, namespace, v1alpha1.PlayService)
// 	stub.Handle(&stub.Handler{})
// 	sdk.Run(context.TODO())
// }
//
func renderMain(w io.Writer, repo, apiVersion, projectName, service, servicePlural string) error {
	file := NewFile(main)

	apiImport := filepath.Join(repo, apisDir, projectName, apiVersion)
	stubImport := filepath.Join(repo, stubDir)
	// func main() {..}
	file.Func().Id("main").Params().Block(
		// namespace := "default"
		Id("namespace").Op(":=").Lit("default"),
		// sdk.Watch(...)
		Qual(sdkImport, "Watch").Call(
			Qual(apiImport, servicePlural),
			Id("namespace"),
			Qual(apiImport, service),
		),
		// sdk.Handle(...)
		Qual(stubImport, "Handle").Call(
			Op("&").Qual(stubImport, "Handler{}"),
		),
		// sdk.Run(...)
		Qual(sdkImport, "Run").Call(
			Qual("context", "TODO").Call(),
		),
	)

	return file.Render(w)
}
