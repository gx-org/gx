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

package reflect

import (
	"go/ast"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

var fieldPathSliceType = &ir.SliceType{
	BaseType: ir.BaseType[ast.Expr]{Src: &ast.ArrayType{}},
	DType:    ir.TypeExpr(nil, ir.FieldPathType()),
	Rank:     1,
}

func evalLeaves(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	ftype := call.Callee.FuncType()
	if len(ftype.GenParams.Values) != 1 {
		return nil, fmterr.Internalf("incorrect number of generic value: got %d but want 1", len(ftype.GenParams.Values))
	}
	genType, err := cast.To[*ir.TypeGenericValue](ftype.GenParams.Values[0])
	if err != nil {
		return nil, err
	}
	defType := genType.DefinedType()
	under := ir.Underlying(defType)
	structType, ok := under.(*ir.StructType)
	if !ok {
		return nil, errors.Errorf("type %s (kind: %s) not a structure", defType.ReferString(env.File()), defType.Kind().String())
	}
	root := storepath.NewProxy()
	paths, err := parseType(root, structType)
	if err != nil {
		return nil, err
	}
	slice, err := elements.NewSlice(fieldPathSliceType, paths)
	if err != nil {
		return nil, err
	}
	return []ir.Element{slice}, nil
}

func parseType(prefix storepath.Path, tp *ir.StructType) ([]ir.Element, error) {
	var leaves []ir.Element
	for _, field := range tp.Fields.Fields() {
		leaves = append(leaves, elements.NewFieldPath(prefix, field))
	}
	return leaves, nil
}
