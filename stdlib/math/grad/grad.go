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
	"go/ast"
	"go/token"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/base/uname"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/math/grad/revgraph"
	"github.com/gx-org/gx/stdlib/math/grad/special"
)

//go:embed *.gx
var fs embed.FS

// Package description of the GX meta/grad package.
var Package = builtin.PackageBuilder{
	FullPath: "math/grad",
	Builders: []builtin.Builder{
		builtin.ParseSource(&fs),
		builtin.RegisterMacro("Func", FuncGrad),
		builtin.RegisterMacro("Reverse", Reverse),
		builtin.RegisterFuncAnnotator("Set", SetGrad),
		builtin.RegisterFuncAnnotator("SetFor", SetGradFor),
	},
}

type gradMacro struct {
	cpevelements.CoreMacroElement
	call   elements.CallAt
	unames *uname.Unique

	wrt      withRespectTo
	revgraph *revgraph.Graph
}

var _ ir.FuncASTBuilder = (*gradMacro)(nil)

func findParamStorage(file *ir.File, src ir.Node, fn ir.Func, name string) (withRespectTo, error) {
	recv := fn.FuncType().ReceiverField()
	if recv != nil && recv.Name != nil {
		if recv.Name.Name == name {
			return newWRT(recv), nil
		}
	}
	field := fn.FuncType().Params.FindField(name)
	if field == nil {
		return nil, fmterr.Errorf(file.FileSet(), src.Node(), "no parameter named %s in %s", name, fn.ShortString())
	}
	return newWRT(field), nil
}

// FuncGrad computes the gradient of a function.
func FuncGrad(file *ir.File, call *ir.FuncCallExpr, mac *ir.Macro, args []ir.Element) (ir.MacroElement, error) {
	fn, err := interp.PkgFuncFromElement(args[0])
	if err != nil {
		return nil, err
	}
	fType := fn.FuncType()
	if fType.Results.Len() != 1 {
		return nil, errors.Errorf("%s expects 1 result but %s returns %d results", mac.ShortString(), fn.ShortString(), fn.FuncType().Results.Len())
	}
	wrtS, err := elements.StringFromElement(args[1])
	if err != nil {
		return nil, err
	}
	wrtF, err := findParamStorage(file, call, fn, wrtS)
	if err != nil {
		return nil, err
	}
	m := &gradMacro{
		CoreMacroElement: cpevelements.MacroElement(mac, file, call),
		call:             elements.NewNodeAt(file, call),
		unames:           uname.New(),
		wrt:              wrtF,
	}
	if m.revgraph, err = revgraph.New(&m.CoreMacroElement, fn); err != nil {
		return nil, err
	}
	m.unames.RegisterFieldNames(fType.Receiver)
	m.unames.RegisterFieldNames(fType.Params)
	m.unames.RegisterFieldNames(fType.Results)
	return m, nil
}

func typeParamsOf(fields *ir.FieldList) *ast.FieldList {
	if fields == nil {
		return nil
	}
	return fields.Src
}

func (m *gradMacro) BuildDecl(fn ir.PkgFunc) (*ir.File, *ast.FuncDecl, bool) {
	target := m.revgraph.Func()
	fType := target.FuncType()
	results := &ast.FieldList{List: []*ast.Field{
		&ast.Field{
			Type: fType.Results.Fields()[0].Group.Type.Expr(),
		},
	}}
	for _, param := range fType.Params.Fields() {
		results.List = append(results.List, &ast.Field{
			Type: param.Group.Type.Expr(),
		})
	}
	fDecl := &ast.FuncDecl{Type: &ast.FuncType{
		TypeParams: typeParamsOf(fType.TypeParams),
		Params:     fType.Params.Src,
		Results:    results,
	}}
	recv := fType.Receiver
	if recv != nil {
		fDecl.Recv = recv.Src
	}
	return target.File(), fDecl, true
}

func (m *gradMacro) BuildBody(fetcher ir.Fetcher, fn ir.Func) (*ast.BlockStmt, bool) {
	gradPkgFullName := m.CoreMacroElement.From().File().Package.FullName()
	imp := m.call.File().FindImport(gradPkgFullName)
	y := &ast.Ident{Name: m.unames.Name("y")}
	resIdent := &ast.Ident{Name: m.unames.Name("res")}
	outType := m.revgraph.Func().FuncType().Results.Fields()[0].Type()
	vjps := m.revgraph.VJPs()
	vjpVars := make([]ast.Expr, len(vjps))
	vjpCalls := make([]ast.Expr, len(vjps))
	for i, vjp := range vjps {
		vjpVars[i] = &ast.Ident{Name: m.unames.Name(vjp.Name() + "VJP")}
		vjpCalls[i] = &ast.CallExpr{
			Fun:  vjpVars[i],
			Args: []ast.Expr{resIdent},
		}
	}
	return &ast.BlockStmt{List: []ast.Stmt{
		&ast.AssignStmt{
			Tok: token.DEFINE,
			Lhs: append([]ast.Expr{y}, vjpVars...),
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   imp.NameDef(),
							Sel: &ast.Ident{Name: "Reverse"},
						},
						Args: []ast.Expr{m.call.Node().Args[0].Expr()},
					},
					Args: []ast.Expr{fn.FuncType().Params.Fields()[0].Name},
				},
			},
		},
		&ast.AssignStmt{
			Tok: token.DEFINE,
			Lhs: []ast.Expr{resIdent},
			Rhs: []ast.Expr{special.OneExpr().CastIfRequired(outType).AST()},
		},
		&ast.ReturnStmt{
			Results: append([]ast.Expr{y}, vjpCalls...),
		},
	}}, true
}
