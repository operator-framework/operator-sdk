package stub

import (
	"context"

	sdkTypes "github.com/coreos/operator-sdk/pkg/sdk/types"
)

type handler struct {
	// custom data structure
}

func (h *handler) Handle(ctx context.Context, events []sdkTypes.Event) []sdkTypes.Action {
	// Filled out by user
	return nil
}
