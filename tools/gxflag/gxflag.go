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

// Package gxflag provides flag types for GX tools.
package gxflag

import (
	"flag"
	"strings"
)

type stringList struct {
	list *[]string
}

func (sl *stringList) String() string {
	return ""
}

func (sl *stringList) Set(values string) error {
	for _, value := range strings.Split(values, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		*sl.list = append(*sl.list, value)
	}
	return nil
}

// StringList returns a flag to pass a list of string from the command line.
func StringList(name, doc string) *[]string {
	var list []string
	sList := stringList{&list}
	flag.Var(&sList, name, doc)
	return sList.list
}
