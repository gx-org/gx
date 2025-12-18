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

	"github.com/gx-org/gx/build/builder/builtins"
	"github.com/gx-org/gx/build/ir"
)

type (
	assignable interface {
		buildStorage(scope resolveScope, typ ir.Type) (_ ir.Storage, newAssign, ok bool)
	}

	identStorage struct {
		target *valueRef
		tok    token.Token
	}
)

var _ assignable = (*identStorage)(nil)

func (asg *identStorage) assign(rscope resolveScope, typ ir.Type) (_ ir.Storage, newName, ok bool) {
	name := asg.target.src
	storage, ok := findStorage(rscope, name)
	if !ok {
		return nil, false, false
	}
	return storage, false, true
}

func (asg *identStorage) define(rscope resolveScope, typ ir.Type) (_ ir.Storage, newName, ok bool) {
	name := asg.target.src.Name
	ns := rscope.nspace()
	_, isDefined := ns.Find(name)
	if builtins.Is(name) {
		// If the name is a builtin, it will already be defined in one of the parent scope.
		// Check if that builtin is defined in the local scope instead.
		isDefined = rscope.nspace().IsLocal(name)
	}
	if ir.IsNumber(typ.Kind()) {
		typ = ir.DefaultNumberType(typ.Kind())
	}
	ext := &ir.LocalVarStorage{Src: asg.target.src, Typ: typ}
	if !ir.ValidName(name) {
		return ext, !isDefined, true
	}
	return ext, !isDefined, true
}

func (asg *identStorage) anonymousStorage(rscope resolveScope, typ ir.Type) (_ ir.Storage, newName, ok bool) {
	if ir.IsNumber(typ.Kind()) {
		typ = ir.DefaultNumberType(typ.Kind())
	}
	return &ir.LocalVarStorage{
		Src: asg.target.src,
		Typ: typ,
	}, false, true
}

func (asg *identStorage) buildStorage(rscope resolveScope, typ ir.Type) (_ ir.Storage, newName, ok bool) {
	if !ir.ValidIdent(asg.target.src) || !ir.IsValid(typ) {
		return asg.anonymousStorage(rscope, typ)
	}
	if asg.tok == token.ASSIGN {
		return asg.assign(rscope, typ)
	}
	return asg.define(rscope, typ)
}

func (asg *identStorage) source() ast.Node {
	return asg.target.source()
}

type selectorStorage struct {
	target *selectorExpr
}

var _ assignable = (*selectorStorage)(nil)

func (asg *selectorStorage) source() ast.Node {
	return asg.target.source()
}

func (asg *selectorStorage) buildStorage(scope resolveScope, typ ir.Type) (_ ir.Storage, newName, ok bool) {
	ext := &ir.StructFieldStorage{}
	ext.Sel, ok = asg.target.buildSelectorExpr(scope)
	if !ok {
		return ext, false, false
	}
	if ext.Sel.Type().Kind() == ir.FuncKind {
		return ext, false, scope.Err().Appendf(asg.source(), "cannot assign to method %s", ext.Sel.Stor.NameDef().Name)
	}
	return ext, false, true
}

type assignment struct {
	target assignable
	expr   exprNode
}

type assignExprStmt struct {
	src *ast.AssignStmt

	assigns []*assignment
}

func processRightExprs(pscope procScope, stmt *ast.AssignStmt) ([]exprNode, bool) {
	rhsExprs := make([]exprNode, len(stmt.Rhs))
	ok := true
	for i, expr := range stmt.Rhs {
		var eOk bool
		if einsumCall := toEinsumCall(expr); einsumCall != nil {
			rhsExprs[i], eOk = processEinsumExpr(pscope, stmt.Lhs[i], einsumCall)
		} else {
			rhsExprs[i], eOk = processExpr(pscope, expr)
		}
		ok = ok && eOk
	}
	return rhsExprs, ok
}

