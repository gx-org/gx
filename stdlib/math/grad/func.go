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

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
)

func (m *gradMacro) gradFunc(fetcher ir.Fetcher, src *ir.FuncValExpr, wrt string) (*ir.FuncValExpr, bool) {
	switch fT := src.F.(type) {
	case *ir.FuncDecl:
		return m.gradFuncDecl(fetcher, src, fT, wrt)
	default:
		return nil, fetcher.Err().Appendf(src.Source(), "cannot compute the gradient of function %T", fT)
	}
}

func (m *gradMacro) gradFuncDecl(fetcher ir.Fetcher, src *ir.FuncValExpr, fn *ir.FuncDecl, wrt string) (*ir.FuncValExpr, bool) {
	name, ok := m.autoBuildSyntheticFuncName(fetcher, fn.Name())
	if !ok {
		return nil, false
	}
	cached, ok := m.aux.Load(name.Name)
	if ok {
		return &ir.FuncValExpr{
			X: &ir.ValueRef{Src: name, Stor: cached.F},
			T: src.T,
			F: cached.F,
		}, true
	}
	gradF := &ir.FuncDecl{
		FFile: m.fn.FFile,
		Src: &ast.FuncDecl{
			Name: name,
		},
	}
	var err error
	gradF.FType, err = m.BuildType()
	if err != nil {
		return nil, fetcher.Err().AppendAt(src.Source(), err)
	}
	grader := m.newMacro(fn, wrt)
	m.aux.Store(name.Name, &cpevelements.SyntheticFuncDecl{
		SyntheticFunc: cpevelements.NewSyntheticFunc(grader),
		F:             gradF,
	})
	return &ir.FuncValExpr{
		X: &ir.ValueRef{Src: name, Stor: gradF},
		T: src.T,
		F: gradF,
	}, true
}

func (m *gradMacro) gradCall(fetcher ir.Fetcher, src *ir.CallExpr, argName string) (*gradExprResult, bool) {
	if len(src.Args) == 0 {
		return zeroValueOf(fetcher, src.Source(), src.Type())
	}
	var gExpr *gradExprResult
	for i, argI := range src.Args {
		gradCallee, ok := m.gradFunc(fetcher, src.Callee, src.Callee.T.Params.Fields()[i].Name.Name)
		if !ok {
			return nil, false
		}
		gradArg, ok := m.gradExpr(fetcher, argI, argName)
		if !ok {
			return nil, false
		}
		gradI := buildMul(
			&gradExprResult{
				expr: &ir.CallExpr{
					Args:   src.Args,
					Callee: gradCallee,
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
