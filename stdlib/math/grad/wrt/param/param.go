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

// Package param builds a function signature to compute the gradient w.r.t. a given field (parameter or in a structure).
package param

import (
	"go/ast"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/astbuilder"
	"github.com/gx-org/gx/stdlib/math/grad/wrt"
)

// Param is a parameter in a function that the gradient is going to be computed with respect to.
type Param struct {
	field *ir.Field
	WRT   *wrt.WithRespectTo
	FType *ast.FuncType
}

// Name of the field.
func (p *Param) Name() string {
	return p.field.Name.Name
}

// Type of the field.
func (p *Param) Type() ir.Type {
	return p.field.Type()
}

// Build builds the gradient parameters given a function type.
func Build(fType *ir.FuncType, backwardValues *ast.FieldList) ([]Param, error) {
	var params []Param
	for _, param := range fType.Params.Fields() {
		fieldParams, err := buildWRTFromField(fType, backwardValues, param)
		if err != nil {
			return nil, err
		}
		params = append(params, fieldParams...)
	}
	return params, nil
}

func buildWRTFromField(fType *ir.FuncType, backwardValues *ast.FieldList, field *ir.Field) ([]Param, error) {
	var params []Param
	for _, wrt := range wrt.New(field) {
		backwardSig, err := buildBackwardSignature(fType, backwardValues, wrt)
		if err != nil {
			return nil, err
		}
		params = append(params, Param{
			field: field,
			WRT:   wrt,
			FType: backwardSig,
		})
	}
	return params, nil
}

func buildBackwardSignature(fType *ir.FuncType, backwardValues *ast.FieldList, wrt *wrt.WithRespectTo) (*ast.FuncType, error) {
	results, err := astbuilder.Clone(&ast.FieldList{
		List: []*ast.Field{&ast.Field{
			Type: wrt.FieldType(),
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
