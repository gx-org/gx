// Copyright 2024 Google LLC
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

// Package shape provides functions to manipulate the shape of tensors.
package shape

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/fun"
	"github.com/gx-org/gx/stdlib/builtin"
)

// Package description of the GX shape package.
var Package = builtin.PackageBuilder{
	FullPath: "shape",
	Builders: []builtin.Builder{
		builtin.ParseSource(),
		builtin.BuildFunc(concat{}),
		builtin.BuildFunc(split{}),
		builtin.BuildFunc(broadcast{}),
		builtin.BuildFunc(gather{}),
		builtin.ImplementBuiltin("SameSlice", sameSlice),
	},
}

func sameSlice(ctx engine.Env, call elements.CallAt, fn fun.Func, irFunc *ir.FuncBuiltin, args []ir.Element) (_ []ir.Element, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cannot call fmt.SameSlice: %w", err)
		}
	}()
	if len(args) != 2 {
		return nil, errors.Errorf("got %d arguments but want 2", len(args))
	}
	x, err := elements.AxesFromElement(args[0])
	if err != nil {
		return nil, fmt.Errorf("cannot get axis from argument 1: %w", err)
	}
	y, err := elements.AxesFromElement(args[1])
	if err != nil {
		return nil, fmt.Errorf("cannot get axis from argument 2: %w", err)
	}
	ok := false
	if len(x) != len(y) {
		goto ret
	}
	for i, xi := range x {
		if xi != y[i] {
			goto ret
		}
	}
	ok = true
ret:
	boolValue, err := values.AtomBoolValue(ir.BoolType(), ok)
	if err != nil {
		return nil, errors.Errorf("cannot create bool value in fmt.SameSlice")
	}
	return []ir.Element{boolValue}, nil
}
