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
)

// FuncLitScope is an interpreter scope to evaluate function literals.
type FuncLitScope struct {
	fitp *FileScope
}

// NewFuncLitScope returns a new interpreter for a function literal.
func (fitp *FileScope) NewFuncLitScope(eval Evaluator) *FuncLitScope {
	itp := *fitp.itp
	itp.eval = eval
	return &FuncLitScope{fitp: &FileScope{
		itp:       &itp,
		initScope: fitp.ctx.File(),
		ctx:       fitp.ctx,
	}}
}

// FileScope returns the interpreter for a file scope.
func (litp *FuncLitScope) FileScope() *FileScope {
	return litp.fitp
}

// RunFuncLit runs a function literal given the current context.
func (litp *FuncLitScope) RunFuncLit(eval Evaluator, fn *ir.FuncLit, args []ir.Element) ([]ir.Element, error) {
	funcFrame, err := litp.fitp.ctx.PushFuncFrame(fn)
	if err != nil {
		return nil, err
	}

	assignArgumentValues(fn.FuncType(), funcFrame, args)
	for _, resultName := range fieldNames(fn.FuncType().Results.List) {
		funcFrame.Define(resultName.Name, nil)
	}
	defer litp.fitp.ctx.PopFrame()
	return evalFuncBody(litp.fitp, fn.Body)
}
