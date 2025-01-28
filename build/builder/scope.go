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

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

type scopePackage struct {
	bpkg *basePackage
	errs *fmterr.Appender
}

func newScopePackage(pkg *basePackage, errs *fmterr.Errors) *scopePackage {
	return &scopePackage{
		bpkg: pkg,
		errs: errs.NewAppender(pkg.repr.FSet),
	}
}

func (s *scopePackage) err() *fmterr.Appender {
	return s.errs
}

func (s *scopePackage) pkg() *basePackage {
	return s.bpkg
}

func (s *scopePackage) String() string {
	return fmt.Sprintf("%s\nerrors:%s", s.bpkg.repr.Name.Name, s.errs.String())
}

type scopeFile struct {
	*scopePackage
	src *file
}

func (s *scopePackage) scopeFile(src *file) *scopeFile {
	return &scopeFile{scopePackage: s, src: src}
}

func (s *scopeFile) file() *file {
	return s.src
}

func (s *scopeFile) namespace() namespace {
	return s.src.ns
}

func (s *scopeFile) fileScope() *scopeFile {
	return s
}

func (s *scopeFile) scope() scoper {
	return s
}

func (s *scopeFile) evalFetcher() *evalFetcher {
	return &evalFetcher{scope: s}
}

func (s *scopeFile) String() string {
	return fmt.Sprintf("%s\n%s", s.src.repr.Name(), s.scopePackage.String())
}

type scopeBlock struct {
	*scopeFile
	fn function
	ns namespace
}

func (s *scopeFile) scopeFunc(fn function, ns namespace) *scopeBlock {
	return &scopeBlock{
		scopeFile: s,
		fn:        fn,
		ns:        ns,
	}
}

func (s *scopeBlock) namespace() namespace {
	return s.ns
}

func (s *scopeBlock) scope() scoper {
	return s
}

func (s *scopeBlock) evalFetcher() *evalFetcher {
	return &evalFetcher{scope: s}
}

func (s *scopeBlock) scopeBlock() *scopeBlock {
	return &scopeBlock{
		scopeFile: s.scopeFile,
		fn:        s.fn,
		ns:        s.ns.newChild(),
	}
}

func (s *scopeBlock) String() string {
	if s.fn == nil {
		return fmt.Sprintf("%s: <anonymous block>", s.scopeFile.String())
	}
	return fmt.Sprintf("%s: %s", s.scopeFile.String(), s.fn.name())
}

type (
	owner interface {
		err() *fmterr.Appender
		fileScope() *scopeFile
		namespace() namespace
		scope() scoper
		String() string
	}

	// fetcher fetches a type given an identifier.
	fetcher interface {
		fetch(scoper, *ast.Ident) (typeNode, bool)
		fetchIdentNode(name string) *identNode
	}

	// scope fetches a type for a given AST block such as
	// a package or a function (represented by the owner).
	scoper interface {
		owner
		// evalFetcher returns a fetcher for evaluating expression.
		evalFetcher() *evalFetcher
	}
)

var (
	_ owner = (*scopeFile)(nil)
	_ owner = (*scopeBlock)(nil)
)

type evalFetcher struct {
	scope scoper
}

func (ev *evalFetcher) FileSet() *token.FileSet {
	return ev.scope.fileScope().file().pkg.repr.FSet
}

func (ev *evalFetcher) Fetch(ident *ast.Ident) (ir.StaticValue, error) {
	node := ev.scope.namespace().fetch(ident.Name)
	if node == nil {
		return nil, nil
	}
	if node.expr == nil {
		return nil, nil
	}
	if _, ok := node.expr.resolveType(ev.scope); !ok {
		return nil, nil
	}
	return node.expr.staticValue(), nil
}

func (ev *evalFetcher) ToGoValue(ir.RuntimeValue) (any, error) {
	return nil, errors.Errorf("GX compiler does not support runtime values")
}
