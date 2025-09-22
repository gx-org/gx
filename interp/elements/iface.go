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
	"github.com/pkg/errors"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
)

type (
	// FixedShape is an (array) element from which the shape has been fully determined.
	FixedShape interface {
		ir.Element
		Shape() *shape.Shape
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
