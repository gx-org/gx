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

package builder

import (
	"go/ast"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

var builtin = map[string]*identNode{
	"bool":    toIdentNode(ir.BoolKind),
	"float32": toIdentNode(ir.Float32Kind),
	"float64": toIdentNode(ir.Float64Kind),
	"int32":   toIdentNode(ir.Int32Kind),
	"int64":   toIdentNode(ir.Int64Kind),
	"uint32":  toIdentNode(ir.Uint32Kind),
	"uint64":  toIdentNode(ir.Uint64Kind),

	"intlen": toIdentNode(ir.AxisLengthKind),
	"intidx": toIdentNode(ir.AxisIndexKind),

	"append":    toBuiltinFunc(&appendFunc{}),
	"axlengths": toBuiltinFunc(&axlengthsFunc{}),
	"set":       toBuiltinFunc(&setFunc{}),
	"trace":     toBuiltinFunc(&traceFunc{}),

	"false": toIdentNode(ir.BoolKind),
	"true":  toIdentNode(ir.BoolKind),
}

func toBuiltinFunc(f function) *identNode {
	return &identNode{
		typeF: func(scoper) (exprNode, typeNode, bool) {
			return nil, f.typeNode(), true
		},
	}
}

type (
	identNode struct {
		ident *ast.Ident
		typeF func(scoper) (exprNode, typeNode, bool)
	}

	namespace struct {
		parent *namespace
		m      map[string]*identNode
	}
)

func newNameSpace(parent *namespace) *namespace {
	return &namespace{
		parent: parent,
		m:      make(map[string]*identNode),
	}
}

func (ns *namespace) assign(ident *ast.Ident, expr exprNode, typ typeNode) *identNode {
	if !ir.ValidIdent(ident) {
		return nil
	}
	return ns.assignName(ident.Name, ident, expr, typ)
}

func (ns *namespace) assignName(name string, ident *ast.Ident, expr exprNode, typ typeNode) *identNode {
	capture := func(scoper) (exprNode, typeNode, bool) {
		typ, ok := typeNodeOk(typ)
		return expr, typ, ok
	}
	return ns.assignTypeF(name, ident, capture)
}

func appendRedeclaredError(errF *fmterr.Appender, cur *ast.Ident, prev *identNode) {
	errF.Appendf(
		cur,
		"%s redeclared in this block\n\t%s: other declaration of %s",
		cur.Name,
		fmterr.PosString(errF.FSet().FSet, prev.ident.Pos()),
		cur.Name,
	)
}

func (ns *namespace) assignTypeF(name string, ident *ast.Ident, typeF func(scoper) (exprNode, typeNode, bool)) *identNode {
	prev := ns.fetchIdentNode(name)
	if prev != nil {
		return prev
	}
	ns.m[name] = &identNode{ident: ident, typeF: typeF}
	return nil
}

func toIdentNode(kind ir.Kind) *identNode {
	typ := &builtinType[*ir.AtomicType]{ext: ir.ScalarTypeK(kind)}
	return &identNode{
		typeF: func(scoper) (exprNode, typeNode, bool) {
			return nil, typ, true
		},
	}
}

func (ns *namespace) fetchIdentNode(name string) *identNode {
	r, ok := ns.m[name]
	if ok {
		return r
	}
	if ns.parent == nil {
		return builtin[name]
	}
	return ns.parent.fetchIdentNode(name)
}

func (ns *namespace) fetch(scope scoper, id *ast.Ident) (exprNode, typeNode, bool) {
	node := ns.fetchIdentNode(id.Name)
	if node == nil {
		scope.err().Appendf(id, "undefined: %s", id.Name)
		return nil, invalid, false
	}
	return node.typeF(scope)
}

func (ns *namespace) newChild() *namespace {
	return newNameSpace(ns)
}

func (ns *namespace) merge(owner owner, o *namespace, override bool) bool {
	ok := true
	for k, v := range o.m {
		if !override {
			prev := ns.assignTypeF(v.ident.Name, v.ident, v.typeF)
			ok = prev == nil && ok
		} else {
			ns.m[k] = v
		}
	}
	return ok
}
