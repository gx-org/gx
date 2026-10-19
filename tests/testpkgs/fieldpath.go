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

package testpkgs

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

// TestFieldPaths is a package to test field paths from the compiler.
var TestFieldPaths = testbuild.DeclarePackage{
	Src: `
package testfieldpaths

//gx:unroll
func Path[T any](path ...string) []fieldpath
`,
	Post: func(pkg *ir.Package) {
		path := pkg.FindFunc("Path").(*ir.FuncBuiltin)
		path.Impl = &pathImpl{f: path}
	},
}

type pathImpl struct{ f *ir.FuncBuiltin }

// BuildFuncType builds the type of a function given how it is called.
func (s *pathImpl) BuildFuncType(ir.TypeEval, *ir.FuncCallExpr) (*ir.FuncType, error) {
	return s.f.FuncType(), nil
}

// Name of the function.
func (s *pathImpl) Name() string {
	return s.f.Name()
}

// Implementation of the function, provided by the backend.
func (s *pathImpl) Implementation() any {
	return elements.FuncBuiltin(evalPath)
}

func evalPath(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	ftype := call.Callee.FuncType()
	genType, err := cast.To[*ir.TypeGenericValue](ftype.GenParams.Values[0])
	if err != nil {
		return nil, err
	}
	pathNames, err := elements.Map(elements.StringFromElement, args[0])
	if err != nil {
		return nil, err
	}
	if len(pathNames) == 0 {
		return nil, errors.Errorf("empty field path")
	}
	curType := genType.DefinedType()
	curPath := storepath.NewProxy()
	var fpath elements.FieldPath
	for _, name := range pathNames {
		under := ir.Underlying(curType)
		structType, ok := under.(*ir.StructType)
		if !ok {
			return nil, errors.Errorf("type %s (kind: %s) not a structure", curType.ReferString(env.File()), curType.Kind().String())
		}
		field := structType.Fields.FindField(name)
		if field == nil {
			return nil, errors.Errorf("type %s has no field %s", curType.ReferString(env.File()), name)
		}
		fpath = elements.NewFieldPath(curPath, field)
		curPath = storepath.NewSelect(curPath, field)
		curType = field.Type()
	}
	slice, err := elements.NewSlice(ir.FieldPathSliceType(), []ir.Element{fpath})
	if err != nil {
		return nil, err
	}
	return []ir.Element{slice}, nil
}
