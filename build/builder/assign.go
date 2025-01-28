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
	"go/token"

	"github.com/gx-org/gx/build/ir"
)

type (
	assignable interface {
		nodePos
		canAssign(scope scoper, exprType typeNode) (newAssign, ok bool)
		buildAssignable() ir.Assignable
		resolveType(scope scoper) (typeNode, bool)
	}

	identAssignable struct {
		target *valueRef
	}
)

var (
	_ assignable = (*identAssignable)(nil)
	_ assignable = (*selectorAssignable)(nil)
)

func (asg *identAssignable) canAssign(scope scoper, exprType typeNode) (newAssign, ok bool) {
	defined := scope.namespace().fetch(asg.target.ident().Name)
	if defined == nil {
		if asg.target.typ == nil {
			asg.target.typ = exprType
		}
		scope.namespace().assign(newIdent(asg.target.ident(), exprType))
		return true, true
	}
	var targetTypeOk bool
	asg.target.typ, targetTypeOk = defined.typeF(scope)
	if !targetTypeOk {
		return false, false
	}
	asg.target.typ, ok = assignableToAt(scope, asg, exprType, asg.target.typ)
	return false, ok
}

func (asg *identAssignable) resolveType(scope scoper) (typeNode, bool) {
	defined := scope.namespace().fetch(asg.target.ident().Name)
	if defined == nil {
		return unknown, true
	}
	typ, ok := defined.typeF(scope)
	if !ok {
		return typ, false
	}
	asg.target.typ, ok = assignableToAt(scope, asg, typ, asg.target.typ)
	return typ, ok
}

func (asg *identAssignable) buildAssignable() ir.Assignable {
	return &ir.LocalVarAssign{
		Src:   asg.target.ext.Src,
		TypeF: asg.target.typ.irType(),
	}
}

func (asg *identAssignable) source() ast.Node {
	return asg.target.source()
}

type selectorAssignable struct {
	target *selectorExpr
	field  *fieldSelectorExpr
}

func (asg *selectorAssignable) buildAssignable() ir.Assignable {
	return &ir.StructFieldAssign{
		Src:       asg.target.src,
		X:         asg.target.x.buildExpr(),
		TypeF:     asg.target.typ.irType(),
		FieldName: asg.field.field.ext.Name.Name,
	}
}

func (asg *selectorAssignable) source() ast.Node {
	return asg.target.source()
}

func (asg *selectorAssignable) canAssign(scope scoper, origType typeNode) (newAssign, ok bool) {
	targetType, ok := asg.target.resolveType(scope)
	if !ok {
		scope.err().Appendf(asg.source(), "undefined: %s", asg.target.String())
		return false, false
	}
	asg.field, ok = asg.target.sel.(*fieldSelectorExpr)
	if !ok {
		scope.err().Appendf(asg.source(), "cannot assign a value to %s", asg.target.sel)
		return false, false
	}
	_, ok = assignableToAt(scope, asg, origType, targetType)
	return false, ok
}

func (asg *selectorAssignable) resolveType(scope scoper) (typeNode, bool) {
	return asg.target.resolveType(scope)
}

type (
	assignment struct {
		target assignable
		expr   exprNode
		typ    typeNode
	}

	// assignExprStmt is a statement assigning expression values to variables.
	assignExprStmt struct {
		ext ir.AssignExprStmt

		assigns []*assignment
	}
)

func processRightExprs(owner owner, stmt *ast.AssignStmt) ([]exprNode, bool) {
	rhsExprs := make([]exprNode, len(stmt.Rhs))
	ok := true
	for i, expr := range stmt.Rhs {
		var eOk bool
		if einsumCall := toEinsumCall(expr); einsumCall != nil {
			rhsExprs[i], eOk = processEinsumExpr(owner, stmt.Lhs[i], einsumCall)
		} else {
			rhsExprs[i], eOk = processExpr(owner, expr)
		}
		ok = ok && eOk
	}
	return rhsExprs, ok
}

func processAssign(owner owner, stmt *ast.AssignStmt) (stmtNode, bool) {
	rhsExprs, ok := processRightExprs(owner, stmt)
	if !ok || len(stmt.Lhs) == 1 || len(stmt.Rhs) > 1 {
		return processAssignStmt(owner, stmt, rhsExprs)
	}
	// We have multiple elements on the left and only one on the right.
	// We check that we call a function (and it is not a type cast) on the right
	// to expand the function return tuple.
	call, ok := rhsExprs[0].(*callExpr)
	if ok {
		return processAssignCall(owner, stmt, call)
	}
	return processAssignStmt(owner, stmt, rhsExprs)
}

func leftExprToTarget(owner owner, stmt *ast.AssignStmt, expr ast.Expr, done map[string]bool) (assignable, bool) {
	ok := true
	switch exprT := expr.(type) {
	case *ast.Ident:
		if done[exprT.Name] {
			ok = owner.err().Appendf(exprT, "%s repeated on left side of %s", exprT.Name, stmt.Tok.String())
		}
		if ir.ValidIdent(exprT) {
			done[exprT.Name] = true
		}
		return &identAssignable{target: processIdentExpr(exprT)}, ok
	case *ast.SelectorExpr:
		if stmt.Tok.String() == ":=" {
			ok = owner.err().Appendf(exprT, "non-name %s on left side of :=", toString(exprT))
		}
		target := &selectorAssignable{}
		var exprOk bool
		target.target, exprOk = processSelectorReference(owner, exprT)
		return target, ok && exprOk
	case *ast.CompositeLit:
		return leftExprToTarget(owner, stmt, exprT.Type, done)
	default:
		return nil, owner.err().Appendf(expr, "%T not supported on left-side of assignment", exprT)
	}
}

