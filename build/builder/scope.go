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
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
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

func elementFromStorage(scope resolveScope, store ir.Storage) (ir.Element, bool) {
	el, err := cpevelements.NewRuntimeValue(scope.fileScope().irFile(), ir.NewIdent(store))
	if err != nil {
		return el, scope.Err().Append(err)
	}
	return el, true
}

func elementFromStorageWithValue(scope resolveScope, store ir.StorageWithValue) (ir.Element, bool) {
	value := store.Value(&ir.Ident{
		Src:  store.NameDef(),
		Stor: store,
	})
	if ir.IsInvalidType(value.Type()) {
		return elementFromStorage(scope, store)
	}
	ev, ok := scope.compEval()
	if !ok {
		return nil, false
	}
	el, err := ev.fitp.EvalExpr(value)
	if err != nil {
		return nil, ev.Err().AppendAt(store.Node(), err)
	}
	return cpevelements.NewStoredValue(ev.File(), store, el), true
}

func defineLocalVar(scope resolveScope, storage ir.Storage) bool {
	if assignExpr, isAssignExpr := storage.(*ir.AssignExpr); isAssignExpr {
		if _, isLocal := assignExpr.Storage.(*ir.LocalVarStorage); !isLocal {
			return true
		}
	}
	lScope, ok := scope.(localScope)
	if !ok {
		return scope.Err().AppendInternalf(storage.Node(), "%T is not a local scope", scope)
	}
	if ir.IsInvalidType(storage.Type()) {
		return lScope.update(storage, ir.InvalidType())
	}
	var el ir.Element
	var elOk bool
	if withValue, hasValue := storage.(ir.StorageWithValue); hasValue {
		el, elOk = elementFromStorageWithValue(scope, withValue)
	} else {
		el, elOk = elementFromStorage(scope, storage)
	}
	if !elOk {
		return false
	}
	return lScope.update(storage, el)
}
