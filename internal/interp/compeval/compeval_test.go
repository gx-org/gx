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

package compeval_test

import (
	"fmt"
	"go/ast"
	"go/token"
	"math/big"
	"testing"

	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irhelper"
	"github.com/gx-org/gx/internal/interp/compeval"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
)

var (
	exprType = ir.Int32Type()
	file     = &ir.File{
		Package: &ir.Package{
			Name:  &ast.Ident{Name: "compeval_test"},
			Decls: &ir.Declarations{},
		},
	}
)

func staticVariable(opts []options.PackageOption, name string) []options.PackageOption {
	decl := &ir.VarSpec{
		FFile: file,
		TypeV: exprType,
	}
	varExpr := &ir.VarExpr{
		Decl:  decl,
		VName: &ast.Ident{Name: name},
	}
	decl.Exprs = []*ir.VarExpr{varExpr}
	file.Package.Decls.Vars = append(file.Package.Decls.Vars, decl)
	opts = append(opts, compeval.NewOptionVariable(varExpr))
	return opts
}

func numberInt(val int) *ir.NumberInt {
	return &ir.NumberInt{
		Src: &ast.BasicLit{Value: fmt.Sprint(val)},
		Val: big.NewInt(int64(val)),
	}
}

func numberInt32(val int) *ir.NumberCastExpr {
	return &ir.NumberCastExpr{
		X:   numberInt(val),
		Typ: exprType,
	}
}

func numberFloat(val float32) *ir.NumberFloat {
	return &ir.NumberFloat{
		Src: &ast.BasicLit{Value: fmt.Sprint(val)},
		Val: big.NewFloat(float64(val)),
	}
}

func atomicFloat32(val float32) ir.AtomicValue {
	return &ir.AtomicValueT[float32]{
		Val: val,
		Typ: ir.Float32Type(),
	}
}

func unaryExpr(op token.Token, x ir.AssignableExpr) *ir.UnaryExpr {
	return &ir.UnaryExpr{
		Src: &ast.UnaryExpr{Op: op},
		X:   x,
	}
}

func castExpr(dt dtype.DataType, x ir.Expr) *ir.CastExpr {
	return &ir.CastExpr{
		Typ: ir.TypeFromKind(ir.Kind(dt)),
		X:   x,
	}
}

func binaryExpr(op token.Token, x, y ir.AssignableExpr) *ir.BinaryExpr {
	return &ir.BinaryExpr{
		Src: &ast.BinaryExpr{Op: op},
		X:   x,
		Y:   y,
		Typ: exprType,
	}
}

func valueRef(name string) *ir.ValueRef {
	store := irhelper.LocalVar(name, exprType)
	return &ir.ValueRef{
		Src:  &ast.Ident{Name: name},
		Stor: store,
	}
}

type wantValue struct {
	eval  bool
	value int32
}

func value(x int32) wantValue {
	return wantValue{eval: true, value: x}
}

func newInterpreter(opts []options.PackageOption) (*interp.FileScope, error) {
	itp, err := interp.New(compeval.NewHostEvaluator(nil, interp.NewRunFunc), opts)
	if err != nil {
		return nil, err
	}
	return itp.ForFile(file)
}

