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

package surrogates

import (
	"go/ast"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/interp/fun"
)

type named struct {
	path  storepath.Path
	ntype *fun.NamedType
}

var _ fun.NamedTypeI = (*named)(nil)

var emptyStruct = &ir.StructType{
	Fields: &ir.FieldList{},
}

func newInterface(path storepath.Path, typ ir.TypeMethods) (Element, error) {
	under, err := newStruct(path, emptyStruct)
	return &named{
		path:  path,
		ntype: fun.NewNamedType(NewFunc, typ, under),
	}, err
}

func newNamedType(path storepath.Path, typ *ir.NamedType) (Element, error) {
	under, err := New(path, typ.Underlying.Val())
	if err != nil {
		return nil, err
	}
	return &named{
		path:  path,
		ntype: fun.NewNamedType(NewFunc, typ, under),
	}, nil
}

func (n *named) Under() (ir.Element, error) {
	return n.ntype.Under()
}

func (n *named) Select(expr *ir.SelectorExpr) (ir.Element, error) {
	return n.ntype.Select(expr)
}

func (n *named) Expr(ir.Evaluator, ast.Expr) ([]ir.Expr, error) {
	return []ir.Expr{n.path.Expr()}, nil
}

func (n *named) Type() ir.Type {
	return n.ntype.Type()
}
