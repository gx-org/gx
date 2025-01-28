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
