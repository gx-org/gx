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

package cpevelements

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/fun"
)

type coreAnnotator struct {
	ann  ir.Func
	recv *fun.Receiver
}

// Func returns the macro function.
func (f *coreAnnotator) Func() ir.Func {
	return f.ann
}

// Recv returns the receiver of the macro function.
func (f *coreAnnotator) Recv() *fun.Receiver {
	return f.recv
}

// Call the macro to build the synthetic element.
func (f *coreAnnotator) Call(fctx *fun.CallEnv, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	return nil, errors.Errorf("annotator gx:@%s only valid in a function annotation context", f.ann.ShortString())
}

// IR of the macro function.
func (f *coreAnnotator) IR() ir.Func {
	return f.ann
}

// ShortString returns the name of the annotator.
func (f *coreAnnotator) ShortString() string {
	return f.ann.ShortString()
}

// Type returns the type of the function.
func (f *coreAnnotator) Type() ir.Type {
	return f.ann.Type()
}

// FuncAnnotator is a macro function to annotate functions.
type FuncAnnotator struct {
	coreAnnotator
	fn *ir.AnnotatorFunc
}

var _ ir.FuncAnnotator = (*FuncAnnotator)(nil)

// NewFuncAnnotator creates a new function annotator.
func NewFuncAnnotator(fn *ir.AnnotatorFunc, recv *fun.Receiver) fun.Func {
	return &FuncAnnotator{
		coreAnnotator: coreAnnotator{
			ann:  fn,
			recv: recv,
		},
		fn: fn,
	}
}

// Annotate a given function for a given call.
func (f *FuncAnnotator) Annotate(fetcher ir.Fetcher, fn ir.PkgFunc, call *ir.FuncCallExpr, args []ir.Element) bool {
	return f.fn.Annotate(fetcher, f.fn, fn, call, args)
}

// FieldAnnotator is a macro function to annotate fields in structures.
type FieldAnnotator struct {
	coreAnnotator
	fn *ir.AnnotatorField
}

// NewFieldAnnotator creates a new field annotator.
func NewFieldAnnotator(fn *ir.AnnotatorField, recv *fun.Receiver) fun.Func {
	return &FieldAnnotator{
		coreAnnotator: coreAnnotator{
			ann:  fn,
			recv: recv,
		},
		fn: fn,
	}
}

// Annotate a given field group for a given call.
func (f *FieldAnnotator) Annotate(fetcher ir.Fetcher, grp *ir.FieldGroup, call *ir.FuncCallExpr, args []ir.Element) (ir.FieldListCheckImpl, bool) {
	return f.fn.Annotate(fetcher, f.fn, grp, call, args)
}
