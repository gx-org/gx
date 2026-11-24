package eagereval

import (
	"testing"

	"github.com/gomlx/compute"
	"github.com/gomlx/compute/notimplemented"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/numbers"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp"
)

type fakeNode struct{ NumericalNode }

type fakeFunction struct{ notimplemented.Function }

func (*fakeFunction) Constant(any, ...int) (compute.Value, error) { return &fakeNode{}, nil }

type fakeBackend struct{}

func (*fakeBackend) ToElement(compute.Value, ir.Type) (NumericalNode, error) {
	return &fakeNode{}, nil
}

func TestConstantCanonical(t *testing.T) {
	bld := builder.New()
	pkg := bld.NewIncrementalPackage("testpkg")
	if err := pkg.Build(`package testpkg`); err != nil {
		t.Fatal(err)
	}
	reg, ops := newRegistry(&fakeBackend{}, bld, &fakeFunction{})
	itp, err := interp.New(reg, nil)
	if err != nil {
		t.Fatal(err)
	}
	fitp, err := itp.ForFile(pkg.IR().File(""))
	if err != nil {
		t.Fatal(err)
	}
	newConst := func(v int) engine.ConstantElement {
		t.Helper()
		cst, err := numbers.NewConstant(ir.IntType(), v)
		if err != nil {
			t.Fatal(err)
		}
		el, err := ops.ElementFromHostValue(fitp, cst)
		if err != nil {
			t.Fatal(err)
		}
		return el
	}
	newSlice := func(els ...ir.Element) *elements.Slice {
		t.Helper()
		s, err := elements.NewSlice(ir.IntSliceType(), els)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}

	c2, c3, c4 := newConst(2), newConst(3), newConst(4)
	can, ok := c2.(cmp.Canonical)
	if !ok {
		t.Fatalf("%T does not implement cmp.Canonical", c2)
	}
	if got, want := can.SourceString(nil), "int(2)"; got != want {
		t.Errorf("SourceString = %q, want %q", got, want)
	}
	expr, err := ir.ToSingleExpr(fitp, nil, c2)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := expr.SourceString(nil), "2"; got != want {
		t.Errorf("ToSingleExpr = %q, want %q", got, want)
	}

	s23a, s23b, s24 := newSlice(c2, c3), newSlice(c2, c3), newSlice(c2, c4)
	if eq, err := cmp.Equal(fitp, s23a, s23b); err != nil || !eq {
		t.Errorf("cmp.Equal(s23a, s23b) = (%v, %v), want (true, nil)", eq, err)
	}
	if eq, err := cmp.Equal(fitp, s23a, s24); err != nil || eq {
		t.Errorf("cmp.Equal(s23a, s24) = (%v, %v), want (false, nil)", eq, err)
	}
}
