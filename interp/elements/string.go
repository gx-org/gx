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
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/engine"
)

// String is a GX string.
type String struct {
	val *values.String
}

var _ ir.Element = (*String)(nil)

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
	gxVal, err := values.NewString(typ, val)
	if err != nil {
		return nil, err
	}
	return &String{
		val: gxVal,
	}, nil
}

// Unflatten consumes the next handles to return a GX value.
func (n *String) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return n.val, nil
}

// StringValue returns the string value as a GX value.
func (n *String) StringValue() string {
	return n.val.StringValue()
}

// Copy returns the receiver.
func (n *String) Copy() engine.Copier {
	return n
}

// Type of the element.
func (n *String) Type() ir.Type {
	return n.val.Type()
}

// StringFromElement returns the string value stored in a element.
func StringFromElement(el ir.Element) (string, error) {
	sEl, ok := Underlying(el).(*String)
	if !ok {
		return "", errors.Errorf("cannot convert element %T is not a string literal", el)
	}
	return sEl.StringValue(), nil
}
