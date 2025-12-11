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

package revgraph

import (
	"go/ast"
	"go/token"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/math/grad/special"
)

type (
	expr interface {
		source() ast.Node
		forwardValue() (*special.Expr, bool)
		buildForward(astmts *astStmts) ([]ast.Expr, bool)
		buildBackward(bckstmts *bckStmts, bck *special.Expr) (*special.Expr, bool)
	}
)

func (p *processor) processExpr(isrc ir.Expr) (expr, bool) {
	switch isrcT := isrc.(type) {
	case *ir.UnaryExpr:
		return p.processUnaryExpr(isrcT)
	case *ir.BinaryExpr:
		return p.processBinaryExpr(isrcT)
	case *ir.NumberCastExpr:
		return p.processNumberCastExpr(isrcT)
	case *ir.ValueRef:
		return p.processValueRef(isrcT)
	case *ir.FuncCallExpr:
		return p.processFuncCallExpr(isrcT)
	case *ir.ParenExpr:
		return p.processParenExpr(isrcT)
	default:
		return nil, p.fetcher.Err().Appendf(isrc.Source(), "%T not supported", isrcT)
	}
}

func buildSingleForward(astmts *astStmts, node expr) (ast.Expr, bool) {
	exprs, ok := node.buildForward(astmts)
	if !ok {
		return nil, false
	}
	if len(exprs) != 1 {
		return nil, astmts.err().Appendf(node.source(), "%T returns multiple forward expressions in  single-value context", node)
	}
	return exprs[0], true
}

type funcCallExpr struct {
	node[*ir.FuncCallExpr]
	args []expr

	fwds []*ast.Ident
	vjps []string
}

func (p *processor) processFuncCallExpr(isrc *ir.FuncCallExpr) (*funcCallExpr, bool) {
	args := make([]expr, len(isrc.Args))
	for i, arg := range isrc.Args {
		var ok bool
		args[i], ok = p.processExpr(arg)
		if !ok {
			return nil, false
		}
	}
	return &funcCallExpr{
		node: newNode[*ir.FuncCallExpr](p, isrc),
		args: args,
	}, true
}

func (n *funcCallExpr) vjpFunc(astmts *astStmts) (string, ast.Expr, bool) {
	src, ok := n.irnode.Callee.(*ir.FuncValExpr)
	if !ok {
		return "", nil, astmts.err().AppendInternalf(n.irnode.Source(), "callee type %T not supported", n.irnode.Callee)
	}
	name := "Fun"
	if pkgFunc, ok := src.F.(ir.PkgFunc); ok {
		name = pkgFunc.Name()
	}
	macro := n.graph.macro.From()
	return name, n.graph.funcNameWithTypeParamsExpr(&ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: macro.File().Package.Name.Name},
			Sel: &ast.Ident{Name: macro.Name()},
		},
		Args: []ast.Expr{src.X.Source().(ast.Expr)},
	}), true
}

func (n *funcCallExpr) buildForward(astmts *astStmts) ([]ast.Expr, bool) {
	ftype := n.irnode.Callee.FuncType()
	if len(n.irnode.Args) == 0 {
		n.fwds = astmts.assignExprs(n.id, []ast.Expr{n.irnode.Src}, ftype.Results.Len(), "")
		return toExprs(n.fwds), true
	}
	args := make([]ast.Expr, len(n.args))
	for i, argI := range n.args {
		var ok bool
		args[i], ok = buildSingleForward(astmts, argI)
		if !ok {
			return nil, false
		}
	}
	fnName, buildVJPCall, ok := n.vjpFunc(astmts)
	if !ok {
		return nil, false
	}
	call := &ast.CallExpr{
		Fun:  buildVJPCall,
		Args: args,
	}
	n.fwds, n.vjps = astmts.assignFuncCall(n.id, ftype, fnName, call)
	return toExprs(n.fwds), true
}

func (n *funcCallExpr) forwardValue() (*special.Expr, bool) {
	if len(n.fwds) < 1 {
		return nil, n.err().Appendf(n.irnode.Src, "%s used as value", n.irnode.Callee.ShortString())
	}
	if len(n.fwds) > 1 {
		return nil, n.err().Appendf(n.irnode.Src, "multiple-value %s in single-value context", n.irnode.Callee.ShortString())
	}
	return special.New(n.fwds[0]), true
}

