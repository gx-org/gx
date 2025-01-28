// Package control provides runtime control flow via the GX `control` standard library.
package control

import (
	"go/ast"

	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

// Package description of the GX control package.
var Package = builtin.PackageBuilder{
	FullPath: "control",
	Builders: []builtin.Builder{builtin.BuildFunc(while{})},
}

type while struct {
	builtin.Func
}

func (f while) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[while]("While", impl.Control.While, pkg), nil
}

// Described in Go syntax, control.While has the signature:
//
//	func While[T any](state T, cond func(state T) bool, body func(state T) T) T
//
// The `cond` and `body` functions must accept the iteration state through identical parameters, and
// the body function must return a new iteration state. Last, the overall While() function takes an
// initial iteration state and returns the final state.
func (f while) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	boolType := ir.TypeFromKind(ir.BoolKind)
	stateType := call.Args[0].Type()
	condType := &ir.FuncType{ // `func(state T) bool`
		Src:     &ast.FuncType{Func: call.Source().Pos()},
		Params:  builtins.Fields(stateType),
		Results: builtins.Fields(boolType),
	}
	bodyType := &ir.FuncType{ // `func(state T) T`
		Src:     &ast.FuncType{Func: call.Source().Pos()},
		Params:  builtins.Fields(stateType),
		Results: builtins.Fields(stateType),
	}
	return &ir.FuncType{
		Src:     &ast.FuncType{Func: call.Source().Pos()},
		Params:  builtins.Fields(stateType, condType, bodyType),
		Results: builtins.Fields(stateType),
	}, nil
}
