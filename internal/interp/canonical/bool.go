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

package canonical

import (
	"fmt"
	"go/token"
	"math/big"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/internal/base/cast"
)

// WithBool can return a boolean value.
type WithBool interface {
	Bool() (bool, error)
}

type bVal struct {
	val bool
}

var (
	trueBVal  = &bVal{val: true}
	falseBVal = &bVal{val: false}
)

func toBVal(v bool) *bVal {
	if v {
		return trueBVal
	}
	return falseBVal
}

func (be *bVal) Compare(other Comparable) (bool, error) {
	otherB, ok := other.(WithBool)
	if !ok {
		return false, errors.Errorf("cannot convert %T to %s", other, reflect.TypeFor[WithBool]().String())
	}
	otherVal, err := otherB.Bool()
	if err != nil {
		return false, err
	}
	return be.val == otherVal, nil
}

func (be *bVal) ShortString() string {
	return fmt.Sprint(be.val)
}

func (be *bVal) Simplify() Simplifier {
	return be
}

func (be *bVal) Bool() (bool, error) {
	return be.val, nil
}

type bExpr struct {
	args []Canonical
	op   token.Token
	eval func(x, y Canonical) (*bVal, error)
}

func (be *bExpr) evalCanonical() (*bVal, error) {
	if len(be.args) != 2 {
		return nil, errors.Errorf("boolean ops expects 2 arguments, got %d", len(be.args))
	}
	return be.eval(be.args[0], be.args[1])
}

func (be *bExpr) Compare(other Comparable) (bool, error) {
	this, err := be.evalCanonical()
	if err != nil {
		return false, err
	}
	return this.Compare(other)
}

func (be *bExpr) ShortString() string {
	strs := make([]string, len(be.args))
	for i, arg := range be.args {
		strs[i] = arg.ShortString()
	}
	return fmt.Sprintf("(%s %s)", be.op, strings.Join(strs, " "))
}

func (be *bExpr) Simplify() Simplifier {
	return be
}

func (be *bExpr) Bool() (bool, error) {
	this, err := be.evalCanonical()
	if err != nil {
		return false, err
	}
	return this.val, nil
}

func eq(args []Canonical) *bExpr {
	return &bExpr{
		args: args,
		op:   token.EQL,
		eval: func(x, y Canonical) (*bVal, error) {
			same, err := x.Compare(y)
			if err != nil {
				return nil, err
			}
			return toBVal(same), nil
		},
	}
}

func neq(args []Canonical) *bExpr {
	return &bExpr{
		args: args,
		op:   token.NEQ,
		eval: func(x, y Canonical) (*bVal, error) {
			same, err := x.Compare(y)
			if err != nil {
				return nil, err
			}
			return toBVal(!same), nil
		},
	}
}

func compareFloat(f func(x, y *big.Float) bool) func(x, y Canonical) (*bVal, error) {
	return func(x, y Canonical) (*bVal, error) {
		xVal, err := cast.To[Evaluable](x)
		if err != nil {
			return nil, err
		}
		yVal, err := cast.To[Evaluable](y)
		if err != nil {
			return nil, err
		}
		bVal := f(xVal.Float(), yVal.Float())
		return toBVal(bVal), nil
	}
}

func lss(args []Canonical) *bExpr {
	return &bExpr{
		args: args,
		op:   token.LSS,
		eval: compareFloat(func(x, y *big.Float) bool {
			cmp := x.Cmp(y)
			return cmp == -1
		}),
	}
}

func gtr(args []Canonical) *bExpr {
	return &bExpr{
		args: args,
		op:   token.GTR,
		eval: compareFloat(func(x, y *big.Float) bool {
			cmp := x.Cmp(y)
			return cmp == 1
		}),
	}
}

func leq(args []Canonical) *bExpr {
	return &bExpr{
		args: args,
		op:   token.LEQ,
		eval: compareFloat(func(x, y *big.Float) bool {
			cmp := x.Cmp(y)
			return cmp == 0 || cmp == -1
		}),
	}
}

func geq(args []Canonical) *bExpr {
	return &bExpr{
		args: args,
		op:   token.GEQ,
		eval: compareFloat(func(x, y *big.Float) bool {
			cmp := x.Cmp(y)
			return cmp == 0 || cmp == 1
		}),
	}
}
