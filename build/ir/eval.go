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

package ir

import (
	"go/ast"

	"github.com/gx-org/gx/build/fmterr"
)

type (
	// Canonical is a canonical value with a IR representation.
	Canonical interface {
		Expr() (AssignableExpr, error)
	}

	// Importer imports packages given their path.
	Importer interface {
		// Import a package given its path.
		Import(pkgPath string) (*Package, error)
	}

	// Element is a value returned by the evaluator.
	Element interface {
		Type() Type
	}

	// Evaluator evaluates IR expressions into canonical values.
	Evaluator interface {
		File() *File
		EvalExpr(Expr) (Element, error)
	}

	// Fetcher represents a scope in the compiler.
	Fetcher interface {
		Evaluator
		fmterr.ErrAppender
		BuildExpr(ast.Expr) (Expr, bool)
		IsDefined(string) bool
	}
)
