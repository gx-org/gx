// Copyright 2026 Google LLC
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

package numbers

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/interp/compeval/cpevops"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/engine"
)

type (
	// boolEl element storing an atomic bool value.
	// Equivalent to numbers.Float and numbers.Int.
	boolEl struct {
		expr ir.Expr
		val  bool
	}

	// Bool represents a boolean value.
	Bool interface {
		cpevops.Element
		Value() bool
	}
)

var (
	_ Bool                = (*boolEl)(nil)
	_ flatten.Unflattener = (*boolEl)(nil)
)

func newBool(expr ir.Expr, val bool) *boolEl {
	return &boolEl{expr: expr, val: val}
}

// NewBool returns a new element to store a boolean value.
func NewBool(expr ir.Expr, val bool) engine.NumericalElement {
	return newBool(expr, val)
}

func (b *boolEl) Type() ir.Type {
	return b.expr.Type()
}

func (b *boolEl) UnaryOp(env engine.Env, expr *ir.UnaryExpr) (engine.NumericalElement, error) {
	switch expr.Src.Op {
	case token.NOT:
		return NewBool(expr, !b.val), nil
	default:
		return nil, fmterr.Errorf(env.File().FileSet(), expr.Src, "bool unary operator %s not implemented", expr.Src.Op)
	}
}

func (b *boolEl) BinaryOp(env engine.Env, expr *ir.BinaryExpr, x, y engine.NumericalElement) (engine.NumericalElement, error) {
	return cpevops.NewBinary(env, expr, x, y)
}

func (b *boolEl) Cast(env engine.Env, expr ir.Expr, target ir.Type) (engine.NumericalElement, error) {
	return NewBool(expr, b.val), nil
}

func (b *boolEl) Reshape(env engine.Env, expr ir.Expr, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	return cpevops.NewReshape(env, expr, b, axisLengths)
}

func (b *boolEl) CanonicalExpr() canonical.Canonical {
	return b
}

// Compare with another number.
func (b *boolEl) Compare(other canonical.Comparable) (bool, error) {
	otherT, isBool := other.(*boolEl)
	if !isBool {
		return false, nil
	}
	return b.val == otherT.val, nil
}

func (b *boolEl) ShortString() string {
	return fmt.Sprint(b.val)
}

// SourceString returns the GX source code to represent the float.
func (b *boolEl) SourceString(from *ir.File) string {
	return b.expr.SourceString(from)
}

// Value returns the current stored value.
func (b *boolEl) Value() bool {
	return b.val
}

// Expr returns the expression representing the boolean value.
func (b *boolEl) Expr(ir.Evaluator, ast.Expr) ([]ir.Expr, error) {
	return []ir.Expr{b.expr}, nil
}

func (b *boolEl) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return values.AtomBoolValue(b.Type(), b.val)
}

// boolStorage is used to store the values of true and false.
type boolStorage struct {
	*boolEl
	storage ir.StorageWithValue
}

var _ ir.WithStore = (*boolStorage)(nil)

// NewBoolFromStorage returns a new element to store a boolean value.
func NewBoolFromStorage(storage ir.StorageWithValue) engine.AtomLitElement {
	val := storage.Value(nil).(*ir.BoolValue).Val
	return &boolStorage{
		boolEl:  newBool(ir.NewIdent(storage), val),
		storage: storage,
	}
}

func (b *boolStorage) Store() ir.Storage {
	return b.storage
}
