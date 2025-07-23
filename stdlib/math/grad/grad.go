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
	"go/token"
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

func (m *gradMacro) syntheticFuncName(fetcher ir.Fetcher, fn ir.Func) (string, bool) {
	callee := m.callSite.Node().Callee
	funcName := callee.Name()
	macroPackage := callee.F.(*ir.Macro).File().Package
	imp := m.callSite.File().FindImport(macroPackage.FullName())
	if imp == nil {
		return "", fetcher.Err().AppendInternalf(callee.Source(), "cannot find import name %s", macroPackage.FullName())
	}
	return fmt.Sprintf("__%s_%s_%s_%s", imp.Name(), funcName, fn.Name(), m.wrt.Field.Name), true
}

func (m *gradMacro) BuildType() (*ast.FuncDecl, error) {
	return &ast.FuncDecl{Type: m.fn.Src.Type, Recv: m.fn.Src.Recv}, nil
}

func (m *gradMacro) BuildBody(fetcher ir.Fetcher) (*ast.BlockStmt, []*cpevelements.SyntheticFuncDecl, bool) {
	m.aux = ordered.NewMap[string, *cpevelements.SyntheticFuncDecl]()
	body, ok := m.gradBlock(fetcher, m.fn.Body)
	if !ok {
		return nil, nil, false
	}
	return body, slices.Collect(m.aux.Values()), true
}

func (m *gradMacro) gradBlock(fetcher ir.Fetcher, src *ir.BlockStmt) (*ast.BlockStmt, bool) {
	var block []ast.Stmt
	for _, stmt := range src.List {
		var ok bool
		stmts, ok := m.gradStmt(fetcher, stmt)
		if !ok {
			return nil, false
		}
		block = append(block, stmts...)
	}
	return &ast.BlockStmt{
		List: block,
	}, true
}

func (m *gradMacro) gradStmt(fetcher ir.Fetcher, src ir.Stmt) ([]ast.Stmt, bool) {
	switch srcT := src.(type) {
	case *ir.ReturnStmt:
		ret, ok := m.gradReturnStmt(fetcher, srcT)
		return []ast.Stmt{ret}, ok
	case *ir.AssignExprStmt:
		return m.gradAssignExprStmt(fetcher, srcT)
	default:
		return nil, fetcher.Err().Appendf(src.Source(), "gradient of %T statement not supported", srcT)
	}
}

func (m *gradMacro) gradReturnStmt(fetcher ir.Fetcher, src *ir.ReturnStmt) (*ast.ReturnStmt, bool) {
	stmt := &ast.ReturnStmt{Results: make([]ast.Expr, len(src.Results))}
	ge := m.newExprGrader(fetcher, false)
	for i, expr := range src.Results {
		res, ok := ge.gradExpr(expr)
		if !ok {
			return nil, false
		}
		if res != nil {
			// The expression depends on arg: nothing left to do.
			stmt.Results[i] = res.expr
			continue
		}
		// The expression does not depend on arg: replace it with a zero value.
		res = zeroValueOf(expr.Source())
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

func (m *gradMacro) gradAssignExprStmt(fetcher ir.Fetcher, src *ir.AssignExprStmt) ([]ast.Stmt, bool) {
	gradStmt := &ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: make([]ast.Expr, len(src.List)),
		Rhs: make([]ast.Expr, len(src.List)),
	}
	ge := m.newExprGrader(fetcher, true)
	for i, aexpr := range src.List {
		gExpr, ok := ge.gradExpr(aexpr.X)
		if !ok {
			return nil, false
		}
		gradStmt.Lhs[i] = gradIdent(aexpr.Storage.NameDef())
		gradStmt.Rhs[i] = gExpr.expr
	}
	return []ast.Stmt{src.Src, gradStmt}, true
}