func TestExprEval(t *testing.T) {
	var opts []options.PackageOption
	opts = staticVariable(opts, "a")
	opts = staticVariable(opts, "b")
	fitp, err := newInterpreter(opts)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		desc  string
		expr  ir.Expr
		value wantValue
		want  string
	}{
		{
			desc:  "basic literal",
			expr:  numberInt32(4),
			value: value(4),
			want:  "int32(4)",
		},
		{
			desc: "unary expression",
			expr: unaryExpr(token.SUB,
				numberInt32(4),
			),
			value: value(-4),
			want:  "-int32(4)",
		},
		{
			desc: "cast expression",
			expr: castExpr(dtype.Int32,
				atomicFloat32(4),
			),
			value: value(4),
			want:  "int32(float32(4))",
		},
		{
			desc: "binary expression",
			expr: binaryExpr(token.ADD,
				numberInt32(4),
				numberInt32(5),
			),
			value: value(9),
			want:  "int32(4)+int32(5)",
		},
		{
			desc: "static variable",
			expr: valueRef("a"),
			want: "a",
		},
		{
			desc: "unary static variable",
			expr: unaryExpr(token.SUB,
				valueRef("a"),
			),
			want: "-a",
		},
		{
			desc: "binary static variable",
			expr: binaryExpr(token.SUB,
				valueRef("a"),
				valueRef("a"),
			),
			want: "a-a",
		},
		{
			desc: "binary static variable, int32",
			expr: binaryExpr(token.SUB,
				valueRef("a"),
				numberInt32(4),
			),
			want: "a-int32(4)",
		},
		{
			desc: "binary int32, static variable",
			expr: binaryExpr(token.SUB,
				numberInt32(4),
				valueRef("a"),
			),
			want: "int32(4)-a",
		},
		{
			desc: "binary int32, unary static variable",
			expr: binaryExpr(token.SUB,
				numberInt32(4),
				unaryExpr(token.SUB, valueRef("a")),
			),
			want: "int32(4)-(-a)",
		},
		{
			desc: "binary unary int32, static variable",
			expr: binaryExpr(token.SUB,
				unaryExpr(token.SUB, numberInt32(4)),
				valueRef("a"),
			),
			want: "-int32(4)-a",
		},
		{
			desc: "numberInt",
			expr: numberInt(5),
			want: "5",
		},
		{
			desc: "numberFloat",
			expr: numberFloat(5.2),
			want: "5.2",
		},
		{
			desc: "binary numberInt numberInt",
			expr: binaryExpr(token.SUB,
				unaryExpr(token.SUB, numberInt(4)),
				numberInt(5),
			),
			want: "-4-5",
		},
		{
			desc: "binary numberInt numberFloat",
			expr: binaryExpr(token.SUB,
				unaryExpr(token.SUB, numberInt(4)),
				numberFloat(5),
			),
			want: "-4-5",
		},
		{
			desc: "binary binary no number",
			expr: binaryExpr(token.SUB,
				binaryExpr(token.ADD, valueRef("a"), valueRef("a")),
				binaryExpr(token.ADD, valueRef("a"), valueRef("a")),
			),
			want: "(a+a)-(a+a)",
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("Test%d", i), func(t *testing.T) {
			element, err := compeval.EvalExpr(fitp, test.expr)
			if err != nil {
				t.Fatalf("\n%+v", err)
			}
			got := element.(fmt.Stringer).String()
			if got != test.want {
				t.Errorf("%s: got %q but want %q", test.desc, got, test.want)
			}
			if !test.value.eval {
				return
			}
			value := element.(elements.ElementWithConstant).NumericalConstant()
			gotValue, err := values.ToAtom[int32](value)
			if err != nil {
				t.Errorf("%s: atom conversion error:\n%+v", test.desc, fmterr.ToStackTraceError(err))
			}
			wantValue := test.value.value
			if gotValue != wantValue {
				t.Errorf("%s: got value %d but want %d", test.desc, gotValue, wantValue)
			}
		})
	}
}

