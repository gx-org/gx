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
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/gx-org/gx/build/ir/irkind"
)

// AxisValue assigns a value to an axis length.
type AxisValue struct {
	Axis  *AxisStmt
	Exprs []AxisLengths
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
		Generic *FuncType

		Receiver   *FieldList
		TypeParams *FieldList
		Params     *FieldList
		Results    *FieldList

		VarArgs *VarArgsType

		AxisLengths      []AxisValue
		TypeParamsValues []TypeParamValue

		// CompEval is set to true if the function can be called at compilation time.
		CompEval bool
	}
)

var (
	_ Type = (*FuncType)(nil)
	_ Expr = (*FuncType)(nil)
)

func (*FuncType) node() {}

// Kind returns the function kind.
func (s *FuncType) Kind() irkind.Kind { return irkind.Func }

// Equal returns true if other is the same type.
func (s *FuncType) Equal(tpcmp TypeCmp, other Type) (bool, CompEvalError, error) {
	otherT, ok := other.(*FuncType)
	if !ok {
		return false, nil, nil
	}
	return s.equal(tpcmp, otherT)
}

// Equal returns true if other is the same type.
func (s *FuncType) equal(tpcmp TypeCmp, other *FuncType) (bool, CompEvalError, error) {
	if s == other {
		return true, nil, nil
	}
	recvOk, cpErr, err := s.Receiver.Type().Equal(tpcmp, other.Receiver.Type())
	if !recvOk || cpErr != nil || err != nil {
		return recvOk, cpErr, err
	}
	return s.equalParamsResults(tpcmp, other)
}

func (s *FuncType) equalParamsResults(tpcmp TypeCmp, other *FuncType) (bool, CompEvalError, error) {
	paramsOk, cpErr, err := s.Params.Type().Equal(tpcmp, other.Params.Type())
	if cpErr != nil || err != nil {
		return false, cpErr, err
	}
	resultsOk, cpErr, err := s.Results.Type().Equal(tpcmp, other.Results.Type())
	if cpErr != nil || err != nil {
		return false, cpErr, err
	}
	return paramsOk && resultsOk, nil, nil
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
func (s *FuncType) AssignableTo(tpcmp TypeCmp, other Type) (bool, CompEvalError, error) {
	otherT, ok := other.(*FuncType)
	if ok {
		return s.equal(tpcmp, otherT)
	}
	aFrom, ok := other.(assignsFrom)
	if !ok {
		return false, nil, nil
	}
	return aFrom.assignableFrom(tpcmp, s)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *FuncType) ConvertibleTo(tpcmp TypeCmp, target Type) (bool, CompEvalError, error) {
	return s.Equal(tpcmp, target)
}

// Value returns a value pointing to the receiver.
func (s *FuncType) Value(x Expr) Expr {
	return TypeExpr(x, s)
}

// Specialise a type to a given target.
func (s *FuncType) Specialise(spec Specialiser) (Type, CompEvalError, error) {
	return s.SpecialiseFType(spec)
}

// String representation of the function type (only used for debugging).
func (s *FuncType) String() string {
	return s.ReferString(nil)
}

func skipIfDefined(spec Specialiser) fieldCloner {
	return func(grp *FieldGroup, i int, field *Field) (*Field, CompEvalError, error) {
		if spec.IsDefined(field.Name.Name) {
			return nil, nil, nil
		}
		return cloneField(grp, i, field)
	}
}

func specialiseGroup(spec Specialiser) groupCloner {
	return func(grp *FieldGroup) (*FieldGroup, CompEvalError, error) {
		specType, cpErr, err := grp.Type.Val().Specialise(spec)
		if cpErr != nil || err != nil {
			return nil, cpErr, err
		}
		return &FieldGroup{
			Src: grp.Src,
			Type: TypeExpr(
				grp.Type.X(),
				specType,
			),
		}, nil, nil
	}
}

// SpecialiseFType specialises a function type.
func (s *FuncType) SpecialiseFType(spec Specialiser) (_ *FuncType, cpErr CompEvalError, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cannot specialise %s: %w", s.ReferString(spec.File()), err)
		}
	}()
	res := *s
	if res.Generic == nil {
		res.Generic = s
	}
	res.Params, cpErr, err = cloneFields(s.Params, &cloner{
		group: specialiseGroup(spec),
		field: cloneField,
	})
	if cpErr != nil || err != nil {
		return nil, cpErr, err
	}
	res.Results, cpErr, err = cloneFields(s.Results, &cloner{
		group: specialiseGroup(spec),
		field: cloneField,
	})
	if cpErr != nil || err != nil {
		return nil, cpErr, err
	}
	res.TypeParams, cpErr, err = cloneFields(s.TypeParams, &cloner{
		group: cloneGroup,
		field: skipIfDefined(spec),
	})
	if cpErr != nil || err != nil {
		return nil, cpErr, err
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
		axes, element := spec.ValueOf(axis.Name())
		res.AxisLengths[i].Exprs = axes
		res.AxisLengths[i].Value = element
	}
	return &res, nil, nil
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
	return s.sourceSignature(from, name, false)
}

func toUnorderedString(m map[string]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "unordered<%s>", b.String())
	keys := slices.Collect(maps.Keys(m))
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "[%s]", m[k])
	}
	return b.String()
}

func axisLengthsString(from *File, axes []AxisLengths) string {
	var b strings.Builder
	for _, axis := range axes {
		fmt.Fprintf(&b, "%s", axis.SourceString(from))
	}
	return b.String()
}

func (s *FuncType) typeParamValuesString(from *File) string {
	if s.Generic == nil {
		return ""
	}
	values := make(map[string]string)
	for _, tp := range s.TypeParamsValues {
		if !ValidIdent(tp.Field.Name) {
			continue
		}
		values[tp.Field.Name.Name] = tp.Typ.ReferString(from)
	}
	for _, tp := range s.AxisLengths {
		if !ValidIdent(tp.Axis.Src) {
			continue
		}
		var src string
		if tp.Exprs != nil {
			src = axisLengthsString(from, tp.Exprs)
		} else {
			src = fmt.Sprintf("%s:<missing expression>", tp.Axis.Src.Name)
		}
		values[tp.Name()] = src
	}
	if s.Generic == nil {
		return toUnorderedString(values)
	}
	var b strings.Builder
	for _, field := range s.Generic.TypeParams.Fields() {
		if !ValidIdent(field.Name) {
			continue
		}
		val, ok := values[field.Name.Name]
		if !ok {
			break
		}
		fmt.Fprintf(&b, "[%s]", val)
	}
	return b.String()
}

// SourceSignature returns a string representation of a signature given a name.
// The name can be empty.
func (s *FuncType) sourceSignature(from *File, name *ast.Ident, skipRecv bool) string {
	var b strings.Builder
	b.WriteString("func")
	if s.Receiver != nil && !skipRecv {
		fmt.Fprintf(&b, " (%s)", s.Receiver.SourceString(from))
	}
	if name != nil && name.Name != "" {
		b.WriteString(" " + name.Name)
	}
	if len(s.AxisLengths) > 0 || len(s.TypeParamsValues) > 0 {
		fmt.Fprint(&b, s.typeParamValuesString(from))
	}
	if s.TypeParams != nil && s.TypeParams.Len() > 0 {
		fmt.Fprintf(&b, "[%s]", s.TypeParams.SourceString(from))
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
