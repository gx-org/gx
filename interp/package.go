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
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
)

func (itn *intern) InitPkgScope(pkg *ir.Package, scope *scope.RWScope[ir.Element]) (ir.Element, error) {
	itp := itn.itp
	for _, f := range pkg.Decls.Funcs {
		scope.Define(f.Name(), itp.eval.NewFunc(itp, f, nil))
	}
	for _, tp := range pkg.Decls.Types {
		scope.Define(tp.Name(), NewNamedType(itn.itp.NewFunc, tp, nil))
	}
	if err := itp.evalPackageConsts(pkg, scope); err != nil {
		return nil, err
	}
	if err := itn.itp.options.Eval(pkg, scope); err != nil {
		return nil, err
	}
	return NewPackage(pkg, scope, itn.itp.NewFunc), nil
}

func (itp *Interpreter) evalPackageConsts(pkg *ir.Package, scope *scope.RWScope[ir.Element]) error {
	exprs, err := pkg.Decls.ConstExprs()
	if err != nil {
		return err
	}
	for _, expr := range exprs {
		if err := itp.evalPackageConstExpr(scope, expr); err != nil {
			return err
		}
	}
	return nil
}

func (itp *Interpreter) evalPackageConstExpr(scope *scope.RWScope[ir.Element], expr *ir.ConstExpr) error {
	fCtx, err := itp.ForFile(expr.Decl.FFile)
	if err != nil {
		return err
	}
	el, err := fCtx.EvalExpr(expr.Val)
	if err != nil {
		return err
	}
	scope.Define(expr.VName.Name, el)
	return nil
}

// Package groups elements exported by a package.
type Package struct {
	pkg  *ir.Package
	defs scope.Scope[ir.Element]
}

var (
	_ ir.Element = (*Package)(nil)
	_ Selector   = (*Package)(nil)
)

// NewPackage returns a package grouping everything that a package exports.
func NewPackage(pkg *ir.Package, defs scope.Scope[ir.Element], newFunc NewFunc) *Package {
	return &Package{pkg: pkg, defs: defs}
}

// Type of the element.
func (pkg *Package) Type() ir.Type {
	return ir.PackageType()
}

// Select a member of the package.
func (pkg *Package) Select(expr *ir.SelectorExpr) (ir.Element, error) {
	name := expr.Stor.NameDef().Name
	el, ok := pkg.defs.Find(name)
	if !ok {
		return nil, errors.Errorf("%s.%s undefined", pkg.pkg.Name, name)
	}
	return el, nil
}

// String returns a string representation of the node.
func (pkg *Package) String() string {
	return "package"
}
