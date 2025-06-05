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

type (
	// MathFactory returns a factory to implement functions from the math package.
	MathFactory interface {
		// Kernelize turns a unary Go math function into a unary kernel.
		Kernelize(func(float64) float64) Unary
	}

	mathFactory[T goAlgebra] struct{}
)

func (m mathFactory[T]) Kernelize(f func(float64) float64) Unary {
	return func(xVal Array) (Array, error) {
		x := toArray[T](xVal)
		z := make([]T, x.shape.Size())
		for i, xi := range x.values {
			z[i] = T(f(float64(xi)))
		}
		return xVal.newArray(&arrayT[T]{
			factory: x.factory,
			shape:   x.shape,
			values:  z,
		}), nil
	}
}
