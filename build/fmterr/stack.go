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
	"io"

	"github.com/pkg/errors"
)

type errorWithStackTrace struct {
	err error
}

func formatVerbose(err error, s fmt.State, verb rune) {
	fmt.Fprintf(s, "%s", err.Error())
	var withSt interface {
		StackTrace() errors.StackTrace
	}
	if !errors.As(err, &withSt) {
		return
	}
	fmt.Fprintf(s, "\nError generated at:%+v\n", withSt.StackTrace())

}

func format(err error, s fmt.State, verb rune) {
	switch verb {
	case 'w':
		fallthrough
	case 'v':
		if s.Flag('+') {
			formatVerbose(err, s, verb)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, err.Error())
	case 'q':
		fmt.Fprintf(s, "%q", err.Error())
	}
}

// ToStackTraceError returns an error that displays its stack trace
// in verbose formatting.
func ToStackTraceError(err error) error {
	if err == nil {
		return nil
	}
	return errorWithStackTrace{err: err}
}

func (err errorWithStackTrace) Unwrap() error {
	return err.err
}

func (err errorWithStackTrace) Format(s fmt.State, verb rune) {
	format(err, s, verb)
}

func (err errorWithStackTrace) Error() string {
	return err.err.Error()
}
