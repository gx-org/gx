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
	"fmt"
	"go/token"
	"strconv"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
)

func areEqual(fetcher Fetcher, x, y Expr) (bool, error) {
	xVal, xName, err := evalDimExpr(fetcher, x)
	if err != nil {
		return false, err
	}
	yVal, yName, err := evalDimExpr(fetcher, y)
	if err != nil {
		return false, err
	}
	// Two expressions are equal if the same value has been
	// computed or/and they have the same string representation.
	return xVal == yVal && xName == yName, nil
}

func areConvertible(fetcher Fetcher, x, y Expr) (bool, error) {
	xVal, xName, err := evalDimExpr(fetcher, x)
	if err != nil {
		return false, err
	}
	yVal, yName, err := evalDimExpr(fetcher, y)
	if err != nil {
		return false, err
	}
	if xName != "" || yName != "" {
		// One of the expression has a static variable,
		// so it can always be converted.
		return true, nil
	}
	// No static variable is used: check that we have computed
	// the same value.
	return xVal == yVal, nil
}

func evalDimBinaryExpr(fetcher Fetcher, x *BinaryExpr) (int, string, error) {
	xInt, xStr, err := evalDimExpr(fetcher, x.X)
	if err != nil {
		return 0, "", err
	}
	yInt, yStr, err := evalDimExpr(fetcher, x.Y)
	if err != nil {
		return 0, "", err
	}
	var val int
	switch x.Src.Op {
	case token.ADD:
		val = xInt + yInt
	case token.SUB:
		val = xInt - yInt
	case token.MUL:
		val = xInt * yInt
	case token.QUO:
		val = xInt / yInt
	default:
		return -1, "", fmterr.Errorf(fetcher.FileSet(), x.Source(), "cannot evaluate dimension: binary op %s not supported", x.Src.Op)
	}
	valStr := ""
	if xStr != "" || yStr != "" {
		if xStr == "" {
			xStr = fmt.Sprint(xInt)
		}
		if yStr == "" {
			yStr = fmt.Sprint(yInt)
		}
		valStr = xStr + x.Src.Op.String() + yStr
		val = 0
	}
	return val, valStr, nil
}

func evalDimExpr(fetcher Fetcher, x Expr) (int, string, error) {
	if x == nil {
		return -1, "", errors.Errorf("cannot evaluate nil expression")
	}
	switch xT := x.(type) {
	case *AtomicValueT[Int]:
		return int(xT.Val), "", nil
	case *BinaryExpr:
		return evalDimBinaryExpr(fetcher, xT)
	case *Number:
		num, err := strconv.Atoi(xT.Src.Value)
		if err != nil {
			return -1, "", fmterr.Errorf(fetcher.FileSet(), x.Source(), "cannot parse number %s", xT.Src.Value)
		}
		return num, "", nil
	case *AtomicExprT[Int]:
		return evalDimExpr(fetcher, xT.X)
	case *ValueRef:
		return 0, xT.Src.Name, nil
	default:
		return -1, "", fmterr.Errorf(fetcher.FileSet(), x.Source(), "cannot evaluate dimension: %T not supported", xT)
	}
}
