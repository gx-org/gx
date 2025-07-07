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

func (ctx *Context) importPackage(imp *ir.ImportDecl) (*elements.Package, error) {
	pkg, err := ctx.evaluator.Importer().Import(imp.Path)
	if err != nil {
		return nil, err
	}
	pFrame, err := ctx.packageFrame(pkg)
	if err != nil {
		return nil, err
	}
	return pFrame.el, nil
}

func (ctx *Context) packageFrame(pkg *ir.Package) (*packageFrame, error) {
	pkgFrame := ctx.packageToFrame[pkg]
	if pkgFrame != nil {
		return pkgFrame, nil
	}
	pkgFrame = &packageFrame{
		baseFrame: baseFrame{
			scope: scope.NewScope(ctx.builtin.scope),
		},
		pkg:         pkg,
		el:          elements.NewPackage(pkg, ctx.evaluator.NewFunc),
		fileToFrame: make(map[*ir.File]*fileFrame),
	}
	ctx.packageToFrame[pkg] = pkgFrame
	for _, f := range pkgFrame.pkg.Decls.Funcs {
		pkgFrame.Define(f.Name(), ctx.evaluator.NewFunc(f, nil))
	}
	if err := pkgFrame.evalPackageConsts(ctx); err != nil {
		return nil, err
	}
	options := ctx.packageOptions[pkg.FullName()]
	if err := pkgFrame.evalPackageOptions(ctx, options); err != nil {
		return nil, err
	}
	return pkgFrame, nil
}

func (fr *packageFrame) evalPackageConstExpr(ctx *Context, expr *ir.ConstExpr) error {
	fCtx, err := ctx.newFileContext(expr.Decl.FFile)
	if err != nil {
		return err
	}
	el, err := fCtx.evalExpr(expr.Val)
	if err != nil {
		return err
	}
	fr.Define(expr.VName.Name, el)
	fr.el.Define(expr.VName.Name, el)
	return nil
}

func (fr *packageFrame) evalPackageConsts(ctx *Context) error {
	exprs, err := fr.pkg.Decls.ConstExprs()
	if err != nil {
		return err
	}
	for _, expr := range exprs {
		if err := fr.evalPackageConstExpr(ctx, expr); err != nil {
			return err
		}
	}
	return nil
}

func (fr *packageFrame) evalPackageOptions(ctx *Context, options []packageOption) error {
	for _, option := range options {
		if err := option(ctx, fr); err != nil {
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

func (fr *packageFrame) fileFrame(ctx *Context, file *ir.File) (*fileFrame, error) {
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
		pkg, err := ctx.importPackage(imp)
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
	pkgFrame, err := ctx.packageFrame(file.Package)
	if err != nil {
		return nil, err
	}
	return pkgFrame.fileFrame(ctx, file)
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
