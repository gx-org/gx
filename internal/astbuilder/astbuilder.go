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

// Package astbuilder provides helper functions to build the AST tree.
package astbuilder

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/src/gx/ast"
)

// Transform an AST node.
type Transform func(ast.Node) error

type cloner struct {
	transform Transform
	errs      fmterr.Errors
}

func (cl *cloner) clone(n ast.Node) ast.Node {
	var out ast.Node
	switch nT := n.(type) {
	case *ast.FieldList:
		out = &ast.FieldList{List: cloneSlice(cl, nT.List)}
	case *ast.Field:
		o := *nT
		o.Type = clone(cl, nT.Type)
		out = &o
	case *ast.Ident:
		o := *nT
		out = &o
	case *ast.ArrayType:
		o := *nT
		o.Elt = clone(cl, nT.Elt)
		if nT.Len != nil {
			o.Len = clone(cl, nT.Len)
		}
		out = &o
	case *ast.Ellipsis:
		var elt ast.Expr
		if nT.Elt != nil {
			elt = clone(cl, nT.Elt)
		}
		out = &ast.Ellipsis{
			Ellipsis: nT.Ellipsis,
			Elt:      elt,
		}
	case *ast.BasicLit:
		o := *nT
		out = &o
	case *ast.CallExpr:
		o := *nT
		out = &o
	case *ast.IndexExpr:
		o := *nT
		o.X = clone(cl, nT.X)
		o.Index = clone(cl, nT.Index)
		out = &o
	case *ast.IndexListExpr:
		o := *nT
		o.X = clone(cl, nT.X)
		o.Indices = cloneSlice(cl, nT.Indices)
		out = &o
	case *ast.SelectorExpr:
		o := *nT
		o.X = clone(cl, nT.X)
		o.Sel = clone(cl, nT.Sel)
		out = &o
	case *ast.BinaryExpr:
		o := *nT
		o.X = clone(cl, nT.X)
		o.Y = clone(cl, nT.Y)
		out = &o
	case *ast.UnaryExpr:
		o := *nT
		o.X = clone(cl, nT.X)
		out = &o
	case *ast.ParenExpr:
		o := *nT
		o.X = clone(cl, nT.X)
		out = &o
	case *ast.CompositeLit:
		o := *nT
		if nT.Type != nil {
			o.Type = clone(cl, nT.Type)
		}
		o.Elts = cloneSlice(cl, nT.Elts)
		out = &o
	case *ast.KeyValueExpr:
		o := *nT
		o.Key = clone(cl, nT.Key)
		o.Value = clone(cl, nT.Value)
		out = &o
	default:
		cl.errs.Append(errors.Errorf("%T not supported", nT))
		return nil
	}
	if cl.transform == nil {
		return out
	}
	if err := cl.transform(out); err != nil {
		cl.errs.Append(err)
	}
	return out
}

func cloneSlice[T ast.Node](cl *cloner, nodes []T) []T {
	if nodes == nil {
		return nil
	}
	out := make([]T, len(nodes))
	for i, node := range nodes {
		out[i] = clone(cl, node)
	}
	return out
}

func clone[T ast.Node](c *cloner, n T) (outT T) {
	var out ast.Node
	out = c.clone(n)
	if out == nil {
		return
	}
	outT, ok := out.(T)
	if !ok {
		c.errs.Append(errors.Errorf("cannot cast %T to %s", out, reflect.TypeFor[T]().Name()))
	}
	return
}

// Clone a node, potentially applying a transformation function.
func Clone[T ast.Node](n T, transform Transform) (outT T, err error) {
	cl := &cloner{transform: transform}
	outT = clone[T](cl, n)
	if cl.errs.Empty() {
		return
	}
	err = fmt.Errorf("cannot clone AST for synthetic code:\n%+v", cl.errs.ToError())
	return
}
