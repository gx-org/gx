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

	"github.com/gx-org/gx/internal/cmd/impgo/gopkgbin"
	"github.com/spf13/cobra"
	"github.com/gx-org/gx/internal/cmd/impgo/generator"
	"github.com/gx-org/gx/internal/cmd/impgo/gengo"
	"github.com/gx-org/gx/internal/cmd/impgo/gengx"
)

const filenameSuffixFlag = "filenamesuffix"
const commandName = "importgo"

// Cmd is the implementation of the impgo command.
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   commandName + " <go_packages>",
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
func (ig *impgo) GenFileName() string {
	return commandName + ig.fileNameSuffix
}

// GoImportPath modifies, if necessary, an import path used in the generated Go file.
func (ig *impgo) GoImportPath(pkgpath string) string {
	return ig.loader.GoImportPath(pkgpath)
}

func (ig *impgo) generate(pkgpath string) error {
	pkg, err := ig.loader.Load(pkgpath)
	if err != nil {
		return err
	}
	target := generator.Target{
		Src:       pkg,
		GoPkgName: pkg.Name + "_" + commandName,
		GxPkgName: pkg.Name,
	}
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
		outName := ig.GenFileName() + "." + gen.FileExtension()
		out, err := ig.loader.BuildPath(pkgpath, outName)
		if err != nil {
			return err
		}
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
