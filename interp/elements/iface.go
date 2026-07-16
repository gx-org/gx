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
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
)

type (
	// EvalShaper is an (array) element from which the shape has been fully determined at evaluation time.
	EvalShaper interface {
		ir.Element
		EvalShape() (*shape.Shape, error)
	}

	// NType is a named type.
	NType interface {
		Under() ir.Element
	}

	// Selector selects a field given its index.
	Selector interface {
		ir.Element
		Select(expr *ir.SelectorExpr) (ir.Element, error)
	}

	// Under is an element with an underlying element.
	Under interface {
		Under() ir.Element
	}
)

// ShapeFromElement returns the shape of a numerical element.
func ShapeFromElement(el ir.Element) (*shape.Shape, error) {
	shaper, err := cast.To[EvalShaper](el)
	if err != nil {
		return nil, err
	}
	return shaper.EvalShape()
}

// Underlying returns the underlying element.
func Underlying(val ir.Element) ir.Element {
	named, ok := val.(Under)
	if !ok {
		return val
	}
	return Underlying(named.Under())
}
