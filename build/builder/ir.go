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
	"fmt"
	"go/ast"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

// GX kinds internal to the compiler.
const packageKind = iota + ir.StructKind

type done int

const (
	notDone done = iota
	doneOk
	doneNotOk
)

func toDone(ok bool) done {
	if ok {
		return doneOk
	}
	return doneNotOk
}

func (d done) ok() bool {
	return d == doneOk
}

func (d done) isDone() bool {
	return d != notDone
}

type (
	// node in the tree
	node any

	// nodePos is a node attached to some GX source code, pointed by `pos()`
	nodePos interface {
		node
		source() ast.Node
	}

	reconciler interface {
		// reconcile two types.
		// Return nil if the type could not be reconciled.
		reconcileWith(scoper, nodePos, typeNode) (typeNode, bool)
	}

	converter interface {
		convertTo(scoper, nodePos, typeNode) (typeNode, bool)
	}

	arrayTypeNode interface {
		converter
		reconciler
		typeNode
		dtype() typeNode
		rank() rankNode
	}

	// typeNode represents a GX type.
	typeNode interface {
		fmt.Stringer
		node
		kind() ir.Kind
		// irType returns the current intermediate representation of the type.
		// The type returned is in its current state and not explicitly built.
		irType() ir.Type
		isGeneric() bool
	}

	typeNodePos interface {
		nodePos
		typeNode
	}

	concreteTypeNode interface {
		typeNode

		// resolveConcreteType recursively calls resolveTypes in the underlying subtree of nodes.
		resolveConcreteType(scoper) (typeNode, bool)
	}

	genericCallTypeNode interface {
		typeNode

		// resolveGenericCallType recursively calls resolveTypes in the underlying subtree of nodes.
		// The type returned by the function may depend on the parameters given to the function.
		resolveGenericCallType(scoper, ir.Fetcher, *callExpr) (*funcType, bool)
	}

	resolverNode interface {
		node

		// resolveType recursively calls resolveTypes in the underlying subtree of nodes.
		resolveType(scoper) (typeNode, bool)
	}

	staticValueNode interface {
		resolverNode
		staticValue() ir.StaticValue
	}

	// exprNode builds a IR expression.
	exprNode interface {
		nodePos
		resolverNode

		expr() ast.Expr

		buildExpr() ir.Expr

		String() string
	}

	// exprScalar is an expression evaluating numbers.
	exprScalar interface {
		exprNode
		scalar() ir.StaticExpr
	}

	// stmtNode is a GX statement.
	stmtNode interface {
		nodePos

		// resolveTypes recursively calls resolveTypes in the underlying subtree of nodes.
		resolveType(*scopeBlock) bool

		buildStmt() ir.Stmt
	}
)

func nodeKindS(node any) string {
	if typ, ok := node.(typeNode); ok {
		return typ.kind().String()
	}
	return fmt.Sprintf("%T", node)
}

func assignableToAt(scope scoper, pos nodePos, orig, target typeNode) (typeNode, bool) {
	if target == nil {
		return orig, true
	}
	typ, assignable, err := assignableTo(scope, pos, orig, target)
	if err != nil {
		scope.err().AppendAt(pos.source(), err)
		return invalid, false
	}
	if !assignable {
		scope.err().Appendf(pos.source(), "cannot use %s as %s value in assignment", orig.String(), target.String())
		return invalid, false
	}
	return typ, true
}

func reconcileWith(scope scoper, pos nodePos, src, dst typeNode) (typeNode, bool, error) {
	if !dst.isGeneric() {
		return dst, true, nil
	}
	dstRe, ok := dst.(reconciler)
	if !ok {
		return dst, false, fmterr.Internal(errors.Errorf("cannot reconcile %s with %s", src.String(), dst.String()), "")
	}
	typ, ok := dstRe.reconcileWith(scope, pos, src)
	return typ, ok, nil
}

func assignableTo(scope scoper, pos nodePos, src, dst typeNode) (typeNode, bool, error) {
	ok, err := src.irType().AssignableTo(scope.evalFetcher(), dst.irType())
	if err != nil {
		return invalid, false, err
	}
	if !ok {
		return invalid, false, nil
	}
	if recType, canReconcile := dst.(reconciler); canReconcile {
		dst, ok = recType.reconcileWith(scope, pos, src)
	}
	return dst, ok, nil
}

func convertTo(scope scoper, pos nodePos, src, dst typeNode) (typeNode, bool) {
	canConvert, err := src.irType().ConvertibleTo(scope.evalFetcher(), dst.irType())
	if err != nil {
		return dst, scope.err().AppendAt(pos.source(), err)
	}
	if !canConvert {
		return dst, scope.err().Appendf(pos.source(), "cannot convert type %s to type %s", src.String(), dst.String())
	}
	converterType, ok := src.(converter)
	if !ok {
		return dst, scope.err().AppendInternalf(pos.source(), "type %T does not implement %T", src, reflect.TypeFor[converter]())
	}
	return converterType.convertTo(scope, pos, dst)
}

// irExprNode encapsulates an expression for the builder.
type irExprNode struct {
	x ir.Expr
}

var _ exprNode = (*irExprNode)(nil)

func toExprNode(expr ir.Expr) *irExprNode {
	return &irExprNode{x: expr}
}

func (n *irExprNode) source() ast.Node {
	return n.expr()
}

func (n *irExprNode) expr() ast.Expr {
	return n.x.Expr()
}

func (n *irExprNode) resolveType(scope scoper) (typeNode, bool) {
	return toTypeNode(scope, n.x.Type())
}

func (n *irExprNode) buildExpr() ir.Expr {
	return n.x
}

func (n *irExprNode) scalar() ir.StaticValue {
	return n.x.(ir.StaticValue)
}

func (n *irExprNode) String() string {
	return n.x.String()
}

func toString(node ast.Node) string {
	switch nodeT := node.(type) {
	case *ast.Ident:
		return nodeT.Name
	case *ast.SelectorExpr:
		return toString(nodeT.X) + "." + nodeT.Sel.Name
	default:
		return fmt.Sprintf("%T", nodeT)
	}
}
