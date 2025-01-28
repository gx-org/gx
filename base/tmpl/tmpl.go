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

// Package tmpl provides helper functions for Go templates.
package tmpl

import (
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

// IterateFunc runs a function to generate a string over a slice of object.
// The result is all the strings joined by a new line.
func IterateFunc[T any](objs []T, f func(int, T) (string, error)) (string, error) {
	var ss []string
	for i, obj := range objs {
		s, err := f(i, obj)
		if err != nil {
			return "", err
		}
		ss = append(ss, s)
	}
	return strings.Join(ss, "\n"), nil
}

// IterateTmpl runs a template to generate a string over a slice of object.
// The result is all the strings joined by a new line.
func IterateTmpl[T any](objs []T, tmpl *template.Template) (string, error) {
	buf := strings.Builder{}
	for _, obj := range objs {
		if err := tmpl.Execute(&buf, obj); err != nil {
			return "", errors.Errorf("cannot generate code for %#v: %v", obj, err)
		}
	}
	return buf.String(), nil
}
