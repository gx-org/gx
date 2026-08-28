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

package testbuild

import (
	"go/ast"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval"
	"github.com/gx-org/gx/internal/interp/compeval/srcstore"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates"
	"github.com/gx-org/gx/interp"
)

// Evaluator used for tests.
type Evaluator struct {
	ferrs *fmterr.Appender
	pkg   *builder.IncrementalPackage
	fitp  ir.Evaluator
	sub   map[ir.Storage]ir.Element
}

// EvaluatorFor returns an evaluator for an incremental package.
func (b *Builder) EvaluatorFor(pkg *builder.IncrementalPackage, sub map[string]ir.Type) (*Evaluator, error) {
	importer := builder.New(b.Importers()...)
	hostEval := compeval.NewHostEvaluator(importer)
	itp, err := interp.New(hostEval, compeval.Runner(), nil)
	if err != nil {
		return nil, err
	}
	file := &ir.File{
		Package: pkg.IR(),
		Src:     &ast.File{},
	}
	fitp, err := itp.ForFile(file)
	if err != nil {
		return nil, err
	}
	var errs fmterr.Errors
	return (&Evaluator{
		ferrs: errs.NewAppender(pkg.IR().FSet),
		pkg:   pkg,
		fitp:  fitp,
	}).subIR(sub)
}

func (ev *Evaluator) subIR(tpsub map[string]ir.Type) (*Evaluator, error) {
	if tpsub == nil {
		return ev, nil
	}
	nameToElt := make(map[string]ir.Element)
	stToElt := make(map[ir.Storage]ir.Element)
	for name, typ := range tpsub {
		field := &ir.Field{
			Group: &ir.FieldGroup{
				Type: ir.TypeExpr(nil, typ),
			},
			Name: &ast.Ident{
				Name: name,
			},
		}
		srVal, err := surrogates.FieldRoot(field, field.Storage())
		if err != nil {
			return ev, err
		}
		lkVal, err := srcstore.Link(field.Storage(), srVal)
		if err != nil {
			return ev, err
		}
		field.Group.Fields = []*ir.Field{field}
		nameToElt[name] = lkVal
		stToElt[field.Storage()] = lkVal
	}
	subEv, err := ev.Sub(ev.File(), nameToElt)
	if err != nil {
		return ev, err
	}
	return &Evaluator{
		ferrs: ev.ferrs,
		pkg:   ev.pkg,
		fitp:  subEv,
		sub:   stToElt,
	}, nil
}

// File in which the expressions are evaluated.
func (ev *Evaluator) File() *ir.File {
	return ev.fitp.File()
}

// Err returns the error appender.
func (ev *Evaluator) Err() *fmterr.Appender {
	return ev.ferrs
}

// BuildExpr builds an expression using the evaluator context.
func (ev *Evaluator) BuildExpr(src string) (ir.Expr, error) {
	return ev.pkg.BuildExprFrom(src, ev.sub)
}

// EvalExpr evaluates an expression.
func (ev *Evaluator) EvalExpr(x ir.Expr) (ir.Element, error) {
	return ev.fitp.EvalExpr(x)
}

// Sub returns a sub-context from a map from name to element.
func (ev *Evaluator) Sub(file *ir.File, sub map[string]ir.Element) (ir.Evaluator, error) {
	return ev.fitp.Sub(file, sub)
}
