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

package grad

import (
	"go/ast"

	"github.com/gx-org/gx/build/ir/annotations"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
)

func (m *gradMacro) gradFunc(fetcher ir.Fetcher, src *ir.FuncValExpr, wrt string) (ast.Expr, bool) {
	ann := annotations.Get[setAnnotation](src.F, m.set)
	if ann != nil {
		return gradFromAnnotation(fetcher, src.F.(ir.Func), ann, wrt)
	}
	switch fT := src.F.(type) {
	case *ir.FuncDecl:
		return gradFuncDecl(fetcher, m, src, fT, wrt)
	default:
		return nil, fetcher.Err().Appendf(src.Source(), "cannot compute the gradient of function %s", fT.ShortString())
	}
}

func gradFuncDecl(fetcher ir.Fetcher, parent *gradMacro, src *ir.FuncValExpr, fn *ir.FuncDecl, wrt string) (ast.Expr, bool) {
	// Build the call to the gradient of a function.
	wrtF, err := findParamStorage(fetcher.File(), src, fn, wrt)
	if err != nil {
		return nil, fetcher.Err().Append(err)
	}
	grader := parent.newMacro(fn, wrtF)
	synthName, ok := grader.syntheticFuncName(fetcher, fn)
	if !ok {
		return nil, false
	}
	ident := &ast.Ident{Name: synthName}
	// Check if we need to build a new synthetic function.
	if fetcher.IsDefined(synthName) {
		// The function has already been built before.
		return ident, true
	}
	if _, ok := grader.aux.Load(synthName); ok {
		// The function has already been registered as a new auxiliary functions.
		return ident, true
	}
	// No function already exists. Prepare to return it as a new auxiliary function.
	gradF, ok := grader.BuildDecl(nil)
	if !ok {
		return nil, false
	}
	gradF.Name = ident
	grader.aux.Store(synthName, &cpevelements.SyntheticFuncDecl{
		Builder: grader,
		F:       gradF,
	})
	return ident, true
}

func astExprs(exprs []ir.AssignableExpr) []ast.Expr {
	as := make([]ast.Expr, len(exprs))
	for i, expr := range exprs {
		as[i] = expr.Source().(ast.Expr)
	}
	return as
}

func (m *stmtGrader) gradCall(src *ir.CallExpr) (*gradExprResult, bool) {
	if len(src.Args) == 0 {
		return zeroValueOf(src.Source()), true
	}
	var gExpr *gradExprResult
	ge := m.newExprGrader(false)
	for i, argI := range src.Args {
		gradCallee, ok := m.macro.gradFunc(m.fetcher, src.Callee, src.Callee.T.Params.Fields()[i].Name.Name)
		if !ok {
			return nil, false
		}
		gradArg, ok := ge.gradExpr(argI)
		if !ok {
			return nil, false
		}
		gradI := buildMul(
			&gradExprResult{
				expr: &ast.CallExpr{
					Fun:  gradCallee,
					Args: astExprs(src.Args),
				},
			},
			gradArg,
		)
		if gExpr == nil {
			gExpr = gradI
			continue
		}
		gExpr = buildAdd(gExpr, gradI)
	}
	return gExpr, true
}
