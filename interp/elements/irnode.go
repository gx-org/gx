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

package elements

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/algexpr"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/src/gx/ast"
)

// IRNode is a GX string value.
type IRNode struct {
	node ir.IR
	typ  ir.Type
	path storepath.Path
}

var _ hostio.Value = (*IRNode)(nil)
var _ cmp.Canonical = (*IRNode)(nil)
var _ engine.NumericalElement = (*IRNode)(nil)

// NewIRNode returns a GX string value from its type and its Go value.
func NewIRNode(node ir.IR) *IRNode {
	return NewIRNodeWithType(node, ir.TypeFromKind(irkind.IR), nil)
}

// NewIRNodeWithType returns a GX string value from its type and its Go value.
func NewIRNodeWithType(node ir.IR, typ ir.Type, path storepath.Path) *IRNode {
	return &IRNode{node: node, typ: typ, path: path}
}

// Node stored in the value.
func (s *IRNode) Node() ir.IR {
	return s.node
}

// Type returns the type of the value.
func (s *IRNode) Type() ir.Type {
	return s.typ
}

// Expr returns the expression of the element.
func (s *IRNode) Expr(ev ir.Evaluator, src ast.Expr) ([]ir.Expr, error) {
	expr, err := cast.To[ir.Expr](s.node)
	if err != nil {
		return nil, err
	}
	return []ir.Expr{expr}, nil
}

// AlgExpr converts the element into an algebra expression.
func (s *IRNode) AlgExpr(ev ir.Evaluator) (cmp.Expr, error) {
	expr, err := cast.To[ir.Expr](s.node)
	if err != nil {
		return nil, err
	}
	if s.path == nil {
		return nil, errors.Errorf("no reference path defined for expression %s", expr.SourceString(ev.File()))
	}
	return algexpr.NewSymbFrom(s, s.same, expr), nil
}

// same returns true if other is an IR node with the same reference path.
func (s *IRNode) same(other ir.Element) bool {
	otherT, ok := other.(*IRNode)
	if !ok {
		return false
	}
	if s.path == nil || otherT.path == nil {
		return false
	}
	return s.path.Same(otherT.path)
}

// UnaryOp is not supported for IR nodes and returns an error.
func (s *IRNode) UnaryOp(env *engine.Env, expr *ir.UnaryExpr, typ ir.Type) (engine.NumericalElement, error) {
	return nil, fmterr.Internalf("UnaryOp not supported on IR node %s", s.SourceString(nil))
}

// BinaryOp is not supported for IR nodes and returns an error.
func (s *IRNode) BinaryOp(env *engine.Env, expr *ir.BinaryExpr, y engine.NumericalElement) (engine.NumericalElement, error) {
	return nil, fmterr.Internalf("BinaryOp not supported on IR node %s", s.SourceString(nil))
}

// Cast is not supported for IR nodes and returns an error.
func (s *IRNode) Cast(env *engine.Env, expr ir.Expr, target ir.Type) (engine.NumericalElement, error) {
	return nil, fmterr.Internalf("Cast not supported on IR node %s", s.SourceString(nil))
}

// Reshape is not supported for IR nodes and returns an error.
func (s *IRNode) Reshape(env *engine.Env, expr ir.Expr, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	return nil, fmterr.Internalf("Reshape not supported on IR node %s", s.SourceString(nil))
}

// ToHost transfers the value to the host.
func (s *IRNode) ToHost() (hostio.Value, error) {
	return s, nil
}

// Finalize is a no-op for IR node values.
func (s *IRNode) Finalize() error {
	return nil
}

// DefineString returns the GX source code to define the value.
// If not possible, fall back to SourceString.
func (s *IRNode) DefineString(from *ir.File) string {
	definer, isDefiner := s.node.(ir.StringDefiner)
	if !isDefiner {
		return s.SourceString(from)
	}
	return definer.DefineString(from)
}

// SourceString returns the GX source code of the implementation.
func (s *IRNode) SourceString(from *ir.File) string {
	return ir.StringerWithFrom(from, s.node).String()
}
