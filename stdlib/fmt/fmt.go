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

// Package fmt provides string formatting functions.
package fmt

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/togo"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/stdlib/builtin"
)

// Package description of the GX fmt package.
var Package = builtin.PackageBuilder{
	FullPath: "fmt",
	Builders: []builtin.Builder{
		builtin.ParseSource(),
		builtin.ImplementBuiltin("sPrintf", sPrintf),
	},
}

func sPrintf(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	if len(args) < 1 {
		return nil, errors.Errorf("unexpected number of arguments to sPrintf: got %d but want (string, varargs)", len(args))
	}
	fString, err := elements.StringFromElement(args[0])
	if err != nil {
		return nil, err
	}
	goArgs, err := elements.Map(togo.Value, args[1])
	if err != nil {
		return nil, err
	}
	res, err := elements.NewString(fmt.Sprintf(fString, goArgs...), ir.StringType())
	if err != nil {
		return nil, err
	}
	return []ir.Element{res}, nil
}
