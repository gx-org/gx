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

// Errors is a set of errors.
type Errors struct {
	errs []error
}

// NewAppender returns a new appender to collect errors.
func (errs *Errors) NewAppender(fset *token.FileSet) *Appender {
	return &Appender{errors: errs, fset: FileSet{FSet: fset}}
}

// Append an error to the list of errors.
func (errs *Errors) Append(err error) {
	errs.errs = append(errs.errs, err)
}

// Empty returns true if no error has been declared.
func (errs *Errors) Empty() bool {
	if errs == nil {
		return true
	}
	return len(errs.errs) == 0
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
