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

// Package template runs a template and write the result into a file.
package template

import (
	"fmt"
	"os"
	"text/template"
)

// Exec parses a template, runs it, and writes the result into a file.
func Exec(src string, target string, data any) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cannot generate %q: %v", src, err)
		}
	}()
	tpl, err := template.New("").Parse(src)
	if err != nil {
		return fmt.Errorf("cannot parse template source: %v", err)
	}

	file, err := os.Create(target)
	if err != nil {
		return err
	}
	defer file.Close()

	return tpl.Execute(file, data)
}
