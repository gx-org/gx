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

type baseFrame struct {
	scope *scope.RWScope[elements.Element]
}

func (fr *baseFrame) Define(name string, value elements.Element) {
	fr.scope.Define(name, value)
}

func (fr *baseFrame) Assign(name string, value elements.Element) error {
	return fr.scope.Assign(name, value)
}

func (fr *baseFrame) Find(key string) (value elements.Element, ok bool) {
	return fr.scope.Find(key)
}

func (fr *baseFrame) String() string {
	return fr.scope.String()
}

type packageFrame struct {
	baseFrame
	pkg         *ir.Package
	el          *elements.Package
	fileToFrame map[*ir.File]*fileFrame
}

func (ectx *EvalContext) importPackage(imp *ir.ImportDecl) (*elements.Package, error) {
	pkg, err := ectx.evaluator.Importer().Import(imp.Path)
	if err != nil {
		return nil, err
	}
	pFrame, err := ectx.packageFrame(pkg)
	if err != nil {
		return nil, err
	}
	return pFrame.el, nil
}

func (ectx *EvalContext) packageFrame(pkg *ir.Package) (*packageFrame, error) {
	pkgFrame := ectx.packageToFrame[pkg]
	if pkgFrame != nil {
		return pkgFrame, nil
	}
	pkgFrame = &packageFrame{
		baseFrame: baseFrame{
			scope: scope.NewScope(ectx.builtin.scope),
		},
		pkg:         pkg,
		el:          elements.NewPackage(pkg, ectx.evaluator.NewFunc),
		fileToFrame: make(map[*ir.File]*fileFrame),
	}
	ectx.packageToFrame[pkg] = pkgFrame
	for _, f := range pkgFrame.pkg.Decls.Funcs {
		pkgFrame.Define(f.Name(), ectx.evaluator.NewFunc(f, nil))
	}
	if err := pkgFrame.evalPackageConsts(ectx); err != nil {
		return nil, err
	}
	options := ectx.packageOptions[pkg.FullName()]
	if err := pkgFrame.evalPackageOptions(ectx, options); err != nil {
		return nil, err
	}
	return pkgFrame, nil
}

func (fr *packageFrame) evalPackageConstExpr(ectx *EvalContext, expr *ir.ConstExpr) error {
	ctx, err := ectx.newFileContext(expr.Decl.FFile)
	if err != nil {
		return err
	}
	el, err := ctx.evalExpr(expr.Val)
	if err != nil {
		return err
	}
	fr.Define(expr.VName.Name, el)
	fr.el.Define(expr.VName.Name, el)
	return nil
}

func (fr *packageFrame) evalPackageConsts(ectx *EvalContext) error {
	exprs, err := fr.pkg.Decls.ConstExprs()
	if err != nil {
		return err
	}
	for _, expr := range exprs {
		if err := fr.evalPackageConstExpr(ectx, expr); err != nil {
			return err
		}
	}
	return nil
}

func (fr *packageFrame) evalPackageOptions(ectx *EvalContext, options []packageOption) error {
	for _, option := range options {
		if err := option(ectx, fr); err != nil {
			return err
		}
	}
	return nil
}

type fileFrame struct {
	baseFrame
	parent *packageFrame
	file   *ir.File
}

func (fr *packageFrame) fileFrame(ectx *EvalContext, file *ir.File) (*fileFrame, error) {
	flFrame := fr.fileToFrame[file]
	if flFrame != nil {
		return flFrame, nil
	}
	flFrame = &fileFrame{
		baseFrame: baseFrame{
			scope: scope.NewScope(fr.scope),
		},
		parent: fr,
		file:   file,
	}
	for _, imp := range file.Imports {
		pkg, err := ectx.importPackage(imp)
		if err != nil {
			return nil, err
		}
		flFrame.Define(imp.NameDef().Name, pkg)
	}
	fr.fileToFrame[file] = flFrame
	return flFrame, nil
}

func (flFrame *fileFrame) pushFuncFrame(ctx *Context, fn ir.Func) *blockFrame {
	fnFrame := &functionFrame{
		parent:   flFrame,
		function: fn,
	}
	return ctx.pushFrame(&blockFrame{
		baseFrame: baseFrame{
			scope: scope.NewScope(flFrame.scope),
		},
		owner: fnFrame,
	})
}

type functionFrame struct {
	parent   *fileFrame
	function ir.Func
}

func (ctx *Context) fileFrame(file *ir.File) (*fileFrame, error) {
	pkgFrame, err := ctx.eval.packageFrame(file.Package)
	if err != nil {
		return nil, err
	}
	return pkgFrame.fileFrame(ctx.eval, file)
}

func (ctx *Context) pushFuncFrame(fn ir.Func) (*blockFrame, error) {
	flFrame, err := ctx.fileFrame(fn.File())
	if err != nil {
		return nil, err
	}
	return flFrame.pushFuncFrame(ctx, fn), nil
}

type blockFrame struct {
	baseFrame
	parent *blockFrame
	owner  *functionFrame
}

func (ctx *Context) pushBlockFrame() *blockFrame {
	parent := ctx.currentFrame()
	return ctx.pushFrame(&blockFrame{
		baseFrame: baseFrame{
			scope: scope.NewScope(parent.scope),
		},
		owner: parent.owner,
	})
}
