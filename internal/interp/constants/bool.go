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

package constants

import (
	"fmt"
	"go/ast"

	"github.com/gx-org/backend/ops"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/interp/engine"
)

// Bool constant.
type Bool interface {
	Constant
	engine.BoolConstant
}

// boolEl element storing an atomic bool value.
// Equivalent to numbers.Float and numbers.Int.
type boolEl struct {
	tp  ir.Type
	val bool
}

func newBool(tp ir.Type, val bool) *boolEl {
	return &boolEl{tp: tp, val: val}
}

var trueEl = newBool(ir.BoolType(), true)

// True returns an element representing the true boolean value.
func True() Bool {
	return trueEl
}

var falseEl = newBool(ir.BoolType(), false)

// False returns an element representing the false boolean value.
func False() Bool {
	return falseEl
}

// NewBool returns a new element to store a boolean value.
func NewBool(val bool) Bool {
	if val {
		return True()
	}
	return False()
}

// NewBoolWithType returns a boolean value for a given type.
func NewBoolWithType(tp ir.Type, val bool) engine.Constant {
	return newBool(tp, val)
}

// AlgExpr returns an algebraic expression.
func (b *boolEl) AlgExpr(eva ir.Evaluator) (cmp.Expr, error) {
	return b, nil
}

func (b *boolEl) Simplify(ir.SourceFile) (cmp.Comparable, error) {
	return b, nil
}

func (b *boolEl) Equal(other cmp.Comparable) bool {
	otherT, isBool := other.(*boolEl)
	if !isBool {
		return false
	}
	return b.val == otherT.val
}

// Atom returns the Go value of the scalar.
func (b *boolEl) Atom(kind irkind.Kind) (any, error) {
	return b.val, nil
}

func (b *boolEl) BuildNode(g ops.Graph, _ irkind.Kind) (ops.Node, error) {
	return g.Core().NewAtomLiteral(b.val)
}

func (b *boolEl) Type() ir.Type {
	return b.tp
}

func (b *boolEl) Bool() bool {
	return b.val
}

func (b *boolEl) AxisLengths() ([]int, error) {
	return nil, nil
}

func (b *boolEl) BuildIR() ir.Expr {
	var storage ir.Storage
	if b.val {
		storage = ir.TrueStorage()
	} else {
		storage = ir.FalseStorage()
	}
	return ir.NewIdent(storage)
}

func (b *boolEl) SourceString(from *ir.File) string {
	return b.String()
}

func (b *boolEl) ShortString(from *ir.File) string {
	return b.String()
}

func (b *boolEl) String() string {
	return fmt.Sprint(b.val)
}

func (b *boolEl) Expr(ir.Evaluator, ast.Expr) ([]ir.Expr, error) {
	return []ir.Expr{b.BuildIR()}, nil
}

type boolStorage struct {
	*boolEl
	storage ir.StorageWithValue
}

var _ ir.WithStore = (*boolStorage)(nil)

// NewBoolFromStorage returns a new element to store a boolean value.
func NewBoolFromStorage(storage ir.StorageWithValue) engine.Constant {
	val := storage.Value(nil).(*ir.BoolValue).Val
	var el *boolEl
	if val {
		el = trueEl
	} else {
		el = falseEl
	}
	return &boolStorage{
		boolEl:  el,
		storage: storage,
	}
}

func (b *boolStorage) Store() ir.Storage {
	return b.storage
}
