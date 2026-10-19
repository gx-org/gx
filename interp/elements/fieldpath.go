// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package elements

import (
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/src/gx/ast"
)

type fieldPath struct {
	path storepath.Path
}

// NewFieldPath returns a new element representing a path in a structure.
func NewFieldPath(parent storepath.Path, field *ir.Field) FieldPath {
	return &fieldPath{
		path: storepath.NewSelect(parent, field),
	}
}

func (f *fieldPath) Root() (*storepath.Proxy, error) {
	root := storepath.Root(f.path)
	return cast.To[*storepath.Proxy](root)
}

func followOn(path storepath.Path, root ir.Element) (ir.Element, error) {
	parent := path.Parent()
	if parent == nil {
		return root, nil
	}
	cur, err := followOn(parent, root)
	if err != nil {
		return nil, err
	}
	sel, err := cast.To[engine.Selector](cur)
	if err != nil {
		return nil, err
	}
	store := path.Store()
	return sel.Select(nil, &ir.SelectorExpr{
		Src:  &ast.SelectorExpr{Sel: store.NameDef()},
		Stor: store,
	})
}

func (f *fieldPath) FollowOn(el ir.Element) (ir.Element, error) {
	return followOn(f.path, el)
}

func (f *fieldPath) Expr(ev ir.Evaluator, src ast.Expr) ([]ir.Expr, error) {
	return []ir.Expr{f.path.Expr()}, nil
}

func (*fieldPath) Type() ir.Type {
	return ir.FieldPathType()
}
