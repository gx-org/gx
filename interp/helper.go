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

package interp

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/numbers"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

// HostArrayToElement converts a host array argument into a scalar or nested slice of constant elements.
func HostArrayToElement(eng engine.Engine, ev ir.Evaluator, field *ir.Field, array hostio.Array) (ir.Element, error) {
	hostArray, err := array.ToHostArray()
	if err != nil {
		return nil, err
	}
	return convertFlatToElement(eng, ev, field, field.Type(), hostArray.Shape().Dimensions, reflect.ValueOf(hostArray.FlatData()))
}

func convertFlatToElement(eng engine.Engine, ev ir.Evaluator, field *ir.Field, typ ir.Type, dims []int, flat reflect.Value) (ir.Element, error) {
	if len(dims) == 0 {
		cst, err := numbers.NewConstant(typ, flat.Index(0).Interface())
		if err != nil {
			return nil, err
		}
		return eng.ElementFromHostValue(ev, cst)
	}
	sliceType, isSlice := typ.(*ir.SliceType)
	if !isSlice {
		return nil, errors.Errorf("cannot convert array argument to %s for parameter %s", typ.ReferString(nil), field.Name.Name)
	}
	subType, ok := sliceType.ElementType()
	if !ok {
		return nil, errors.Errorf("cannot get element type of %s for parameter %s", typ.ReferString(nil), field.Name.Name)
	}
	numEls := dims[0]
	subDims := dims[1:]
	stride := 1
	for _, dim := range subDims {
		stride *= dim
	}
	els := make([]ir.Element, numEls)
	for i := range numEls {
		var err error
		els[i], err = convertFlatToElement(eng, ev, field, subType, subDims, flat.Slice(i*stride, (i+1)*stride))
		if err != nil {
			return nil, err
		}
	}
	return elements.NewSlice(typ, els)
}

// PkgFuncFromElement extracts a function declaration from an element.
func PkgFuncFromElement(el ir.Element) (ir.PkgFunc, error) {
	fIR, err := FuncDeclFromElement(el)
	if err != nil {
		return nil, err
	}
	return fIR, nil
}

// FuncDeclFromElement extracts a function declaration from an element.
func FuncDeclFromElement(el ir.Element) (ir.PkgFunc, error) {
	fEl, ok := el.(engine.Func)
	if !ok {
		return nil, errors.Errorf("%s is not a function", el.Type().ReferString(nil))
	}
	fIR, ok := fEl.FuncIR().(ir.PkgFunc)
	if !ok {
		return nil, errors.Errorf("%s is not a package level function", fEl.FuncIR().ReferString(nil))
	}
	return fIR, nil
}
