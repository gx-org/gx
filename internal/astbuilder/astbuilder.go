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
	"go/ast"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
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
	case *ast.BasicLit:
		o := *nT
		out = &o
	default:
		cl.errs.Append(errors.Errorf("%T not supported", nT))
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

// AssignToExpandShape transforms shapes definitions like ___Shape to
// shape expansion like Shape___.
var AssignToExpandShape = toTransform(assignToExpandShape)

func assignToExpandShape(id *ast.Ident) error {
	if strings.HasPrefix(id.Name, ir.DefineAxisGroup) {
		id.Name = strings.TrimPrefix(id.Name, ir.DefineAxisGroup)
		id.Name += ir.DefineAxisGroup
	}
	return nil
}

func toTransform[T ast.Node](f func(T) error) Transform {
	return func(n ast.Node) error {
		nT, ok := n.(T)
		if !ok {
			return nil
		}
		return f(nT)
	}
}

type paramNamer struct {
	pos int
}

// NameParams names parameters that have no names.
func NameParams() Transform {
	return toTransform((&paramNamer{}).transform)
}

func (pn *paramNamer) name() string {
	return fmt.Sprintf("p%d", pn.pos)
}

func (pn *paramNamer) transform(n *ast.Field) error {
	defer func() {
		pn.pos++
	}()
	if len(n.Names) == 0 {
		n.Names = []*ast.Ident{&ast.Ident{
			Name: pn.name(),
		}}
		return nil
	}
	for _, name := range n.Names {
		if ir.ValidName(name.Name) {
			continue
		}
		name.Name = pn.name()
	}
	return nil
}
