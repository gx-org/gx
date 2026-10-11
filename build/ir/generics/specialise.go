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
	"fmt"
	"strings"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/src/gx/ast"
)

type specialiser struct {
	from         *ir.File
	x            ir.Expr
	gtype        ir.GenericType
	defined      []ir.GenericValue
	tparamFields []*ir.Field
	fetcher      ir.Fetcher

	defaultInstErr func(*specialiser, ast.Node, error) bool

	instantiatingResults *ir.FuncType
}

var _ ir.Instantiator = (*specialiser)(nil)

func newSpecialiser(from *ir.File, x ir.Expr, fetcher ir.Fetcher, defaultInstErr func(*specialiser, ast.Node, error) bool, gtype ir.GenericType, defined []ir.GenericValue) *specialiser {
	return &specialiser{
		from:           from,
		x:              x,
		fetcher:        fetcher,
		gtype:          gtype,
		defined:        defined,
		tparamFields:   gtype.Origin().Generics().Fields.Fields(),
		defaultInstErr: defaultInstErr,
	}
}

func (s *specialiser) StartResults(ftype *ir.FuncType) {
	if s.instantiatingResults != nil {
		return
	}
	s.instantiatingResults = ftype
}

func (s *specialiser) InstantiantingResults() bool {
	return s.instantiatingResults != nil
}

func (s *specialiser) StopResults(ftype *ir.FuncType) {
	if ftype != s.instantiatingResults {
		return
	}
	s.instantiatingResults = nil
}

func (s *specialiser) IsDefined(pos int) bool {
	return s.defined[pos] != nil
}

func (s *specialiser) Fetcher() ir.Fetcher {
	return s.fetcher
}

func (s *specialiser) From() (*ir.File, ir.Expr) {
	return s.from, s.x
}

func (s *specialiser) valueOf(genParam ir.GenericParam) ir.GenericValue {
	paramField := genParam.OrigField()
	if paramField.Pos >= len(s.tparamFields) {
		return nil
	}
	pos := genParam.OrigField().Pos
	if paramField != s.tparamFields[pos] {
		return nil
	}
	return s.defined[pos]
}

func (s *specialiser) NonTypeFor(genParam ir.GenericParam) *ir.NonTypeGenericValue {
	val := s.valueOf(genParam)
	if val == nil {
		return nil
	}
	axisVal, isAxis := val.(*ir.NonTypeGenericValue)
	if !isAxis {
		return nil
	}
	return axisVal
}

func (s *specialiser) TypeFor(genParam ir.GenericParam) *ir.TypeGenericValue {
	val := s.valueOf(genParam)
	if val == nil {
		return nil
	}
	typeVal, isType := val.(*ir.TypeGenericValue)
	if !isType {
		return nil
	}
	return typeVal
}

func (s *specialiser) Values() []ir.GenericValue {
	return s.defined
}

func (s *specialiser) Source() ast.Expr {
	return s.x.Expr()
}

func (s *specialiser) File() *ir.File {
	return s.fetcher.File()
}

func (s *specialiser) Err() *fmterr.Appender {
	return s.fetcher.Err()
}

func (s *specialiser) InstantiateError(src ast.Node, err error) bool {
	if s.defaultInstErr != nil {
		return s.defaultInstErr(s, src, err)
	}
	return s.Err().AppendAt(src, err)
}

func funcTypeInstantiateError(s *specialiser, fn *ir.FuncValExpr, sigSrc ast.Node, err error) bool {
	// sigSrc is the call from the function signature.
	src := sigSrc
	if fn != nil {
		// set the location of the error to the call site of the function instead of its signature.
		src = fn.Node()
	}
	compileErr, isCompileErr := err.(*ir.CompileError)
	if !isCompileErr {
		return s.Err().AppendAt(src, err)
	}
	if fn == nil {
		// The error is from the function signature.
		return s.Err().AppendAt(src, err)
	}
	return s.Err().Appendf(src, "in call to %s: %v", fn.ShortString(s.File()), compileErr)
}

func fTypeError(fn *ir.FuncValExpr) func(*specialiser, ast.Node, error) bool {
	return func(s *specialiser, src ast.Node, err error) bool {
		return funcTypeInstantiateError(s, fn, src, err)
	}
}

func (s *specialiser) String() string {
	b := strings.Builder{}
	fmt.Fprintln(&b, "Scope:")
	fields := s.gtype.Origin().Generics().Fields.Fields()
	for i, v := range s.defined {
		fmt.Fprintf(&b, "\t%d:%s %s->", i, fields[i].Name.Name, fields[i].Type().ReferString(nil))
		s := "nil"
		if v != nil {
			s = v.SourceString(nil)
		}
		fmt.Fprintln(&b, s)
	}
	fmt.Fprintln(&b, "Fetcher:")
	fmt.Fprint(&b, s.Fetcher())
	return b.String()
}

func toTypeValue(fetcher ir.Fetcher, field *ir.Field, x ir.Expr) (ir.GenericValue, bool) {
	typeValExpr := ir.TypeFromExpr(x)
	genericIdentType := ir.NewGenericTypeParam(field)
	if typeValExpr == nil {
		return invalidGenericType(genericIdentType), fetcher.Err().Appendf(x.Node(), "%s is not a type", x.SourceString(fetcher.File()))
	}
	gotType, wantType := typeValExpr.Val(), field.Group.Type.Val()
	assignedOk, err := gotType.AssignableTo(fetcher, wantType)
	if err != nil {
		return invalidGenericType(genericIdentType), fetcher.Err().AppendAt(x.Node(), err)
	}
	if !assignedOk {
		return invalidGenericType(genericIdentType), fetcher.Err().Appendf(x.Node(), "%s does not satisfy %s", gotType.ReferString(fetcher.File()), wantType.ReferString(fetcher.File()))
	}
	return ir.NewTypeExprGenericValue(genericIdentType, typeValExpr), true
}

// SpecialiseStruct specialises a structure given type values.
func SpecialiseStruct(fetcher ir.Fetcher, expr ir.Expr, st *ir.StructType, typArgs []ir.GenericValue) (*ir.TypeValExpr, bool) {
	spec := newSpecialiser(fetcher.File(), expr, fetcher, nil, st, typArgs)
	specType, ok := st.SpecialiseStruct(spec)
	return ir.TypeExpr(expr.Expr(), specType), ok
}

// SpecialiseParams a function signature for a given type.
func SpecialiseParams(fetcher ir.Fetcher, expr ir.Expr, fun *ir.FuncValExpr, typArgs []ir.GenericValue) (*ir.FuncValExpr, bool) {
	ftype := fun.FuncType()
	spec := newSpecialiser(fetcher.File(), fun, fetcher, fTypeError(fun), ftype, typArgs)
	specType, ok := ftype.SpecialiseFType(spec)
	return ir.NewFuncValExpr(expr, fun.Func()).NewFType(specType), ok
}

// Instantiate specialises the result of a function.
// fun can be nil when the type is instantiated for its definition.
func Instantiate(from *ir.File, fun *ir.FuncValExpr, fetcher ir.Fetcher, ftype *ir.FuncType) (*ir.FuncType, bool) {
	typArgs := ftype.GenParams.Values
	spec := newSpecialiser(from, fun, fetcher, fTypeError(fun), ftype, typArgs)
	instantiateFType, ok := ftype.InstantiateFType(fetcher, spec)
	return instantiateFType, ok
}
