package framework

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

type TestCtx struct {
	ID         string
	cleanUpFns []finalizerFn
}

type finalizerFn func() error

func (f *Framework) NewTestCtx(t *testing.T) TestCtx {
	// TestCtx is used among others for namespace names where '/' is forbidden
	prefix := strings.TrimPrefix(
		strings.Replace(
			strings.ToLower(t.Name()),
			"/",
			"-",
			-1,
		),
		"test",
	)

	id := prefix + "-" + strconv.FormatInt(time.Now().Unix(), 10)
	return TestCtx{
		ID: id,
	}
}

// GetObjID returns an ascending ID based on the length of cleanUpFns. It is
// based on the premise that every new object also appends a new finalizerFn on
// cleanUpFns. This can e.g. be used to create multiple namespaces in the same
// test context.
func (ctx *TestCtx) GetObjID() string {
	return ctx.ID + "-" + strconv.Itoa(len(ctx.cleanUpFns))
}

func (ctx *TestCtx) Cleanup(t *testing.T) {
	for i := len(ctx.cleanUpFns) - 1; i >= 0; i-- {
		err := ctx.cleanUpFns[i]()
		if err != nil {
			t.Errorf("A cleanup function failed with error: %v\n", err)
		}
	}
}

func (ctx *TestCtx) AddFinalizerFn(fn finalizerFn) {
	ctx.cleanUpFns = append(ctx.cleanUpFns, fn)
}
