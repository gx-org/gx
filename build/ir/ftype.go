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
	"fmt"
	"go/ast"
	"strings"

	"github.com/gx-org/gx/build/ir/irkind"
)

type (

	// FuncType defines a function signature.
	FuncType struct {
		BaseType[*ast.FuncType]
		origin *FuncType

		Receiver   *FieldList
		TypeParams *FieldList
		Params     *FieldList
		Results    *FieldList

		VarArgs *VarArgsType

		GenericValues []GenericValue

		// CompEval is set to true if the function can be called at compilation time.
		CompEval bool
	}
)

var _ Type = (*FuncType)(nil)

func (*FuncType) node() {}

// Kind returns the function kind.
func (s *FuncType) Kind() irkind.Kind { return irkind.Func }

// Equal returns true if other is the same type.
func (s *FuncType) Equal(tpcmp TypeCmp, other Type) (bool, error) {
	otherT, ok := other.(*FuncType)
	if !ok {
		return false, nil
	}
	return s.equal(tpcmp, otherT)
}

// Equal returns true if other is the same type.
func (s *FuncType) equal(tpcmp TypeCmp, other *FuncType) (bool, error) {
	if s == other {
		return true, nil
	}
	recvOk, err := s.Receiver.Type().Equal(tpcmp, other.Receiver.Type())
	if !recvOk || err != nil {
		return recvOk, err
	}
	return s.equalParamsResults(tpcmp, other)
}

