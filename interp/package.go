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
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/interp/elements"
)

func (itn *intern) NewFunc(fn ir.Func, recv *elements.Receiver) elements.Func {
	return itn.itp.eval.NewFunc(itn.itp, fn, recv)
}

func (itn *intern) InitPkgScope(pkg *ir.Package, scope *scope.RWScope[ir.Element]) (ir.Element, error) {
	itp := itn.itp
	for _, f := range pkg.Decls.Funcs {
		scope.Define(f.Name(), itp.eval.NewFunc(itp, f, nil))
	}
	for _, tp := range pkg.Decls.Types {
		scope.Define(tp.Name(), elements.NewNamedType(itn.NewFunc, tp, nil))
	}
	if err := itp.evalPackageConsts(pkg, scope); err != nil {
		return nil, err
	}
	if err := itn.itp.evalOptions(pkg, scope); err != nil {
		return nil, err
	}
	return elements.NewPackage(pkg, scope, itn.NewFunc), nil
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
