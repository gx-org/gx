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
	"go/ast"

	"github.com/gx-org/gx/build/ir"
)

var (
	nilIdent   = &ast.Ident{Name: "nil"}
	nilStorage = &ir.Nil{Src: nilIdent}
	nilExpr    = &ir.Ident{
		Src:  nilIdent,
		Stor: nilStorage,
	}
)

type nilEl struct {
	x *ir.NilCastExpr
}

var (
	_ ir.Element  = (*nilEl)(nil)
	_ ir.WithExpr = (*nilEl)(nil)
)

var nilError = NewNil(&ir.NilCastExpr{
	X:   nilExpr,
	Typ: ir.ErrorType(),
})

// NilError returns a GX nil error element.
func NilError() ir.Element {
	return nilError
}

// NilStorage returns the nil built-in storage singleton.
func NilStorage() *ir.Nil {
	return nilStorage
}

// NilFromType returns a nil element for a given type.
func NilFromType(tp ir.Type) ir.Element {
	return NewNil(&ir.NilCastExpr{
		X:   nilExpr,
		Typ: tp,
	})
}

// NewNil returns a new nil element.
func NewNil(x *ir.NilCastExpr) ir.Element {
	return &nilEl{x: x}
}

func (el *nilEl) Type() ir.Type {
	return el.x.Typ
}

func (el *nilEl) Expr(ev ir.Evaluator, expr ast.Expr) ([]ir.Expr, error) {
	return []ir.Expr{el.x}, nil
}

// IsNil returns true if the element is a nil element (whatever the type).
func IsNil(el ir.Element) bool {
	_, ok := el.(*nilEl)
	return ok
}
