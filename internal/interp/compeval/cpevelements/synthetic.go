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

package cpevelements

import (
	"go/ast"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

type (
	// FuncASTBuilder builds GX functions programmatically.
	FuncASTBuilder interface {
		BuildType() (*ast.FuncDecl, error)
		BuildBody(ir.Fetcher) (*ast.BlockStmt, []*SyntheticFuncDecl, bool)
	}

	// FuncIRBuilder builds a synthetic function.
	FuncIRBuilder interface {
		BuildIR(fmterr.ErrAppender, *ast.FuncDecl, *ir.File, *ir.FuncType) (ir.PkgFunc, bool)
	}

	// SyntheticFunc is a function that is being built by a macro.
	SyntheticFunc struct {
		ir  FuncIRBuilder
		ast FuncASTBuilder
	}

	// SyntheticFuncDecl is a synthetic package function declaration.
	SyntheticFuncDecl struct {
		*SyntheticFunc
		F *ast.FuncDecl
	}
)

var _ ir.Element = (*SyntheticFunc)(nil)

// NewSyntheticFunc returns a state element storing a string GX value.
func NewSyntheticFunc(irBuilder FuncIRBuilder) *SyntheticFunc {
	astBuilder, _ := irBuilder.(FuncASTBuilder)
	return &SyntheticFunc{ir: irBuilder, ast: astBuilder}
}

// BuildType builds the function type of a synthetic function.
func (n *SyntheticFunc) BuildType(src *ast.FuncDecl) (*ast.FuncDecl, error) {
	if n.ast == nil {
		return src, nil
	}
	return n.ast.BuildType()
}

// BuildBody builds the source code body of a synthetic function.
func (n *SyntheticFunc) BuildBody(fetcher ir.Fetcher, src *ast.FuncDecl) (*ast.BlockStmt, []*SyntheticFuncDecl, bool) {
	if n.ast == nil {
		return src.Body, nil, true
	}
	return n.ast.BuildBody(fetcher)
}

// BuildIR builds the IR of a function.
func (n *SyntheticFunc) BuildIR(err fmterr.ErrAppender, src *ast.FuncDecl, file *ir.File, fType *ir.FuncType) (ir.PkgFunc, bool) {
	return n.ir.BuildIR(err, src, file, fType)
}

// Type of the element.
func (n *SyntheticFunc) Type() ir.Type {
	return ir.UnknownType()
}