func (n *funcCallExpr) buildBackward(bckstmts *bckStmts, bck *special.Expr) (*special.Expr, bool) {
	if len(n.args) == 0 {
		return special.ZeroExpr(), true
	}
	calleeParams := n.irnode.Callee.FuncType().Params.Fields()
	backwardIdents := make([]*special.Expr, len(calleeParams))
	for i, param := range calleeParams {
		vjpCall := &ast.CallExpr{
			Fun:  &ast.Ident{Name: n.vjps[i]},
			Args: []ast.Expr{bck.AST()},
		}
		bckSuffix := ""
		if len(calleeParams) > 1 {
			bckSuffix = param.Name.Name
		}
		bckIdent := bckstmts.assignExprs(n.id, []ast.Expr{vjpCall}, 1, bckSuffix)
		backwardIdents[i] = special.New(bckIdent[0])
	}
	bckstmts.callTraceSpecials(backwardIdents)
	argsGrad := make([]*special.Expr, len(n.args))
	for i := len(n.args) - 1; i >= 0; i-- {
		var ok bool
		argsGrad[i], ok = n.args[i].buildBackward(bckstmts, backwardIdents[i])
		if !ok {
			return nil, false
		}
	}
	if len(argsGrad) == 1 {
		return argsGrad[0], true
	}
	return special.Add(argsGrad...), true
}

type unaryExpr struct {
	node[*ir.UnaryExpr]
	x expr

	fwd *ast.Ident
}

func (p *processor) processUnaryExpr(isrc *ir.UnaryExpr) (*unaryExpr, bool) {
	x, ok := p.processExpr(isrc.X)
	return &unaryExpr{
		node: newNode[*ir.UnaryExpr](p, isrc),
		x:    x,
	}, ok
}

func (n *unaryExpr) buildForward(astmts *astStmts) ([]ast.Expr, bool) {
	x, ok := buildSingleForward(astmts, n.x)
	if !ok {
		return nil, false
	}
	n.fwd = astmts.assignExpr(n.id, &ast.UnaryExpr{
		Op: n.irnode.Src.Op,
		X:  x,
	})
	return []ast.Expr{n.fwd}, true
}

func (n *unaryExpr) forwardValue() (*special.Expr, bool) {
	return special.New(n.fwd), true
}

func (n *unaryExpr) buildBackward(bckstmts *bckStmts, bck *special.Expr) (*special.Expr, bool) {
	var xbck *special.Expr
	switch n.irnode.Src.Op {
	case token.SUB:
		xbck = special.UnarySub(bck)
	default:
		return nil, bckstmts.err().Appendf(n.irnode.Src, "gradient of unary operator %s not supported", n.irnode.Src.Op)
	}
	xbck = bckstmts.assignSpecialExpr(n.id, xbck)
	return n.x.buildBackward(bckstmts, xbck)
}

type binaryExpr struct {
	node[*ir.BinaryExpr]
	x expr
	y expr

	fwd *ast.Ident
}

func (p *processor) processBinaryExpr(isrc *ir.BinaryExpr) (*binaryExpr, bool) {
	x, xOk := p.processExpr(isrc.X)
	y, yOk := p.processExpr(isrc.Y)
	return &binaryExpr{
		node: newNode[*ir.BinaryExpr](p, isrc),
		x:    x,
		y:    y,
	}, xOk && yOk
}

func (n *binaryExpr) buildForward(astmts *astStmts) ([]ast.Expr, bool) {
	x, xOk := buildSingleForward(astmts, n.x)
	y, yOk := buildSingleForward(astmts, n.y)
	if !xOk || !yOk {
		return nil, false
	}
	n.fwd = astmts.assignExpr(n.id, &ast.BinaryExpr{
		Op: n.irnode.Src.Op,
		X:  x,
		Y:  y,
	})
	return []ast.Expr{n.fwd}, true
}

func (n *binaryExpr) forwardValue() (*special.Expr, bool) {
	return special.New(n.fwd), true
}

