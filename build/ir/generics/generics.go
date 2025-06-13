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

// Package generics provides functions to handle generics with the GX IR.
package generics

import (
	"go/ast"
	"reflect"

	"github.com/gx-org/gx/build/ir"
)

type genericType interface {
	name() string
	specialise(fetcher ir.Fetcher, typ ir.Type) ir.Type
	instantiate(fetcher ir.Fetcher) ir.Type
}

func typeInclude(fetcher ir.Fetcher, set ir.Type, typ ir.Type) bool {
	isIn, err := ir.TypeInclude(fetcher, set, typ)
	if err != nil {
		return fetcher.Err().Append(err)
	}
	if !isIn {
		return fetcher.Err().Appendf(typ.Source(), "%s does not satisfy %s", typ.String(), set.String())
	}
	return true
}

type ident struct {
	set     ir.Type
	typName string
}

func (i *ident) name() string {
	return i.typName
}

func (i *ident) specialise(fetcher ir.Fetcher, typ ir.Type) ir.Type {
	if !typeInclude(fetcher, i.set, typ) {
		return nil
	}
	return typ
}

func (i *ident) instantiate(ir.Fetcher) ir.Type {
	return i.set
}

type array struct {
	src     *ast.ArrayType
	typ     ir.ArrayType
	typName string
	set     ir.Type
}

func (i *array) name() string {
	return i.typName
}

func (i *array) instantiate(fetcher ir.Fetcher) ir.Type {
	rank, ok := i.instantiateRank(fetcher, i.typ.Rank())
	if !ok {
		return nil
	}
	return ir.NewArrayType(i.src, i.typ.DataType(), rank)
}

func (i *array) specialise(fetcher ir.Fetcher, typ ir.Type) ir.Type {
	if !typeInclude(fetcher, i.set, typ) {
		return nil
	}
	return ir.NewArrayType(i.typ.ArrayType(), typ, i.typ.Rank())
}

func dtypeElement(expr ast.Expr) ast.Expr {
	aType, ok := expr.(*ast.ArrayType)
	if !ok {
		return expr
	}
	return dtypeElement(aType.Elt)
}

func extractTypeParamName(field *ir.FieldGroup) genericType {
	switch typT := field.Type.Typ.(type) {
	case *ir.TypeParam:
		return &ident{set: typT.Field.Type(), typName: typT.Field.Name.Name}
	case *ir.TypeSet:
		return nil
	case ir.ArrayType:
		dtype := typT.DataType()
		gType := &array{src: typT.ArrayType(), typ: typT}
		typeParam, isTypeParam := dtype.(*ir.TypeParam)
		if !isTypeParam {
			return gType
		}
		gType.set = typeParam.Field.Type()
		gType.typName = typeParam.Field.Name.Name
		return gType
	}
	return nil
}

func instantiatExpr(fetcher ir.Fetcher, expr ir.Expr) (ir.Value, bool) {
	val, err := fetcher.Eval(expr)
	if err != nil {
		return expr, fetcher.Err().Appendf(expr.Source(), "cannot evaluate expression %q: %v", expr.String(), err)
	}
	irVal, ok := val.(ir.Canonical)
	if !ok {
		return expr, fetcher.Err().AppendInternalf(expr.Source(), "cannot convert %T to %s", val, reflect.TypeFor[ir.Canonical]())
	}
	return irVal.Expr(), true
}
