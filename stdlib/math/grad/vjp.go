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
	"go/token"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/base/uname"
	"github.com/gx-org/gx/build/ir/annotations"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/astbuilder"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/internal/interp/flatten"
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
		set         *ir.Macro
		exprToName  map[ir.Expr]forwardValues
		resultNames []string

		unames  *uname.Unique
		fwdRoot *uname.Root
		bckRoot *uname.Root

		fn     ir.PkgFunc
		params []vjpParam
	}
)

var _ ir.FuncASTBuilder = (*vjpMacro)(nil)

// VJP computes the vector-jacobian product of a function.
func VJP(file *ir.File, call *ir.CallExpr, macro *ir.Macro, args []ir.Element) (ir.MacroElement, error) {
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
		CoreMacroElement: cpevelements.MacroElement(macro, file, call),
		unames:           unames,
		set:              setMacro(macro),
		fwdRoot:          unames.Root("fwd"),
		bckRoot:          unames.Root("bck"),
	}.newMacro(fnT)
}

func (m vjpMacro) newMacro(fn ir.PkgFunc) (*vjpMacro, error) {
	m.fn = fn
	fType := m.fn.FuncType()
	m.params = make([]vjpParam, fType.Params.Len())
	var namedResultFields *ast.FieldList
	m.resultNames, namedResultFields = m.nameResultFields(fType.Results.Src)
	for i, param := range fType.Params.Fields() {
		wrt := newWRT(param)
		backwardSig, err := m.buildBackwardSignature(fType, wrt, namedResultFields)
		if err != nil {
			return nil, err
		}
		m.params[i] = vjpParam{
			field:    param,
			wrt:      newWRT(param),
			vjpFType: backwardSig,
		}
	}
	return &m, nil
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

func (m *vjpMacro) buildBackwardSignature(fType *ir.FuncType, wrt withRespectTo, namedResultFields *ast.FieldList) (*ast.FuncType, error) {
	results, err := astbuilder.Clone(&ast.FieldList{
		List: []*ast.Field{&ast.Field{
			Type: wrt.fieldType(),
		}},
	}, astbuilder.AssignToExpandShape)
	if err != nil {
		return nil, err
	}

	return &ast.FuncType{
		// Same as the original function.
		TypeParams: fType.Src.TypeParams,
		// Gradient coming from the output values of the function.
		Params: namedResultFields,
		// Return the gradient.
		Results: results,
	}, nil
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

func (sg *stmtVJP) buildVJPFunctionWRTFromAnn(grad ir.PkgFunc, param vjpParam) (*ast.FuncLit, bool) {
	backwarder := sg.newExprBackwardVJP(param.wrt)
	ret := &ast.ReturnStmt{Results: make([]ast.Expr, len(sg.macro.resultNames))}
	for i, res := range sg.macro.resultNames {
		ret.Results[i] = buildMul(
			&gradExprResult{
				expr: &ast.Ident{Name: res},
			},
			&gradExprResult{
				expr: &ast.Ident{Name: grad.Name()},
			},
		).expr
	}
	var body []ast.Stmt
	body = append(body, backwarder.stmts...)
	body = append(body, ret)
	return &ast.FuncLit{
		Type: param.vjpFType,
		Body: &ast.BlockStmt{List: body},
	}, true
}

func (m *vjpMacro) buildBodyFromSetAnnotation(fetcher ir.Fetcher, sg *stmtVJP, ann *setAnnotation) (*ast.BlockStmt, bool) {
	forwarder := sg.newExprForwardVJP()
	// Call the original function to get the forward values.
	nVals := m.fn.FuncType().Results.Len()
	fv := forwarder.newForwardValues(nil, nVals)
	stmt := &ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: fv.idents(),
		Rhs: []ast.Expr{&ast.CallExpr{
			Fun: &ast.Ident{Name: m.fn.Name()},
		}},
	}
	forwarder.appendMainStmt(stmt)

	// Generate forward statements for the expressions in the statement.
	ret := &ast.ReturnStmt{Results: make([]ast.Expr, nVals)}
	for i, expr := range fv.idents() {
		ret.Results[i] = expr
	}

	// Build a backward function for each function parameter.
	names := make([]ast.Expr, len(sg.macro.params))
	for i, param := range sg.macro.params {
		vjpFuncLit, ok := sg.buildVJPFunctionWRTFromAnn(ann.partials[i], param)
		if !ok {
			return nil, false
		}
		root := "selfVJPFunc"
		if len(names) > 1 {
			root += "WRT" + param.wrt.name()
		}
		vjpFuncName := sg.macro.unames.Name(root)
		sg.appendMainStmt(&ast.AssignStmt{
			Tok: token.DEFINE,
			Lhs: []ast.Expr{&ast.Ident{Name: vjpFuncName}},
			Rhs: []ast.Expr{vjpFuncLit},
		})
		ret.Results = append(ret.Results, &ast.Ident{Name: vjpFuncName})
	}
	stmts := append([]ast.Stmt{}, sg.stmts...)
	stmts = append(stmts, ret)
	return &ast.BlockStmt{List: stmts}, true
}

func (m *vjpMacro) BuildBody(fetcher ir.Fetcher, _ ir.Func) (*ast.BlockStmt, bool) {
	sg := m.newStmt(fetcher, nil)
	fType := m.fn.FuncType()
	sg.registerFieldNames(fType.Receiver)
	sg.registerFieldNames(fType.Params)
	sg.registerFieldNames(fType.Results)
	if ann := annotations.Get[*setAnnotation](m.fn, m.set); ann != nil {
		return m.buildBodyFromSetAnnotation(fetcher, sg, ann)
	}
	fnWithBody, ok := m.fn.(*ir.FuncDecl)
	if !ok {
		return nil, fetcher.Err().Appendf(m.Source(), "function %s requires a gradient specification", m.fn.ShortString())
	}
	body, ok := sg.processBlock(fnWithBody.Body)
	if !ok {
		return nil, false
	}
	return body, true
}

// Unflatten creates a GX value from the next handles available in the parser.
func (m *vjpMacro) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return values.NewIRNode(m.fn)
}
