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
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/constants"
	"github.com/gx-org/gx/internal/interp/coreiface"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

func mergeSubs(els []engine.NumericalElement) ([]engine.AtomConstant, bool) {
	var all []engine.AtomConstant
	for _, el := range els {
		cst, isConstant := el.(engine.ConstantElement)
		if !isConstant {
			return nil, false
		}
		elVals, ok := constants.ArrayElements(cst.Constant())
		if !ok {
			return nil, false
		}
		all = append(all, elVals...)
	}
	return all, true
}

func buildStaticNonZeroArray(fitp *Interpreter, lit *ir.ArrayLitExpr, axes, vals []engine.NumericalElement) (engine.Constant, bool, error) {
	var valsT []engine.AtomConstant
	var err error
	var ok bool
	if len(axes) == 1 {
		// Literal of atomic elements.
		valsT, ok = coreiface.MapSliceOk(func(el ir.Element) (engine.AtomConstant, bool) {
			cst, isConstant := el.(engine.ConstantElement)
			if !isConstant {
				return nil, false
			}
			var atom engine.AtomConstant
			atom, err = coreiface.Cast[engine.AtomConstant](cst.Constant())
			if err != nil {
				return nil, false
			}
			return atom, true
		}, vals)
	} else {
		// Literal of other composite literals.
		valsT, ok = mergeSubs(vals)
	}
	if !ok || err != nil {
		return nil, false, err
	}
	axesInt, err := coreiface.MapSlice(elements.IntFromElement, axes)
	if err != nil {
		return nil, false, err
	}
	size := 1
	for _, axis := range axesInt {
		size *= int(axis)
	}
	if len(vals) > 0 && len(valsT) != size {
		return nil, false, errors.Errorf("array has dimensions %v (size=%d) but has %d elements", axes, size, len(vals))
	}
	return constants.NewArray(lit, valsT, axes), true, nil

}

func buildStaticArray(fitp *Interpreter, lit *ir.ArrayLitExpr, axes, vals []engine.NumericalElement) (ir.Element, bool, error) {
	var array engine.Constant
	static := true
	var err error
	if len(vals) > 0 {
		array, static, err = buildStaticNonZeroArray(fitp, lit, axes, vals)
	} else {
		array = constants.NewArray(lit, nil, axes)
	}
	if err != nil {
		return nil, false, err
	}
	if !static {
		// Not all elements of the array are known.
		return nil, false, nil
	}
	node, err := fitp.Engine().ArrayOps().ElementFromHostValue(fitp.env.ExprEval(), array)
	return node, true, err
}

func evalArrayLiteral(fitp *Interpreter, lit *ir.ArrayLitExpr) (ir.Element, error) {
	axes, err := evalArrayAxes(fitp, lit, lit.Typ)
	if err != nil {
		return nil, err
	}
	irVals := lit.Values()
	elVals := make([]engine.NumericalElement, len(irVals))
	for i, expr := range irVals {
		elVals[i], err = evalNumExpr(fitp, expr)
		if err != nil {
			return nil, err
		}
	}
	staticArray, staticOk, err := buildStaticArray(fitp, lit, axes, elVals)
	if staticOk || err != nil {
		return staticArray, err
	}
	// Some values will be known at runtime. We create one node for each element
	// and concatenates everything into an array.
	array1d, err := fitp.Engine().ArrayOps().Concat(fitp, lit, elVals)
	if err != nil {
		return nil, err
	}
	if len(axes) == 1 {
		return array1d, nil
	}
	return array1d.Reshape(fitp.env, lit, axes)
}
