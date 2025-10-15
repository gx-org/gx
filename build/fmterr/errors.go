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

package fmterr

import (
	"fmt"
	"go/token"
	"strings"
)

type (
	contextError struct {
		f      func(error) error
		errors Errors
	}

	// Errors is a set of errors.
	Errors struct {
		stack []contextError
		errs  []error
	}
)

// NewAppender returns a new appender to collect errors.
func (errs *Errors) NewAppender(fset *token.FileSet) *Appender {
	return &Appender{errors: errs, fset: FileSet{FSet: fset}}
}

// Push a new context in the error stack.
func (errs *Errors) Push(f func(error) error) {
	errs.stack = append(errs.stack, contextError{f: f})
}

// Pop removes the last error context in the stack.
func (errs *Errors) Pop() {
	last := errs.stack[len(errs.stack)-1]
	errs.stack = errs.stack[:len(errs.stack)-1]
	if last.errors.Empty() {
		return
	}
	errs.Append(last.f(&last.errors))
}

// Append an error to the list of errors.
func (errs *Errors) Append(err error) bool {
	if len(errs.stack) == 0 {
		errs.errs = append(errs.errs, err)
	} else {
		errs.stack[len(errs.stack)-1].errors.Append(err)
	}
	return false
}

// Empty returns true if no error has been declared.
func (errs *Errors) Empty() bool {
	if len(errs.errs) > 0 {
		return false
	}
	for _, st := range errs.stack {
		if !st.errors.Empty() {
			return false
		}
	}
	return true
}

// Error returns the current set of errors as a string.
func (errs *Errors) Error() string {
	var ss []string
	if len(errs.errs) > 0 {
		ss = []string{""}
	}
	for _, err := range errs.errs {
		ss = append(ss, err.Error())
	}
	return strings.Join(ss, "\n")
}

// Errors returns the list of all collected errors.
func (errs *Errors) Errors() []error {
	all := append([]error{}, errs.errs...)
	for _, st := range errs.stack {
		if st.errors.Empty() {
			continue
		}
		all = append(all, st.f(&st.errors))
	}
	return all
}

// ToError returns the errors as an error interface.
func (errs *Errors) ToError() error {
	if errs == nil || errs.Empty() {
		return nil
	}
	return errs
}

// Transform the set of errors into a new set.
func (errs *Errors) Transform(f func(error) error) *Errors {
	if errs == nil {
		return nil
	}
	nw := &Errors{}
	for _, err := range errs.errs {
		err = f(err)
		if err == nil {
			continue
		}
		nw.errs = append(nw.errs, err)
	}
	if nw.Empty() {
		return nil
	}
	return nw
}

// Format writes the error into the state of the formatter.
func (errs *Errors) Format(s fmt.State, verb rune) {
	flag := ""
	if s.Flag('+') {
		flag = "+"
	}
	for _, e := range errs.errs {
		format := fmt.Sprintf("%%%s%s\n", flag, string(verb))
		fmt.Fprintf(s, format, e)
	}
}

// String representation of the error.
func (errs *Errors) String() string {
	return errs.Error()
}
