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

	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

// String is a GX string.
type String struct {
	str *ir.StringLiteral
	val *values.String
}

var _ Element = (*String)(nil)

// NewString returns a state element storing a string GX value.
func NewString(str *ir.StringLiteral) (*String, error) {
	val, err := strconv.Unquote(str.Src.Value)
	if err != nil {
		return nil, err
	}
	gxVal, err := values.NewString(str.Type(), val)
	if err != nil {
		return nil, err
	}
	return &String{
		str: str,
		val: gxVal,
	}, nil
}

// Flatten returns the element in a slice of elements.
func (n *String) Flatten() ([]Element, error) {
	return []Element{n}, nil
}

// Unflatten consumes the next handles to return a GX value.
func (n *String) Unflatten(handles *Unflattener) (values.Value, error) {
	return n.val, nil
}

// StringValue returns the string value as a GX value.
func (n *String) StringValue() *values.String {
	return n.val
}

// Kind of the element.
func (*String) Kind() ir.Kind {
	return ir.StringKind
}
