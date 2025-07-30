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
	for node := range pkg.last.decls.Values() {
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
		processIdent(*ast.Ident) (exprNode, bool)
		pkgScope() *pkgProcScope
		axisLengthScope() procAxLenScope
	}

	fileScope struct {
		*pkgProcScope
		f *file

		// fileNames maps all declarations to their identifier. It is used to check if a name has already been declared in the package.
		fileNames map[string]*ast.Ident
	}
)

func (s *pkgProcScope) newScope(f *file) *fileScope {
	return &fileScope{pkgProcScope: s, f: f}
}

func (s *fileScope) declareFileName(src *ast.Ident) (ok bool) {
	prevPkg, exist := s.decls().declarations.Load(src.Name)
	if exist {
		return appendRedeclaredError(s.Err(), src.Name, src, prevPkg.ident())
	}
	prevFile := s.fileNames[src.Name]
	if prevFile != nil {
		return appendRedeclaredError(s.Err(), src.Name, src, prevFile)
	}
	s.fileNames[src.Name] = src
	return true
}

func (s *fileScope) pkgScope() *pkgProcScope {
	return s.pkgProcScope
}

func (s *fileScope) file() *file {
	return s.f
}

func (s *fileScope) decls() *decls {
	return s.dcls
}

func (s *fileScope) processIdent(src *ast.Ident) (exprNode, bool) {
	return processIdentExpr(s, src)
}

func (s *fileScope) String() string {
	return fmt.Sprintf("errs: %s\n%s", s.errs.String(), s.pkgProcScope.String())
}

type (
	procAxLenScope interface {
		axlenScope()
		procScope
	}

	// axLenDefaultScope is the process scope used inside all axis length expressions,
	// except in function parameters.
	// It checks that no identifier starts with _.
	// Expressions are processed like any other expressions.
	axLenDefaultScope struct {
		procScope
	}
)

func (s *fileScope) axisLengthScope() procAxLenScope {
	return &axLenDefaultScope{procScope: s}
}

func checkAxisLengthIdent(pscope procScope, ident *ast.Ident) bool {
	if strings.HasPrefix(ident.Name, "_") {
		return pscope.Err().Appendf(ident, "invalid character _ in axis length name %s", ident.Name)
	}
	return true
}

func (*axLenDefaultScope) axlenScope() {}

func (s *axLenDefaultScope) checkIdent(ident *ast.Ident) bool {
	return checkAxisLengthIdent(s, ident)
}

func (s *axLenDefaultScope) processIdent(ident *ast.Ident) (exprNode, bool) {
	if strings.HasSuffix(ident.Name, ir.DefineAxisGroup) {
		grpIdent := *ident
		grpIdent.Name = strings.TrimSuffix(grpIdent.Name, ir.DefineAxisGroup)
		return processIdentExpr(s, &grpIdent)
	}
	if ident.Name == ir.DefineAxisLength {
		return nil, s.Err().Appendf(ident, "cannot use %s as an axis identifier", ident.Name)
	}
	return processIdentExpr(s, ident)
}

type (
	funcParamScope struct {
		procScope
	}

	// axLenParamScope is a process scope used inside all axis length expressions
	// in the parameters section of a function signature.
	// It returns axis length name definition when _ and ___ prefixes identifiers.
	// Also checks that names are not defined twice.
	axLenParamScope struct {
		procScope
		defined map[string]bool
	}
)

func (s *funcParamScope) axisLengthScope() procAxLenScope {
	return &axLenParamScope{
		procScope: s,
		defined:   make(map[string]bool),
	}
}

func (*axLenParamScope) axlenScope() {}

func (s *axLenParamScope) checkIfAlreadyDefine(src ast.Node, name string) bool {
	if s.defined[name] {
		return s.Err().Appendf(src, "axis length %s assignment repeated", name)
	}
	s.defined[name] = true
	return true
}

func (s *axLenParamScope) processIdent(ident *ast.Ident) (exprNode, bool) {
	if strings.HasPrefix(ident.Name, ir.DefineAxisGroup) {
		name := strings.TrimPrefix(ident.Name, ir.DefineAxisGroup)
		return &defineAxisLength{
			src:  ident,
			name: name,
			typ:  ir.IntLenSliceType(),
		}, s.checkIfAlreadyDefine(ident, name)
	}
	if strings.HasPrefix(ident.Name, ir.DefineAxisLength) {
		name := strings.TrimPrefix(ident.Name, ir.DefineAxisLength)
		return &defineAxisLength{
			src:  ident,
			name: name,
			typ:  ir.IntLenType(),
		}, s.checkIfAlreadyDefine(ident, name)
	}
	return processIdentExpr(s, ident)
}