func leftToTargets(owner owner, stmt *ast.AssignStmt) ([]assignable, bool) {
	targets := make([]assignable, len(stmt.Lhs))
	done := make(map[string]bool)
	ok := true
	for i, expr := range stmt.Lhs {
		var exprOk bool
		targets[i], exprOk = leftExprToTarget(owner, stmt, expr, done)
		ok = ok && exprOk
	}
	return targets, ok
}

func processAssignStmt(owner owner, stmt *ast.AssignStmt, rhsExprs []exprNode) (stmtNode, bool) {
	n := assignExprStmt{
		ext: ir.AssignExprStmt{
			Src: stmt,
		},
	}
	if len(stmt.Lhs) != len(rhsExprs) {
		owner.err().Appendf(stmt, "assignment mismatch: %d variable(s) but %d value(s)", len(stmt.Lhs), len(stmt.Rhs))
		return nil, false
	}
	targets, ok := leftToTargets(owner, stmt)
	n.assigns = make([]*assignment, len(targets))
	for i, expr := range rhsExprs {
		n.assigns[i] = &assignment{
			target: targets[i],
			expr:   expr,
		}
	}
	return &n, ok
}

func checkNewVariables(scope scoper, stmt *ast.AssignStmt, newVariables bool) bool {
	if !newVariables && stmt.Tok != token.ASSIGN {
		return scope.err().Appendf(stmt, "no new variables on left side of :=")
	}
	return true
}

func (n *assignExprStmt) resolveType(scope *scopeBlock) bool {
	ok := true
	newVariables := false
	for _, asgm := range n.assigns {
		var exprOk bool
		if asgm.typ, exprOk = asgm.expr.resolveType(scope); !exprOk {
			ok = false
			continue
		}
		if tpl, tplOk := asgm.typ.(*tupleType); tplOk {
			if tpl.len() != 1 {
				scope.err().Appendf(n.source(), "multiple-value (value of type %s) in single-value context", tpl.String())
				ok = false
				continue
			}
			asgm.typ = tpl.elt(0).typ()
		}
		targetTyp, targetTypOk := asgm.target.resolveType(scope)
		if !targetTypOk {
			ok = false
		}
		if targetTypOk && ir.IsNumber(asgm.typ.kind()) {
			asgm.expr, asgm.typ, exprOk = castNumber(scope, asgm.expr, targetTyp.irType())
			ok = ok && exprOk
		}
		asgmNew, asgmOk := asgm.target.canAssign(scope, asgm.typ)
		newVariables = newVariables || asgmNew
		ok = ok && asgmOk
	}
	return ok && checkNewVariables(scope, n.ext.Src, newVariables)
}

func (n *assignExprStmt) buildStmt() ir.Stmt {
	n.ext.List = make([]ir.AssignExpr, len(n.assigns))
	for i, asg := range n.assigns {
		n.ext.List[i] = ir.AssignExpr{
			Expr: asg.expr.buildExpr(),
			Dest: asg.target.buildAssignable(),
		}
	}
	return &n.ext
}

func (n *assignExprStmt) source() ast.Node {
	return n.ext.Source()
}

// assignCallStmt is a statement assigning the results of a function call to one or more variables.
type assignCallStmt struct {
	ext        ir.AssignCallStmt
	call       exprNode
	calleeType *funcType
	targets    []assignable
}

func processAssignCall(owner owner, stmt *ast.AssignStmt, call *callExpr) (stmtNode, bool) {
	targets, ok := leftToTargets(owner, stmt)
	n := &assignCallStmt{
		ext: ir.AssignCallStmt{
			Src: stmt,
		},
		call:    call,
		targets: targets,
	}
	return n, ok
}

func (n *assignCallStmt) buildStmt() ir.Stmt {
	n.ext.Call = n.call.buildExpr().(*ir.CallExpr)
	n.ext.List = make([]ir.Assignable, len(n.targets))
	for i, target := range n.targets {
		n.ext.List[i] = target.buildAssignable()
	}
	return &n.ext
}

func (n *assignCallStmt) resolveType(scope *scopeBlock) bool {
	typ, ok := n.call.resolveType(scope)
	if !ok {
		return false
	}
	callResults, ok := typ.(*tupleType)
	if !ok {
		scope.err().AppendInternalf(n.source(), "cannot call non-function %s (variable of type %s)", n.call.String(), typ.String())
		return false
	}
	n.calleeType = callResults.fn
	if len(n.targets) != callResults.len() {
		scope.err().Appendf(n.source(), "assignment mismatch: %d variable(s) but %s returns %d values", len(n.targets), n.call.String(), callResults.len())
		return false
	}
	newVariables := false
	for i, target := range n.targets {
		asgmNew, asgmOk := target.canAssign(scope, callResults.elt(i).typ())
		newVariables = newVariables || asgmNew
		ok = ok && asgmOk
	}
	return ok && checkNewVariables(scope, n.ext.Src, newVariables)
}

// Pos returns the position of the statement in the file.
func (n *assignCallStmt) source() ast.Node {
	return n.ext.Source()
}
