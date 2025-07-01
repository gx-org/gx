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

	"github.com/pkg/errors"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp"
)

type pkgScope struct {
	bpkg *basePackage
	errs *fmterr.Appender
}

func (s *pkgScope) err() *fmterr.Appender {
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
	ev    evaluator.Context

	file *ir.File
}

var _ ir.Fetcher = (*compileEvaluator)(nil)

func newEvaluator(scope resolveScope, irb *irBuilder, ctx evaluator.Context) (*compileEvaluator, bool) {
	file, ok := buildParentNode[*ir.File](irb, nil, scope.fileScope().bFile)
	return &compileEvaluator{
		scope: scope,
		irb:   irb,
		ev:    ctx,
		file:  file,
	}, ok
}

func (ev *compileEvaluator) update(rscope resolveScope, store ir.Storage, el elements.Element) (*compileEvaluator, bool) {
	name := store.NameDef().Name
	subEval, err := ev.ev.Sub(map[string]elements.Element{name: el})
	if err != nil {
		return ev, rscope.err().AppendInternalf(store.Source(), "cannot create compilation evaluation context: %v", err)
	}
	return newEvaluator(rscope, ev.irb, subEval)
}

func (ev *compileEvaluator) sub(src ast.Node, vals map[string]elements.Element) (*compileEvaluator, bool) {
	ctx, err := ev.ev.Sub(vals)
	if err != nil {
		return nil, ev.scope.err().AppendInternalf(src, "cannot create subcontext: %v", err)
	}
	return newEvaluator(ev.scope, ev.irb, ctx)
}

func (ev *compileEvaluator) File() *ir.File {
	return ev.file
}

func (ev *compileEvaluator) BuildExpr(src ast.Expr) (ir.Expr, bool) {
	file := ev.scope.fileScope().bFile
	pscope := ev.scope.fileScope().pkgProcScope.newScope(file)
	expr, ok := processExpr(pscope, src)
	if !ok {
		return nil, false
	}
	return expr.buildExpr(ev.scope)
}

func (ev *compileEvaluator) EvalExpr(expr ir.Expr) (canonical.Canonical, error) {
	val, err := interp.EvalExprInContext(ev.ev, expr)
	if err != nil {
		return nil, err
	}
	canonicalVal, ok := val.(canonical.Canonical)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to %s", val, reflect.TypeFor[canonical.Canonical]().String())
	}
	return canonicalVal, nil
}

func (ev *compileEvaluator) Err() *fmterr.Appender {
	return ev.scope.err()
}

func (ev *compileEvaluator) String() string {
	return gxfmt.String(ev.ev)
}

func defineGlobal(s *scope.RWScope[processNode], tok token.Token, name string, node ir.Storage) {
	s.Define(name, pNodeFromIR(tok, name, node, nil))
}

func elementFromStorage(scope resolveScope, ev *compileEvaluator, node ir.Storage) (elements.Element, bool) {
	el, err := cpevelements.NewRuntimeValue(ev.ev, node)
	if err != nil {
		return el, scope.err().Append(err)
	}
	return el, true
}

func elementFromStorageWithValue(ev *compileEvaluator, node ir.StorageWithValue) (elements.Element, bool) {
	value := node.Value(&ir.ValueRef{
		Src:  node.NameDef(),
		Stor: node,
	})
	if _, isType := value.(*ir.TypeValExpr); isType {
		return nil, true
	}
	el, err := interp.EvalExprInContext(ev.ev, value)
	if err != nil {
		return nil, ev.scope.err().Appendf(node.Source(), "cannot evaluate expression %s. Original error:\n%+v", value.String(), err)
	}
	return el, true
}

func defineLocalVar(scope resolveScope, storage ir.Storage) bool {
	lScope, ok := scope.(localScope)
	if !ok {
		return scope.err().AppendInternalf(storage.Source(), "%T is not a local scope", scope)
	}
	ev, ok := scope.compEval()
	if !ok {
		return false
	}
	var el elements.Element
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
