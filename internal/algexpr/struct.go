// Copyright 2026 Google LLC
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

package algexpr

import (
	"fmt"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
)

type structExpr struct {
	structType ir.Type
	elements   []cmp.Expr
}

// NewStruct of algebraic expressions.
func NewStruct(structType ir.Type, elements []cmp.Expr) cmp.Expr {
	return &structExpr{structType: structType, elements: elements}
}

func (x *structExpr) Simplify(sourceFile ir.SourceFile) (cmp.Comparable, error) {
	elements := make([]cmp.Comparable, len(x.elements))
	for i, element := range x.elements {
		var err error
		elements[i], err = element.Simplify(sourceFile)
		if err != nil {
			return nil, err
		}
	}
	return &structCmp{structType: x.structType, elements: elements}, nil
}

func (x *structExpr) String() string {
	return fmt.Sprintf("{%s}", toString(x.elements, ", "))
}

type structCmp struct {
	structType ir.Type
	elements   []cmp.Comparable
}

func (c *structCmp) Equal(other cmp.Comparable) bool {
	otherStruct, ok := other.(*structCmp)
	if !ok {
		return false
	}
	if len(c.elements) != len(otherStruct.elements) {
		return false
	}
	for i, element := range c.elements {
		if !element.Equal(otherStruct.elements[i]) {
			return false
		}
	}
	return true
}

func (c *structCmp) BuildIR() ir.Expr {
	fields := ir.Underlying(c.structType).(*ir.StructType).Fields.Fields()
	fieldLiterals := make([]*ir.FieldLit, len(c.elements))
	for i, element := range c.elements {
		fieldLiterals[i] = &ir.FieldLit{
			FieldStorage: fields[i].Storage(),
			X:            element.BuildIR(),
		}
	}
	return &ir.StructLitExpr{
		Typ:  c.structType,
		Elts: fieldLiterals,
	}
}

func (c *structCmp) String() string {
	return fmt.Sprintf("{%s}", toString(c.elements, ", "))
}
