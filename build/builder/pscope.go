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
	"go/token"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder/irb"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

type (
	processNode interface {
		irb.Node[*pkgResolveScope]
		ident() *ast.Ident
		token() token.Token
		bNode() any
	}

	nodeID struct {
		tok   token.Token
		ident *ast.Ident
	}

	processNodeT[T any] struct {
		id   nodeID
		node T
	}
)

var (
	_ processNode = (*processNodeT[bool])(nil)
	_ cachedIR    = (*processNodeT[bool])(nil)
)

func newProcessNode[T any](tok token.Token, id *ast.Ident, node T) *processNodeT[T] {
	return &processNodeT[T]{
		id: nodeID{
			tok:   tok,
			ident: id,
		},
		node: node,
	}
}

func (p *processNodeT[T]) ir() ir.Node {
	node, ok := any(p.node).(ir.Node)
	if !ok {
		return nil
	}
	return node
}

func (p *processNodeT[T]) ident() *ast.Ident {
	return p.id.ident
}

func (p *processNodeT[T]) token() token.Token {
	return p.id.tok
}

func (p *processNodeT[T]) bNode() any {
	return p.node
}

func (p *processNodeT[T]) Build(ibld irBuilder) (ir.Node, bool) {
	iNode, ok := any(p.node).(irb.Node[*pkgResolveScope])
	if !ok {
		return nil, ibld.Scope().Err().Append(fmterr.Internal(errors.Errorf("cannot cast %T to %s", p.node, reflect.TypeFor[irb.Node[*pkgResolveScope]]().Name())))
	}
	return ibld.Build(iNode)
}

// pkgProcScope is a package and its namespace with an error accumulator.
// This context is used in the process phase.
type pkgProcScope struct {
	pkgScope
	dcls *decls
}

func newPackageProcScope(overwriteOk bool, pkg *basePackage, errs *fmterr.Errors) *pkgProcScope {
	scope := &pkgProcScope{
		pkgScope: pkgScope{
			bpkg: pkg,
			errs: errs.NewAppender(pkg.fset),
		},
	}
	scope.dcls = newDecls(overwriteOk, scope)
	for node := range pkg.last.decls().Values() {
		scope.dcls.declarePackageName(node)
	}
	scope.dcls.methods = append([]*processNodeT[function]{}, pkg.last.methods...)
	return scope
}

func (s *pkgProcScope) decls() *decls {
	return s.dcls
}

type (
	procScope interface {
		fmterr.ErrAppender
		file() *file
		decls() *decls
		processIdentExpr(*ast.Ident) (exprNode, bool)
		pkg() *basePackage
		typeScope() typeProcScope
	}

	fileProcScope struct {
		*pkgProcScope
		f *file

		// fileNames maps all declarations to their identifier.
		// It is used to check if a name has already been declared in the package.
		fileNames map[string]*ast.Ident
	}

	fieldProcScope interface {
		procScope
	}
)

func (s *pkgProcScope) newFilePScope(f *file) *fileProcScope {
	return &fileProcScope{pkgProcScope: s, f: f}
}

func (s *fileProcScope) pkgScope() *pkgProcScope {
	return s.pkgProcScope
}

func (s *fileProcScope) file() *file {
	return s.f
}

func (s *fileProcScope) decls() *decls {
	return s.dcls
}

func (s *fileProcScope) processIdentExpr(src *ast.Ident) (exprNode, bool) {
	return processIdent(s, src)
}

func (s *fileProcScope) typeScope() typeProcScope {
	return defaultTypeProcScope(s)
}

func (s *fileProcScope) String() string {
	return fmt.Sprintf("errs: %s\n%s", s.errs.String(), s.pkgProcScope.String())
}

type typeProcScope interface {
	procScope
	axisLengthScope() procAxLenScope
}

type procAxLenScope interface {
	procScope
	processAxisExpr(ast.Expr) (axisLengthNode, bool)
}

// defaultAxLenTypeScope is the process scope used inside all axis length expressions,
// except in function parameters.
// It checks that no identifier starts with _.
// Expressions are processed like any other expressions.
type defaultAxLenTypeScope struct {
	procScope
}

func defaultTypeProcScope(s procScope) typeProcScope {
	return &defaultAxLenTypeScope{
		procScope: s,
	}
}

func (s *defaultAxLenTypeScope) axisLengthScope() procAxLenScope {
	return s
}

func (s *defaultAxLenTypeScope) processAxisExpr(expr ast.Expr) (axisLengthNode, bool) {
	return processExprAxisLength(s, expr)
}

func (s *defaultAxLenTypeScope) processIdentExpr(ident *ast.Ident) (exprNode, bool) {
	if strings.HasPrefix(ident.Name, ir.DefineAxisGroup) {
		name := strings.TrimPrefix(ident.Name, ir.DefineAxisGroup)
		return nil, s.Err().Appendf(ident, "shape %s using %s can only be defined in function parameters", name, ident.Name)
	}
	if strings.HasPrefix(ident.Name, "_") {
		return nil, s.Err().Appendf(ident, "axis length %s can only be defined in function parameters", ident.Name)
	}
	if strings.HasSuffix(ident.Name, ir.DefineAxisGroup) {
		grpIdent := *ident
		grpIdent.Name = strings.TrimSuffix(grpIdent.Name, ir.DefineAxisGroup)
		return processIdent(s, &grpIdent)
	}
	return processIdent(s, ident)
}

// funcParamScope is a process scope used inside all axis length expressions
// in the parameters section of a function signature.
// It returns axis length name definition when _ and ___ prefixes identifiers.
// Also checks that names are not defined twice.
type funcParamScope struct {
	*defaultAxLenTypeScope
	ftype *funcType
}

var _ typeProcScope = (*funcParamScope)(nil)

func (s *funcParamScope) registerAxis(axis *defineAxisLength) (*defineAxisLength, bool) {
	if _, has := s.ftype.genShapes.Load(axis.name); has {
		return axis, s.Err().Appendf(axis.src, "axis length %s can only be defined once", axis.name)
	}
	s.ftype.genShapes.Store(axis.name, newProcessNode(token.VAR, axis.src, axis))
	return axis, true
}

func (s *funcParamScope) processAxisIdent(ident *ast.Ident) (axisLengthNode, bool) {
	if strings.HasPrefix(ident.Name, ir.DefineAxisGroup) {
		name := strings.TrimPrefix(ident.Name, ir.DefineAxisGroup)
		src := &ast.Ident{NamePos: ident.NamePos, Name: name}
		return s.registerAxis(&defineAxisLength{
			src:  src,
			name: name,
			typ:  ir.IntLenSliceType(),
		})
	}
	if strings.HasPrefix(ident.Name, ir.DefineAxisLength) {
		name := strings.TrimPrefix(ident.Name, ir.DefineAxisLength)
		src := &ast.Ident{NamePos: ident.NamePos, Name: name}
		return s.registerAxis(&defineAxisLength{
			src:  src,
			name: name,
			typ:  ir.IntLenType(),
		})
	}
	return processExprAxisLength(s.defaultAxLenTypeScope, ident)
}

func (s *funcParamScope) processAxisExpr(expr ast.Expr) (axisLengthNode, bool) {
	ident, isIdent := expr.(*ast.Ident)
	if isIdent {
		return s.processAxisIdent(ident)
	}
	return processExprAxisLength(s.defaultAxLenTypeScope, expr)
}

func (s *funcParamScope) axisLengthScope() procAxLenScope {
	return s
}
