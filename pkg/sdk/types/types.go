package types

import "context"

type Object interface{}

type Event struct {
	Object      Object
	ObjectExist bool
}

type Action struct {
	Object Object
	Actor  string
}

type Handler interface {
	Handle(ctx context.Context, events []Event) []Action
}
