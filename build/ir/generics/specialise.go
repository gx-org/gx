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
	axes    map[string]ir.Element
	defined map[string]ir.Type
}

func (s *specialiser) TypeOf(name string) ir.Type {
	return s.defined[name]
}

func (s *specialiser) ValueOf(name string) ir.Element {
	return s.axes[name]
}

// Specialise a function signature for a given type.
func Specialise(fetcher ir.Fetcher, expr ir.Expr, fun *ir.FuncValExpr, typs []*ir.TypeValExpr) (*ir.FuncValExpr, bool) {
	fType := fun.T
	if fType == nil {
		// This is a builtin function with the type built later.
		// That should not be specialised by the user.
		return nil, fetcher.Err().Appendf(expr.Source(), "builtin function does not support type arguments")
	}
	typeParams := fType.TypeParams.Fields()
	gotN, wantN := len(typs), len(typeParams)
	if gotN > wantN {
		return nil, fetcher.Err().Appendf(expr.Source(), "got %d type arguments but want %d", gotN, wantN)
	}
	definedTypeParams := make(map[string]ir.Type)
	ok := true
	for i, typeValExpr := range typs {
		typeParam := typeParams[i]
		if !ir.ValidName(typeParam.Name.Name) {
			continue
		}
		gotType, wantType := typeValExpr.Typ, typeParam.Group.Type.Typ
		assignedOk, err := gotType.AssignableTo(fetcher, wantType)
		if err != nil {
			ok = fetcher.Err().Append(err)
			continue
		}
		if !assignedOk {
			ok = fetcher.Err().Appendf(expr.Source(), "%s does not satisfy %s", ir.TypeString(gotType), ir.TypeString(wantType))
			continue
		}
		definedTypeParams[typeParam.Name.Name] = typeValExpr.Typ
	}
	if !ok {
		return nil, false
	}
	specType, err := fType.SpecialiseFType(&specialiser{
		Fetcher: fetcher,
		defined: definedTypeParams,
	})
	if err != nil {
		return nil, fetcher.Err().AppendAt(fun.X.Source(), err)
	}
	if specType == nil {
		return nil, false
	}
	return &ir.FuncValExpr{
		X: expr,
		F: fun.F,
		T: specType,
	}, ok
}

// Instantiate replaces data types either specified or inferred.
func Instantiate(fetcher ir.Fetcher, ftype *ir.FuncType) (*ir.FuncType, error) {
	return ftype.SpecialiseFType(&specialiser{
		Fetcher: fetcher,
		defined: newTypeParamDefinition(ftype),
		axes:    newAxisLengthsDefinition(ftype),
	})
}
