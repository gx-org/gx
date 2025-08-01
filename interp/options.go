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

package interp

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/interp/elements"
)

type packageOption func(itp *Interpreter, pkg *ir.Package, scope *scope.RWScope[ir.Element]) error

func (itp *Interpreter) evalOptions(pkg *ir.Package, scope *scope.RWScope[ir.Element]) error {
	options := itp.packageOptions[pkg.FullName()]
	for _, option := range options {
		if err := option(itp, pkg, scope); err != nil {
			return err
		}
	}
	return nil
}

func processOptions(opts []options.PackageOption) (map[string][]packageOption, error) {
	packageOptions := make(map[string][]packageOption)
	for _, option := range opts {
		var optFunc packageOption
		var err error
		switch optionT := option.(type) {
		case options.PackageVarSetValue:
			optFunc, err = processPackageVarSetGXValue(optionT)
		case elements.PackageVarSetElement:
			optFunc, err = processPackageVarSetElement(optionT)
		default:
			err = errors.Errorf("option of type %T not supported", optionT)
		}
		if err != nil {
			return nil, err
		}
		pkg := option.Package()
		options := packageOptions[pkg]
		options = append(options, optFunc)
		packageOptions[pkg] = options
	}
	return packageOptions, nil
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

func processPackageVarSetGXValue(opt options.PackageVarSetValue) (packageOption, error) {
	return func(itp *Interpreter, pkg *ir.Package, scope *scope.RWScope[ir.Element]) error {
		vrExpr, err := findVarExpr(pkg, opt.Var)
		if err != nil {
			return err
		}
		fitp, err := itp.ForFile(vrExpr.Decl.FFile)
		if err != nil {
			return err
		}
		ident := opt.Var
		array, ok := opt.Value.(values.Array)
		if !ok {
			return errors.Errorf("package variables of type %T (used in %s.%s) not supported", opt.Value, pkg.Name, opt.Var)
		}
		node, err := itp.eval.ElementFromAtom(fitp, &ir.ValueRef{
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

func processPackageVarSetElement(opt elements.PackageVarSetElement) (packageOption, error) {
	return func(itp *Interpreter, pkg *ir.Package, scope *scope.RWScope[ir.Element]) error {
		varExpr, err := findVarExpr(pkg, opt.Var)
		if err != nil {
			return err
		}
		scope.Define(varExpr.VName.Name, opt.Value)
		return nil
	}, nil
}
