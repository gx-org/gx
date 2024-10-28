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

package kernels

import (
	"go/token"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
)

type boolFactory struct {
	arrayFactory[bool]
}

var _ Factory = (*boolFactory)(nil)

func (boolFactory) BinaryOp(op token.Token, x, y *shape.Shape) (Binary, *shape.Shape, error) {
	return nil, nil, errors.Errorf("not implemented")
}

// UnaryOp creates a new kernel for a unary operator.
func (boolFactory) UnaryOp(op token.Token, x *shape.Shape) (Unary, *shape.Shape, error) {
	return nil, nil, errors.Errorf("operator %q supported", op.String())
}

// Cast an array to another
func (boolFactory) Cast(dtype.DataType, []int) (Unary, *shape.Shape, Factory, error) {
	return nil, nil, nil, errors.Errorf("not implemented")
}

// ToBoolAtom converts a value into an atom owned by a backend.
func ToBoolAtom(val bool) *ArrayT[bool] {
	return &ArrayT[bool]{
		factory: boolFactory{},
		shape:   &shape.Shape{DType: dtype.Bool},
		values:  []bool{val},
	}
}

// ToBoolArray converts values and a shape into a native multi-dimensional array owned by a backend.
func ToBoolArray(values []bool, dims []int) *ArrayT[bool] {
	return &ArrayT[bool]{
		factory: boolFactory{},
		shape: &shape.Shape{
			DType:       dtype.Bool,
			AxisLengths: dims,
		},
		values: values,
	}
}
