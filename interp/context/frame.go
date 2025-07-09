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

package context

import (
	"go/ast"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/interp/elements"
)

// Frame in the context.
type Frame struct {
	file    *ir.File
	current *blockFrame
}

// Define a new variable in the frame.
func (fr *Frame) Define(name string, value ir.Element) {
	fr.current.Define(name, value)
}

// Assign a value to an existing name in the frame owning the value.
func (fr *Frame) Assign(name string, value ir.Element) error {
	return fr.current.Assign(name, value)
}

// Find the element in the stack of frame given its identifier.
func (fr *Frame) Find(id *ast.Ident) (ir.Element, error) {
	value, exists := fr.current.Find(id.Name)
	if !exists {
		return nil, fmterr.Errorf(fr.file.FileSet(), id, "undefined: %s", id.Name)
	}
	return value, nil
}

type baseFrame struct {
	scope *scope.RWScope[ir.Element]
}

func (fr *baseFrame) Define(name string, value ir.Element) {
	fr.scope.Define(name, value)
}

func (fr *baseFrame) Assign(name string, value ir.Element) error {
	return fr.scope.Assign(name, value)
}

func (fr *baseFrame) Find(key string) (value ir.Element, ok bool) {
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

func (core *Core) importPackage(imp *ir.ImportDecl) (*elements.Package, error) {
	pkg, err := core.evaluator.Importer().Import(imp.Path)
	if err != nil {
		return nil, err
	}
	pFrame, err := core.packageFrame(pkg)
	if err != nil {
		return nil, err
	}
	return pFrame.el, nil
}

func (core *Core) packageFrame(pkg *ir.Package) (*packageFrame, error) {
	pkgFrame := core.packageToFrame[pkg]
	if pkgFrame != nil {
		return pkgFrame, nil
	}
	pkgFrame = &packageFrame{
		baseFrame: baseFrame{
			scope: scope.NewScope(core.builtin.scope),
		},
		pkg:         pkg,
		el:          elements.NewPackage(pkg, core.NewFunc),
		fileToFrame: make(map[*ir.File]*fileFrame),
	}
	core.packageToFrame[pkg] = pkgFrame
	for _, f := range pkgFrame.pkg.Decls.Funcs {
		pkgFrame.Define(f.Name(), core.evaluator.NewFunc(core, f, nil))
	}
	if err := pkgFrame.evalPackageConsts(core); err != nil {
		return nil, err
	}
	options := core.packageOptions[pkg.FullName()]
	if err := pkgFrame.evalPackageOptions(core, options); err != nil {
		return nil, err
	}
	return pkgFrame, nil
}

func (fr *packageFrame) evalPackageConstExpr(core *Core, expr *ir.ConstExpr) error {
	fCtx, err := core.NewFileContext(expr.Decl.FFile)
	if err != nil {
		return err
	}
	el, err := fCtx.EvalExpr(expr.Val)
	if err != nil {
		return err
	}
	fr.Define(expr.VName.Name, el)
	fr.el.Define(expr.VName.Name, el)
	return nil
}

func (fr *packageFrame) evalPackageConsts(core *Core) error {
	exprs, err := fr.pkg.Decls.ConstExprs()
	if err != nil {
		return err
	}
	for _, expr := range exprs {
		if err := fr.evalPackageConstExpr(core, expr); err != nil {
			return err
		}
	}
	return nil
}

func (fr *packageFrame) evalPackageOptions(core *Core, options []packageOption) error {
	for _, option := range options {
		if err := option(core, fr); err != nil {
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

func (fr *packageFrame) fileFrame(core *Core, file *ir.File) (*fileFrame, error) {
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
		pkg, err := core.importPackage(imp)
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

func (core *Core) fileFrame(file *ir.File) (*fileFrame, error) {
	pkgFrame, err := core.packageFrame(file.Package)
	if err != nil {
		return nil, err
	}
	return pkgFrame.fileFrame(core, file)
}

func (ctx *Context) pushFuncFrame(fn ir.Func) (*blockFrame, error) {
	flFrame, err := ctx.core.fileFrame(fn.File())
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

// PushBlockFrame pushes an empty new frame on the stack.
func (ctx *Context) PushBlockFrame() *blockFrame {
	parent := ctx.currentFrame()
	return ctx.pushFrame(&blockFrame{
		baseFrame: baseFrame{
			scope: scope.NewScope(parent.scope),
		},
		owner: parent.owner,
	})
}
