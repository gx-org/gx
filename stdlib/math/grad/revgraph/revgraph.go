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

// Package revgraph computes a reverse graph from IR expressions.
package revgraph

import (
	"fmt"
	"go/ast"

	"github.com/gx-org/gx/base/uname"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/astbuilder"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/stdlib/math/grad/setann"
)

var traceAll = false

type (
	nodeID int

	node[T ir.Node] struct {
		graph   *Graph
		fetcher ir.Fetcher
		id      nodeID
		irnode  T
	}
)

func newNode[T ir.Node](p *processor, isrc T) node[T] {
	n := newNodeNoID[T](p, isrc)
	n.id = nodeID(p.Graph.nextID)
	p.Graph.nextID++
	return n
}

func newNodeNoID[T ir.Node](proc *processor, isrc T) node[T] {
	return node[T]{graph: proc.Graph, fetcher: proc.fetcher, irnode: isrc, id: -1}
}

func (n *node[T]) source() ast.Node {
	return n.irnode.Node()
}

func (n *node[T]) err() *fmterr.Appender {
	return n.fetcher.Err()
}

func (n *node[T]) String() string {
	return fmt.Sprint(n.irnode)
}

// VJPParam is a parameter in a function that the gradient is going to be computed with respect to.
type VJPParam struct {
	field    *ir.Field
	wrt      withRespectTo
	vjpFType *ast.FuncType
}

// Name of the field.
func (p *VJPParam) Name() string {
	return p.field.Name.Name
}

// Type of the field.
func (p *VJPParam) Type() ir.Type {
	return p.field.Type()
}

// Graph representing the compute done in a root block.
type Graph struct {
	macro  *cpevelements.CoreMacroElement
	fn     ir.Func
	unames *uname.Unique
	root   stmt
	nextID int

	params         []VJPParam
	nResults       *namedFields
	nParams        *namedFields
	typeParamsExpr []ast.Expr
}

// New reverse graph.
func New(macro *cpevelements.CoreMacroElement, fn ir.Func) (*Graph, error) {
	g := &Graph{
		macro:  macro,
		unames: uname.New(),
		fn:     fn,
	}
	fType := g.fn.FuncType()
	g.params = make([]VJPParam, fType.Params.Len())
	g.nResults = nameFields(g.unames, "res", fType.Results.Src)
	g.nParams = nameFields(g.unames, "par", fType.Params.Src)
	for i, param := range fType.Params.Fields() {
		wrt := newWRT(param)
		backwardSig, err := g.buildBackwardSignature(fType, wrt, g.nResults.fields)
		if err != nil {
			return nil, err
		}
		g.params[i] = VJPParam{
			field:    param,
			wrt:      wrt,
			vjpFType: backwardSig,
		}
	}
	for _, axisVal := range fType.AxisLengths {
		g.unames.Register(axisVal.Name())
	}
	g.unames.RegisterFieldNames(fType.TypeParams)
	return g, nil
}

func concatFieldList(lists ...*ast.FieldList) *ast.FieldList {
	all := ast.FieldList{}
	for _, list := range lists {
		all.List = append(all.List, list.List...)
	}
	return &all
}

// BuildType builds the type of the function.
func (g *Graph) BuildType() *ast.FuncType {
	fType := g.fn.FuncType()
	vjpFuncs := make([]*ast.Field, len(g.params))
	for i, param := range g.params {
		vjpFuncs[i] = &ast.Field{
			Type: param.vjpFType,
		}
	}
	vjpType := &ast.FuncType{
		// Same as the original function.
		TypeParams: fType.Src.TypeParams,
		// Same parameters (but named) as the original function.
		Params: g.nParams.fields,
		// Return the result of the original function as well as
		// a backward function to compute the gradient.
		Results: concatFieldList(
			fType.Src.Results,
			&ast.FieldList{List: vjpFuncs},
		),
	}
	return vjpType
}

// VJPs returns all the parameters with which the gradient is going to be computed with respect to.
func (g *Graph) VJPs() []VJPParam {
	return g.params
}

// Func returns the function represented by the graph.
func (g *Graph) Func() ir.Func {
	return g.fn
}

type processor struct {
	*Graph
	fetcher ir.Fetcher
}

// Process a block statement to build its corresponding graph.
func (g *Graph) Process(fetcher ir.Fetcher) (*ast.BlockStmt, bool) {
	p := &processor{Graph: g, fetcher: fetcher}
	var root stmt
	var ok bool
	if ann := setann.Get(g.fn); ann != nil {
		root, ok = p.processFuncWithAnn(ann)
	} else {
		root, ok = p.processFuncWithoutAnn()
	}
	if !ok {
		return nil, false
	}
	astmts := g.newASTOut(fetcher)
	if ok := root.build(astmts); !ok {
		return nil, false
	}
	return &ast.BlockStmt{List: astmts.stmts}, true
}

func (p *processor) processFuncWithoutAnn() (stmt, bool) {
	fnWithBody, ok := p.fn.(*ir.FuncDecl)
	if !ok {
		return nil, p.fetcher.Err().Appendf(p.fn.Node(), "function %s requires a gradient specification", p.fn.ShortString())
	}
	root, ok := p.processBlockStmt(fnWithBody.Body)
	typeParams := p.fn.FuncType().TypeParams.Fields()
	if len(typeParams) == 0 {
		return root, ok
	}
	p.typeParamsExpr = make([]ast.Expr, len(typeParams))
	for i, typeParam := range typeParams {
		p.typeParamsExpr[i] = &ast.Ident{Name: typeParam.Name.Name}
	}
	return root, ok
}

func (g *Graph) buildBackwardSignature(fType *ir.FuncType, wrt withRespectTo, namedResultFields *ast.FieldList) (*ast.FuncType, error) {
	results, err := astbuilder.Clone(&ast.FieldList{
		List: []*ast.Field{&ast.Field{
			Type: wrt.fieldType(),
		}},
	}, astbuilder.AssignToExpandShape)
	if err != nil {
		return nil, err
	}

	return &ast.FuncType{
		// Gradient coming from the output values of the function.
		Params: namedResultFields,
		// Return the gradient.
		Results: results,
	}, nil
}

func (g *Graph) funcNameWithTypeParamsExpr(fn ast.Expr) ast.Expr {
	var callee ast.Expr = fn
	switch len(g.typeParamsExpr) {
	case 0:
		return callee
	case 1:
		return &ast.IndexExpr{
			X:     callee,
			Index: g.typeParamsExpr[0],
		}
	default:
		return &ast.IndexListExpr{
			X:       callee,
			Indices: g.typeParamsExpr,
		}
	}
}
