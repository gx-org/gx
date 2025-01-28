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
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

type (
	// PackageOptionFactory creates options given a backend.
	PackageOptionFactory func(platform.Platform) PackageOption

	// PackageOption is an option specific to a package.
	PackageOption interface {
		Package() string
	}

	// PackageVarSetValue sets the value of a package level static variable.
	PackageVarSetValue struct {
		// Pck is the package owning the variable.
		Pkg string
		// Index of the variable in the package definition.
		Var string
		// Value of the static variable for the compiler.
		Value values.Value
	}
)

// Package for which the option has been built.
func (p PackageVarSetValue) Package() string {
	return p.Pkg
}

type packageOption func(ctx Context, pkg *ir.Package, fr *frame) error

func (itrp *Interpreter) processOptions(options []PackageOption) error {
	for _, option := range options {
		var optFunc packageOption
		var err error
		switch optionT := option.(type) {
		case PackageVarSetValue:
			optFunc, err = itrp.processPackageVarSetValue(optionT)
		default:
			err = errors.Errorf("option of type %T not supported", optionT)
		}
		if err != nil {
			return err
		}
		pkg := option.Package()
		options := itrp.packageOptions[pkg]
		options = append(options, optFunc)
		itrp.packageOptions[pkg] = options
	}
	return nil
}

func findVarDecl(pkg *ir.Package, name string) *ir.VarDecl {
	for _, vr := range pkg.Vars {
		if vr.VName.Name == name {
			return vr
		}
	}
	return nil
}

func (itrp *Interpreter) processPackageVarSetValue(opt PackageVarSetValue) (packageOption, error) {
	return func(ctx Context, pkg *ir.Package, fr *frame) error {
		varDecl := findVarDecl(pkg, opt.Var)
		if varDecl == nil {
			return errors.Errorf("cannot find static variable %s in package %s", opt.Var, opt.Pkg)
		}
		ident := opt.Var
		node, err := ctx.State().PkgVar(varDecl, opt.Value)
		if err != nil {
			return err
		}
		fr.Define(ident, node)
		return nil
	}, nil
}
