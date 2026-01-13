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

// Package procoptions processes the options for the interpreter.
package procoptions

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

type packageOption func(pkg *ir.Package, scope *scope.RWScope[ir.Element]) error

// Options set for an evaluation.
type Options struct {
	eval           evaluator.Evaluator
	options        []options.PackageOption
	packageOptions map[string][]packageOption
}

// New returns an option set.
func New(eval evaluator.Evaluator, opts []options.PackageOption) (*Options, error) {
	o := &Options{
		eval:           eval,
		options:        opts,
		packageOptions: make(map[string][]packageOption),
	}
	for _, option := range opts {
		var optFunc packageOption
		var err error
		switch optionT := option.(type) {
		case options.PackageVarSetValue:
			optFunc, err = o.processPackageVarSetGXValue(optionT)
		case elements.PackageVarSetElement:
			optFunc, err = o.processPackageVarSetElement(optionT)
		default:
			err = errors.Errorf("option of type %T not supported", optionT)
		}
		if err != nil {
			return nil, err
		}
		pkg := option.Package()
		options := o.packageOptions[pkg]
		options = append(options, optFunc)
		o.packageOptions[pkg] = options
	}
	return o, nil
}

// Eval evaluates options for a given package.
func (o *Options) Eval(pkg *ir.Package, scope *scope.RWScope[ir.Element]) error {
	options := o.packageOptions[pkg.FullName()]
	for _, option := range options {
		if err := option(pkg, scope); err != nil {
			return err
		}
	}
	return nil
}

func findVarExpr(pkg *ir.Package, name string) (*ir.VarExpr, error) {
	for _, vr := range pkg.Decls.Vars {
		for _, vrExpr := range vr.Exprs {
			if vrExpr.VName.Name == name {
				return vrExpr, nil
			}
		}
	}
	return nil, errors.Errorf("cannot find static variable %s in package %s", name, pkg.FullName())
}

func (o *Options) processPackageVarSetGXValue(opt options.PackageVarSetValue) (packageOption, error) {
	return func(pkg *ir.Package, scope *scope.RWScope[ir.Element]) error {
		vrExpr, err := findVarExpr(pkg, opt.Var)
		if err != nil {
			return err
		}
		ident := opt.Var
		array, ok := opt.Value.(values.Array)
		if !ok {
			return errors.Errorf("package variables of type %T (used in %s.%s) not supported", opt.Value, pkg.Name, opt.Var)
		}
		node, err := o.eval.ElementFromAtom(vrExpr.Decl.FFile, &ir.Ident{
			Src:  vrExpr.VName,
			Stor: vrExpr,
		}, array)
		if err != nil {
			return err
		}
		scope.Define(ident, node)
		return nil
	}, nil
}

func (o *Options) processPackageVarSetElement(opt elements.PackageVarSetElement) (packageOption, error) {
	return func(pkg *ir.Package, scope *scope.RWScope[ir.Element]) error {
		varExpr, err := findVarExpr(pkg, opt.Var)
		if err != nil {
			return err
		}
		scope.Define(varExpr.VName.Name, opt.Value)
		return nil
	}, nil
}
