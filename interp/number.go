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

package interp

import (
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/numbers"
	"github.com/gx-org/gx/interp/engine"
)

func (fitp *Interpreter) evalNumberExpr(expr ir.Expr) (engine.ScalarNumber, error) {
	switch exprT := expr.(type) {
	case *ir.NumberFloat:
		return numbers.NewFloatNumber(exprT), nil
	case *ir.NumberInt:
		return numbers.NewIntNumber(exprT), nil
	case *ir.UnaryExpr:
		return evalNumberUnaryExpr(fitp, exprT)
	case *ir.BinaryExpr:
		return evalNumberBinaryExpr(fitp, exprT)
	case *ir.Ident:
		return evalNumberIdent(fitp, exprT)
	case *ir.ParenExpr:
		return fitp.evalNumberExpr(exprT.X)
	case *ir.SelectorExpr:
		return evalNumberSelectorExpr(fitp, exprT)
	default:
		return nil, fmterr.Errorf(fitp.File().FileSet(), expr.Node(), "cannot evaluate GX constant expression %s: %T not supported", expr.SourceString(fitp.File()), expr)
	}
}

func evalNumberIdent(fitp *Interpreter, ref *ir.Ident) (engine.ScalarNumber, error) {
	el, err := evalIdent(fitp, ref)
	if err != nil {
		return nil, err
	}
	n, err := cast.To[engine.ScalarNumber](ir.BareValue(el))
	if err != nil {
		return nil, err
	}
	return n, nil
}

func evalNumberUnaryExpr(fitp *Interpreter, expr *ir.UnaryExpr) (engine.ScalarNumber, error) {
	x, err := fitp.evalNumberExpr(expr.X)
	if err != nil {
		return nil, err
	}
	return x.UnaryOp(fitp.env, expr)
}

func evalNumberBinaryExpr(fitp *Interpreter, expr *ir.BinaryExpr) (engine.ScalarNumber, error) {
	x, err := fitp.evalNumberExpr(expr.X)
	if err != nil {
		return nil, err
	}
	y, err := fitp.evalNumberExpr(expr.Y)
	if err != nil {
		return nil, err
	}
	return x.BinaryOp(fitp.env, expr, y)
}

func evalNumberSelectorExpr(fitp *Interpreter, expr *ir.SelectorExpr) (engine.ScalarNumber, error) {
	el, err := evalSelectorExpr(fitp, expr)
	if err != nil {
		return nil, err
	}
	return cast.To[engine.ScalarNumber](el)
}
