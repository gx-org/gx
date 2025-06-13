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

package cpevelements

import (
	"github.com/pkg/errors"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

type slice struct {
	src elements.ExprAt
	typ *ir.SliceType
}

var (
	_ ir.Canonical    = (*slice)(nil)
	_ Element         = (*slice)(nil)
	_ elements.Slicer = (*slice)(nil)
)

func newSlice(src elements.ExprAt, typ *ir.SliceType) elements.Element {
	return &slice{src: src, typ: typ}
}

func (f *slice) Flatten() ([]elements.Element, error) {
	return []elements.Element{f}, nil
}

func (f *slice) Unflatten(handles *elements.Unflattener) (values.Value, error) {
	return nil, errors.Errorf("not implemented")
}

func (f *slice) UnaryOp(ctx elements.FileContext, expr *ir.UnaryExpr) (elements.NumericalElement, error) {
	return newUnary(ctx, expr, f)
}

func (f *slice) BinaryOp(ctx elements.FileContext, expr *ir.BinaryExpr, x, y elements.NumericalElement) (elements.NumericalElement, error) {
	return newBinary(ctx, expr, x, y)
}

func (f *slice) Cast(ctx elements.FileContext, expr ir.AssignableExpr, target ir.Type) (elements.NumericalElement, error) {
	return newCast(ctx, expr, f, target)
}

func (f *slice) Shape() *shape.Shape {
	return nil
}

func (f *slice) Compare(x canonical.Comparable) bool {
	return f == x
}

func (f *slice) Slice(ctx elements.FileContext, expr ir.AssignableExpr, index elements.NumericalElement) (elements.Element, error) {
	return NewRuntimeValue(ctx.(evaluator.Context), expr)
}

func (f *slice) Expr() ir.AssignableExpr {
	return f.src.Node()
}

func (f *slice) Kind() ir.Kind {
	return ir.SliceKind
}

// CanonicalExpr returns the canonical expression used for comparison.
func (f *slice) CanonicalExpr() canonical.Canonical {
	return f
}

func (f *slice) String() string {
	return f.typ.String()
}
