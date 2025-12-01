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
	"github.com/gx-org/gx/stdlib/math/grad/special"
)

type (
	vjpParam struct {
		field    *ir.Field
		wrt      withRespectTo
		vjpFType *ast.FuncType
	}

	vjpMacro struct {
		cpevelements.CoreMacroElement
		exprToName map[ir.Expr]forwardValues

		unames  *uname.Unique
		fwdRoot *uname.Root
		bckRoot *uname.Root

		fn     ir.PkgFunc
		params []vjpParam

		nResults       *namedFields
		nParams        *namedFields
		typeParamsExpr []ast.Expr
	}
)

var _ ir.FuncASTBuilder = (*vjpMacro)(nil)

// VJP computes the vector-jacobian product of a function.
func VJP(file *ir.File, call *ir.FuncCallExpr, macro *ir.Macro, args []ir.Element) (ir.MacroElement, error) {
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
		fwdRoot:          unames.Root("fwd"),
		bckRoot:          unames.Root("bck"),
	}.newMacro(fnT)
}

func (m vjpMacro) newMacro(fn ir.PkgFunc) (*vjpMacro, error) {
	m.fn = fn
	fType := m.fn.FuncType()
	m.params = make([]vjpParam, fType.Params.Len())
	m.nResults = nameFields(m.unames, "res", fType.Results.Src)
	m.nParams = nameFields(m.unames, "par", fType.Params.Src)
	for i, param := range fType.Params.Fields() {
		wrt := newWRT(param)
		backwardSig, err := m.buildBackwardSignature(fType, wrt, m.nResults.fields)
		if err != nil {
			return nil, err
		}
		m.params[i] = vjpParam{
			field:    param,
			wrt:      newWRT(param),
			vjpFType: backwardSig,
		}
	}
	typeParams := fType.TypeParams.Fields()
	if len(typeParams) == 0 {
		return &m, nil
	}
	m.typeParamsExpr = make([]ast.Expr, len(typeParams))
	for i, typeParam := range typeParams {
		m.typeParamsExpr[i] = &ast.Ident{Name: typeParam.Name.Name}
	}
	return &m, nil
}

func (m *vjpMacro) start() {
	m.exprToName = make(map[ir.Expr]forwardValues)
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
		// Same parameters (but named) as the original function.
		Params: m.nParams.fields,
		// Return the result of the original function as well as
		// a backward function to compute the gradient.
		Results: concatFieldList(
			fType.Src.Results,
			&ast.FieldList{List: vjpFuncs},
		),
	}
	return vjpType
}

func (m *vjpMacro) BuildDecl(ir.PkgFunc) (*ir.File, *ast.FuncDecl, bool) {
	m.start()
	fDecl := &ast.FuncDecl{Type: m.buildType()}
	recv := m.fn.FuncType().Receiver
	if recv != nil {
		fDecl.Recv = recv.Src
	}
	return m.fn.File(), fDecl, true
}

func (sg *stmtVJP) buildVJPFunctionWRTFromAnn(grad ir.PkgFunc, param vjpParam, args []ast.Expr) (*ast.FuncLit, bool) {
	backwarder := sg.newExprBackwardVJP(param.wrt)
	ret := &ast.ReturnStmt{Results: make([]ast.Expr, len(sg.macro.nResults.names))}
	for i, res := range sg.macro.nResults.names {
		ret.Results[i] = special.Mul(
			&special.Expr{
				Expr: &ast.Ident{Name: res},
			},
			&special.Expr{
				Expr: &ast.CallExpr{
					Fun: sg.macro.funcNameWithTypeParamsExpr(
						&ast.Ident{Name: grad.Name()},
					),
					Args: args,
				},
			},
		).Expr
	}
	var body []ast.Stmt
	body = append(body, backwarder.stmts...)
	body = append(body, ret)
	return &ast.FuncLit{
		Type: param.vjpFType,
		Body: &ast.BlockStmt{List: body},
	}, true
}

func (m *vjpMacro) funcNameWithTypeParamsExpr(fn ast.Expr) ast.Expr {
	var callee ast.Expr = fn
	switch len(m.typeParamsExpr) {
	case 0:
		return callee
	case 1:
		return &ast.IndexExpr{
			X:     callee,
			Index: m.typeParamsExpr[0],
		}
	default:
		return &ast.IndexListExpr{
			X:       callee,
			Indices: m.typeParamsExpr,
		}
	}
}

func (m *vjpMacro) buildBodyFromSetAnnotation(fetcher ir.Fetcher, sg *stmtVJP, ann *setAnnotation) (*ast.BlockStmt, bool) {
	forwarder := sg.newExprForwardVJP()
	// Build the arguments to call the forward functions.
	args := make([]ast.Expr, len(m.nParams.names))
	for i, fieldName := range m.nParams.names {
		args[i] = &ast.Ident{Name: fieldName}
	}
	// Call the original function to get the forward values.
	fType := m.fn.FuncType()
	nVals := fType.Results.Len()
	fv := forwarder.newForwardValues(nil, nVals)
	stmt := &ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: fv.exprs(),
		Rhs: []ast.Expr{&ast.CallExpr{
			Fun: m.funcNameWithTypeParamsExpr(
				&ast.Ident{Name: m.fn.Name()},
			),
			Args: args,
		}},
	}
	forwarder.appendMainStmt(stmt)

	// Generate forward statements for the expressions in the statement.
	ret := &ast.ReturnStmt{Results: make([]ast.Expr, nVals)}
	for i, expr := range fv.exprs() {
		ret.Results[i] = expr
	}

	// Build a backward function for each function parameter.
	names := make([]ast.Expr, len(sg.macro.params))
	for i, param := range sg.macro.params {
		vjpFuncLit, ok := sg.buildVJPFunctionWRTFromAnn(ann.partials[i], param, args)
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
	if ann := annotations.Get[*setAnnotation](m.fn, setKey); ann != nil {
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
