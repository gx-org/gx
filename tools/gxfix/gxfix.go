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

// Package gxfix fixes GX code.
package gxfix

import (
	"flag"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers/fsimporter"
	"github.com/gx-org/gx/stdlib/impl"
	"github.com/gx-org/gx/stdlib"
	"github.com/gx-org/gx/tools/gxfix/fixers"
)

var (
	folder = flag.String("folder", "", "folder where the GX code needs to be fixed")
	dryRun = flag.Bool("dry_run", true, "output on the standard output if true")
)

type embedFS struct {
}

func (embedFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return nil, fmt.Errorf("ReadDir not implemented")
}

func (embedFS) Open(name string) (fs.File, error) {
	return nil, fmt.Errorf("Open not implemented")
}

// NewBuilder returns a new builder given a standard library implementation.
func NewBuilder(impl *impl.Stdlib) (*builder.Builder, error) {
	return builder.New([]builder.Importer{
		stdlib.Importer(impl),
		fsimporter.New(&embedFS{}),
	}), nil
}

// Fix the GX code base.
func Fix(fw fixers.FileWriter, rtm *api.Runtime) error {
	if *folder == "" {
		return fmt.Errorf("no folder specified: please use --folder to specify a target folder")
	}
	walker := fixers.NewWalker(fw, rtm, *dryRun)
	if err := filepath.Walk(*folder, walker.Fix); err != nil {
		return err
	}
	if err := walker.Close(); err != nil {
		return err
	}
	return nil
}
