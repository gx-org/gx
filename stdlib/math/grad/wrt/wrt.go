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

// Package wrt builds a list of fields for which we are computing the gradient for.
package wrt

import (
	"go/ast"
	"slices"

	"github.com/gx-org/gx/build/ir"
)

// WithRespectTo refers a field to which the gradient is being computed with respect to.
type WithRespectTo struct {
	parent *wrtStruct
	field  *ir.Field
}

// New builds the list of field to which gradient are computed with respect to.
func New(field *ir.Field) []*WithRespectTo {
	return newWRTs(nil, field)
}

func newWRTs(parent *wrtStruct, field *ir.Field) []*WithRespectTo {
	switch typeT := ir.Underlying(field.Type()).(type) {
	case ir.ArrayType:
		return []*WithRespectTo{
			&WithRespectTo{
				parent: parent,
				field:  field,
			}}
	case *ir.StructType:
		return parseStructure(parent, field, typeT)
	}
	return nil
}

func toName(field *ir.Field) string {
	if !ir.ValidIdent(field.Name) {
		return ""
	}
	return field.Name.Name
}

// Same returns true if src matches the field of the receiver.
func (w *WithRespectTo) Same(src *ir.Field) bool {
	return src == w.field
}

// Name pointing to the field.
func (w *WithRespectTo) Name() []string {
	return append(slices.Clone(w.parent.name()), toName(w.field))
}

// FieldType returns the type of the field.
func (w *WithRespectTo) FieldType() ast.Expr {
	return w.field.Group.Src.Type
}

type wrtStruct struct {
	parent *wrtStruct
	field  *ir.Field
	typ    *ir.StructType
}

func (w *wrtStruct) name() []string {
	if w == nil {
		return nil
	}
	return append(slices.Clone(w.parent.name()), toName(w.field))
}

func parseStructure(parent *wrtStruct, field *ir.Field, tp *ir.StructType) (wrts []*WithRespectTo) {
	current := &wrtStruct{
		parent: parent,
		field:  field,
		typ:    tp,
	}
	for _, field := range tp.Fields.Fields() {
		wrts = append(wrts, newWRTs(current, field)...)
	}
	return
}