func processOpAssignStmt(pscope procScope, stmt *ast.AssignStmt) (stmtNode, bool) {
	if len(stmt.Lhs) > 1 {
		pscope.Err().Appendf(stmt, "unexpected %s, expected := or = or comma", stmt.Tok)
		return nil, false
	}
	if len(stmt.Rhs) != 1 {
		pscope.Err().Appendf(stmt, "unexpected comma at end of statement")
		return nil, false
	}

	var op token.Token
	switch stmt.Tok {
	case token.ADD_ASSIGN:
		op = token.ADD
	case token.MUL_ASSIGN:
		op = token.MUL
	case token.SUB_ASSIGN:
		op = token.SUB
	case token.QUO_ASSIGN:
		op = token.QUO
	case token.REM_ASSIGN:
		op = token.REM
	case token.AND_ASSIGN:
		op = token.AND
	case token.OR_ASSIGN:
		op = token.OR
	case token.XOR_ASSIGN:
		op = token.XOR
	case token.SHL_ASSIGN:
		op = token.SHL
	case token.SHR_ASSIGN:
		op = token.SHR
	default:
		return nil, pscope.Err().Appendf(stmt, "%s not supported in assignment", stmt.Tok)
	}

	target, targetOk := leftExprToTarget(pscope, stmt, stmt.Lhs[0], make(map[string]bool))
	expr, exprOk := processBinaryExpr(pscope, &ast.BinaryExpr{X: stmt.Lhs[0], Y: stmt.Rhs[0], OpPos: stmt.TokPos, Op: op})
	n := assignExprStmt{
		src:     stmt,
		assigns: []*assignment{&assignment{target: target, expr: expr}},
	}
	return &n, targetOk && exprOk
}

func processAssign(pscope procScope, stmt *ast.AssignStmt) (stmtNode, bool) {
	if stmt.Tok != token.ASSIGN && stmt.Tok != token.DEFINE {
		return processOpAssignStmt(pscope, stmt)
	}

	rhsExprs, ok := processRightExprs(pscope, stmt)
	if !ok {
		return nil, false
	}
	if len(stmt.Lhs) == 1 || len(stmt.Rhs) > 1 {
		return processAssignStmt(pscope, stmt, rhsExprs)
	}
	// We have multiple elements on the left and only one on the right.
	// We check that we call a function (and it is not a type cast) on the right
	// to expand the function return tuple.
	call, ok := rhsExprs[0].(*callExpr)
	if ok {
		return processAssignCall(pscope, stmt, call)
	}
	return processAssignStmt(pscope, stmt, rhsExprs)
}

func leftExprToTarget(pscope procScope, stmt *ast.AssignStmt, expr ast.Expr, done map[string]bool) (assignable, bool) {
	ok := true
	switch exprT := expr.(type) {
	case *ast.Ident:
		if done[exprT.Name] {
			ok = pscope.Err().Appendf(exprT, "%s repeated on left side of %s", exprT.Name, stmt.Tok.String())
		}
		if ir.ValidIdent(exprT) {
			done[exprT.Name] = true
		}
		identExpr, identExprOk := processIdent(pscope, exprT)
		return &identStorage{target: identExpr, tok: stmt.Tok}, identExprOk && ok
	case *ast.SelectorExpr:
		target := &selectorStorage{}
		var exprOk bool
		target.target, exprOk = processSelectorExpr(pscope, exprT)
		if stmt.Tok == token.DEFINE {
			ok = pscope.Err().Appendf(exprT, "non-name %s on left side of :=", target.target.String())
		}
		return target, ok && exprOk
	case *ast.CompositeLit:
		return leftExprToTarget(pscope, stmt, exprT.Type, done)
	default:
		return nil, pscope.Err().Appendf(expr, "non-name on left side of %s", stmt.Tok)
	}
}

func leftToTargets(pscope procScope, stmt *ast.AssignStmt) ([]assignable, bool) {
	targets := make([]assignable, len(stmt.Lhs))
	done := make(map[string]bool)
	ok := true
	for i, expr := range stmt.Lhs {
		var exprOk bool
		targets[i], exprOk = leftExprToTarget(pscope, stmt, expr, done)
		ok = ok && exprOk
	}
	return targets, ok
}

func processAssignStmt(pscope procScope, src *ast.AssignStmt, rhsExprs []exprNode) (stmtNode, bool) {
	n := assignExprStmt{src: src}
	lenLeft := len(src.Lhs)
	lenRight := len(rhsExprs)
	equalOk := true
	if lenLeft != lenRight {
		equalOk = pscope.Err().Appendf(src, "assignment mismatch: %d variable(s) but %d value(s)", lenLeft, lenRight)
	}
	total := min(lenLeft, lenRight)
	targets, targetsOk := leftToTargets(pscope, src)
	n.assigns = make([]*assignment, total)
	for i := range total {
		n.assigns[i] = &assignment{
			target: targets[i],
			expr:   rhsExprs[i],
		}
	}
	return &n, equalOk && targetsOk
}

