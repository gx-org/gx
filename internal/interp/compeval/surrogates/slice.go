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

package surrogates

import (
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

type slice struct {
	core
	typ *ir.SliceType
}

var _ elements.ISlice = (*slice)(nil)

func newSliceType(path storepath.Path, typ *ir.SliceType) *slice {
	return &slice{
		core: core{path: path},
		typ:  typ,
	}
}

func (*slice) Type() ir.Type {
	return ir.StringType()
}

func (f *slice) Unpack(ev ir.Evaluator) (ir.Element, error) {
	return newTuple(f)
}

func (f *slice) SliceAt(_ engine.Env, expr *ir.IndexExpr, index engine.NumericalElement) (ir.Element, error) {
	return New(storepath.NewUniqueIR(expr), f.typ.DType.Val())
}

func (f *slice) Slice(_ engine.Env, expr *ir.SliceExpr, low, high engine.NumericalElement) (ir.Element, error) {
	return newSliceType(storepath.NewUniqueIR(expr), f.typ), nil
}

func (f *slice) Append(call *ir.FuncCallExpr, el []ir.Element) engine.Slice {
	return newSliceType(storepath.NewUniqueIR(call), f.typ)
}

func (f *slice) ShortString(from *ir.File) string {
	return f.path.SourceString(from)
}
