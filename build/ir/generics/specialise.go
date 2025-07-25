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

func specialiseGroupType(nameToType map[string]ir.Type) groupCloner {
	return func(fetcher ir.Fetcher, group *ir.FieldGroup) *ir.FieldGroup {
		ext := &ir.FieldGroup{
			Src:  group.Src,
			Type: group.Type,
		}
		gType := extractTypeParamName(group)
		if gType == nil {
			return ext
		}
		replaceWith, ok := nameToType[gType.name()]
		if !ok {
			return ext
		}
		specialisedType := gType.specialise(fetcher, replaceWith)
		if specialisedType == nil {
			return nil
		}
		if !ok {
			return nil
		}
		ext.Type = &ir.TypeValExpr{X: group.Type.X, Typ: specialisedType}
		return ext
	}
}

func cloneGroup(_ ir.Fetcher, grp *ir.FieldGroup) *ir.FieldGroup {
	return &ir.FieldGroup{Src: grp.Src, Type: grp.Type}
}

func cloneField(_ ir.Fetcher, grp *ir.FieldGroup, field *ir.Field) (*ir.Field, bool) {
	return &ir.Field{Name: field.Name, Group: grp}, true
}

func skipField(nameToType map[string]ir.Type) fieldCloner {
	return func(fetcher ir.Fetcher, grp *ir.FieldGroup, field *ir.Field) (*ir.Field, bool) {
		if _, done := nameToType[field.Name.Name]; !done {
			return cloneField(fetcher, grp, field)
		}
		return nil, true
	}
}

func specialise(fetcher ir.Fetcher, fType *ir.FuncType, nameToType map[string]ir.Type) (*ir.FuncType, bool) {
	typeParams, ok := cloneFields(fetcher, fType.TypeParams, cloneGroup, skipField(nameToType))
	if !ok {
		return fType, false
	}
	params, ok := cloneFields(fetcher, fType.Params, specialiseGroupType(nameToType), cloneField)
	if !ok {
		return fType, false
	}
	results, ok := cloneFields(fetcher, fType.Results, specialiseGroupType(nameToType), cloneField)
	if !ok {
		return fType, false
	}
	return &ir.FuncType{
		BaseType:         fType.BaseType,
		Receiver:         fType.Receiver,
		TypeParams:       typeParams,
		TypeParamsValues: append([]ir.TypeParamValue{}, fType.TypeParamsValues...),
		CompEval:         fType.CompEval,
		Params:           params,
		Results:          results,
	}, true
}

// Specialise a function signature for a given type.
func Specialise(fetcher ir.Fetcher, expr ir.Expr, fun *ir.FuncValExpr, typs []*ir.TypeValExpr) (*ir.SpecialisedFunc, bool) {
	fType := fun.T
	if fType == nil {
		// This is a builtin function with the type builtin later.
		// That should not be specialised by the user.
		return nil, fetcher.Err().Appendf(expr.Source(), "builtin function does not support type arguments")
	}
	gotN, wantN := len(typs), fType.TypeParams.Len()
	if gotN > wantN {
		return nil, fetcher.Err().Appendf(expr.Source(), "got %d type arguments but want %d", gotN, wantN)
	}
	nameToType := make(map[string]ir.Type)
	typeParamFields := fType.TypeParams.Fields()
	ok := true
	for i, typeArgRef := range typs {
		typeParamField := typeParamFields[i]
		genericName := typeParamField.Name.Name
		if !ir.ValidName(genericName) {
			continue
		}
		typeArg := typeArgRef.Typ
		typeParam := typeParamField.Group.Type.Typ
		assignedOk, err := typeArgRef.Typ.AssignableTo(fetcher, typeParam)
		if err != nil {
			ok = fetcher.Err().Append(err)
			continue
		}
		if !assignedOk {
			ok = fetcher.Err().Appendf(expr.Source(), "%s does not satisfy %s", ir.TypeString(typeArg), ir.TypeString(typeParam))
			continue
		}
		nameToType[genericName] = typeArgRef.Typ
	}
	if !ok {
		return nil, false
	}
	specType, ok := specialise(fetcher, fType, nameToType)
	return &ir.SpecialisedFunc{
		X: expr,
		F: fun,
		T: specType,
	}, ok
}

type (
	groupCloner func(ir.Fetcher, *ir.FieldGroup) *ir.FieldGroup
	fieldCloner func(ir.Fetcher, *ir.FieldGroup, *ir.Field) (*ir.Field, bool)
)

func cloneFields(fetcher ir.Fetcher, list *ir.FieldList, groupFun groupCloner, fieldFun fieldCloner) (*ir.FieldList, bool) {
	if list == nil {
		return nil, true
	}
	ext := &ir.FieldList{Src: list.Src}
	ok := true
	for _, group := range list.List {
		cloneGroup := groupFun(fetcher, group)
		if cloneGroup == nil {
			ok = false
			continue
		}
		for _, field := range group.Fields {
			cloneField, ok := fieldFun(fetcher, cloneGroup, field)
			if !ok {
				ok = false
				continue
			}
			if cloneField == nil {
				continue
			}
			cloneField.Group = cloneGroup
			cloneGroup.Fields = append(cloneGroup.Fields, cloneField)
		}
		if len(cloneGroup.Fields) == 0 && len(group.Fields) > 0 {
			continue
		}
		ext.List = append(ext.List, cloneGroup)
	}
	return ext, ok
}