func (s *FuncType) equalParamsResults(tpcmp TypeCmp, other *FuncType) (bool, error) {
	paramsOk, err := s.Params.Type().Equal(tpcmp, other.Params.Type())
	if err != nil {
		return false, err
	}
	resultsOk, err := s.Results.Type().Equal(tpcmp, other.Results.Type())
	if err != nil {
		return false, err
	}
	return paramsOk && resultsOk, nil
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
func (s *FuncType) AssignableTo(tpcmp TypeCmp, other Type) (bool, error) {
	otherT, ok := other.(*FuncType)
	if ok {
		return s.equal(tpcmp, otherT)
	}
	aFrom, ok := other.(assignsFrom)
	if !ok {
		return false, nil
	}
	return aFrom.assignableFrom(tpcmp, s)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *FuncType) ConvertibleTo(tpcmp TypeCmp, target Type) (bool, error) {
	return s.Equal(tpcmp, target)
}

// Value returns a value pointing to the receiver.
func (s *FuncType) Value(x Expr) Expr {
	return TypeExpr(x, s)
}

// Specialise a type to a given target.
func (s *FuncType) Specialise(spec Specialiser) (Type, bool) {
	return s.SpecialiseFType(spec, false)
}

// String representation of the function type (only used for debugging).
func (s *FuncType) String() string {
	return s.ReferString(nil)
}

func skipIfDefined(spec Specialiser) fieldCloner {
	return func(grp *FieldGroup, field *Field) *Field {
		if spec.IsDefined(field.Pos) {
			return nil
		}
		return cloneField(grp, field)
	}
}

func specialiseGroup(spec Specialiser, ok *bool) groupCloner {
	*ok = true
	return func(grp *FieldGroup) *FieldGroup {
		grpType, grpOk := grp.Type.Specialise(spec)
		*ok = *ok && grpOk
		return &FieldGroup{
			Src:  grp.Src,
			Type: grpType,
		}
	}
}

// Origin returns the original function type from which this type was defined, typically through specialisation.
// If origin is nil, then it return itself.
func (s *FuncType) Origin() *FuncType {
	if s.origin == nil {
		return s
	}
	return s.origin
}

func (s *FuncType) varArgs() *VarArgsType {
	if s.origin.VarArgs == nil {
		return nil
	}
	paramGroups := s.Params.List
	if len(paramGroups) == 0 {
		return nil
	}
	lastGroupType := paramGroups[len(paramGroups)-1].Type
	lastAsSlice, isSlice := lastGroupType.Val().(*SliceType)
	if !isSlice {
		return nil
	}
	va := *s.origin.VarArgs
	va.Typ = lastAsSlice
	return &va
}

// SpecialiseFType specialises a function type.
func (s *FuncType) SpecialiseFType(spec Specialiser, skipResult bool) (*FuncType, bool) {
	res := *s
	res.origin = s.Origin()
	res.TypeParams = cloneFields(s.TypeParams, &cloner{
		group: cloneGroup,
		field: skipIfDefined(spec),
	})
	res.GenericValues = spec.Values()
	var paramsOk bool
	res.Params = cloneFields(s.Params, &cloner{
		group: specialiseGroup(spec, &paramsOk),
		field: cloneField,
	})
	if skipResult {
		return &res, paramsOk
	}
	var resultsOk bool
	res.Results = cloneFields(s.Results, &cloner{
		group: specialiseGroup(spec, &resultsOk),
		field: cloneField,
	})
	res.VarArgs = res.varArgs()
	return &res, paramsOk && resultsOk
}

// ArgIndexToParamField returns the field corresponding to an argument index.
func (s *FuncType) ArgIndexToParamField(i int) (*Field, bool) {
	fields := s.Params.Fields()
	if i >= len(fields) {
		i = len(fields) - 1
	}
	return fields[i], s.VarArgs != nil && i+1 == len(fields)
}

// UnifyWith recursively unifies a type parameters with types.
func (s *FuncType) UnifyWith(unifier Unifier, typ Type) bool {
	return true
}

// Instantiate a function type.
func (s *FuncType) Instantiate(fetcher Fetcher, spec Specialiser) (Type, bool) {
	return s.InstantiateFType(fetcher, spec)
}

// InstantiateFType instantiate function type.
func (s *FuncType) InstantiateFType(fetcher Fetcher, spec Specialiser) (*FuncType, bool) {
	res := *s
	res.origin = s.Origin()
	var paramsOk bool
	res.Params = cloneFields(s.Params, &cloner{
		group: specialiseGroup(spec, &paramsOk),
		field: cloneField,
	})
	resultsOk := true
	res.Results = cloneFields(s.Results, &cloner{
		group: func(grp *FieldGroup) *FieldGroup {
			grp = cloneGroup(grp)
			var grpOk bool
			grp.Type, grpOk = grp.Type.Instantiate(fetcher, spec)
			resultsOk = resultsOk && grpOk
			return grp
		},
		field: cloneField,
	})
	res.TypeParams = cloneFields(s.TypeParams, &cloner{
		group: cloneGroup,
		field: cloneField,
	})
	res.VarArgs = res.varArgs()
	return &res, paramsOk && resultsOk
}

// IndexForVarArgs returns a type specific to a given index in varargs.
func (s *FuncType) IndexForVarArgs(ErrSource, int) (Type, bool) {
	return s, true
}

// Expr returns the expression AST.
func (s *FuncType) Expr() ast.Expr { return s.Src }

// Node returns the node in the AST tree.
func (s *FuncType) Node() ast.Node { return s.Src }

// DefineString returns a string representation of the signature of a function.
func (s *FuncType) DefineString(from *File) string {
	return s.SourceSignature(from, nil)
}

// ReferString returns a string representation of the signature of a function.
func (s *FuncType) ReferString(from *File) string {
	return s.SourceSignature(from, nil)
}

// SourceSignature returns a string representation of a signature given a name.
// The name can be empty.
func (s *FuncType) SourceSignature(from *File, name *ast.Ident) string {
	nameS := ""
	if name != nil {
		nameS = name.Name
	}
	return s.sourceSignature(from, nameS, false)
}

func (s *FuncType) specializeString(from *File) string {
	var b strings.Builder
	for _, gval := range s.GenericValues {
		if gval == nil {
			continue
		}
		fmt.Fprintf(&b, "%s", gval.SourceString(from))
	}
	if s.TypeParams != nil && s.TypeParams.Len() > 0 {
		fmt.Fprintf(&b, "[%s]", s.TypeParams.SourceString(from))
	}
	return b.String()
}

// SourceSignature returns a string representation of a signature given a name.
// The name can be empty.
func (s *FuncType) sourceSignature(from *File, name string, skipRecv bool) string {
	var b strings.Builder
	b.WriteString("func")
	if s.Receiver != nil && !skipRecv {
		fmt.Fprintf(&b, " (%s)", s.Receiver.SourceString(from))
	}
	if name != "" {
		fmt.Fprintf(&b, " %s", name)
	}
	b.WriteString(s.specializeString(from))
	b.WriteString("(" + s.Params.SourceString(from) + ")")
	if s.Results.Len() == 0 {
		return b.String()
	}
	b.WriteRune(' ')
	if s.Results.Len() > 1 {
		b.WriteString("(")
	}
	b.WriteString(s.Results.SourceString(from))
	if s.Results.Len() > 1 {
		b.WriteString(")")
	}
	return b.String()
}

// SourceString returns GX code defining the function type.
func (s *FuncType) SourceString(from *File) string {
	return s.SourceSignature(from, nil)
}
