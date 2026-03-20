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
	"github.com/pkg/errors"
	"github.com/gx-org/gx/internal/interp/canonical"
)

// TupleElement is a tuple evaluated by the interpreter.
type TupleElement interface {
	Element
	TupleElements() []Element
}

func evalExpr(fetcher Fetcher, x Expr) (Element, CompEvalError, error) {
	el, err := fetcher.EvalExpr(x)
	if err != nil {
		return el, nil, err
	}
	tuple, isTuple := el.(TupleElement)
	if !isTuple {
		return el, nil, nil
	}
	els := tuple.TupleElements()
	if len(els) != 2 {
		return el, nil, errors.Errorf("invalid tuple length: got %d but want 2", len(els))
	}
	cpErr, err := fetcher.ToCompEvalError(els[1])
	return els[0], cpErr, err
}

func areEqual(fetcher Fetcher, x, y Expr) (bool, CompEvalError, error) {
	xExpr, xCPErr, err := evalExpr(fetcher, x)
	if xCPErr != nil || err != nil {
		return false, xCPErr, err
	}
	yExpr, yCPErr, err := evalExpr(fetcher, y)
	if yCPErr != nil || err != nil {
		return false, yCPErr, err
	}
	ok, err := ElementEqual(xExpr, yExpr)
	return ok, nil, err
}

// ElementEqual compares if two runtime elements are equal.
func ElementEqual(x, y Element) (bool, error) {
	xCan, ok := x.(canonical.Canonical)
	if !ok {
		return false, nil
	}
	yCan, ok := y.(canonical.Canonical)
	if !ok {
		return false, nil
	}
	return xCan.Compare(yCan)
}
