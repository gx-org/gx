// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package grad implement GX functions to compute the gradient of GX functions.
package grad

import (
	"embed"
	"go/ast"
	"go/token"
	"math/big"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/stdlib/builtin"
)

//go:embed *.gx
var fs embed.FS

// Package description of the GX meta/grad package.
var Package = builtin.PackageBuilder{
	FullPath: "math/grad",
	Builders: []builtin.Builder{
		builtin.ParseSource(&fs),
		builtin.RegisterMacro("Func", FuncGrad),
	},
}

type gradMacro struct {
	fn  *ir.FuncDecl
	wrt string
}

// FuncGrad computes the gradient of a function.
func FuncGrad(call elements.CallAt, macro *cpevelements.Macro, args []elements.Element) (*cpevelements.SyntheticFunc, error) {
	fn, err := elements.FuncDeclFromElement(args[0])
	if err != nil {
		return nil, err
	}
	wrt, err := elements.StringFromElement(args[1])
	if err != nil {
		return nil, err
	}
	return cpevelements.NewSyntheticFunc(macro, &gradMacro{
		fn:  fn,
		wrt: wrt,
	}), nil
}

func (m *gradMacro) BuildType() (*ir.FuncType, error) {
	return m.fn.FType, nil
}

func (m *gradMacro) BuildBody(fetcher ir.Fetcher) (*ir.BlockStmt, bool) {
	return gradBlock(fetcher, m.fn.Body, m.wrt)
}

func gradBlock(fetcher ir.Fetcher, src *ir.BlockStmt, argName string) (*ir.BlockStmt, bool) {
	block := &ir.BlockStmt{List: make([]ir.Stmt, len(src.List))}
	for i, stmt := range src.List {
		var ok bool
		block.List[i], ok = gradStmt(fetcher, stmt, argName)
		if !ok {
			return nil, false
		}
	}
	return block, true
}

func gradStmt(fetcher ir.Fetcher, src ir.Stmt, argName string) (ir.Stmt, bool) {
	switch srcT := src.(type) {
	case *ir.ReturnStmt:
		return gradReturnStmt(fetcher, srcT, argName)
	default:
		return nil, fetcher.Err().Appendf(src.Source(), "gradient of %T statement not supported", srcT)
	}
}

func gradReturnStmt(fetcher ir.Fetcher, src *ir.ReturnStmt, argName string) (*ir.ReturnStmt, bool) {
	stmt := &ir.ReturnStmt{Results: make([]ir.Expr, len(src.Results))}
	for i, expr := range src.Results {
		res, ok := gradExpr(fetcher, expr, argName)
		if !ok {
			return nil, false
		}
		if res != nil {
			// The expression depends on arg: nothing left to do.
			stmt.Results[i] = res
			continue
		}
		// The expression does not depend on arg: replace it with a zero value.
		res, ok = zeroValueOf(fetcher, expr.Source(), expr.Type())
		if !ok {
			return nil, false
		}
		stmt.Results[i] = res
	}
	return stmt, true
}

func gradExpr(fetcher ir.Fetcher, src ir.Expr, argName string) (ir.AssignableExpr, bool) {
	switch srcT := src.(type) {
	case *ir.ArrayLitExpr:
		return gradArrayLitExpr(fetcher, srcT, argName)
	case *ir.NumberCastExpr:
		return gradNumberCastExpr(fetcher, srcT, argName)
	case *ir.ValueRef:
		return gradValueRef(fetcher, srcT, argName)

	case *ir.NumberFloat:
		return nil, true
	default:
		return nil, fetcher.Err().Appendf(src.Source(), "gradient of %T expression not supported", srcT)
	}
}

func gradNumberCastExpr(fetcher ir.Fetcher, src *ir.NumberCastExpr, argName string) (ir.AssignableExpr, bool) {
	gExpr, ok := gradExpr(fetcher, src.X, argName)
	if !ok {
		return nil, ok
	}
	if gExpr == nil {
		return nil, true
	}
	return &ir.NumberCastExpr{Typ: src.Typ, X: gExpr}, true
}

func gradValueRef(fetcher ir.Fetcher, src *ir.ValueRef, argName string) (ir.AssignableExpr, bool) {
	if src.Src.Name != argName {
		// The ident does not correspond to the variable
		// for which we are differentiating: return zero.
		return nil, true
	}
	return oneValueOf(fetcher, src.Source(), src.Type())
}

func gradArrayLitExpr(fetcher ir.Fetcher, src *ir.ArrayLitExpr, argName string) (ir.AssignableExpr, bool) {
	allZero := true
	gValues := make([]ir.AssignableExpr, len(src.Values()))
	for i, expr := range src.Values() {
		var ok bool
		gValues[i], ok = gradExpr(fetcher, expr, argName)
		if !ok {
			return nil, false
		}
		if gValues[i] != nil {
			allZero = false
			continue
		}
		gValues[i], ok = zeroValueOf(fetcher, expr.Source(), expr.Type())
		if !ok {
			return nil, false
		}
	}
	if allZero {
		return nil, true
	}
	return src.NewFromValues(gValues), true
}

func zeroValueOf(fetcher ir.Fetcher, node ast.Node, typ ir.Type) (ir.AssignableExpr, bool) {
	zero, ok := typ.(ir.Zeroer)
	if !ok {
		return nil, fetcher.Err().Appendf(node, "zero expression of %T not supported", typ)
	}
	return zero.Zero(), true
}

var one = &ir.NumberInt{Src: &ast.BasicLit{Value: "1"}, Val: big.NewInt(1)}

func oneValueOf(fetcher ir.Fetcher, node ast.Node, typ ir.Type) (ir.AssignableExpr, bool) {
	zero, ok := zeroValueOf(fetcher, node, typ)
	if !ok {
		return nil, false
	}
	return &ir.ParenExpr{
		X: &ir.BinaryExpr{
			Src: &ast.BinaryExpr{Op: token.ADD},
			X:   zero,
			Y: &ir.NumberCastExpr{
				X:   one,
				Typ: ir.TypeFromKind(typ.Kind()),
			},
			Typ: typ,
		},
	}, true
}
