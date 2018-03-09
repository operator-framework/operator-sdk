package main

import (
	"context"

	sdk "github.com/coreos/operator-sdk/pkg/sdk"
	api "github.com/coreos/play/pkg/apis/play/v1alpha1"
	stub "github.com/coreos/play/pkg/apis/stub"
)

func main() {
	namespace := "default"
	sdk.Watch(api.PlayServicePlural, namespace, api.PlayService)
	sdk.Handle(&stub.Handler{})
	sdk.Run(context.TODO())
}
