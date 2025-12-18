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

package ir

import (
	"go/ast"
	"slices"
)

// AxisValue assigns a value to an axis length.
type AxisValue struct {
	Axis  *AxisStmt
	Value Element
}

// Name of the axis length.
func (ax *AxisValue) Name() string {
	return ax.Axis.NameDef().Name
}

type (
	// TypeParamValue assigns a type to a field of a more generic type.
	TypeParamValue struct {
		Field *Field
		Typ   Type
	}

	// FuncType defines a function signature.
	FuncType struct {
		BaseType[*ast.FuncType]

		Receiver   *FieldList
		TypeParams *FieldList
		Params     *FieldList
		Results    *FieldList

		AxisLengths      []AxisValue
		TypeParamsValues []TypeParamValue

		// CompEval is set to true if the function can be called at compilation time.
		CompEval bool
	}
)

var _ Type = (*FuncType)(nil)

func (*FuncType) node() {}

// Kind returns the function kind.
func (s *FuncType) Kind() Kind { return FuncKind }

// Equal returns true if other is the same type.
func (s *FuncType) Equal(fetcher Fetcher, other Type) (bool, error) {
	otherT, ok := other.(*FuncType)
	if !ok {
		return false, nil
	}
	return s.equal(fetcher, otherT)
}

// Equal returns true if other is the same type.
func (s *FuncType) equal(fetcher Fetcher, other *FuncType) (bool, error) {
	if s == other {
		return true, nil
	}
	recvOk, err := s.Receiver.Type().Equal(fetcher, other.Receiver.Type())
	if err != nil {
		return false, err
	}
	paramsOk, err := s.Params.Type().Equal(fetcher, other.Params.Type())
	if err != nil {
		return false, err
	}
	resultsOk, err := s.Results.Type().Equal(fetcher, other.Results.Type())
	if err != nil {
		return false, err
	}
	return recvOk && paramsOk && resultsOk, nil
}

// ReceiverField returns a field representing the receiver of the function, or nil if the function has no receiver.
func (s *FuncType) ReceiverField() *Field {
	if s == nil || s.Receiver == nil { // The function type can be nil for builtin functions.
		return nil
	}
	grp := s.Receiver.List[0]
	if len(grp.Fields) == 0 {
		return &Field{Group: grp}
	}
	return grp.Fields[0]
}

// AssignableTo reports if the type can be assigned to other.
func (s *FuncType) AssignableTo(fetcher Fetcher, other Type) (bool, error) {
	otherT, ok := other.(*FuncType)
	if ok {
		return s.equal(fetcher, otherT)
	}
	aFrom, ok := other.(assignsFrom)
	if !ok {
		return false, nil
	}
	return aFrom.assignableFrom(fetcher, s)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *FuncType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	return s.Equal(fetcher, target)
}

// Value returns a value pointing to the receiver.
func (s *FuncType) Value(x Expr) AssignableExpr {
	return &TypeValExpr{X: x, Typ: s}
}

// Specialise a type to a given target.
func (s *FuncType) Specialise(spec Specialiser) (Type, error) {
	return s.SpecialiseFType(spec)
}

func skipIfDefined(spec Specialiser) fieldCloner {
	return func(grp *FieldGroup, i int, field *Field) (*Field, error) {
		if spec.TypeOf(field.Name.Name) != nil {
			return nil, nil
		}
		return cloneField(grp, i, field)
	}
}

// SpecialiseFType specialises a function type.
func (s *FuncType) SpecialiseFType(spec Specialiser) (*FuncType, error) {
	res := *s
	var err error
	specialiser := &cloner{
		group: func(grp *FieldGroup) (*FieldGroup, error) {
			specType, err := grp.Type.Typ.Specialise(spec)
			if err != nil {
				return nil, err
			}
			return &FieldGroup{
				Src: grp.Src,
				Type: &TypeValExpr{
					X:   grp.Type.X,
					Typ: specType,
				},
			}, nil
		},
		field: cloneField,
	}
	res.Params, err = cloneFields(s.Params, specialiser)
	if err != nil {
		return nil, err
	}
	res.Results, err = cloneFields(s.Results, specialiser)
	if err != nil {
		return nil, err
	}
	res.TypeParams, err = cloneFields(s.TypeParams, &cloner{
		group: cloneGroup,
		field: skipIfDefined(spec),
	})
	if err != nil {
		return nil, err
	}

	for _, typeParam := range s.TypeParams.Fields() {
		defined := spec.TypeOf(typeParam.Name.Name)
		if defined == nil {
			break
		}
		res.TypeParamsValues = append(res.TypeParamsValues, TypeParamValue{
			Field: typeParam,
			Typ:   defined,
		})
	}
	res.AxisLengths = slices.Clone(s.AxisLengths)
	for i, axis := range s.AxisLengths {
		res.AxisLengths[i].Value = spec.ValueOf(axis.Name())
	}
	return &res, nil
}

// UnifyWith recursively unifies a type parameters with types.
func (s *FuncType) UnifyWith(unifier Unifier, typ Type) bool {
	return true
}

// Source returns the node in the AST tree.
func (s *FuncType) Source() ast.Node { return s.Src }