func (n *binaryExpr) buildBackward(bckstmts *bckStmts, bck *special.Expr) (*special.Expr, bool) {
	x, xOk := n.x.forwardValue()
	y, yOk := n.y.forwardValue()
	if !xOk || !yOk {
		return nil, false
	}
	var xres, yres *special.Expr
	switch n.irnode.Src.Op {
	case token.ADD:
		xres = bck
		yres = bck
	case token.SUB:
		xres = bck
		yres = special.UnarySub(bck)
	case token.MUL:
		xres = special.Mul(bck, y)
		yres = special.Mul(x, bck)
	case token.QUO:
		xres = special.Quo(bck, y)
		yres = special.Quo(
			special.UnarySub(special.Mul(x, bck)),
			special.Mul(y, y),
		)
	default:
		return nil, bckstmts.err().Appendf(n.irnode.Src, "gradient of binary operator %s not supported", n.irnode.Src.Op)
	}
	if xres != bck {
		xres = bckstmts.assignSpecialExprSuffix(n.id, xres, "x")
	}
	if yres != bck {
		yres = bckstmts.assignSpecialExprSuffix(n.id, yres, "y")
	}
	ybck, yok := n.y.buildBackward(bckstmts, yres)
	xbck, xok := n.x.buildBackward(bckstmts, xres)
	return special.Paren(special.Add(xbck, ybck)), xok && yok
}

type numberCastExpr struct {
	node[*ir.NumberCastExpr]
}

func (p *processor) processNumberCastExpr(isrc *ir.NumberCastExpr) (*numberCastExpr, bool) {
	return &numberCastExpr{
		node: newNodeNoID[*ir.NumberCastExpr](p, isrc),
	}, true
}

func (n *numberCastExpr) buildForward(astmts *astStmts) ([]ast.Expr, bool) {
	return []ast.Expr{n.irnode.X.Source().(ast.Expr)}, true
}

func (n *numberCastExpr) forwardValue() (*special.Expr, bool) {
	return special.NewFromIR(n.irnode.X), true
}

func (n *numberCastExpr) buildBackward(bckstmts *bckStmts, bck *special.Expr) (*special.Expr, bool) {
	return special.ZeroExpr(), true
}

type valueRef struct {
	node[*ir.ValueRef]
}

func (p *processor) processValueRef(isrc *ir.ValueRef) (*valueRef, bool) {
	return &valueRef{
		node: newNodeNoID[*ir.ValueRef](p, isrc),
	}, true
}

func (n *valueRef) buildForward(astmts *astStmts) ([]ast.Expr, bool) {
	return []ast.Expr{n.irnode.Src}, true
}

func (n *valueRef) forwardValue() (*special.Expr, bool) {
	return special.New(n.irnode.Src), true
}

func (n *valueRef) gradFieldStorage(bckstmts *bckStmts, bck *special.Expr, stor *ir.FieldStorage) (*special.Expr, bool) {
	if bckstmts.wrt.same(stor.Field) {
		return bck, true
	}
	return special.ZeroExpr(), true
}

func (n *valueRef) buildBackward(bckstmts *bckStmts, bck *special.Expr) (*special.Expr, bool) {
	fieldStorage, isField := n.irnode.Stor.(*ir.FieldStorage)
	if isField {
		return n.gradFieldStorage(bckstmts, bck, fieldStorage)
	}
	name := n.irnode.Stor.NameDef().Name
	expr := bckstmts.identToExpr[name]
	if expr == nil {
		return nil, n.fetcher.Err().AppendInternalf(n.irnode.Source(), "cannot find revgraph node for %q", name)
	}
	bckExpr, ok := expr.buildBackward(bckstmts, bck)
	if !ok {
		return nil, false
	}
	return bckstmts.assignSpecialExprSuffix(n.id, bckExpr, name), true
}

type parenExpr struct {
	node[*ir.ParenExpr]
	x   expr
	fwd ast.Expr
}

func (p *processor) processParenExpr(isrc *ir.ParenExpr) (*parenExpr, bool) {
	x, ok := p.processExpr(isrc.X)
	return &parenExpr{
		node: newNodeNoID[*ir.ParenExpr](p, isrc),
		x:    x,
	}, ok
}

func (n *parenExpr) buildForward(astmts *astStmts) ([]ast.Expr, bool) {
	var ok bool
	n.fwd, ok = buildSingleForward(astmts, n.x)
	return []ast.Expr{n.fwd}, ok
}

func (n *parenExpr) forwardValue() (*special.Expr, bool) {
	return special.New(n.fwd), true
}

func (n *parenExpr) buildBackward(bckstmts *bckStmts, bck *special.Expr) (*special.Expr, bool) {
	xBack, xOk := n.x.buildBackward(bckstmts, bck)
	if !xOk {
		return nil, false
	}
	return special.Paren(xBack), true
}
