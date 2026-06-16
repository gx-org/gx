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

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir/irkind"
)

type errorType struct {
	iface Interface
}

var errorIdent = &ast.Ident{Name: "Error"}

var errorSrc = &ast.Field{
	Names: []*ast.Ident{errorIdent},
	Type:  &ast.FuncType{},
}

var errorFType = &FuncType{
	BaseType: BaseType[*ast.FuncType]{
		Src: errorSrc.Type.(*ast.FuncType),
	},
	Params: &FieldList{},
	Results: &FieldList{
		List: []*FieldGroup{
			&FieldGroup{Type: TypeExpr(nil, StringType())},
		},
	},
}

var errorTyp = &errorType{
	iface: Interface{
		methods: []*IMethod{
			// Define method: Error() string
			&IMethod{
				Src:   errorSrc,
				FType: errorFType,
			},
		},
	},
}

var (
	_ Type        = errorTyp
	_ assigner    = errorTyp
	_ TypeMethods = errorTyp
)

// ErrorType returns the type for the keyword error.
func ErrorType() TypeMethods {
	return errorTyp
}

func (*errorType) node()         {}
func (*errorType) storage()      {}
func (*errorType) storageValue() {}

// Kind returns the scalar kind.
func (s *errorType) Kind() irkind.Kind { return irkind.Interface }

// Equal returns true if other is the exact same type set.
func (s *errorType) Equal(tpcmp TypeCmp, target Type) (bool, error) {
	return s.iface.Equal(tpcmp, target)
}

// AssignableTo reports whether a value of the type can be assigned to another.
func (s *errorType) AssignableTo(tpcmp TypeCmp, target Type) (bool, error) {
	if s.Same(target) {
		return true, nil
	}
	return s.iface.AssignableTo(tpcmp, target)
}

func (s *errorType) assignableFrom(tpcmp TypeCmp, x Type) (bool, error) {
	return s.iface.assignableFromWithName(tpcmp, x, s.DefineString)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *errorType) ConvertibleTo(tpcmp TypeCmp, target Type) (bool, error) {
	return s.iface.ConvertibleTo(tpcmp, target)
}

// DefineString returns the GX source code to define the type.
func (s *errorType) DefineString(from *File) string {
	return "error"
}

func (s *errorType) Methods() []PkgFunc {
	return s.iface.Methods()
}

// NameDef of the base type always returns a nil name definition.
func (s *errorType) NameDef() *ast.Ident { return errorIdent }

// Node returns the source node defining the type.
func (s *errorType) Node() ast.Node {
	return s.iface.Node()
}

// Same returns true if the other storage is this storage.
func (s *errorType) Same(o Storage) bool {
	return Storage(s) == o
}

// Specialise a type to a given target.
func (s *errorType) Specialise(spec Specialiser) (Type, bool) {
	return s, true
}

// Value returns a value pointing to the receiver.
func (s *errorType) Value(x Expr) Expr {
	return TypeExpr(x, s)
}

// Instantiate a function type.
func (s *errorType) Instantiate(Fetcher, Specialiser) (Type, bool) {
	return s, true
}

// UnifyWith recursively unifies a type parameters with types.
func (s *errorType) UnifyWith(unifier Unifier, typ Type) bool {
	return true
}

// Type of a type: always return metatype.
func (s *errorType) Type() Type {
	return MetaType()
}

func (s *errorType) IndexForVarArgs(fmterr.ErrAppender, int) (Type, bool) {
	return s, true
}

// ReferString returns the GX source to refer to the type.
func (s *errorType) ReferString(from *File) string {
	return s.DefineString(from)
}

type errorCallee struct {
	src   ast.Expr
	ftype *FuncType
}

func (*errorCallee) node() {}

func (ec *errorCallee) Node() ast.Node {
	return ec.src
}

func (*errorCallee) Func() Func {
	return nil
}

func (ec *errorCallee) FuncType() *FuncType {
	return ec.ftype
}

func (*errorCallee) Type() Type {
	return errorFType
}

func (*errorCallee) Expr() ast.Expr {
	return nil
}

func (*errorCallee) SourceString(from *File) string {
	return "<cperror>"
}

// ErrorCallee returns a proxy callee to call the Error method.
func ErrorCallee(src ast.Expr, ftype *FuncType) Callee {
	return &errorCallee{src: src, ftype: ftype}
}

// UnifyErr returns a system error if not nil, return the compile error otherwise.
func UnifyErr(cpErr CompEvalError, err error) error {
	if err != nil {
		return err
	}
	return cpErr
}

// CompileError is a normal error to be reported to the user as a compile error of the source code.
type CompileError struct {
	error
}

// CompileErrorF converts an error into a compilation error.
func CompileErrorF(s string, as ...any) *CompileError {
	return &CompileError{error: errors.Errorf(s, as...)}
}

// SplitErr checks if an error is a compile error.
func SplitErr(err error) (*CompileError, error) {
	if err == nil {
		return nil, nil
	}
	cpErr, isCompileError := err.(*CompileError)
	if isCompileError {
		return cpErr, nil
	}
	return nil, err
}
