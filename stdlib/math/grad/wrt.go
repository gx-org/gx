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

package grad

import (
	"go/ast"

	"github.com/gx-org/gx/build/ir"
)

type withRespectTo interface {
	same(storage *ir.Field) bool
	name() string
	fieldType() ast.Expr
}

type wrtArray struct {
	field *ir.Field
}

func newWRT(field *ir.Field) withRespectTo {
	tp := field.Type()
	if tp.Kind() != ir.StructKind {
		return &wrtArray{field: field}
	}
	under := ir.Underlying(tp).(*ir.StructType)
	if under.NumFields() > 1 {
		// TODO(degris): the gradient computation does not make sense at the moment,
		// but the core does not return an error. So, temporarily panicking.
		// This will be fixed soon.
		panic("structure with more than one field not supported")
	}
	return &wrtStruct{field: field, tp: under}
}

func (f *wrtArray) same(src *ir.Field) bool {
	return src == f.field
}

func (f *wrtArray) name() string {
	return f.field.Name.Name
}

func (f *wrtArray) fieldType() ast.Expr {
	return f.field.Group.Src.Type
}

type wrtStruct struct {
	tp    *ir.StructType
	field *ir.Field
}

func (f *wrtStruct) same(src *ir.Field) bool {
	return src == f.tp.Fields.FindField(src.Name.Name)
}

func (f *wrtStruct) name() string {
	return f.field.Name.Name
}

func (f *wrtStruct) fieldType() ast.Expr {
	return f.field.Group.Src.Type
}
