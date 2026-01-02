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

package values

import (
	"github.com/pkg/errors"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
)

var _ Value = (*String)(nil)

// String is a GX string value.
type String struct {
	typ ir.Type
	str string
}

// NewString returns a GX string value from its type and its Go value.
func NewString(typ ir.Type, str string) (*String, error) {
	if typ.Kind() != irkind.String {
		return nil, errors.Errorf("%s is an invalid kind for a string value", typ.Kind().String())
	}
	return &String{typ: typ, str: str}, nil
}

func (s *String) value() {}

// Type returns the type of the value.
func (s *String) Type() ir.Type {
	return s.typ
}

// ToHost transfers the value to host given an allocator.
func (s *String) ToHost(platform.Allocator) (Value, error) {
	return s, nil
}

// String representation of the value.
// The returned string is a string reported to the user.
func (s *String) String() string {
	return s.str
}
