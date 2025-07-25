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

// Infer the type parameters of a function given a list of argument expressions.
func Infer(fetcher ir.Fetcher, fExpr *ir.FuncValExpr, args []ir.AssignableExpr) (*ir.FuncValExpr, bool) {
	fType := fExpr.T
	nameToTypeParam := make(map[string]ir.Type)
	for _, typeParam := range fType.TypeParams.Fields() {
		nameToTypeParam[typeParam.Name.Name] = typeParam.Type()
	}
	if len(nameToTypeParam) == 0 {
		// Nothing to infer.
		return fExpr, true
	}
	nameToType := make(map[string]ir.Type)
	inferredOk := true
	for i, param := range fType.Params.Fields() {
		arg := args[i]
		if ir.IsNumber(arg.Type().Kind()) {
			continue
		}
		gType := extractTypeParamName(param.Group)
		if gType == nil {
			continue
		}
		assigned := nameToType[gType.name()]
		if assigned != nil {
			assignedOk, err := assigned.Equal(fetcher, arg.Type())
			if err != nil {
				inferredOk = fetcher.Err().Append(err)
				continue
			}
			if !assignedOk {
				inferredOk = fetcher.Err().Appendf(arg.Source(), "type %s of %s does not match inferred type %s for %s", arg.Type(), arg.String(), assigned.String(), gType)
				continue
			}
		}
		targetType := nameToTypeParam[gType.name()]
		if targetType == nil {
			continue
		}
		canAssign, err := arg.Type().AssignableTo(fetcher, targetType)
		if err != nil {
			inferredOk = fetcher.Err().Append(err)
		}
		if !canAssign {
			inferredOk = fetcher.Err().Appendf(arg.Source(), "%s does not satisfy %s", ir.TypeString(arg.Type()), ir.TypeString(targetType))
		}
		nameToType[gType.name()] = arg.Type()
	}
	if !inferredOk {
		return fExpr, false
	}
	if len(nameToType) == 0 {
		return fExpr, true
	}
	specialised, ok := specialise(fetcher, fExpr.T, nameToType)
	return &ir.FuncValExpr{
		X: fExpr.X,
		F: fExpr.F,
		T: specialised,
	}, ok
}
