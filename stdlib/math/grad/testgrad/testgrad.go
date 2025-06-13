// Package testgrad provides function to test autograd.
package testgrad

import (
	"fmt"

	"github.com/gx-org/gx/build/builder/testbuild"
)

// Func tests the computation of the gradient of a function.
type Func struct {
	// GradOf declares the function (which must be named `F`) to compute the gradient of.
	GradOf string

	// Want stores the source code of the expected gradient of the function
	Want string

	// Err is the substring expected if the compiler returns an error.
	Err string
}

// Source code of the declarations.
func (tt Func) Source() string {
	return tt.GradOf
}

// Run builds the declarations as a package, then compare to an expected outcome.
func (tt Func) Run(b *testbuild.Builder) error {
	pkg, err := b.Build(fmt.Sprintf(`
package test

import "grad"

//gx:=grad.Func(F, "x")
func gradF()

%s
`, tt.GradOf))
	if err != nil {
		return err
	}
	gotF := pkg.IR().FindFunc("gradF")
	if err := testbuild.CompareString(gotF.String(), tt.Want); err != nil {
		return err
	}
	return nil
}
