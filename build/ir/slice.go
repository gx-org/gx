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

package ir

import (
	"fmt"
	"go/ast"
)

// SliceLitExpr is a slice literal.
type SliceLitExpr struct {
	Src  ast.Expr
	Typ  Type
	Elts []Expr
}

var (
	_ ExprUnpacker   = (*SliceLitExpr)(nil)
	_ VarArgsIndexer = (*SliceLitExpr)(nil)
)

func (s *SliceLitExpr) node() {}

// Node returns the node in the AST tree.
func (s *SliceLitExpr) Node() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *SliceLitExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *SliceLitExpr) Expr() ast.Expr { return s.Src }

// Unpack the slice literal expressions.
func (s *SliceLitExpr) Unpack() []Expr { return s.Elts }

// IndexForVarArgs returns the expression at the ith position.
func (s *SliceLitExpr) IndexForVarArgs(errapp ErrSource, i int) (Expr, bool) {
	if i >= len(s.Elts) {
		return InvalidIdent, false
	}
	return s.Elts[i], true
}

// SourceString returns the GX source code of the node.
func (s *SliceLitExpr) SourceString(from *File) string {
	return fmt.Sprintf("%s%s", s.Typ.ReferString(from), sourceStringLiteral(from, s.Elts))
}

// SliceExpr is a slice expression to select elements in a sliceable elements.
type SliceExpr struct {
	Src  *ast.SliceExpr
	X    Expr
	Low  Expr
	High Expr
}

var _ Expr = (*SliceLitExpr)(nil)

func (s *SliceExpr) node() {}

// Node returns the node in the AST tree.
func (s *SliceExpr) Node() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *SliceExpr) Type() Type { return s.X.Type() }

// Expr returns the AST expression.
func (s *SliceExpr) Expr() ast.Expr { return s.Src }

// SourceString returns the GX source code of the node.
func (s *SliceExpr) SourceString(from *File) string {
	low := ""
	if s.Low != nil {
		low = s.Low.SourceString(from)
	}
	high := ""
	if s.High != nil {
		high = s.High.SourceString(from)
	}
	return fmt.Sprintf("%s[%s:%s]", s.X.SourceString(from), low, high)
}

// Store returns a storage for the expression.
func (s *SliceExpr) Store() Storage {
	return &AnonymousStorage{
		Typ: s.Type(),
	}
}
