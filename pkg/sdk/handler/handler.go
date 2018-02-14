package handler

import (
	sdkTypes "github.com/coreos/operator-sdk/pkg/sdk/types"
)

// Handler reacts to events and outputs actions.
// If any intended action failed, the event would be re-triggered.
// For actions done before the failed action, there is no rollback.
type Handler interface {
	Handle(sdkTypes.Context, sdkTypes.Event) []sdkTypes.Action
}

var (
	// RegisteredHandler is the user registered handler set by sdk.Handle()
	RegisteredHandler Handler
)
