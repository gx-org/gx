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

// Package storepath keeps track of storage path to distingued surrogate values from one another.
package storepath

import (
	"fmt"
	"go/ast"

	"github.com/gx-org/gx/build/ir"
)

// Path to a store.
type Path interface {
	ir.WithStore
	// Same returns true if the paths are the same.
	Same(Path) bool
	// Expr returns the IR expression.
	Expr() ir.Expr
	// SourceString returns the path as a string.
	SourceString(from *ir.File) string
}

type unique struct {
	x ir.Expr
}

// NewUnique returns a new unique source path that is different from everything else except itself.
func NewUnique(el ir.Element) (Path, error) {
	x, err := ir.ToSingleExpr(nil, nil, el)
	return &unique{x: x}, err
}

// NewUniqueIR returns a new unique source path given an IR expressions.
func NewUniqueIR(x ir.Expr) Path {
	return &unique{x: x}
}

func (p *unique) Same(other Path) bool {
	otherT, isRoot := other.(*unique)
	if !isRoot {
		return false
	}
	return p == otherT
}

func (p *unique) Expr() ir.Expr {
	return p.x
}

func (p *unique) Store() ir.Storage {
	return nil
}

func (p *unique) SourceString(from *ir.File) string {
	return p.x.SourceString(from)
}

// fieldRoot of the path.
type fieldRoot struct {
	field *ir.Field
}

// NewRoot returns a new root path.
func NewRoot(r *ir.Field) Path {
	return &fieldRoot{r}
}

func (p *fieldRoot) Same(other Path) bool {
	otherT, isRoot := other.(*fieldRoot)
	if !isRoot {
		return false
	}
	return p.field.Origin() == otherT.field.Origin()
}

func (p *fieldRoot) Expr() ir.Expr {
	return ir.NewIdent(p.field.Storage())
}

func (p *fieldRoot) Store() ir.Storage {
	return p.field.Storage()
}

func (p *fieldRoot) SourceString(from *ir.File) string {
	return p.field.Name.Name
}

type selectField struct {
	parent Path
	field  *ir.Field
}

// NewSelect returns a path from selecting a field given a parent.
func NewSelect(parent Path, field *ir.Field) Path {
	return &selectField{parent: parent, field: field}
}

func (p *selectField) Same(other Path) bool {
	otherT, isSelectField := other.(*selectField)
	if !isSelectField {
		return false
	}
	if p.field.Origin() != otherT.field.Origin() {
		return false
	}
	return p.parent.Same(otherT.parent)
}

func (p *selectField) Expr() ir.Expr {
	parentExpr := p.parent.Expr()
	return &ir.SelectorExpr{
		Src: &ast.SelectorExpr{
			X:   parentExpr.Expr(),
			Sel: p.field.Name,
		},
		X:    parentExpr,
		Stor: p.field.Storage(),
	}
}

func (p *selectField) Store() ir.Storage {
	return p.field.Storage()
}

func (p *selectField) SourceString(from *ir.File) string {
	return fmt.Sprintf("%s.%s", p.parent.SourceString(from), p.field.Name.Name)
}

// varRoot of the path.
type varRoot struct {
	vr *ir.VarExpr
}

// NewVar returns a new root path.
func NewVar(vr *ir.VarExpr) Path {
	return &varRoot{vr: vr}
}

func (p *varRoot) Same(other Path) bool {
	otherT, isRoot := other.(*varRoot)
	if !isRoot {
		return false
	}
	return p.vr == otherT.vr
}

func (p *varRoot) Expr() ir.Expr {
	return ir.NewIdent(p.vr)
}

func (p *varRoot) Store() ir.Storage {
	return p.vr
}

func (p *varRoot) SourceString(from *ir.File) string {
	return p.vr.VName.Name
}

type localVar struct {
	s *ir.LocalVarStorage
}

// NewLocal is a path starting from a local variable.
func NewLocal(s *ir.LocalVarStorage) Path {
	return &localVar{s: s}
}

func (p *localVar) Same(other Path) bool {
	otherT, isRoot := other.(*localVar)
	if !isRoot {
		return false
	}
	return p.s == otherT.s
}

func (p *localVar) Expr() ir.Expr {
	return ir.NewIdent(p.s)
}

func (p *localVar) Store() ir.Storage {
	return p.s
}

func (p *localVar) SourceString(from *ir.File) string {
	return p.s.NameDef().Name
}
