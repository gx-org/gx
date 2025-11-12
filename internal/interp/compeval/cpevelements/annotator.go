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

// Annotator is a macro function to build synthetic functions.
type Annotator struct {
	ann  *ir.Annotator
	recv *fun.Receiver
}

var _ ir.FuncAnnotator = (*Annotator)(nil)

// NewAnnotator creates a new macro given its definition and a receiver.
func NewAnnotator(fn *ir.Annotator, recv *fun.Receiver) fun.Func {
	return &Annotator{ann: fn, recv: recv}
}

// Func returns the macro function.
func (f *Annotator) Func() ir.Func {
	return f.ann
}

// Recv returns the receiver of the macro function.
func (f *Annotator) Recv() *fun.Receiver {
	return f.recv
}

// Call the macro to build the synthetic element.
func (f *Annotator) Call(fctx *fun.CallEnv, call *ir.CallExpr, args []ir.Element) ([]ir.Element, error) {
	return nil, errors.Errorf("annotator gx:@%s only valid in a function annotation context", f.Name())
}

// IR of the macro function.
func (f *Annotator) IR() ir.Func {
	return f.ann
}

// Name of the annotator.
func (f *Annotator) Name() string {
	return f.ann.File().Package.Name.Name + "." + f.ann.Name()
}

// ShortString returns the name of the annotator.
func (f *Annotator) ShortString() string {
	return f.Name()
}

// Annotate a given function for a given call.
func (f *Annotator) Annotate(fetcher ir.Fetcher, fn ir.PkgFunc, call *ir.CallExpr, args []ir.Element) bool {
	return f.ann.Annotate(fetcher, f.ann, fn, call, args)
}

// Type returns the type of the function.
func (f *Annotator) Type() ir.Type {
	return f.ann.Type()
}
