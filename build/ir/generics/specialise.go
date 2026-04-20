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

package generics

import (
	"github.com/gx-org/gx/build/ir"
)

type specialiser struct {
	ir.Fetcher
	fun     ir.Func
	defined map[string]ir.Type
	axes    map[string]ir.Element
}

func newSpecialiser(fetcher ir.Fetcher, fun ir.Func, defined map[string]ir.Type, axes map[string]ir.Element) *specialiser {
	return &specialiser{
		Fetcher: fetcher,
		fun:     fun,
		defined: defined,
		axes:    axes,
	}
}

func (s *specialiser) TypeOf(name string) ir.Type {
	return s.defined[name]
}

func (s *specialiser) ValueOf(name string) ir.Element {
	return s.axes[name]
}

func toTypeValue(fetcher ir.Fetcher, typeParam *ir.Field, x ir.Expr) (ir.Type, bool) {
	typeValExpr := ir.TypeFromExpr(x)
	if typeValExpr == nil {
		return ir.InvalidType(), fetcher.Err().Appendf(x.Node(), "%s is not a type", x.SourceString(fetcher.File()))
	}
	gotType, wantType := typeValExpr.Val(), typeParam.Group.Type.Val()
	assignedOk, cpErr, err := gotType.AssignableTo(fetcher, wantType)
	if err != nil {
		return ir.InvalidType(), fetcher.Err().AppendAt(x.Node(), err)
	}
	if cpErr != nil {
		return ir.InvalidType(), fetcher.Err().AppendAt(x.Node(), cpErr)
	}
	if !assignedOk {
		return ir.InvalidType(), fetcher.Err().Appendf(x.Node(), "%s does not satisfy %s", gotType.ReferString(fetcher.File()), wantType.ReferString(fetcher.File()))
	}
	return typeValExpr.Val(), true
}

// Specialise a function signature for a given type.
func Specialise(fetcher ir.Fetcher, expr ir.Expr, fun *ir.FuncValExpr, typs []ir.Expr) (*ir.FuncValExpr, bool) {
	fType := fun.FuncType()
	if fType == nil {
		// This is a builtin function with the type built later.
		// That should not be specialised by the user.
		return nil, fetcher.Err().Appendf(expr.Node(), "builtin function does not support type arguments")
	}
	typeParams := fType.TypeParams.Fields()
	gotN, wantN := len(typs), len(typeParams)
	if gotN > wantN {
		return nil, fetcher.Err().Appendf(expr.Node(), "got %d type arguments but want %d", gotN, wantN)
	}
	definedTypeParams := make(map[string]ir.Type)
	ok := true
	for i, typeParam := range typeParams {
		if i >= len(typs) {
			// Not all type parameters are defined as in:
			// f[float32]() for f[T, U floats]()
			break
		}
		typeValExpr := typs[i]
		if !ir.ValidName(typeParam.Name.Name) {
			continue
		}
		paramValue, paramOk := toTypeValue(fetcher, typeParam, typeValExpr)
		definedTypeParams[typeParam.Name.Name] = paramValue
		ok = ok && paramOk
	}
	if !ok {
		return nil, false
	}
	spec := newSpecialiser(fetcher, fun.Func(), definedTypeParams, nil)
	specType, cpErr, err := fType.SpecialiseFType(spec)
	if cpErr != nil {
		return nil, fetcher.Err().AppendAt(fun.Node(), cpErr)
	}
	if err != nil {
		return nil, fetcher.Err().AppendAt(fun.Node(), err)
	}
	if specType == nil {
		return nil, false
	}
	return ir.NewFuncValExpr(expr, fun.Func()).NewFType(specType), ok
}

// Instantiate replaces data types either specified or inferred.
func Instantiate(fetcher ir.Fetcher, fexpr *ir.FuncValExpr) (*ir.FuncType, ir.CompEvalError, error) {
	ftype := fexpr.FuncType()
	defined := newTypeParamDefinition(ftype)
	axes := newAxisLengthsDefinition(ftype)
	spec := newSpecialiser(fetcher, fexpr.Func(), defined, axes)
	return ftype.SpecialiseFType(spec)
}
