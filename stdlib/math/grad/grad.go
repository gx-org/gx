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
	"fmt"
	"go/ast"
	"slices"

	"github.com/gx-org/gx/base/ordered"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
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
	macro    *cpevelements.Macro
	callSite elements.CallAt
	aux      *ordered.Map[string, *cpevelements.SyntheticFuncDecl]

	fn  *ir.FuncDecl
	wrt *ir.FieldStorage
}

func findParamStorage(file *ir.File, src ir.SourceNode, fn ir.Func, name string) (*ir.FieldStorage, error) {
	field := fn.FuncType().Params.FindField(name)
	if field == nil {
		return nil, fmterr.Errorf(file.FileSet(), src.Source(), "no parameter named %s in %s", name, fn.Name())
	}
	return field.Storage(), nil
}

// FuncGrad computes the gradient of a function.
func FuncGrad(call elements.CallAt, macro *cpevelements.Macro, args []ir.Element) (*cpevelements.SyntheticFunc, error) {
	fn, err := interp.FuncDeclFromElement(args[0])
	if err != nil {
		return nil, err
	}
	wrtS, err := elements.StringFromElement(args[1])
	if err != nil {
		return nil, err
	}
	wrtF, err := findParamStorage(call.File(), call.Node(), fn, wrtS)
	if err != nil {
		return nil, err
	}
	return cpevelements.NewSyntheticFunc(gradMacro{
		callSite: call,
		macro:    macro,
	}.newMacro(fn, wrtF)), nil
}

func (m gradMacro) newMacro(fn *ir.FuncDecl, wrt *ir.FieldStorage) *gradMacro {
	var n gradMacro = m
	n.fn = fn
	n.wrt = wrt
	return &n
}

func (m gradMacro) clone() *gradMacro {
	return m.newMacro(m.fn, m.wrt)
}

func (m *gradMacro) autoBuildSyntheticFuncName(fetcher ir.Fetcher, name string) (*ast.Ident, bool) {
	callee := m.callSite.Node().Callee
	funcName := callee.Name()
	macroPackage := callee.F.(*ir.Macro).File().Package
	imp := m.callSite.File().FindImport(macroPackage.FullName())
	if imp == nil {
		return nil, fetcher.Err().AppendInternalf(callee.Source(), "cannot find import name %s", macroPackage.FullName())
	}
	return &ast.Ident{Name: fmt.Sprintf("__%s_%s_%s_%s", imp.Name(), funcName, name, m.wrt.Field.Name)}, true
}

func (m *gradMacro) BuildType() (*ir.FuncType, error) {
	return m.fn.FType, nil
}

func (m *gradMacro) BuildBody(fetcher ir.Fetcher) (*ir.BlockStmt, []*cpevelements.SyntheticFuncDecl, bool) {
	m.aux = ordered.NewMap[string, *cpevelements.SyntheticFuncDecl]()
	body, ok := m.gradBlock(fetcher, m.fn.Body)
	if !ok {
		return nil, nil, false
	}
	return body, slices.Collect(m.aux.Values()), true
}

func (m *gradMacro) gradBlock(fetcher ir.Fetcher, src *ir.BlockStmt) (*ir.BlockStmt, bool) {
	var block []ir.Stmt
	for _, stmt := range src.List {
		var ok bool
		stmts, ok := m.gradStmt(fetcher, stmt)
		if !ok {
			return nil, false
		}
		block = append(block, stmts...)
	}
	return &ir.BlockStmt{
		Src:  src.Src,
		List: block,
	}, true
}

func (m *gradMacro) gradStmt(fetcher ir.Fetcher, src ir.Stmt) ([]ir.Stmt, bool) {
	switch srcT := src.(type) {
	case *ir.ReturnStmt:
		ret, ok := m.gradReturnStmt(fetcher, srcT)
		return []ir.Stmt{ret}, ok
	case *ir.AssignExprStmt:
		return m.gradAssignExprStmt(fetcher, srcT)
	default:
		return nil, fetcher.Err().Appendf(src.Source(), "gradient of %T statement not supported", srcT)
	}
}

func (m *gradMacro) gradReturnStmt(fetcher ir.Fetcher, src *ir.ReturnStmt) (*ir.ReturnStmt, bool) {
	stmt := &ir.ReturnStmt{Results: make([]ir.Expr, len(src.Results))}
	for i, expr := range src.Results {
		res, ok := m.gradExpr(fetcher, expr)
		if !ok {
			return nil, false
		}
		if res != nil {
			// The expression depends on arg: nothing left to do.
			stmt.Results[i] = res.expr
			continue
		}
		// The expression does not depend on arg: replace it with a zero value.
		res, ok = zeroValueOf(fetcher, expr.Source(), expr.Type())
		if !ok {
			return nil, false
		}
		stmt.Results[i] = res.expr
	}
	return stmt, true
}

func gradIdent(src *ast.Ident) *ast.Ident {
	return &ast.Ident{
		NamePos: src.NamePos,
		Name:    "__grad_" + src.Name,
	}
}

func (m *gradMacro) gradStorage(fetcher ir.Fetcher, src ir.SourceNode, store ir.Storage) (ir.Storage, bool) {
	switch storeT := store.(type) {
	case *ir.LocalVarStorage:
		return &ir.LocalVarStorage{
			Src: gradIdent(storeT.Src),
			Typ: storeT.Typ,
		}, true
	default:
		return nil, fetcher.Err().Appendf(store.Source(), "gradient of %T storage not supported", storeT)
	}
}

func (m *gradMacro) gradAssignExprStmt(fetcher ir.Fetcher, src *ir.AssignExprStmt) ([]ir.Stmt, bool) {
	gradStmt := &ir.AssignExprStmt{
		Src:  src.Src,
		List: make([]*ir.AssignExpr, len(src.List)),
	}
	for i, aexpr := range src.List {
		gExpr, ok := m.gradExpr(fetcher, aexpr.X)
		if !ok {
			return nil, false
		}
		gStorage, ok := m.gradStorage(fetcher, aexpr.X, aexpr.Storage)
		if !ok {
			return nil, false
		}
		gradStmt.List[i] = &ir.AssignExpr{
			Storage: gStorage,
			X:       gExpr.expr,
		}
	}
	return []ir.Stmt{src, gradStmt}, true
}
