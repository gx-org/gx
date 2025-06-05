// Package trace is the API to use the builtin `trace` in GX.
package trace

import (
	"go/token"

	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

// Callback is called when the trace builtin is invoked
// in GX source code.
type Callback interface {
	Trace(fset *token.FileSet, call *ir.CallExpr, values []values.Value) error
}
