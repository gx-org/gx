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

package fixers

import (
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"strings"

	"github.com/gx-org/gx/api"
)

type (
	// FileWriter writes a file given its content.
	FileWriter interface {
		Write(path string, content string) error
		Close() error
	}

	// Walker walks across a file system to find GX files.
	Walker struct {
		rtm *api.Runtime
		fw  FileWriter
	}

	dummyFileWriter struct{}
)

func (dummyFileWriter) Write(path string, content string) error {
	fmt.Println(path + ":")
	fmt.Println(content)
	fmt.Println()
	return nil
}

func (dummyFileWriter) Close() error {
	return nil
}

// NewWalker returns a new walker.
func NewWalker(fw FileWriter, rtm *api.Runtime, dryRun bool) *Walker {
	if dryRun {
		fw = dummyFileWriter{}
	}
	return &Walker{rtm: rtm, fw: fw}
}

type fileInfo interface {
	IsDir() bool
	Name() string
}

func isGXFile(fi fileInfo) bool {
	return !fi.IsDir() && strings.HasSuffix(fi.Name(), ".gx")
}

// Fix a path.
func (w *Walker) Fix(path string, info fs.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if !isGXFile(info) {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, data, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return err
	}
	fixed := false
	for _, fix := range Fixers {
		entryFixed, err := fix(w.rtm, path, f)
		if err != nil {
			return err
		}
		fixed = fixed || entryFixed
	}
	if !fixed {
		return nil
	}
	src := strings.Builder{}
	if err := format.Node(&src, fset, f); err != nil {
		return fmt.Errorf("cannot write formatted code: %w", err)
	}

	if srcStr := src.String(); string(data) != srcStr {
		w.fw.Write(path, srcStr)
	}
	return nil
}

// Close the walker.
func (w *Walker) Close() error {
	return w.fw.Close()
}
