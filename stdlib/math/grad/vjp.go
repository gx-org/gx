// Copyright 2025 Google LLC
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
	"fmt"
	"go/ast"
	"slices"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/base/ordered"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
)

type vjpMacro struct {
	cpevelements.CoreMacroElement
	callSite elements.CallAt
	aux      *ordered.Map[string, *cpevelements.SyntheticFuncDecl]

	fn  ir.PkgFunc
	wrt withRespectTo
}

var _ cpevelements.FuncASTBuilder = (*vjpMacro)(nil)

// VJP computes the vector-jacobian product of a function.
func VJP(call elements.CallAt, macro *cpevelements.Macro, args []ir.Element) (cpevelements.MacroElement, error) {
	fn, err := interp.PkgFuncFromElement(args[1])
	if err != nil {
		return nil, err
	}
	fnT, ok := fn.(ir.PkgFunc)
	if !ok {
		return nil, errors.Errorf("cannot compute the gradient of function %T", fn)
	}
	wrtS, err := elements.StringFromElement(args[2])
	if err != nil {
		return nil, err
	}
	wrtF, err := findParamStorage(call.File(), call.Node(), fn, wrtS)
	if err != nil {
		return nil, err
	}
	return vjpMacro{
		CoreMacroElement: cpevelements.CoreMacroElement{Mac: macro},
		callSite:         call,
	}.newMacro(fnT, wrtF), nil
}

func (m vjpMacro) newMacro(fn ir.PkgFunc, wrt withRespectTo) *vjpMacro {
	var n vjpMacro = m
	n.fn = fn
	n.wrt = wrt
	return &n
}

func argAt(i int) *ast.Ident {
	return &ast.Ident{
		Name: fmt.Sprintf("__bck%d", i),
	}
}

func nameResultFields(results *ast.FieldList) *ast.FieldList {
	all := ast.FieldList{
		List: make([]*ast.Field, len(results.List)),
	}
	var num int
	for iField, field := range results.List {
		named := *field
		if len(named.Names) == 0 {
			named.Names = []*ast.Ident{&ast.Ident{}}
		}
		for iIdent := range named.Names {
			named.Names[iIdent] = argAt(num)
			num++
		}
		all.List[iField] = &named
	}
	return &all
}

func concatFieldList(lists ...*ast.FieldList) *ast.FieldList {
	all := ast.FieldList{}
	for _, list := range lists {
		all.List = append(all.List, list.List...)
	}
	return &all
}

func (m *vjpMacro) BuildDecl() (*ast.FuncDecl, bool) {
	fType := m.fn.FuncType()
	namedResultFields := nameResultFields(fType.Src.Results)
	fDecl := &ast.FuncDecl{Type: &ast.FuncType{
		// Same as the original function.
		TypeParams: fType.Src.TypeParams,
		// Same parameters than the original function plus all the output values.
		Params: concatFieldList(fType.Src.Params, namedResultFields),
		// Return the result of the original function as well as the gradient.
		Results: concatFieldList(
			fType.Src.Results,
			&ast.FieldList{
				List: []*ast.Field{&ast.Field{
					Type: m.wrt.fieldType(),
				}},
			}),
	}}
	recv := fType.Receiver
	if recv != nil {
		fDecl.Recv = recv.Src
	}
	return fDecl, true
}

func (m *vjpMacro) buildBodyFromSetAnnotation(fetcher ir.Fetcher, fn ir.Func, ann *setAnnotation) (*ast.BlockStmt, []*cpevelements.SyntheticFuncDecl, bool) {
	return nil, nil, fetcher.Err().Appendf(fn.Source(), "function with set directives not supported yet")
}

func (m *vjpMacro) BuildBody(fetcher ir.Fetcher, fn ir.Func) (*ast.BlockStmt, []*cpevelements.SyntheticFuncDecl, bool) {
	if ann := findSetAnnotation(m.fn); ann != nil {
		return m.buildBodyFromSetAnnotation(fetcher, fn, ann)
	}
	fnWithBody, ok := m.fn.(*ir.FuncDecl)
	if !ok {
		return nil, nil, fetcher.Err().Appendf(fn.Source(), "function has no body")
	}
	m.aux = ordered.NewMap[string, *cpevelements.SyntheticFuncDecl]()
	sg := m.newStmt(fetcher, nil)
	fType := m.fn.FuncType()
	sg.registerFieldNames(fType.Receiver)
	sg.registerFieldNames(fType.Params)
	sg.registerFieldNames(fType.Results)
	body, ok := sg.processBlock(fnWithBody.Body)
	if !ok {
		return nil, nil, false
	}
	return body, slices.Collect(m.aux.Values()), true
}
