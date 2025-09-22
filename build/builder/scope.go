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

	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp"
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

type compileEvaluator struct {
	scope resolveScope
	irb   *irBuilder
	fitp  *interp.FileScope
}

var _ ir.Fetcher = (*compileEvaluator)(nil)

func newEvaluator(scope resolveScope, ctx *interp.FileScope) *compileEvaluator {
	return &compileEvaluator{
		scope: scope,
		fitp:  ctx,
	}
}

func (ev *compileEvaluator) update(rscope resolveScope, store ir.Storage, el ir.Element) (*compileEvaluator, bool) {
	name := store.NameDef().Name
	subEval := ev.fitp.Sub(map[string]ir.Element{name: el})
	return newEvaluator(rscope, subEval), true
}

func (ev *compileEvaluator) sub(src ast.Node, vals map[string]ir.Element) (*compileEvaluator, bool) {
	ctx := ev.fitp.Sub(vals)
	return newEvaluator(ev.scope, ctx), true
}

func (ev *compileEvaluator) File() *ir.File {
	return ev.fitp.File()
}

func (ev *compileEvaluator) BuildExpr(src ast.Expr) (ir.Expr, bool) {
	file := ev.scope.fileScope().bFile()
	pscope := ev.scope.fileScope().pkgProcScope.newScope(file)
	expr, ok := processExpr(pscope, src)
	if !ok {
		return nil, false
	}
	return expr.buildExpr(ev.scope)
}

func (ev *compileEvaluator) EvalExpr(expr ir.Expr) (ir.Element, error) {
	return ev.fitp.EvalExpr(expr)
}

func (ev *compileEvaluator) IsDefined(name string) bool {
	_, found := ev.scope.find(name)
	return found
}

func (ev *compileEvaluator) Err() *fmterr.Appender {
	return ev.scope.Err()
}

func (ev *compileEvaluator) String() string {
	return gxfmt.String(ev.fitp)
}

func defineGlobal(s *scope.RWScope[processNode], tok token.Token, name *ast.Ident, node ir.Storage) {
	s.Define(name.Name, newProcessNode(tok, name, node))
}

func elementFromStorage(scope resolveScope, ev *compileEvaluator, node ir.Storage) (ir.Element, bool) {
	el, err := cpevelements.NewRuntimeValue(ev.fitp.File(), ev.fitp.NewFunc, node)
	if err != nil {
		return el, scope.Err().Append(err)
	}
	return el, true
}

func elementFromStorageWithValue(ev *compileEvaluator, node ir.StorageWithValue) (ir.Element, bool) {
	value := node.Value(&ir.ValueRef{
		Src:  node.NameDef(),
		Stor: node,
	})
	if _, isType := value.(*ir.TypeValExpr); isType {
		return nil, true
	}
	el, err := ev.fitp.EvalExpr(value)
	if err != nil {
		return nil, ev.scope.Err().Appendf(node.Source(), "cannot evaluate expression %s. Original error:\n%+v", value.String(), err)
	}
	return el, true
}

func defineLocalVar(scope resolveScope, storage ir.Storage) bool {
	lScope, ok := scope.(localScope)
	if !ok {
		return scope.Err().AppendInternalf(storage.Source(), "%T is not a local scope", scope)
	}
	if isInvalid(storage.Type()) {
		return lScope.update(storage, ir.InvalidType())
	}
	ev, ok := scope.compEval()
	if !ok {
		return false
	}
	var el ir.Element
	var elOk bool
	if withValue, hasValue := storage.(ir.StorageWithValue); hasValue {
		el, elOk = elementFromStorageWithValue(ev, withValue)
	} else {
		el, elOk = elementFromStorage(scope, ev, storage)
	}
	if !elOk {
		return false
	}
	return lScope.update(storage, el)
}
