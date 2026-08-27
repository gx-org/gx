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

// Package srcstore links element values to a store.
// The compiler uses such a link to point identifier to the storage being referenced.
package srcstore

import (
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/compeval/cpevops"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

// Element returned after being linked.
type Element interface {
	ir.Element
	ir.WithStore
}

type numEl struct {
	cpevops.Element
	store ir.Storage
}

var _ ir.WithBareValue = (*numEl)(nil)

func (n *numEl) Store() ir.Storage {
	return n.store
}

// AlgExpr returns an algebraic expression.
func (n *numEl) AlgExpr(eva ir.Evaluator) (cmp.Expr, error) {
	return cmp.ToAlgExpr(eva, n.Element)
}

func (n *numEl) BareValue() ir.Element {
	return n.Element
}

type number struct {
	engine.ScalarNumber
	store ir.Storage
}

func (n *number) Store() ir.Storage {
	return n.store
}

func (n *number) BareValue() ir.Element {
	return n.ScalarNumber
}

type named struct {
	named engine.NamedType
	store ir.Storage
}

var _ engine.NamedType = (*named)(nil)

func (n *named) Store() ir.Storage {
	return n.store
}

func (n *named) Under() (ir.Element, error) {
	under, err := n.named.Under()
	if err != nil {
		return under, err
	}
	return Link(n.store, under)
}

func (n *named) Copy() engine.Copier {
	return n
}

func (n *named) Select(expr *ir.SelectorExpr) (ir.Element, error) {
	field, err := n.named.Select(expr)
	if err != nil {
		return field, err
	}
	return Link(expr.Stor, field)
}

func (n *named) Type() ir.Type {
	return n.named.Type()
}

type slice struct {
	elements.ISlice
	store ir.Storage
}

func (f *slice) Store() ir.Storage {
	return f.store
}

type function struct {
	engine.Func
	store ir.Storage
}

func (f *function) Store() ir.Storage {
	return f.store
}

type generic struct {
	elements.Generic
	store ir.Storage
}

func (f *generic) Store() ir.Storage {
	return f.store
}

type invalid struct {
	el    ir.Element
	store ir.Storage
}

func (f *invalid) Store() ir.Storage {
	return f.store
}

func (f *invalid) Type() ir.Type {
	return ir.InvalidType()
}

type str struct {
	elements.IString
	store ir.Storage
}

func (f *str) Store() ir.Storage {
	return f.store
}

// Link an element with a storage.
// Let's say we have the following code:
//
//	func [a,b int](x [a][b]float32) {
//		c := a+b
//		d := f(c)
//		...
//	}
//
// Link links the value of surrogates 'a+b' to a local storage 'c'
// such that, when building 'f(c)', the compiler can find that 'c' references
// the 'c' local variable on the line above.
func Link(store ir.Storage, el ir.Element) (ir.Element, error) {
	if el.Type().Kind() == irkind.Invalid {
		return &invalid{store: store, el: el}, nil
	}
	var linkEl Element
	switch elT := el.(type) {
	case cpevops.Element:
		linkEl = &numEl{Element: elT, store: store}
	case engine.ScalarNumber:
		linkEl = &number{ScalarNumber: elT, store: store}
	case elements.ISlice:
		linkEl = &slice{ISlice: elT, store: store}
	case engine.NamedType:
		linkEl = &named{named: elT, store: store}
	case engine.Func:
		linkEl = &function{Func: elT, store: store}
	case elements.Generic:
		linkEl = &generic{Generic: elT, store: store}
	case elements.IString:
		linkEl = &str{IString: elT, store: store}
	default:
		return el, fmterr.Internalf("cannot link %T to a storage", elT)
	}
	return linkEl, nil
}
