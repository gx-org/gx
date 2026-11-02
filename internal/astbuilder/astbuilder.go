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
		o := ast.FieldList{List: make([]*ast.Field, len(nT.List))}
		for i, field := range nT.List {
			o.List[i] = clone(cl, field)
		}
		out = &o
	case *ast.Field:
		o := *nT
		o.Type = clone(cl, nT.Type)
		out = &o
	case *ast.Ident:
		o := *nT
		out = &o
	case *ast.ArrayType:
		out = &ast.ArrayType{
			Elt: clone(cl, nT.Elt),
			Len: clone(cl, nT.Len),
		}
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
	default:
		cl.errs.Append(errors.Errorf("%T not supported", nT))
	}
	if cl.transform == nil {
		return out
	}
	if err := cl.transform(out); err != nil {
		cl.errs.Append(err)
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
