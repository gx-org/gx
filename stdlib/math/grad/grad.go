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

	"github.com/pkg/errors"
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
		builtin.RegisterMacro("Set", SetGrad),
		builtin.RegisterMacro("SetFor", SetGradFor),
	},
}

type gradMacro struct {
	cpevelements.CoreMacroElement
	callSite elements.CallAt
	aux      *ordered.Map[string, *cpevelements.SyntheticFuncDecl]

	fn  ir.PkgFunc
	wrt withRespectTo
}

var _ cpevelements.FuncASTBuilder = (*gradMacro)(nil)

func findParamStorage(file *ir.File, src ir.SourceNode, fn ir.Func, name string) (withRespectTo, error) {
	recv := fn.FuncType().ReceiverField()
	if recv != nil && recv.Name != nil {
		if recv.Name.Name == name {
			return newWRT(recv), nil
		}
	}
	field := fn.FuncType().Params.FindField(name)
	if field == nil {
		return nil, fmterr.Errorf(file.FileSet(), src.Source(), "no parameter named %s in %s", name, fn.ShortString())
	}
	return newWRT(field), nil
}

// FuncGrad computes the gradient of a function.
func FuncGrad(call elements.CallAt, macro *cpevelements.Macro, args []ir.Element) (cpevelements.MacroElement, error) {
	fn, err := interp.PkgFuncFromElement(args[0])
	if err != nil {
		return nil, err
	}
	if fn.FuncType().Results.Len() > 1 {
		return nil, errors.Errorf("cannot compute the gradient of function with more than one result")
	}
	fnT, ok := fn.(ir.PkgFunc)
	if !ok {
		return nil, errors.Errorf("cannot compute the gradient of function %T", fn)
	}
	wrtS, err := elements.StringFromElement(args[1])
	if err != nil {
		return nil, err
	}
	wrtF, err := findParamStorage(call.File(), call.Node(), fn, wrtS)
	if err != nil {
		return nil, err
	}
	return gradMacro{
		CoreMacroElement: cpevelements.CoreMacroElement{Mac: macro},
		callSite:         call,
	}.newMacro(fnT, wrtF), nil
}

func (m gradMacro) newMacro(fn ir.PkgFunc, wrt withRespectTo) *gradMacro {
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
	funcName := callee.ShortString()
	macroPackage := callee.F.(*ir.Macro).File().Package
	imp := m.callSite.File().FindImport(macroPackage.FullName())
	if imp == nil {
		return "", fetcher.Err().AppendInternalf(callee.Source(), "cannot find import name %s", macroPackage.FullName())
	}
	return fmt.Sprintf("__%s_%s_%s_%s", imp.Name(), funcName, fn.ShortString(), m.wrt.name()), true
}

func (m *gradMacro) BuildDecl(ir.PkgFunc) (*ast.FuncDecl, bool) {
	fType := m.fn.FuncType()
	fDecl := &ast.FuncDecl{Type: fType.Src}
	fDecl.Type.Results = &ast.FieldList{
		List: []*ast.Field{&ast.Field{
			Type: m.wrt.fieldType(),
		}},
	}
	recv := fType.Receiver
	if recv != nil {
		fDecl.Recv = recv.Src
	}
	return fDecl, true
}

func (m *gradMacro) buildBodyFromSetAnnotation(fetcher ir.Fetcher, fn ir.Func, ann *setAnnotation) (*ast.BlockStmt, []*cpevelements.SyntheticFuncDecl, bool) {
	params := fn.FuncType().Params.Fields()
	args := make([]ast.Expr, len(params))
	for i, param := range params {
		args[i] = param.Name
	}
	identToGradFunc, ok := gradFromAnnotation(fetcher, fn, ann, m.wrt.name())
	if !ok {
		return nil, nil, false
	}
	return &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{
		Results: []ast.Expr{&ast.CallExpr{
			Fun:  identToGradFunc,
			Args: args,
		}},
	}}}, nil, true
}

func (m *gradMacro) BuildBody(fetcher ir.Fetcher, fn ir.Func) (*ast.BlockStmt, []*cpevelements.SyntheticFuncDecl, bool) {
	if ann := findSetAnnotation(m.fn); ann != nil {
		return m.buildBodyFromSetAnnotation(fetcher, fn, ann)
	}
	fnWithBody, ok := m.fn.(*ir.FuncDecl)
	if !ok {
		return nil, nil, fetcher.Err().Appendf(fn.Source(), "function has no body")
	}
	m.aux = ordered.NewMap[string, *cpevelements.SyntheticFuncDecl]()
	sg := m.newStmtGrader(fetcher, nil)
	fType := m.fn.FuncType()
	sg.registerFieldNames(fType.Receiver)
	sg.registerFieldNames(fType.Params)
	sg.registerFieldNames(fType.Results)
	body, ok := sg.gradBlock(fetcher, fnWithBody.Body)
	if !ok {
		return nil, nil, false
	}
	return body, slices.Collect(m.aux.Values()), true
}

func (m *gradMacro) gradIdent(src *ast.Ident) *ast.Ident {
	return &ast.Ident{
		NamePos: src.NamePos,
		Name:    "__grad_" + src.Name,
	}
}
