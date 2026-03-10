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
		buildForward(astmts *fwdStmts) ([]ast.Expr, bool)
		buildBackward(bckstmts *astOutWRT, bck *special.Expr) (*special.Expr, bool)
		dependsOn(bckstmts *astOutWRT) bool
		fieldPath() []*ir.Field
		String() string
	}
)

func (p *processor) processExpr(isrc ir.Expr) (expr, bool) {
	if !ir.IsValid(isrc.Type()) {
		// Invalid type: abort and an error should already has been reported.
		return nil, false
	}
	switch isrcT := isrc.(type) {
	case *ir.ArrayLitExpr:
		return p.processArrayLitExpr(isrcT)
	case *ir.UnaryExpr:
		return p.processUnaryExpr(isrcT)
	case *ir.BinaryExpr:
		return p.processBinaryExpr(isrcT)
	case *ir.NumberCastExpr:
		return p.processNumberCastExpr(isrcT)
	case *ir.Ident:
		return p.processIdent(isrcT)
	case *ir.FuncCallExpr:
		return p.processFuncCallExpr(isrcT)
	case *ir.ParenExpr:
		return p.processParenExpr(isrcT)
	case *ir.SelectorExpr:
		return p.processSelectorExpr(isrcT)
	default:
		return nil, p.fetcher.Err().Appendf(isrc.Node(), "%T not supported", isrcT)
	}
}

func buildSingleForward(astmts *fwdStmts, node expr) (ast.Expr, bool) {
	exprs, ok := node.buildForward(astmts)
	if !ok {
		return nil, false
	}
	if len(exprs) != 1 {
		return nil, astmts.err().Appendf(node.source(), "%T returns multiple forward expressions in  single-value context", node)
	}
	return exprs[0], true
}

func buildBackward(bckstmts *astOutWRT, bck *special.Expr, x expr) (*special.Expr, bool) {
	if !x.dependsOn(bckstmts) {
		return special.ZeroExpr(), true
	}
	return x.buildBackward(bckstmts, bck)
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

func (n *funcCallExpr) vjpFunc(astmts *fwdStmts) (string, ast.Expr, bool) {
	src, ok := n.irnode.Callee.(*ir.FuncValExpr)
	if !ok {
		return "", nil, astmts.err().AppendInternalf(n.irnode.Node(), "callee type %T not supported", n.irnode.Callee)
	}
	name := "Fun"
	if pkgFunc, ok := src.Func().(ir.PkgFunc); ok {
		name = pkgFunc.Name()
	}
	macro := n.graph.macro.From()
	return name, n.graph.funcNameWithTypeParamsExpr(&ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: macro.File().Package.Name.Name},
			Sel: &ast.Ident{Name: macro.Name()},
		},
		Args: []ast.Expr{src.Expr()},
	}), true
}

func (n *funcCallExpr) buildForward(astmts *fwdStmts) ([]ast.Expr, bool) {
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
		return nil, n.err().Appendf(n.irnode.Src, "%s used as value", n.irnode.Callee.SourceString(n.file()))
	}
	if len(n.fwds) > 1 {
		return nil, n.err().Appendf(n.irnode.Src, "multiple-value %s in single-value context", n.irnode.Callee.SourceString(n.file()))
	}
	return special.New(n.fwds[0]), true
}

func (n *funcCallExpr) dependsOn(bckstmts *astOutWRT) bool {
	for _, arg := range n.args {
		if arg.dependsOn(bckstmts) {
			return true
		}
	}
	return false
}

func (n *funcCallExpr) fieldPath() []*ir.Field {
	return nil
}

