// Copyright 2026 Google LLC
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

// Package impgo implements the command to generate the code to import a Go package into Gx.
package impgo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gx-org/gx/internal/cmd/impgo/gopkgbin"
	"github.com/spf13/cobra"
	"github.com/gx-org/gx/internal/cmd/impgo/generator"
	"github.com/gx-org/gx/internal/cmd/impgo/gengo"
	"github.com/gx-org/gx/internal/cmd/impgo/gengx"
)

const filenameSuffixFlag = "filenamesuffix"

// Cmd is the implementation of the impgo command.
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "importgo <go_packages>",
		Short: "Generate the code to use a Go package from GX",
		Args:  cobra.ExactArgs(1),
		RunE:  goImport,
	}
	cmd.PersistentFlags().StringP(filenameSuffixFlag, "o", "", "name of the output files")
	return cmd
}

type impgo struct {
	cmd            *cobra.Command
	fileNameSuffix string
	loader         *gopkgbin.Loader
}

// FileNameSuffix returns the suffix used for the generated file names.
func (ig *impgo) FileNameSuffix() string {
	return ig.fileNameSuffix
}

func (ig *impgo) generate(pkgpath string) (err error) {
	outFolder := filepath.Dir(pkgpath)
	pkg := generator.Pkg{}
	pkg.Dir, pkg.Name, pkg.Pkg, err = ig.loader.Load(pkgpath)
	target := generator.Target{Src: pkg, Name: pkg.Name}
	generators := []generator.New{
		gengo.New,
		gengx.New,
	}
	for _, newGen := range generators {
		gen := newGen(ig, target)
		src, err := gen.Generate()
		if err != nil {
			return err
		}
		outName := "impgo" + ig.fileNameSuffix + "." + gen.FileExtension()
		out := filepath.Join(outFolder, outName)
		if err := os.WriteFile(out, []byte(src), 0644); err != nil {
			return err
		}
	}
	return nil
}

func goImport(cmd *cobra.Command, args []string) (err error) {
	cmd.SetErrPrefix(fmt.Sprintf("gx %s:", cmd.Name()))
	ig := impgo{cmd: cmd}
	ig.loader, err = gopkgbin.New()
	if err != nil {
		return
	}
	ig.fileNameSuffix, err = ig.cmd.Flags().GetString(filenameSuffixFlag)
	if err != nil {
		return
	}
	for _, arg := range args {
		if err = ig.generate(arg); err != nil {
			return
		}
	}
	return
}
