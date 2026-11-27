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

package reflect

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

func evalArrays(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	if len(genArgs) != 2 {
		return nil, fmterr.Internalf("incorrect number of generic arguments: got %d but want 2", len(genArgs))
	}
	genType, err := cast.To[*elements.GenericType](ir.BareValue(genArgs[0]))
	if err != nil {
		return nil, err
	}
	nums, err := cast.To[*elements.GenericType](ir.BareValue(genArgs[1]))
	if err != nil {
		return nil, err
	}
	return findLeaves(env, call, genType.Value(), nums.Value(), checkFieldType)
}

func checkFieldType(env *engine.Env, field *ir.Field, nums ir.Type) bool {
	fieldType := field.Type()
	satisfied := false
	arrayType := ir.ToArrayType(fieldType)
	if arrayType == nil {
		return false
	}
	tpcmp, isTypeEval := env.ExprEval().(ir.TypeEval)
	if !isTypeEval {
		return false
	}
	satisfied, err := ir.AssignableTo(tpcmp, arrayType.DataType(), nums)
	return satisfied && err == nil
}

// keepFunc returns true if the path to a field is included in the result.
// nums is the type argument constraining the data type of the fields.
type keepFunc func(env *engine.Env, field *ir.Field, nums ir.Type) bool

func findLeaves(env *engine.Env, call *ir.FuncCallExpr, defType, nums ir.Type, keep keepFunc) ([]ir.Element, error) {
	under := ir.Underlying(defType)
	structType, ok := under.(*ir.StructType)
	if !ok {
		return nil, errors.Errorf("type %s (kind: %s) not a structure", defType.ReferString(env.File()), defType.Kind().String())
	}
	root := storepath.NewProxy()
	paths, err := parseStruct(env, root, structType, nums, keep)
	if err != nil {
		return nil, err
	}
	slice, err := elements.NewSlice(call.Type(), paths)
	if err != nil {
		return nil, err
	}
	return []ir.Element{slice}, nil
}

func parseStruct(env *engine.Env, prefix storepath.Path, tp *ir.StructType, nums ir.Type, keep keepFunc) ([]ir.Element, error) {
	var leaves []ir.Element
	for _, field := range tp.Fields.Fields() {
		if keep(env, field, nums) {
			leaves = append(leaves, elements.NewFieldPath(prefix, field))
		}
		subStruct, isStruct := ir.Underlying(field.Type()).(*ir.StructType)
		if !isStruct {
			continue
		}
		subLeaves, err := parseStruct(env, storepath.NewSelect(prefix, field), subStruct, nums, keep)
		if err != nil {
			return nil, err
		}
		leaves = append(leaves, subLeaves...)
	}
	return leaves, nil
}

func evalArrayFromPath(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	fieldIndex, err := cast.To[engine.FieldIndex](args[1])
	if err != nil {
		return nil, err
	}
	arrayEl, err := fieldIndex.ArrayCast(call, ir.Float32Type())
	if err != nil {
		return nil, err
	}
	return []ir.Element{arrayEl}, nil
}
