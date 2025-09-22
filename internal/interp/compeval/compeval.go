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

// Package compeval runs GX code at compile time.
package compeval

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
)

// EvalExpr evaluates a GX expression into an interpreter element.
func EvalExpr(eval ir.Evaluator, expr ir.Expr) (cpevelements.Element, error) {
	val, err := eval.EvalExpr(expr)
	if err != nil {
		return nil, err
	}
	el, ok := val.(cpevelements.Element)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to %s", val, reflect.TypeFor[cpevelements.Element]().String())
	}
	return el, nil
}

// NewOptionVariable creates a package option to set a static variable of a package with its corresponding symbolic element.
func NewOptionVariable(vr *ir.VarExpr) options.PackageOption {
	src := elements.NewNodeAt[ir.Storage](vr.Decl.FFile, vr)
	return elements.PackageVarSetElement{
		Pkg:   vr.Decl.FFile.Package.FullName(),
		Var:   vr.VName.Name,
		Value: cpevelements.NewVariable(src),
	}
}
