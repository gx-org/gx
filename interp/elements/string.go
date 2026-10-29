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
	"strconv"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/coreiface"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/internal/togo"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/src/gx/ast"
)

// String is a GX string.
type String struct {
	typ ir.Type
	str string
}

var (
	_ IString           = (*String)(nil)
	_ cmp.Canonical     = (*String)(nil)
	_ togo.CompileValue = (*String)(nil)
	_ hostio.Value      = (*String)(nil)
)

// NewStringFromLit returns a state element storing a string GX value.
func NewStringFromLit(str *ir.StringLiteral) (*String, error) {
	val, err := strconv.Unquote(str.Src.Value)
	if err != nil {
		return nil, err
	}
	return NewString(val, str.Type())
}

// NewString returns a new element containing a string.
func NewString(val string, typ ir.Type) (*String, error) {
	if typ.Kind() != irkind.String {
		return nil, errors.Errorf("%s is an invalid kind for a string value", typ.Kind().String())
	}
	return &String{typ: typ, str: val}, nil
}

// StrEl is only useful to implement the IString interface and has no other purpose.
func (*String) StrEl() {}

// Unflatten consumes the next handles to return a GX value.
func (n *String) Unflatten(handles *flatten.Parser) (hostio.Value, error) {
	return n, nil
}

// ToHost transfers the value to host given an allocator.
func (n *String) ToHost(kernels.Allocator) (hostio.Value, error) {
	return n, nil
}

// Finalize is a no-op for string values.
func (n *String) Finalize() error {
	return nil
}

// Copy returns the receiver.
func (n *String) Copy() engine.Copier {
	return n
}

// Simplify the expression.
func (n *String) Simplify(ir.SourceFile) (cmp.Comparable, error) {
	return n, nil
}

// Equal compare the receiver with the argument.
func (n *String) Equal(other cmp.Comparable) bool {
	otherT, isString := other.(*String)
	if !isString {
		return false
	}
	return n.str == otherT.str
}

// AlgExpr converts the element into an algebra expression.
func (n *String) AlgExpr(ir.Evaluator) (cmp.Expr, error) {
	return n, nil
}

// Type of the element.
func (n *String) Type() ir.Type {
	return n.typ
}

// BuildIR returns a string literal.
func (n *String) BuildIR() ir.Expr {
	return &ir.StringLiteral{
		Src: &ast.BasicLit{Value: strconv.Quote(n.str)},
	}
}

// Expr returns the string as an IR expression.
func (n *String) Expr(ir.Evaluator, ast.Expr) ([]ir.Expr, error) {
	return []ir.Expr{n.BuildIR()}, nil
}

// GoValue returns the string as a Go value.
func (n *String) GoValue() (any, error) {
	return n.String(), nil
}

// SourceString returns the string value as a GX string literal.
func (n *String) SourceString(*ir.File) string {
	return strconv.Quote(n.str)
}

// DefineString returns the GX source code of the implementation.
func (n *String) DefineString(*ir.File) string {
	return n.String()
}

// String returns the string value as a GX value.
func (n *String) String() string {
	return n.str
}

// StringFromElement returns the string value stored in a element.
func StringFromElement(el ir.Element) (string, error) {
	under, err := coreiface.Underlying(el)
	if err != nil {
		return "", err
	}
	sEl, err := cast.To[*String](under)
	if err != nil {
		return "", err
	}
	return sEl.String(), nil
}
