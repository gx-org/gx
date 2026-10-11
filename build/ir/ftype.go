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
	"strings"

	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/src/gx/ast"
)

type (
	// FuncNature is the nature of a function.
	FuncNature struct {
		// CompEval is set to true if the function can be called at compilation time.
		CompEval bool
		// Unroll indicates to the compiler to unroll the for-loop iterating over the function.
		// This forces the function to be used only as an iterator.
		Unroll bool
		// Unknown indicates is not known at compile time.
		// Disable compile checks for the function.
		Unknown bool
	}

	// FuncType defines a function signature.
	FuncType struct {
		BaseType[*ast.FuncType]
		Nature FuncNature
		origin *FuncType

		GenParams GenericParams
		Receiver  *FieldList
		Params    *FieldList
		Results   *FieldList

		VarArgs *VarArgsType
	}
)

var _ GenericType = (*FuncType)(nil)

func (*FuncType) node() {}

// Kind returns the function kind.
func (s *FuncType) Kind() irkind.Kind { return irkind.Func }

// Refer to the function type.
func (s *FuncType) Refer(file *File) ast.Expr {
	return s.Src
}

// Generics returns the generic values.
func (s *FuncType) Generics() *GenericParams {
	return &s.GenParams
}

// Equal returns true if other is the same type.
func (s *FuncType) Equal(tpcmp TypeEval, other Type) (bool, error) {
	otherT, ok := other.(*FuncType)
	if !ok {
		return false, nil
	}
	return s.equal(tpcmp, otherT)
}

// Equal returns true if other is the same type.
func (s *FuncType) equal(tpcmp TypeEval, other *FuncType) (bool, error) {
	if s == other {
		return true, nil
	}
	recvOk, err := s.Receiver.Type().Equal(tpcmp, other.Receiver.Type())
	if !recvOk || err != nil {
		return recvOk, err
	}
	return s.equalParamsResults(tpcmp, other)
}

func (s *FuncType) equalParamsResults(tpcmp TypeEval, other *FuncType) (bool, error) {
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
func (s *FuncType) AssignableTo(tpcmp TypeEval, other Type) (bool, error) {
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
func (s *FuncType) ConvertibleTo(tpcmp TypeEval, target Type) (bool, error) {
	return s.Equal(tpcmp, target)
}

// Value returns a value pointing to the receiver.
func (s *FuncType) Value(x Expr) Expr {
	return TypeExpr(x.Expr(), s)
}

// Specialise a type to a given target.
func (s *FuncType) Specialise(spec Specialiser) (Type, bool) {
	return s.SpecialiseFType(spec)
}

// AllParams returns a list of all the parameters of the function, including non-type generic parameters.
func (s *FuncType) AllParams() []*Field {
	all := s.Params.Fields()
	// Fields already create a new slice. So we can reuse it.
	return append(all, s.GenParams.Fields.Fields()...)
}

// NumAllParams returns the number of parameters of the function
// including generic parameters.
func (s *FuncType) NumAllParams() int {
	return s.Params.Len() + s.GenParams.Fields.Len()
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

// originFType returns the original function type from which this type was defined, typically through specialisation.
// If origin is nil, then it return itself.
func (s *FuncType) originFType() *FuncType {
	if s.origin == nil {
		return s
	}
	return s.origin
}

// Origin generic type.
func (s *FuncType) Origin() GenericType {
	return s.originFType()
}

// Unroll the type.
func (s *FuncType) Unroll(ev Fetcher, urlr Unroller) (ast.Expr, bool) {
	return s.Src, true
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
func (s *FuncType) SpecialiseFType(spec Specialiser) (*FuncType, bool) {
	res := *s
	res.origin = s.originFType()
	res.GenParams.Fields = cloneFields(s.GenParams.Fields, &cloner{
		group: cloneGroup,
		field: skipIfDefined(spec),
	})
	res.GenParams.Values = spec.Values()
	var paramsOk bool
	res.Params = cloneFields(s.Params, &cloner{
		group: specialiseGroup(spec, &paramsOk),
		field: cloneField,
	})
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
	if s.Params == nil {
		return nil, false
	}
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
func (s *FuncType) Instantiate(fetcher Fetcher, spec Instantiator) (Type, bool) {
	return s.InstantiateFType(fetcher, spec)
}

func instantiateGroup(fetcher Fetcher, spec Instantiator, ok *bool) func(grp *FieldGroup) *FieldGroup {
	return func(grp *FieldGroup) *FieldGroup {
		grp = cloneGroup(grp)
		var grpOk bool
		grp.Type, grpOk = grp.Type.Instantiate(fetcher, spec)
		*ok = *ok && grpOk
		return grp
	}
}

// InstantiateFType instantiate function type.
func (s *FuncType) InstantiateFType(fetcher Fetcher, spec Instantiator) (*FuncType, bool) {
	res := *s
	res.origin = s.originFType()
	var paramsOk bool
	paramGroup := specialiseGroup(spec, &paramsOk)
	if spec.InstantiantingResults() {
		paramGroup = instantiateGroup(fetcher, spec, &paramsOk)
	}
	res.Params = cloneFields(s.Params, &cloner{
		group: paramGroup,
		field: cloneField,
	})
	resultsOk := true
	spec.StartResults(s)
	res.Results = cloneFields(s.Results, &cloner{
		group: instantiateGroup(fetcher, spec, &resultsOk),
		field: cloneField,
	})
	spec.StopResults(s)
	res.GenParams.Fields = cloneFields(s.GenParams.Fields, &cloner{
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
	return s.sourceSignature(from, "", false, true)
}

// SourceSignature returns a string representation of a signature given a name.
// The name can be empty.
func (s *FuncType) SourceSignature(from *File, name *ast.Ident) string {
	nameS := ""
	if name != nil {
		nameS = name.Name
	}
	return s.sourceSignature(from, nameS, false, false)
}

func (s *FuncType) shortString(from *File) string {
	return s.Params.ShortString(from)
}

func (s *FuncType) calleeString(from *File) string {
	var b strings.Builder
	vals := s.GenParams.Values
	for i := range len(vals) {
		gval := s.GenParams.Values[len(vals)-i-1]
		if gval == nil {
			continue
		}
		fmt.Fprintf(&b, "%s", gval.SourceString(from))
		break
	}
	return b.String()
}

func (s *FuncType) specializeString(from *File) string {
	var b strings.Builder
	for _, gval := range s.GenParams.Values {
		if gval == nil {
			continue
		}
		fmt.Fprintf(&b, "%s", gval.SourceString(from))
	}
	if s.GenParams.Fields != nil && s.GenParams.Fields.Len() > 0 {
		fmt.Fprintf(&b, "[%s]", s.GenParams.Fields.SourceString(from))
	}
	return b.String()
}

// SourceSignature returns a string representation of a signature given a name.
// The name can be empty.
func (s *FuncType) sourceSignature(from *File, name string, skipRecv, skipSpec bool) string {
	var b strings.Builder
	b.WriteString("func")
	if s.Receiver != nil && !skipRecv {
		fmt.Fprintf(&b, " (%s)", s.Receiver.SourceString(from))
	}
	if name != "" {
		fmt.Fprintf(&b, " %s", name)
	}
	if s.Params == nil || s.Results == nil {
		return b.String()
	}
	if !skipSpec {
		b.WriteString(s.specializeString(from))
	}
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

// UnknownFType returns the function type for a function unknown at compile time.
func UnknownFType() *FuncType {
	return &FuncType{
		Nature: FuncNature{
			Unknown: true,
		},
	}
}
