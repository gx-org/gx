// Package options specifies options for packages.
package options

import (
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/api/values"
)

type (
	// PackageOptionFactory creates options given a backend.
	PackageOptionFactory func(platform.Platform) PackageOption

	// PackageOption is an option specific to a package.
	PackageOption interface {
		Package() string
	}

	// PackageVarSetValue sets the value of a package level static variable.
	PackageVarSetValue struct {
		// Pck is the package owning the variable.
		Pkg string
		// Index of the variable in the package definition.
		Var string
		// Value of the static variable for the compiler.
		Value values.Value
	}
)

// Package for which the option has been built.
func (p PackageVarSetValue) Package() string {
	return p.Pkg
}
