package cgxasync_test

import (
	"testing"

	"github.com/gx-org/gx/golang/backend"
	"github.com/gx-org/gx/golang/binder/cgx/testing/async"
)

func TestAsyncCGXGoBackend(t *testing.T) {
	rtm := backend.New(async.NewBuilder())
	async.RunTestAsyncCGX(t, rtm, async.RunFunctionReturnFloat32)
}
