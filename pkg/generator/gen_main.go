package generator

import (
	"io"
	"path/filepath"
	"text/template"
)

const (
	// sdkImport is the operator-sdk import path.
	sdkImport = "github.com/coreos/operator-sdk/pkg/sdk"
)

// Main contains all the customized data needed to generate cmd/<projectName>/main.go for a new operator
// when pairing with mainTmpl template.
type Main struct {
	// imports
	OperatorSDKImport string
	APIImport         string
	StubImport        string

	// Service defintion in pkg/apis/..
	ServicePlural string
	Service       string
}

// renderMain generates the cmd/<projectName>/main.go file given a repo path ("github.com/coreos/play"), apiVersion ("v1alpha1"),
// api dir name ("play"), service ("PlayService"), and servicePlural ("PlayServicePlural").
//
// for example:
// renderMain(w, "github.com/coreos/play", "v1alpha1", "play", "PlayService", "PlayServicePlural" )
// output:
//
// package main
//
// import (
// 	"context"
//
// 	sdk "github.com/coreos/operator-sdk/pkg/sdk"
// 	api "github.com/coreos/play/pkg/apis/play/v1alpha1"
// 	stub "github.com/coreos/play/pkg/stub"
// )
//
// func main() {
// 	namespace := "default"
// 	sdk.Watch(api.PlayServicePlural, namespace, api.PlayService)
//  	sdk.Handle(&stub.Handler{})
//  	sdk.Run(context.TODO())
// }
//
func renderMain(w io.Writer, repo, version, apiDirName, service, servicePlural string) error {
	t := template.New("cmd/<projectName>/main.go")
	t, err := t.Parse(mainTmpl)
	if err != nil {
		return err
	}

	m := Main{
		OperatorSDKImport: sdkImport,
		APIImport:         filepath.Join(repo, apisDir, apiDirName, version),
		StubImport:        filepath.Join(repo, stubDir),
		ServicePlural:     servicePlural,
		Service:           service,
	}
	return t.Execute(w, m)
}
