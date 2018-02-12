package dispatcher

import (
	"context"

	sdkTypes "github.com/coreos/operator-sdk/pkg/sdk/types"
)

type Dispatcher interface {
	Run(ctx context.Context) error
}

type dispatcher struct {
	eventChans []<-chan *sdkTypes.Event
}

func New(eventChans []<-chan *sdkTypes.Event, handler sdkTypes.Handler) *dispatcher {
	panic("TODO")
}

// Run runs the dispatcher which collects events from all the event channels and sends it to the handler
func (d *dispatcher) Run(ctx context.Context) error {
	panic("TODO")
}
