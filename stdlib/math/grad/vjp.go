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
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/src/gx/ast"
	"github.com/gx-org/gx/stdlib/math/grad/revgraph"
)

type vjpMacro struct {
	elements.MacroCall
	graph *revgraph.Graph
	recv  ir.Expr
}

var _ ir.FuncASTBuilder = (*vjpMacro)(nil)

// Reverse computes the vector-jacobian product of a function.
func Reverse(file *ir.File, call *ir.FuncCallExpr, macro *ir.Macro, args []ir.Element) (ir.MacroElement, error) {
	fnIR, err := interp.PkgFuncFromElement(args[0])
	if err != nil {
		return nil, err
	}
	recv, err := extractReceiverIR(fnIR, call.Args[0])
	if err != nil {
		return nil, err
	}
	return vjpMacro{
		MacroCall: elements.NewMacroCall(macro, file, call),
		recv:      recv,
	}.newMacro(fnIR)
}

func (m vjpMacro) newMacro(fnIR ir.PkgFunc) (*vjpMacro, error) {
	var err error
	m.graph, err = revgraph.New(&m.MacroCall, fnIR)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func unwrapExpr(x ir.Expr) ir.Expr {
	switch xT := x.(type) {
	case *ir.FuncValExpr:
		return unwrapExpr(xT.X())
	case *ir.SpecTypeExpr:
		return unwrapExpr(xT.X)
	case *ir.ParenExpr:
		return unwrapExpr(xT.X)
	default:
		return x
	}
}

func extractReceiverIR(fnIR ir.PkgFunc, x ir.Expr) (ir.Expr, error) {
	if fnIR.FuncType().ReceiverField() == nil {
		return nil, nil
	}
	sel, err := cast.To[*ir.SelectorExpr](unwrapExpr(x))
	if err != nil {
		return nil, err
	}
	if sel.X.Type().Kind() == irkind.MetaType {
		return nil, nil
	}
	return sel.X, nil
}

func (m *vjpMacro) BuildDecl(ir.PkgFunc) (*ir.File, ir.Expr, *ast.FuncDecl, bool) {
	fDecl := &ast.FuncDecl{Type: m.graph.BuildType()}
	fn := m.graph.Func()
	recvType := fn.FuncType().Receiver
	if recvType != nil {
		fDecl.Recv = recvType.Src
	}
	return fn.File(), m.recv, fDecl, true
}

func (m *vjpMacro) BuildBody(fetcher ir.Fetcher, _ ir.Func) (*ast.BlockStmt, bool) {
	return m.graph.Process(fetcher)
}

// Unflatten creates a GX value from the next handles available in the parser.
func (m *vjpMacro) Unflatten(handles hostio.Parser) (hostio.Value, error) {
	return elements.NewIRNode(m.graph.Func()), nil
}
