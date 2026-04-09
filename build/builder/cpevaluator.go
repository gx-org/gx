// Copyright 2026 Google LLC
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
	"go/ast"

	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp"
)

type compileEvaluator struct {
	fitp *interp.FileScope
	ferr *fmterr.Appender
}

var _ ir.Fetcher = (*compileEvaluator)(nil)

func newEvaluator(pkgitp *interp.Interpreter, file *ir.File, ferr *fmterr.Appender) (*compileEvaluator, bool) {
	fitp, err := pkgitp.ForFile(file)
	ok := true
	if err != nil {
		ok = ferr.Append(err)
	}
	return newFileEvaluator(fitp, ferr), ok
}

func newFileEvaluator(ctx *interp.FileScope, ferr *fmterr.Appender) *compileEvaluator {
	return &compileEvaluator{
		fitp: ctx,
		ferr: ferr,
	}
}

func (ev *compileEvaluator) update(store ir.Storage, el ir.Element) (*compileEvaluator, bool) {
	storeEl := cpevelements.NewStoredValue(ev.fitp.File(), store, el)
	sm := context.NewSubMap(nil)
	sm.Define(store.NameDef(), storeEl)
	return ev.sub(nil, sm)
}

func (ev *compileEvaluator) sub(file *ir.File, vals *context.SubMap) (*compileEvaluator, bool) {
	if file == nil {
		file = ev.File()
	}
	ctx, err := ev.fitp.Sub(file, vals)
	return newFileEvaluator(ctx, ev.ferr), ev.Err().Append(err)
}

func (ev *compileEvaluator) Sub(file *ir.File, vals map[string]ir.Element) (ir.Fetcher, bool) {
	return ev.sub(file, context.NewSubMap(vals))
}

func (ev *compileEvaluator) File() *ir.File {
	return ev.fitp.File()
}

func (ev *compileEvaluator) ToCompEvalError(src ast.Expr, el ir.Element) (ir.CompEvalError, error) {
	return ev.fitp.ToCompEvalError(src, el)
}

func (ev *compileEvaluator) EvalExpr(expr ir.Expr) (ir.Element, error) {
	return ev.fitp.EvalExpr(expr)
}

func (ev *compileEvaluator) Err() *fmterr.Appender {
	return ev.ferr
}

func (ev *compileEvaluator) String() string {
	return gxfmt.String(ev.fitp)
}