func (n *funcCallExpr) buildBackward(bckstmts *astOutWRT, bck *special.Expr) (*special.Expr, bool) {
	if !n.dependsOn(bckstmts) {
		return special.ZeroExpr(), true
	}
	calleeParams := n.irnode.Callee.FuncType().Params.Fields()
	backwardIdents := make([]*special.Expr, len(calleeParams))
	for i, calleeParam := range calleeParams {
		if !n.args[i].dependsOn(bckstmts) {
			continue
		}
		vjpCall := &ast.CallExpr{
			Fun:  &ast.Ident{Name: n.vjps[i]},
			Args: []ast.Expr{bck.AST()},
		}
		bckSuffix := ""
		if len(calleeParams) > 1 {
			bckSuffix = calleeParam.Name.Name
		}
		bckIdent := bckstmts.assignExprs(n.id, []ast.Expr{vjpCall}, 1, bckSuffix)
		backwardIdents[i] = special.New(bckIdent[0])
	}
	bckstmts.callTraceSpecials(backwardIdents)
	argsGrad := make([]*special.Expr, len(n.args))
	for i := len(n.args) - 1; i >= 0; i-- {
		var ok bool
		argsGrad[i], ok = buildBackward(bckstmts, backwardIdents[i], n.args[i])
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

func (n *unaryExpr) dependsOn(bckstmts *astOutWRT) bool {
	return n.x.dependsOn(bckstmts)
}

func (n *unaryExpr) buildForward(astmts *fwdStmts) ([]ast.Expr, bool) {
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

func (n *unaryExpr) buildBackward(bckstmts *astOutWRT, bck *special.Expr) (*special.Expr, bool) {
	var xbck *special.Expr
	switch n.irnode.Src.Op {
	case token.SUB:
		xbck = special.UnarySub(bck)
	default:
		return nil, bckstmts.err().Appendf(n.irnode.Src, "gradient of unary operator %s not supported", n.irnode.Src.Op)
	}
	xbck = bckstmts.assignSpecialExpr(n.id, xbck)
	return buildBackward(bckstmts, xbck, n.x)
}

func (n *unaryExpr) fieldPath() []*ir.Field {
	return nil
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

func (n *binaryExpr) dependsOn(bckstmts *astOutWRT) bool {
	return n.x.dependsOn(bckstmts) || n.y.dependsOn(bckstmts)
}

func (n *binaryExpr) fieldPath() []*ir.Field {
	return nil
}

func (n *binaryExpr) buildForward(astmts *fwdStmts) ([]ast.Expr, bool) {
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

func (n *binaryExpr) buildBackward(bckstmts *astOutWRT, bck *special.Expr) (*special.Expr, bool) {
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
	if yres != bck && n.y.dependsOn(bckstmts) {
		yres = bckstmts.assignSpecialExprSuffix(n.id, yres, "y")
	}
	ybck, yok := buildBackward(bckstmts, yres, n.y)
	if xres != bck && n.x.dependsOn(bckstmts) {
		xres = bckstmts.assignSpecialExprSuffix(n.id, xres, "x")
	}
	xbck, xok := buildBackward(bckstmts, xres, n.x)
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

func (n *numberCastExpr) dependsOn(bckstmts *astOutWRT) bool {
	return false
}

func (n *numberCastExpr) fieldPath() []*ir.Field {
	return nil
}

func (n *numberCastExpr) buildForward(astmts *fwdStmts) ([]ast.Expr, bool) {
	return []ast.Expr{n.irnode.X.Expr()}, true
}

func (n *numberCastExpr) forwardValue() (*special.Expr, bool) {
	return special.NewFromIR(n.irnode.X), true
}

func (n *numberCastExpr) buildBackward(bckstmts *astOutWRT, bck *special.Expr) (*special.Expr, bool) {
	return special.ZeroExpr(), true
}

type fieldRef struct {
	node[*ir.FieldStorage]
}

func (n *fieldRef) dependsOn(bckstmts *astOutWRT) bool {
	return bckstmts.wrt.Same(n.fieldPath())
}

func (n *fieldRef) fieldPath() []*ir.Field {
	return []*ir.Field{n.irnode.Field}
}

func (n *fieldRef) buildForward(astmts *fwdStmts) ([]ast.Expr, bool) {
	return []ast.Expr{n.irnode.Field.Name}, true
}

func (n *fieldRef) forwardValue() (*special.Expr, bool) {
	return special.New(n.irnode.Field.Name), true
}

func (n *fieldRef) buildBackward(bckstmts *astOutWRT, bck *special.Expr) (*special.Expr, bool) {
	if !n.dependsOn(bckstmts) {
		return special.ZeroExpr(), true
	}
	return bck, true
}

type ident struct {
	node[*ir.Ident]

	fwd expr
}

func (p *processor) processIdent(isrc *ir.Ident) (expr, bool) {
	fieldStorage, isField := isrc.Stor.(*ir.FieldStorage)
	if isField {
		return &fieldRef{
			node: newNodeNoID[*ir.FieldStorage](p, fieldStorage),
		}, true
	}
	return &ident{
		node: newNodeNoID[*ir.Ident](p, isrc),
	}, true
}

func (n *ident) dependsOn(bckstmts *astOutWRT) bool {
	return n.fwd.dependsOn(bckstmts)
}

func (n *ident) fieldPath() []*ir.Field {
	return n.fwd.fieldPath()
}

func (n *ident) buildForward(astmts *fwdStmts) ([]ast.Expr, bool) {
	var ok bool
	n.fwd, ok = astmts.idents.Find(n.irnode.Src.Name)
	if !ok {
		return nil, n.err().Appendf(n.irnode.Src, "identifier node %s has not been registered in the reverse graph", n.irnode.Src.Name)
	}
	return n.fwd.buildForward(astmts)
}

func (n *ident) forwardValue() (*special.Expr, bool) {
	return n.fwd.forwardValue()
}

func (n *ident) buildBackward(bckstmts *astOutWRT, bck *special.Expr) (*special.Expr, bool) {
	bckExpr, ok := buildBackward(bckstmts, bck, n.fwd)
	if !ok {
		return nil, false
	}
	return bckstmts.assignSpecialExprSuffix(n.id, bckExpr, n.irnode.Src.Name), true
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

func (n *parenExpr) dependsOn(bckstmts *astOutWRT) bool {
	return n.x.dependsOn(bckstmts)
}

func (n *parenExpr) fieldPath() []*ir.Field {
	return n.x.fieldPath()
}

func (n *parenExpr) buildForward(astmts *fwdStmts) ([]ast.Expr, bool) {
	var ok bool
	n.fwd, ok = buildSingleForward(astmts, n.x)
	return []ast.Expr{n.fwd}, ok
}

func (n *parenExpr) forwardValue() (*special.Expr, bool) {
	return special.New(n.fwd), true
}

func (n *parenExpr) buildBackward(bckstmts *astOutWRT, bck *special.Expr) (*special.Expr, bool) {
	xBack, xOk := buildBackward(bckstmts, bck, n.x)
	if !xOk {
		return nil, false
	}
	return special.Paren(xBack), true
}

type arrayLitExpr struct {
	node[*ir.ArrayLitExpr]
	xs  []expr
	fwd ast.Expr
}

func (p *processor) processArrayLitExpr(isrc *ir.ArrayLitExpr) (*arrayLitExpr, bool) {
	xs := make([]expr, len(isrc.Elts))
	ok := true
	for i, el := range isrc.Elts {
		var elOk bool
		xs[i], elOk = p.processExpr(el)
		ok = ok && elOk
	}
	return &arrayLitExpr{
		node: newNodeNoID[*ir.ArrayLitExpr](p, isrc),
		xs:   xs,
	}, ok
}

func (n *arrayLitExpr) dependsOn(bckstmts *astOutWRT) bool {
	for _, x := range n.xs {
		if x.dependsOn(bckstmts) {
			return true
		}
	}
	return false
}

func (n *arrayLitExpr) fieldPath() []*ir.Field {
	return nil
}

func (n *arrayLitExpr) buildForward(astmts *fwdStmts) ([]ast.Expr, bool) {
	ok := true
	fwds := make([]ast.Expr, len(n.xs))
	for i, x := range n.xs {
		var fwdOk bool
		fwds[i], fwdOk = buildSingleForward(astmts, x)
		ok = ok && fwdOk
	}
	n.fwd = &ast.CompositeLit{
		Type: n.node.irnode.Src.Type,
		Elts: fwds,
	}
	return []ast.Expr{n.fwd}, ok
}

func (n *arrayLitExpr) forwardValue() (*special.Expr, bool) {
	return special.New(n.fwd), true
}

func (n *arrayLitExpr) buildBackward(bckstmts *astOutWRT, bck *special.Expr) (*special.Expr, bool) {
	bcks := make([]*special.Expr, len(n.xs))
	for i, x := range n.xs {
		xBack, xOk := buildBackward(bckstmts, bck.Index(i), x)
		if !xOk {
			return nil, false
		}
		bcks[i] = xBack
	}
	return special.Add(bcks...), true
}

type selectorExpr struct {
	node[*ir.SelectorExpr]
	x            expr
	fieldStorage *ir.FieldStorage
	fwd          ast.SelectorExpr
}

func (p *processor) processSelectorExpr(isrc *ir.SelectorExpr) (*selectorExpr, bool) {
	fieldStorage, isFieldStorage := isrc.Store().(*ir.FieldStorage)
	if !isFieldStorage {
		return nil, p.fetcher.Err().Appendf(isrc.Src, "reverse of type %T not supported", isrc.Store())
	}
	x, xOk := p.processExpr(isrc.X)
	return &selectorExpr{
		fwd:          *isrc.Src,
		fieldStorage: fieldStorage,
		node:         newNodeNoID[*ir.SelectorExpr](p, isrc),
		x:            x,
	}, xOk
}

func (n *selectorExpr) dependsOn(bckstmts *astOutWRT) bool {
	return bckstmts.wrt.Same(n.fieldPath())
}

func (n *selectorExpr) fieldPath() []*ir.Field {
	return append(n.x.fieldPath(), n.fieldStorage.Field)
}

func (n *selectorExpr) buildForward(astmts *fwdStmts) ([]ast.Expr, bool) {
	var ok bool
	n.fwd.X, ok = buildSingleForward(astmts, n.x)
	return []ast.Expr{&n.fwd}, ok
}

func (n *selectorExpr) forwardValue() (*special.Expr, bool) {
	return special.New(&n.fwd), true
}

func (n *selectorExpr) buildBackward(bckstmts *astOutWRT, bck *special.Expr) (*special.Expr, bool) {
	if !bckstmts.wrt.Same(n.fieldPath()) {
		return special.ZeroExpr(), true
	}
	return bck, true
}
