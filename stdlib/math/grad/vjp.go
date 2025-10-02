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
	"go/ast"
	"slices"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/base/ordered"
	"github.com/gx-org/gx/base/uname"
	"github.com/gx-org/gx/build/ir/annotations"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
)

type (
	vjpParam struct {
		field    *ir.Field
		wrt      withRespectTo
		vjpFType *ast.FuncType
	}

	vjpMacro struct {
		cpevelements.CoreMacroElement
		callSite    elements.CallAt
		set         *ir.Macro
		aux         *ordered.Map[string, *cpevelements.SyntheticFuncDecl]
		exprToName  map[ir.Expr]forwardValues
		resultNames []string

		unames  *uname.Unique
		fwdRoot *uname.Root
		bckRoot *uname.Root

		fn     ir.PkgFunc
		params []vjpParam
	}
)

var _ cpevelements.FuncASTBuilder = (*vjpMacro)(nil)

// VJP computes the vector-jacobian product of a function.
func VJP(call elements.CallAt, macro *cpevelements.Macro, args []ir.Element) (cpevelements.MacroElement, error) {
	fn, err := interp.PkgFuncFromElement(args[0])
	if err != nil {
		return nil, err
	}
	fnT, ok := fn.(ir.PkgFunc)
	if !ok {
		return nil, errors.Errorf("cannot compute the gradient of function %T", fn)
	}
	unames := uname.New()
	return vjpMacro{
		CoreMacroElement: macro.Element(call),
		unames:           unames,
		callSite:         call,
		set:              setMacro(macro),
		fwdRoot:          unames.Root("fwd"),
		bckRoot:          unames.Root("bck"),
	}.newMacro(fnT), nil
}

func (m vjpMacro) newMacro(fn ir.PkgFunc) *vjpMacro {
	m.fn = fn
	fType := m.fn.FuncType()
	params := fType.Params.Fields()
	m.params = make([]vjpParam, len(params))
	var namedResultFields *ast.FieldList
	m.resultNames, namedResultFields = m.nameResultFields(fType.Results.Src)
	for i, param := range params {
		wrt := newWRT(param)
		m.params[i] = vjpParam{
			field:    param,
			wrt:      newWRT(param),
			vjpFType: m.buildBackwardSignature(fType, wrt, namedResultFields),
		}
	}
	return &m
}

func (m *vjpMacro) start() {
	m.exprToName = make(map[ir.Expr]forwardValues)
}

func (m *vjpMacro) nameResultFields(results *ast.FieldList) ([]string, *ast.FieldList) {
	all := ast.FieldList{
		List: make([]*ast.Field, len(results.List)),
	}
	var names []string
	for iField, field := range results.List {
		named := *field
		if len(named.Names) == 0 {
			named.Names = []*ast.Ident{&ast.Ident{}}
		}
		for iName, name := range named.Names {
			retName := uname.DefaultIdent(name, "res")
			named.Names[iName] = m.unames.Ident(retName)
			names = append(names, retName.Name)
		}
		all.List[iField] = &named
	}
	return names, &all
}

func concatFieldList(lists ...*ast.FieldList) *ast.FieldList {
	all := ast.FieldList{}
	for _, list := range lists {
		all.List = append(all.List, list.List...)
	}
	return &all
}

func (m *vjpMacro) buildBackwardSignature(fType *ir.FuncType, wrt withRespectTo, namedResultFields *ast.FieldList) *ast.FuncType {
	return &ast.FuncType{
		// Same as the original function.
		TypeParams: fType.Src.TypeParams,
		// Gradient coming from the output values of the function.
		Params: namedResultFields,
		// Return the gradient.
		Results: &ast.FieldList{
			List: []*ast.Field{&ast.Field{
				Type: wrt.fieldType(),
			}},
		},
	}
}

func (m *vjpMacro) buildType() *ast.FuncType {
	fType := m.fn.FuncType()
	vjpFuncs := make([]*ast.Field, len(m.params))
	for i, param := range m.params {
		vjpFuncs[i] = &ast.Field{
			Type: param.vjpFType,
		}
	}
	vjpType := &ast.FuncType{
		// Same as the original function.
		TypeParams: fType.Src.TypeParams,
		// Same parameters than the original function.
		Params: fType.Src.Params,
		// Return the result of the original function as well as
		// a backward function to compute the gradient.
		Results: concatFieldList(
			fType.Src.Results,
			&ast.FieldList{List: vjpFuncs},
		),
	}
	return vjpType
}

func (m *vjpMacro) BuildDecl(ir.PkgFunc) (*ast.FuncDecl, bool) {
	m.start()
	fDecl := &ast.FuncDecl{Type: m.buildType()}
	recv := m.fn.FuncType().Receiver
	if recv != nil {
		fDecl.Recv = recv.Src
	}
	return fDecl, true
}

func (m *vjpMacro) buildBodyFromSetAnnotation(fetcher ir.Fetcher, ann setAnnotation) (*ast.BlockStmt, []*cpevelements.SyntheticFuncDecl, bool) {
	return nil, nil, fetcher.Err().Appendf(m.Source(), "function with set directives not supported yet")
}

func (m *vjpMacro) BuildBody(fetcher ir.Fetcher, _ ir.Func) (*ast.BlockStmt, []*cpevelements.SyntheticFuncDecl, bool) {
	if ann := annotations.Get[setAnnotation](m.fn, m.set); ann != nil {
		return m.buildBodyFromSetAnnotation(fetcher, ann)
	}
	fnWithBody, ok := m.fn.(*ir.FuncDecl)
	if !ok {
		return nil, nil, fetcher.Err().Appendf(m.Source(), "function %s has no body", m.fn.ShortString())
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