func checkNewVariables(scope resolveScope, stmt *ast.AssignStmt, newVariables bool) bool {
	if !newVariables && stmt.Tok == token.DEFINE {
		return scope.Err().Appendf(stmt, "no new variables on left side of :=")
	}
	return true
}

func buildAssignExpr(rscope resolveScope, asgm *assignment) (*ir.AssignExpr, bool, bool) {
	ext := &ir.AssignExpr{}
	var exprOk bool
	ext.X, exprOk = buildAExpr(rscope, asgm.expr)
	if tpl, isTuple := ext.X.Type().(*ir.TupleType); exprOk && isTuple {
		if len(tpl.Types) != 1 {
			exprOk = rscope.Err().Appendf(asgm.expr.source(), "multiple-value (value of type %s) in single-value context", tpl.String())
		}
	}
	var newAsgm, targetOk bool
	ext.Storage, newAsgm, targetOk = asgm.target.buildStorage(rscope, ext.X.Type())
	if !targetOk {
		return ext, false, false
	}
	if ir.IsNumber(ext.X.Type().Kind()) {
		ext.X, exprOk = castNumber(rscope, ext.X, ext.Storage.Type())
	}
	assignOk := assignableToAt(rscope, asgm.expr.source(), ext.X.Type(), ext.Storage.Type())
	definedOk := defineLocalVar(rscope, ext)
	return ext, newAsgm, exprOk && targetOk && definedOk && assignOk
}

func (n *assignExprStmt) buildStmt(scope fnResolveScope) (ir.Stmt, bool, bool) {
	ext := &ir.AssignExprStmt{Src: n.src, List: make([]*ir.AssignExpr, len(n.assigns))}
	ok := true
	newVariables := false
	for i, asgm := range n.assigns {
		var asgmNew, asgmOk bool
		ext.List[i], asgmNew, asgmOk = buildAssignExpr(scope, asgm)
		newVariables = newVariables || asgmNew
		ok = ok && asgmOk
	}
	return ext, false, ok && checkNewVariables(scope, n.src, newVariables)
}

func (n *assignExprStmt) source() ast.Node {
	return n.src
}

// assignCallStmt is a statement assigning the results of a function call to one or more variables.
type assignCallStmt struct {
	src     *ast.AssignStmt
	call    exprNode
	targets []assignable
}

func processAssignCall(pscope procScope, src *ast.AssignStmt, call *callExpr) (stmtNode, bool) {
	targets, ok := leftToTargets(pscope, src)
	return &assignCallStmt{
		src:     src,
		call:    call,
		targets: targets,
	}, ok
}
func (n *assignCallStmt) defineLeftAsInvalid(rscope fnResolveScope) bool {
	for _, target := range n.targets {
		storage, _, _ := target.buildStorage(rscope, invalidExpr().Type())
		defineLocalVar(rscope, storage)
	}
	return false
}

func (n *assignCallStmt) buildStmt(rscope fnResolveScope) (ir.Stmt, bool, bool) {
	ext := &ir.AssignCallStmt{Src: n.src}
	var callOk bool
	ext.Call, callOk = buildCall(rscope, n.call)
	lenTargets := len(n.targets)
	var funType *ir.FuncType
	if callOk {
		funType = ext.Call.FuncCall().Callee.FuncType()
	} else {
		return ext, false, n.defineLeftAsInvalid(rscope)
	}
	lenCallResults := funType.Results.Len()
	if lenTargets != funType.Results.Len() {
		return ext, false, rscope.Err().Appendf(n.source(), "assignment mismatch: %d variable(s) but %s returns %d values", lenTargets, ext.Call.String(), lenCallResults)
	}
	ext.List = make([]*ir.AssignCallResult, lenCallResults)
	total := min(lenTargets, lenCallResults)
	ok := true
	newVariables := false
	results := funType.Results.Fields()
	for i := range total {
		storage, asgmNew, asgmOk := n.targets[i].buildStorage(rscope, results[i].Type())
		ok = ok && asgmOk
		if !asgmOk {
			continue
		}
		ext.List[i] = &ir.AssignCallResult{
			Storage:     storage,
			Call:        ext.Call,
			ResultIndex: i,
		}
		if !defineLocalVar(rscope, ext.List[i]) {
			ok = false
		}
		newVariables = newVariables || asgmNew
	}
	return ext, false, ok && callOk && checkNewVariables(rscope, n.src, newVariables)
}

// Pos returns the position of the statement in the file.
func (n *assignCallStmt) source() ast.Node {
	return n.src
}
