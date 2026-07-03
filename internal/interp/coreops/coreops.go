// Copyright 2026 Google LLC
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

// Package coreops provides core operators for numerical elements.
package coreops

import (
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

// Element returned after an evaluation at compeval.
type Element interface {
	engine.NumericalElement
	canonical.Comparable
	ir.StringSourcer

	ShortString() string

	// CanonicalExpr returns the canonical expression used for comparison.
	CanonicalExpr() canonical.Canonical
}

type releaseFunc func()

func toKernelArray(array *values.HostArray) (kernels.Array, releaseFunc, error) {
	// Convert the GX value into a Go array with a kernel factory.
	data := array.Buffer().Acquire()
	kArray, err := kernels.NewArrayFromRaw(data, array.Shape())
	if err != nil {
		array.Buffer().Release()
		return nil, nil, err
	}
	return kArray, array.Buffer().Release, nil
}

func valEqual(x, y Element) (bool, error) {
	xEl, err := elements.ConstantFromElement(x)
	if err != nil {
		return false, err
	}
	if xEl == nil {
		return false, nil
	}
	yEl, err := elements.ConstantFromElement(y)
	if err != nil {
		return false, err
	}
	if yEl == nil {
		return false, nil
	}
	return EqualArray(xEl, yEl), nil
}

// EqualArray returns true if two arrays are equal.
func EqualArray(x, y *values.HostArray) bool {
	if !x.Shape().Equal(y.Shape()) {
		return false
	}
	xBuf := x.Buffer()
	yBuf := y.Buffer()
	xData := xBuf.Acquire()
	defer xBuf.Release()
	yData := yBuf.Acquire()
	defer yBuf.Release()
	for i, xi := range xData {
		if yData[i] != xi {
			return false
		}
	}
	return true
}

// AxesFromType returns a slice element of axis lengths from an array type.
func AxesFromType(ev ir.Evaluator, typ ir.Type) (*elements.Slice, error) {
	aTyp, ok := typ.(ir.ArrayType)
	if !ok {
		return nil, nil
	}
	rank := aTyp.Rank()
	axes := rank.Axes()
	elts := make([]ir.Element, len(axes))
	for i, ax := range axes {
		var err error
		elts[i], err = ev.EvalExpr(ax.AsExpr())
		if err != nil {
			return nil, err
		}
	}
	return elements.NewSlice(ir.IntSliceType(), elts)
}
