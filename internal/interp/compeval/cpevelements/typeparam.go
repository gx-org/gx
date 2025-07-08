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
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/elements"
)

type typeParam struct {
	src elements.ExprAt
	typ *ir.TypeParam
}

func newTypeParam(src elements.ExprAt, typ *ir.TypeParam) elements.Element {
	return &typeParam{src: src, typ: typ}
}

func (tp *typeParam) Flatten() ([]ir.Element, error) {
	return []ir.Element{tp}, nil
}

// Unflatten creates a GX value from the next handles available in the parser.
func (tp *typeParam) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return nil, errors.Errorf("not implemented")
}

// Type returns the type of the element.
func (tp *typeParam) Type() ir.Type {
	return tp.typ
}
