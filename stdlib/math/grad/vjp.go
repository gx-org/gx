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

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/astbuilder"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/stdlib/math/grad/revgraph"
)

type (
	vjpParam struct {
		field    *ir.Field
		wrt      withRespectTo
		vjpFType *ast.FuncType
	}

	vjpMacro struct {
		cpevelements.CoreMacroElement
		graph *revgraph.Graph
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
	return vjpMacro{
		CoreMacroElement: cpevelements.MacroElement(macro, file, call),
	}.newMacro(fnT)
}

func (m vjpMacro) newMacro(fn ir.PkgFunc) (*vjpMacro, error) {
	var err error
	m.graph, err = revgraph.New(&m.CoreMacroElement, fn)
	if err != nil {
		return nil, err
	}
	return &m, nil
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

func (m *vjpMacro) BuildDecl(ir.PkgFunc) (*ir.File, *ast.FuncDecl, bool) {
	fDecl := &ast.FuncDecl{Type: m.graph.BuildType()}
	fn := m.graph.Func()
	recv := fn.FuncType().Receiver
	if recv != nil {
		fDecl.Recv = recv.Src
	}
	return fn.File(), fDecl, true
}

func (m *vjpMacro) BuildBody(fetcher ir.Fetcher, _ ir.Func) (*ast.BlockStmt, bool) {
	return m.graph.Process(fetcher)
}

// Unflatten creates a GX value from the next handles available in the parser.
func (m *vjpMacro) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return values.NewIRNode(m.graph.Func())
}
