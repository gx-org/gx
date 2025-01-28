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
	"any":     toIdentNode(ir.UnknownKind),
	"bool":    toIdentNode(ir.BoolKind),
	"float32": toIdentNode(ir.Float32Kind),
	"float64": toIdentNode(ir.Float64Kind),
	"int32":   toIdentNode(ir.Int32Kind),
	"int64":   toIdentNode(ir.Int64Kind),
	"string":  toIdentNode(ir.StringKind),
	"uint32":  toIdentNode(ir.Uint32Kind),
	"uint64":  toIdentNode(ir.Uint64Kind),

	"intlen": toIdentNode(ir.IntLenKind),
	"intidx": toIdentNode(ir.IntIdxKind),

	"append":    toBuiltinFunc(&appendFunc{}),
	"axlengths": toBuiltinFunc(&axlengthsFunc{}),
	"set":       toBuiltinFunc(&setFunc{}),
	"trace":     toBuiltinFunc(&traceFunc{}),

	"false": toIdentNode(ir.BoolKind),
	"true":  toIdentNode(ir.BoolKind),
}

func toIdentNode(kind ir.Kind) *identNode {
	typ := &builtinType[ir.Type]{ext: ir.TypeFromKind(kind)}
	return &identNode{
		typeF: func(scoper) (typeNode, bool) {
			return typ, true
		},
	}
}

func toBuiltinFunc(f function) *identNode {
	return &identNode{
		typeF: func(scope scoper) (typeNode, bool) {
			return f.resolveType(scope)
		},
	}
}

type (
	namespaceFetcher interface {
		fetch(name string) *identNode
	}

	namespaceBase interface {
		namespaceFetcher
		assign(*identNode)
	}

	namespace interface {
		namespaceBase
		newChild() *blockNamespace
	}

	nameToIdent map[string]*identNode

	identNode struct {
		ident   *ast.Ident
		expr    staticValueNode
		typeF   func(scoper) (typeNode, bool)
		builder irBuilder
	}
)

func newIdent(ident *ast.Ident, typ typeNode) *identNode {
	return &identNode{
		ident: ident,
		typeF: func(scoper) (typeNode, bool) {
			return typeNodeOk(typ)
		},
	}
}

func newIdentExpr(ident *ast.Ident, rnode staticValueNode) *identNode {
	return &identNode{
		ident: ident,
		expr:  rnode,
		typeF: func(s scoper) (typeNode, bool) {
			return rnode.resolveType(s)
		},
	}
}

func (ns nameToIdent) assign(node *identNode) {
	ident := node.ident
	if !ir.ValidIdent(ident) {
		return
	}
	ns[ident.Name] = node
}

func (ns nameToIdent) fetch(name string) (*identNode, bool) {
	node, ok := ns[name]
	return node, ok
}

type blockNamespace struct {
	parent namespace
	nameToIdent
}

var _ namespace = (*blockNamespace)(nil)

func newNamespace(parent namespace) *blockNamespace {
	return &blockNamespace{
		nameToIdent: make(map[string]*identNode),
		parent:      parent,
	}
}

func (ns *blockNamespace) newChild() *blockNamespace {
	return newNamespace(ns)
}

func (ns *blockNamespace) fetch(name string) *identNode {
	return fetch(ns.nameToIdent.fetch, ns.parent, name)
}

func (ns *blockNamespace) merge(owner owner, o *blockNamespace, override bool) bool {
	ok := true
	for k, v := range o.nameToIdent {
		if !override {
			if prev := ns.fetch(k); prev != nil {
				ok = false
			}
		}
		ns.nameToIdent[k] = v
	}
	return ok
}

func fetch(fetcher func(string) (*identNode, bool), parent namespaceFetcher, name string) *identNode {
	r, ok := fetcher(name)
	if ok {
		return r
	}
	if parent == nil {
		return builtin[name]
	}
	return parent.fetch(name)
}

func fetchType(scope scoper, ns namespaceFetcher, id *ast.Ident) (typeNode, bool) {
	node := ns.fetch(id.Name)
	if node == nil {
		scope.err().Appendf(id, "undefined: %s", id.Name)
		return invalid, false
	}
	return node.typeF(scope)
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
