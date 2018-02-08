package sdk

import "context"

type Operator interface {
	// Start starts the operator.
	Start(Context context.Context) error
}
