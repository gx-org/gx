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
	macro    *cpevelements.Macro
	callSite elements.CallAt
	aux      *ordered.Map[string, *cpevelements.SyntheticFuncDecl]

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
	return cpevelements.NewSyntheticFunc(gradMacro{
		callSite: call,
		macro:    macro,
	}.newMacro(fn, wrt)), nil
}

func (m gradMacro) newMacro(fn *ir.FuncDecl, wrt string) *gradMacro {
	var n gradMacro = m
	n.fn = fn
	n.wrt = wrt
	n.aux = ordered.NewMap[string, *cpevelements.SyntheticFuncDecl]()
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
	return &ast.Ident{Name: fmt.Sprintf("__%s_%s_%s_%s", imp.Name(), funcName, name, m.wrt)}, true
}

func (m *gradMacro) BuildType() (*ir.FuncType, error) {
	return m.fn.FType, nil
}

func (m *gradMacro) BuildBody(fetcher ir.Fetcher) (*ir.BlockStmt, []*cpevelements.SyntheticFuncDecl, bool) {
	body, ok := m.gradBlock(fetcher, m.fn.Body, m.wrt)
	if !ok {
		return nil, nil, false
	}
	return body, slices.Collect(m.aux.Values()), true
}

func (m *gradMacro) gradBlock(fetcher ir.Fetcher, src *ir.BlockStmt, argName string) (*ir.BlockStmt, bool) {
	block := &ir.BlockStmt{List: make([]ir.Stmt, len(src.List))}
	for i, stmt := range src.List {
		var ok bool
		block.List[i], ok = m.gradStmt(fetcher, stmt, argName)
		if !ok {
			return nil, false
		}
	}
	return block, true
}

func (m *gradMacro) gradStmt(fetcher ir.Fetcher, src ir.Stmt, argName string) (ir.Stmt, bool) {
	switch srcT := src.(type) {
	case *ir.ReturnStmt:
		return m.gradReturnStmt(fetcher, srcT, argName)
	default:
		return nil, fetcher.Err().Appendf(src.Source(), "gradient of %T statement not supported", srcT)
	}
}

func (m *gradMacro) gradReturnStmt(fetcher ir.Fetcher, src *ir.ReturnStmt, argName string) (*ir.ReturnStmt, bool) {
	stmt := &ir.ReturnStmt{Results: make([]ir.Expr, len(src.Results))}
	for i, expr := range src.Results {
		res, ok := m.gradExpr(fetcher, expr, argName)
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
