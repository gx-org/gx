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
	"strconv"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/builtin"
)

//go:embed *.gx
var fs embed.FS

// Package description of the GX meta/grad package.
var Package = builtin.PackageBuilder{
	FullPath: "math/grad",
	Builders: []builtin.Builder{
		builtin.ParseSource(&fs),
		builtin.RegisterMacro("Func", computeGrad),
	},
}

func computeGrad(fetcher ir.Fetcher, target *ir.FuncDecl, args []ir.AssignableExpr) (*ir.FuncDecl, bool) {
	ext := *target
	if len(args) != 2 {
		return &ext, fetcher.Err().Appendf(target.Src, "%d argument(s) given to grad.Func, want 2", len(args))
	}
	funcRef, ok := args[0].(*ir.ValueRef)
	if !ok {
		return &ext, fetcher.Err().Appendf(target.Src, "%T not supported, want a function name", args[0])
	}
	funcExpr, ok := fetcher.BuildExpr(funcRef.Src)
	if !ok {
		return &ext, false
	}
	funcSrc, ok := funcExpr.(*ir.FuncDecl)
	if !ok {
		return &ext, fetcher.Err().Appendf(target.Src, "cannot compute the gradient of %s, want a GX declared function", funcExpr.Type().String())
	}
	argName, ok := args[1].(*ir.StringLiteral)
	if !ok {
		return &ext, fetcher.Err().Appendf(target.Src, "%s not supported, want a string literal", args[1].Type().String())
	}
	argValue, err := strconv.Unquote(argName.Src.Value)
	if err != nil {
		return &ext, fetcher.Err().Appendf(target.Src, "%v", err)
	}
	ext.FType = funcSrc.FType
	ext.Body, err = gradBlock(fetcher, target, funcSrc.Body, argValue)
	if err != nil {
		return &ext, fetcher.Err().AppendAt(target.Src, err)
	}
	return &ext, true
}

func gradBlock(fetcher ir.Fetcher, target *ir.FuncDecl, src *ir.BlockStmt, argName string) (*ir.BlockStmt, error) {
	block := &ir.BlockStmt{List: make([]ir.Stmt, len(src.List))}
	for i, stmt := range src.List {
		var err error
		block.List[i], err = gradStmt(fetcher, stmt, argName)
		if err != nil {
			return nil, err
		}
	}
	return block, nil
}

func gradStmt(fetcher ir.Fetcher, src ir.Stmt, argName string) (ir.Stmt, error) {
	switch srcT := src.(type) {
	case *ir.ReturnStmt:
		return gradReturnStmt(fetcher, srcT, argName)
	default:
		return nil, fmterr.Errorf(fetcher.File().FileSet(), src.Source(), "gradient of %T statement not supported", srcT)
	}
}

func gradReturnStmt(fetcher ir.Fetcher, src *ir.ReturnStmt, argName string) (*ir.ReturnStmt, error) {
	stmt := &ir.ReturnStmt{Results: make([]ir.Expr, len(src.Results))}
	for i, expr := range src.Results {
		res, err := gradExpr(fetcher, expr, argName)
		if err != nil {
			return nil, err
		}
		if res != nil {
			// The expression depends on arg: nothing left to do.
			stmt.Results[i] = res
			continue
		}
		// The expression does not depend on arg: replace it with a zero value.
		res, err = zeroValueOf(fetcher, expr.Source(), expr.Type())
		if err != nil {
			return nil, err
		}
		stmt.Results[i] = res
	}
	return stmt, nil
}

func gradExpr(fetcher ir.Fetcher, src ir.Expr, argName string) (ir.AssignableExpr, error) {
	switch srcT := src.(type) {
	case *ir.ArrayLitExpr:
		return gradArrayLitExpr(fetcher, srcT, argName)
	case *ir.NumberCastExpr:
		return gradNumberCastExpr(fetcher, srcT, argName)
	case *ir.ValueRef:
		return gradValueRef(fetcher, srcT, argName)

	case *ir.NumberFloat:
		return nil, nil
	default:
		return nil, fmterr.Errorf(fetcher.File().FileSet(), src.Source(), "gradient of %T expression not supported", srcT)
	}
}

func gradNumberCastExpr(fetcher ir.Fetcher, src *ir.NumberCastExpr, argName string) (ir.AssignableExpr, error) {
	gExpr, err := gradExpr(fetcher, src.X, argName)
	if err != nil {
		return nil, err
	}
	if gExpr == nil {
		return nil, nil
	}
	return &ir.NumberCastExpr{Typ: src.Typ, X: gExpr}, nil
}

func gradValueRef(fetcher ir.Fetcher, src *ir.ValueRef, argName string) (ir.AssignableExpr, error) {
	if src.Src.Name != argName {
		// The ident does not correspond to the variable
		// for which we are differentiating: return zero.
		return nil, nil
	}
	return oneValueOf(fetcher, src.Source(), src.Type())
}

func gradArrayLitExpr(fetcher ir.Fetcher, src *ir.ArrayLitExpr, argName string) (ir.AssignableExpr, error) {
	allZero := true
	gValues := make([]ir.AssignableExpr, len(src.Values()))
	for i, expr := range src.Values() {
		var err error
		gValues[i], err = gradExpr(fetcher, expr, argName)
		if err != nil {
			return nil, err
		}
		if gValues[i] != nil {
			allZero = false
			continue
		}
		gValues[i], err = zeroValueOf(fetcher, expr.Source(), expr.Type())
		if err != nil {
			return nil, err
		}
	}
	if allZero {
		return nil, nil
	}
	return src.NewFromValues(gValues), nil
}

func zeroValueOf(fetcher ir.Fetcher, node ast.Node, typ ir.Type) (ir.AssignableExpr, error) {
	zero, ok := typ.(ir.Zeroer)
	if !ok {
		return nil, fmterr.Errorf(fetcher.File().FileSet(), node, "zero expression of %T not supported", typ)
	}
	return zero.Zero(), nil
}

var one = &ir.NumberInt{Src: &ast.BasicLit{Value: "1"}, Val: big.NewInt(1)}

func oneValueOf(fetcher ir.Fetcher, node ast.Node, typ ir.Type) (ir.AssignableExpr, error) {
	zero, err := zeroValueOf(fetcher, node, typ)
	if err != nil {
		return nil, err
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
	}, nil
}
