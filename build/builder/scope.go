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

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/internal/interp/compeval/srcstore"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates"
	"github.com/gx-org/gx/interp/elements"
)

type pkgScope struct {
	bpkg *basePackage
	errs *fmterr.Appender
}

func (s *pkgScope) Err() *fmterr.Appender {
	return s.errs
}

func (s *pkgScope) pkg() *basePackage {
	return s.bpkg
}

func (s *pkgScope) String() string {
	return fmt.Sprintf("%s\nerrors:%s", s.bpkg.name.Name, s.errs.String())
}

func defineGlobal(s *scope.RWScope[processNode], tok token.Token, name *ast.Ident, node ir.Storage) {
	s.Define(name.Name, newProcessNode(tok, name, node))
}

func evalExpr(scope resolveScope, x ir.Expr) (ir.Element, bool) {
	if isInvalidExpr(x) {
		return surrogates.NewInvalid(), false
	}
	ev, ok := scope.compEval()
	if !ok {
		return ir.InvalidType(), false
	}
	el, err := ev.fitp.EvalExpr(x)
	if err != nil {
		return ir.InvalidType(), ev.Err().AppendAt(x.Node(), err)
	}
	return el, true
}

func defineStoreWithValue(scope localScope, store ir.Storage, value ir.Expr) bool {
	el, evalOk := evalExpr(scope, value)
	el, err := srcstore.Link(store, el)
	linkOk := true
	if err != nil {
		linkOk = scope.Err().AppendAt(store.Node(), err)
	}
	return scope.update(store, el) && evalOk && linkOk
}

func defineStoresFromCall(scope localScope, stmt *ir.AssignCallStmt) bool {
	el, ok := evalExpr(scope, stmt.Call)
	if !ok {
		return false
	}
	tuple, err := cast.To[*elements.Tuple](el)
	if err != nil {
		return false
	}
	elts := tuple.Elements()
	for i, el := range tuple.Elements() {
		storage := stmt.List[i]
		if storage == nil {
			continue
		}
		elts[i], err = srcstore.Link(storage, el)
		if err != nil {
			ok = scope.Err().AppendAt(stmt.Node(), err)
			continue
		}
		if !scope.update(storage, elts[i]) {
			ok = false
		}
	}
	return ok
}

func defineLocalVar(scope localScope, store ir.Storage) bool {
	local, err := cast.To[*ir.LocalVarStorage](store)
	if err != nil {
		scope.update(store, ir.InvalidType())
		return scope.Err().AppendAt(store.Node(), err)
	}
	path := storepath.NewLocal(local)
	ok := true
	el, err := surrogates.New(path, store.Type())
	if err != nil {
		ok = scope.Err().AppendAt(store.Node(), err)
	}
	return scope.update(store, el) && ok
}
