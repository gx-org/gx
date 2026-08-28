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

// Package surrogates creates surrogate interpreter elements given
// a storage path and a type. The storage path is used to compare
// surrogates value to one another.
//
// Surrogate values typically need to be linked to a storage
// using srcstore.Link so that the compiler knows where the value
// is stored.
package surrogates

import (
	"go/ast"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
)

// Element is a surrogate element.
type Element interface {
	ir.Element
	ir.WithExpr
}

// FieldRoot returns a new surrogates from a root field.
func FieldRoot(field *ir.Field, storage ir.Storage) (Element, error) {
	return New(storepath.NewRoot(field, storage), field.Type())
}

type core struct {
	path storepath.Path
}

func (c *core) Expr(ir.Evaluator, ast.Expr) ([]ir.Expr, error) {
	return []ir.Expr{c.path.Expr()}, nil
}

// invalid element.
type invalid struct{}

var invalidEl = &invalid{}

// NewInvalid returns an invalid element.
func NewInvalid() Element {
	return invalidEl
}

// Type returns an invalid type.
func (i *invalid) Type() ir.Type {
	return ir.InvalidType()
}

// Expr of the invalid element.
func (*invalid) Expr(ir.Evaluator, ast.Expr) ([]ir.Expr, error) {
	return nil, errors.Errorf("invalid element has no expression")
}

// New surrogate value given a path and a type.
func New(path storepath.Path, typ ir.Type) (Element, error) {
	switch typ.Kind() {
	case irkind.String:
		return NewString(path), nil
	case irkind.Invalid:
		return NewInvalid(), nil
	}
	switch typT := typ.(type) {
	case ir.ArrayType:
		return NewArray(path, typT), nil
	case *ir.FuncType:
		return newSurrogateFunc(path, typT), nil
	case *ir.StructType:
		return newStruct(path, typT)
	case *ir.NamedType:
		return newNamedType(path, typT)
	case *ir.VarArgsType:
		return newSliceType(path, typT.Typ), nil
	case *ir.SliceType:
		return newSliceType(path, typT), nil
	case *ir.GenericTypeParam:
		return newGenericType(path, typT), nil
	case ir.TypeMethods:
		return newInterface(path, typT)
	default:
		return NewInvalid(), fmterr.Internalf("cannot convert %T:%s to a surrogate element", typT, typT.ReferString(nil))
	}
}
