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

package interp

import (
	"github.com/pkg/errors"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/evaluator"
)

type (
	// Slicer is a state element that can be sliced.
	Slicer interface {
		Slice(fitp *FileScope, expr *ir.IndexExpr, index evaluator.NumericalElement) (ir.Element, error)
	}

	// ArraySlicer is a state element with an array that can be sliced.
	ArraySlicer interface {
		evaluator.NumericalElement
		SliceArray(fitp *FileScope, expr ir.AssignableExpr, index evaluator.NumericalElement) (evaluator.NumericalElement, error)
		Type() ir.Type
	}

	// WithAxes is an element able to return its axes as a slice of element.
	WithAxes interface {
		Axes(ev ir.Evaluator) (*Slice, error)
	}

	// FixedShape is an (array) element from which the shape has been fully determined.
	FixedShape interface {
		ir.Element
		Shape() *shape.Shape
	}

	// FixedSlice is a slice
	FixedSlice interface {
		ir.Element
		Elements() []ir.Element
		Len() int
	}

	// NType is a named type.
	NType interface {
		Under() ir.Element
	}

	// Copier is an interface implemented by nodes that need to be copied when passed to a function.
	Copier interface {
		ir.Element
		Copy() Copier
	}

	// Selector selects a field given its index.
	Selector interface {
		ir.Element
		Select(expr *ir.SelectorExpr) (ir.Element, error)
	}
)

// ShapeFromElement returns the shape of a numerical element.
func ShapeFromElement(node ir.Element) (*shape.Shape, error) {
	numerical, ok := node.(FixedShape)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to a numerical element", node)
	}
	return numerical.Shape(), nil
}
