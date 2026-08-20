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

package engine

import (
	"google3/third_party/cel/go/common/ast/ast"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

type proxyEnv struct {
	eng  Engine
	file *ir.File
}

func (e *proxyEnv) File() *ir.File {
	return e.file
}

func (e *proxyEnv) ExprEval() ir.Evaluator {
	return e
}
func (e *proxyEnv) EvalExpr(ir.Expr) (ir.Element, error) {
	return nil, fmterr.Internalf("not implemented")
}

func (e *proxyEnv) Sub(*ir.File, map[string]ir.Element) (ir.Evaluator, error) {
	return nil, fmterr.Internalf("not implemented")
}

func (e *proxyEnv) Engine() Engine {
	return e.eng
}

func (*proxyEnv) ToConcrete(_ ast.Expr, tp ir.Type) (ir.Type, ir.CompEvalError, error) {
	return tp, nil, nil
}

var env proxyEnv

// ProxyEnv returns a proxy implementation of the Env interface.
func ProxyEnv(eng Engine, file *ir.File) Env {
	return &proxyEnv{eng: eng, file: file}
}
