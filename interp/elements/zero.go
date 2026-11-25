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

package elements

import (
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/src/gx/ast"
)

// Zero is the zero value of a type.
// For example, the zero value of a generic type parameter
// for which the concrete type is not known yet.
type Zero struct {
	typ ir.Zeroer
}

var (
	_ ir.WithExpr      = (*Zero)(nil)
	_ ir.StringSourcer = (*Zero)(nil)
)

// NewZero returns an element representing the zero value of typ.
// An error is returned if the type has no zero value.
func NewZero(typ ir.Type) (*Zero, error) {
	zeroer, err := cast.To[ir.Zeroer](typ)
	if err != nil {
		return nil, err
	}
	return &Zero{typ: zeroer}, nil
}

// Type of the element.
func (z *Zero) Type() ir.Type {
	return z.typ
}

// Expr returns an expression evaluating to the zero value.
func (z *Zero) Expr(ir.Evaluator, ast.Expr) ([]ir.Expr, error) {
	return []ir.Expr{z.typ.Zero()}, nil
}

// SourceString returns the GX source code of the zero value.
func (z *Zero) SourceString(from *ir.File) string {
	return z.typ.Zero().SourceString(from)
}
