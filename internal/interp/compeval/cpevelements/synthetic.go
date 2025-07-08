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
	"github.com/gx-org/gx/build/ir"
)

type (
	// SyntheticBuilder builds a synthetic function.
	SyntheticBuilder interface {
		BuildType() (*ir.FuncType, error)
		BuildBody(ir.Fetcher) (*ir.BlockStmt, []*SyntheticFuncDecl, bool)
	}

	// SyntheticFunc is a GX string.
	SyntheticFunc struct {
		builder SyntheticBuilder
	}

	// SyntheticFuncDecl is a synthetic package function declaration.
	SyntheticFuncDecl struct {
		*SyntheticFunc
		F *ir.FuncDecl
	}
)

var _ ir.Element = (*SyntheticFunc)(nil)

// NewSyntheticFunc returns a state element storing a string GX value.
func NewSyntheticFunc(builder SyntheticBuilder) *SyntheticFunc {
	return &SyntheticFunc{builder: builder}
}

// Builder returns the builder responsible for building the synthetic function.
func (n *SyntheticFunc) Builder() SyntheticBuilder {
	return n.builder
}

// Type of the element.
func (n *SyntheticFunc) Type() ir.Type {
	return ir.UnknownType()
}
