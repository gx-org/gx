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
	"go/ast"
	"go/token"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/state"
)

type frame struct {
	ctx      *context
	function *ir.FuncDecl

	parent *frame
	vars   map[string]state.Element
}

func (ctx *context) newPackageFrame(pkg *ir.Package) (*frame, error) {
	fr := &frame{
		ctx:    ctx,
		parent: ctx.builtin,
		vars:   make(map[string]state.Element),
	}
	for _, f := range pkg.Funcs {
		fr.define(f.Name(), ctx.state.Func(f, nil))
	}
	for _, cst := range pkg.Consts {
		for _, expr := range cst.Exprs {
			if expr.Decl.Type.Kind() == ir.NumberKind {
				// TODO(degris): this is a hack.
				// Constant should be registered and then casted in place.
				continue
			}
			opVal, err := evalValue(ctx, expr.Value)
			if err != nil {
				return nil, err
			}
			fr.define(expr.VName.Name, opVal)
		}
	}
	options := ctx.itrp.packageOptions[pkg.FullName()]
	for _, option := range options {
		if err := option(ctx, pkg, fr); err != nil {
			return nil, err
		}
	}
	return fr, nil
}

func (fr *frame) newFileFrame(file *ir.File) *frame {
	fileFrame := &frame{
		ctx:    fr.ctx,
		parent: fr,
		vars:   make(map[string]state.Element),
	}
	fset := fmterr.FileSet{FSet: file.Package.FSet}
	for _, imp := range file.Imports {
		fileFrame.define(imp.Name().Name, fr.ctx.state.Package(fset.Pos(imp.Source()), imp.Package))
	}
	return fileFrame
}

func (fr *frame) newFunctionFrame(fn *ir.FuncDecl) *frame {
	return &frame{
		function: fn,
		ctx:      fr.ctx,
		parent:   fr,
		vars:     make(map[string]state.Element),
	}
}

func (fr *frame) newBlockFrame() *frame {
	return &frame{
		function: fr.function,
		ctx:      fr.ctx,
		parent:   fr,
		vars:     make(map[string]state.Element),
	}
}

func (fr *frame) set(tok token.Token, id *ast.Ident, value state.Element) error {
	if !ir.ValidIdent(id) {
		return nil
	}
	if tok == token.ILLEGAL {
		return nil
	}
	if tok == token.DEFINE {
		fr.define(id.Name, value)
		return nil
	}
	return fr.assign(id, value)
}

func (fr *frame) define(name string, value state.Element) {
	fr.vars[name] = value
}

func (fr *frame) assign(id *ast.Ident, value state.Element) error {
	_, has := fr.vars[id.Name]
	if has {
		fr.define(id.Name, value)
		return nil
	}
	if fr.parent == nil {
		return errors.Errorf("cannot assign %s: not defined in any frame", id)
	}
	return fr.parent.assign(id, value)
}

func (fr *frame) find(id *ast.Ident) (state.Element, error) {
	if asg := fr.vars[id.Name]; asg != nil {
		return asg, nil
	}
	if fr.parent == nil {
		return nil, fr.ctx.FileSet().Errorf(id, "undefined: %s", id.Name)
	}
	return fr.parent.find(id)
}
