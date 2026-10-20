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
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/interp/engine"
)

type (
	// EvalShaper is an (array) element from which the shape has been fully determined at evaluation time.
	EvalShaper interface {
		ir.Element
		EvalShape() (shapes.Shape, error)
	}

	// Generic is an instance of a generic type.
	Generic interface {
		engine.NumericalElement
	}

	// IString is an element representing a string value.
	IString interface {
		ir.Element
		StrEl()
	}

	// IStruct is an interface implemented by all structure elements.
	IStruct interface {
		engine.Selector
		StructType() *ir.StructType
	}

	// FieldPath indicates a path to a field in a structure.
	FieldPath interface {
		ir.Element
		// Root of the field path.
		Root() (*storepath.Proxy, error)
		// FollowOn follow the path on a given element.
		FollowOn(ir.Element) (ir.Element, error)
	}
)
