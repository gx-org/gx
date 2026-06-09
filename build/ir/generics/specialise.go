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
	"go/ast"
	"strings"

	"github.com/gx-org/gx/build/ir"
)

type specialiser struct {
	ir.Fetcher
	src          ast.Node
	ftype        *ir.FuncType
	defined      []ir.GenericValue
	tparamFields []*ir.Field
}

var _ ir.Specialiser = (*specialiser)(nil)

func checkTypeParams(ftype *ir.FuncType, defined []ir.GenericValue) {
	if len(defined) != ftype.Origin().TypeParams.Len() {
		panic(fmt.Sprintf("defined: %d but TypeParams is %d", len(defined), ftype.Origin().TypeParams.Len()))
	}
}

func newSpecialiser(fetcher ir.Fetcher, src ast.Node, ftype *ir.FuncType, defined []ir.GenericValue) *specialiser {
	checkTypeParams(ftype, defined)
	return &specialiser{
		src:          src,
		Fetcher:      fetcher,
		ftype:        ftype,
		defined:      defined,
		tparamFields: ftype.Origin().TypeParams.Fields(),
	}
}

func (s *specialiser) IsDefined(pos int) bool {
	return s.defined[pos] != nil
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

func (s *specialiser) Source() ast.Node {
	return s.src
}

func (s *specialiser) String() string {
	b := strings.Builder{}
	fmt.Fprintln(&b, "Scope:")
	fields := s.ftype.Origin().TypeParams.Fields()
	for i, v := range s.defined {
		fmt.Fprintf(&b, "\t%d:%s %s->", i, fields[i].Name.Name, fields[i].Type().ReferString(nil))
		s := "nil"
		if v != nil {
			s = v.SourceString(nil)
		}
		fmt.Fprintln(&b, s)
	}
	fmt.Fprintln(&b, "Fetcher:")
	fmt.Fprint(&b, s.Fetcher)
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

// SpecialiseParams a function signature for a given type.
func SpecialiseParams(fetcher ir.Fetcher, expr ir.Expr, fun *ir.FuncValExpr, typArgs []ir.GenericValue) *ir.FuncValExpr {
	ftype := fun.FuncType()
	spec := newSpecialiser(fetcher, expr.Node(), ftype, typArgs)
	specType := ftype.SpecialiseFType(spec, true)
	checkTypeParams(specType, specType.GenericValues)
	return ir.NewFuncValExpr(expr, fun.Func()).NewFType(specType)
}

// Instantiate specialises the result of a function.
func Instantiate(fetcher ir.Fetcher, src ast.Node, ftype *ir.FuncType, typArgs []ir.GenericValue) (*ir.FuncType, bool) {
	spec := newSpecialiser(fetcher, src, ftype, typArgs)
	instantiateFType, ok := ftype.InstantiateFType(fetcher, spec)
	checkTypeParams(instantiateFType, instantiateFType.GenericValues)
	return instantiateFType, ok
}
