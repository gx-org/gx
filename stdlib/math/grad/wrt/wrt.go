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

// Package wrt builds a function signature to compute the gradient w.r.t. a given field (parameter or in a structure).
package wrt

import (
	"go/ast"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/astbuilder"
)

type core struct {
	parent         *Struct
	ftype          *ast.FuncType
	backwardValues *ast.FieldList

	fieldPath []*ir.Field
}

func newCore(backwardValues *ast.FieldList, parent *Struct, field *ir.Field) (*core, error) {
	var parentPath []*ir.Field
	if parent != nil {
		parentPath = parent.fPath()
	}
	cr := &core{
		parent:         parent,
		fieldPath:      append(parentPath, field),
		backwardValues: backwardValues,
	}
	var err error
	cr.ftype, err = cr.buildBackwardSignature(backwardValues)
	return cr, err
}

func (cr *core) fPath() []*ir.Field {
	return append([]*ir.Field{}, cr.fieldPath...)
}

func (cr *core) field() *ir.Field {
	return cr.fieldPath[len(cr.fieldPath)-1]
}

// FuncType returns the type of the function computing the gradient.
func (cr *core) FuncType() *ast.FuncType {
	return cr.ftype
}

// Name of the field.
func (cr *core) Name() []string {
	if cr == nil {
		return nil
	}
	field := cr.field()
	var name string
	if field != nil && ir.ValidIdent(field.Name) {
		name = field.Name.Name
	}
	if cr.parent == nil {
		return []string{name}
	}
	return append(slices.Clone(cr.parent.Name()), name)
}

// Type of the field.
func (cr *core) Type() ir.Type {
	return cr.field().Type()
}

// Same returns true if src matches the field of the receiver.
func (cr *core) Same(other []*ir.Field) bool {
	if len(cr.fieldPath) != len(other) {
		return false
	}
	for i, field := range cr.fieldPath {
		if field != other[i] {
			return false
		}
	}
	return true
}
func (cr *core) buildBackwardSignature(backwardValues *ast.FieldList) (*ast.FuncType, error) {
	results, err := astbuilder.Clone(&ast.FieldList{
		List: []*ast.Field{&ast.Field{
			Type: cr.field().Group.Src.Type,
		}},
	}, astbuilder.AssignToExpandShape)
	if err != nil {
		return nil, err
	}

	return &ast.FuncType{
		// Gradient coming from the output values of the function.
		Params: backwardValues,
		// Return the gradient.
		Results: results,
	}, nil
}

func (cr *core) String() string {
	parent := ""
	if cr.parent != nil {
		parent = "(" + cr.parent.String() + ")."
	}
	return parent + cr.field().Name.Name + ":" + cr.Type().ReferString(nil)
}

type (
	// WRT is a parameter in a function with the array type.
	WRT interface {
		Arrays() []*Array
		FuncType() *ast.FuncType
		Name() []string
		Type() ir.Type
		Same([]*ir.Field) bool
		String() string
	}

	// WRTs groups all the parameters of a function.
	WRTs []WRT
)

// BuildFromField build the gradient parameters from a single field.
func BuildFromField(backwardValues *ast.FieldList, field *ir.Field) (WRT, error) {
	return parse(backwardValues, nil, field)
}

// Build builds the gradient parameters given a function type.
func Build(fType *ir.FuncType, backwardValues *ast.FieldList) ([]WRT, error) {
	fields := fType.Params.Fields()
	params := make([]WRT, len(fields))
	for i, field := range fields {
		param, err := BuildFromField(backwardValues, field)
		if err != nil {
			return nil, err
		}
		params[i] = param
	}
	return params, nil
}

// Arrays returns all the arrays for which the gradient needs to be computed for.
func (p WRTs) Arrays() (arrays []*Array) {
	for _, param := range p {
		arrays = append(arrays, param.Arrays()...)
	}
	return arrays
}

func parse(backwardValues *ast.FieldList, parent *Struct, field *ir.Field) (WRT, error) {
	cr, err := newCore(backwardValues, parent, field)
	if err != nil {
		return nil, err
	}
	switch typeT := ir.Underlying(field.Type()).(type) {
	case ir.ArrayType:
		return &Array{core: cr}, nil
	case *ir.StructType:
		return cr.parseStructure(backwardValues, typeT)
	default:
		return nil, errors.Errorf("%T not supported", typeT)
	}
}

// ToName joins strings to form an identifier name
func ToName(names []string) string {
	return strings.Join(names, "_")
}
