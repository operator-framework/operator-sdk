package controller

import (
	"github.com/operator-framework/operator-sdk/test/test-framework/pkg/controller/memcached"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, memcached.Add)
}