func TestExprEvalAndCompare(t *testing.T) {
	var opts []options.PackageOption
	opts = staticVariable(opts, "a")
	opts = staticVariable(opts, "b")
	tests := []struct {
		desc  string
		x     ir.Expr
		is    ir.Expr
		isNot ir.Expr
	}{
		{
			desc:  "basic literal",
			x:     numberInt32(4),
			is:    numberInt32(4),
			isNot: numberInt32(5),
		},
		{
			desc: "unary expression",
			x: unaryExpr(token.SUB,
				numberInt32(4),
			),
			is:    numberInt32(-4),
			isNot: numberInt32(4),
		},
		{
			desc: "cast expression",
			x: castExpr(dtype.Int32,
				atomicFloat32(4),
			),
			is:    numberInt32(4),
			isNot: atomicFloat32(4),
		},
		{
			desc: "binary, atom",
			x: binaryExpr(token.ADD,
				numberInt32(4),
				numberInt32(5),
			),
			is:    numberInt32(9),
			isNot: numberInt32(10),
		},
		{
			desc: "atom, binary",
			x:    numberInt32(9),
			is: binaryExpr(token.ADD,
				numberInt32(4),
				numberInt32(5),
			),
			isNot: numberInt32(10),
		},
		{
			desc: "binary, binary",
			x: binaryExpr(token.ADD,
				numberInt32(4),
				numberInt32(5),
			),
			is: binaryExpr(token.ADD,
				numberInt32(4),
				numberInt32(5),
			),
			isNot: binaryExpr(token.ADD,
				numberInt32(5),
				numberInt32(5),
			),
		},
		{
			desc:  "static variable",
			x:     valueRef("a"),
			is:    valueRef("a"),
			isNot: valueRef("b"),
		},
		{
			desc: "unary static variable",
			x: unaryExpr(token.SUB,
				valueRef("a"),
			),
			is: unaryExpr(token.SUB,
				valueRef("a"),
			),
			isNot: numberInt32(5),
		},
		{
			desc: "binary static variable",
			x: binaryExpr(token.SUB,
				valueRef("a"),
				valueRef("a"),
			),
			is: binaryExpr(token.SUB,
				valueRef("a"),
				valueRef("a"),
			),
			isNot: binaryExpr(token.SUB,
				valueRef("a"),
				valueRef("b"),
			),
		},
		{
			desc: "binary static variable, number",
			x: binaryExpr(token.SUB,
				valueRef("a"),
				numberInt32(4),
			),
			is: binaryExpr(token.SUB,
				valueRef("a"),
				numberInt32(4),
			),
			isNot: binaryExpr(token.SUB,
				valueRef("a"),
				numberInt32(5),
			),
		},
		{
			desc: "number, number",
			x: binaryExpr(token.ADD,
				binaryExpr(token.ADD,
					numberInt32(1),
					numberInt32(1),
				),
				numberInt32(1),
			),
			is: binaryExpr(token.MUL,
				numberInt32(1),
				numberInt32(3),
			),
			isNot: valueRef("a"),
		},
	}
	itp, err := newInterpreter(opts)
	if err != nil {
		t.Fatal(err)
	}
	for i, test := range tests {
		xEl, err := compeval.EvalExpr(itp, test.x)
		if err != nil {
			t.Fatalf("\n%+v", err)
		}
		isEl, err := compeval.EvalExpr(itp, test.is)
		if err != nil {
			t.Fatalf("\n%+v", err)
		}
		if !xEl.Compare(isEl) {
			t.Errorf("test %d:%s: %s == %s is false", i, test.desc, xEl, isEl)
			continue
		}
		isNotEl, err := compeval.EvalExpr(itp, test.isNot)
		if err != nil {
			t.Fatalf("\n%+v", err)
		}
		if !xEl.Compare(isEl) {
			t.Errorf("test %d:%s: %s != %s is false", i, test.desc, xEl, isNotEl)
			continue
		}
	}
}

func TestSubContext(t *testing.T) {
	var opts []options.PackageOption
	opts = staticVariable(opts, "a")
	itp, err := newInterpreter(opts)
	if err != nil {
		t.Fatal(err)
	}
	expr := binaryExpr(token.SUB, valueRef("a"), valueRef("b"))
	_, err = compeval.EvalExpr(itp, expr)
	if err == nil {
		t.Errorf("expected an error but got nil")
	}
	cstVal, err := values.AtomIntegerValue(ir.Int32Type(), int32(3))
	if err != nil {
		t.Fatal(err)
	}
	bValue, err := cpevelements.NewAtom(
		elements.NewExprAt(nil, irhelper.IntNumberAs(3, ir.Int32Type())),
		cstVal,
	)
	if err != nil {
		t.Fatal(err)
	}
	sub := itp.Sub(map[string]ir.Element{
		"b": bValue,
	})
	val, err := compeval.EvalExpr(sub, expr)
	if err != nil {
		t.Error(err)
	}
	want := "a-int32(3)"
	got := val.String()
	if got != want {
		t.Errorf("incorrect evaluation expression: got %s but want %s", got, want)
	}
}
