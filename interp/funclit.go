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

package interp

import (
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/fun"
)

// FuncLitScope is an interpreter scope to evaluate function literals.
type FuncLitScope struct {
	eval  fun.Evaluator
	ctx   *context.Context
	lit   *ir.FuncLit
	frame *context.Frame
}

// NewFuncLitScope returns a new interpreter for a function literal.
func NewFuncLitScope(eval fun.Evaluator, ctx *context.Context, lit *ir.FuncLit) *FuncLitScope {
	ctx, frame := ctx.FuncLitFrame(lit)
	litp := &FuncLitScope{
		eval:  eval,
		ctx:   ctx,
		lit:   lit,
		frame: frame,
	}
	return litp
}

// RunFuncLit runs a function literal given the current context.
func (litp *FuncLitScope) RunFuncLit(args []ir.Element) ([]ir.Element, error) {
	funcFrame := litp.ctx.PushBlockFrame()
	fType := litp.lit.FuncType()
	assignArgumentValues(fType, funcFrame, args)
	for _, resultName := range fieldNames(fType.Results.List) {
		funcFrame.Define(resultName.Name, nil)
	}
	defer litp.ctx.PopFrame()
	fitp := newFileScope(litp.ctx, litp.eval, litp.lit.File())
	return evalFuncBody(fitp, litp.lit.Body)
}
