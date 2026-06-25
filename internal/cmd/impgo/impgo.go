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
	"path"
	"path/filepath"
	"strings"

	"google3/gdm/gxlang/binder/impgo/importer"
	"google3/third_party/golang/cobra/cobra"
	"github.com/gx-org/gx/internal/cmd/impgo/generator"
	"github.com/gx-org/gx/internal/cmd/impgo/gengo"
	"github.com/gx-org/gx/internal/cmd/impgo/gengx"
)

const sourceFilenameFlag = "source_filename"

// Cmd is the implementation of the impgo command.
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "importgo <go_package>",
		Short: "Generate the code to use a Go package from GX",
		Args:  cobra.ExactArgs(1),
		RunE:  goImport,
	}
	cmd.PersistentFlags().StringP(sourceFilenameFlag, "o", "", "name of the output files")
	return cmd
}

// splitPath splits a path to a binary library file into a Go package path and a Go package name.
// For instance "blaze-out/host/bin/learning/deepmind/golang/tools/packpy/tests/init/value.a"
// is split into "learning/deepmind/golang/tools/packpy/tests/init" and "value".
func splitPath(binPath string) (blazeOut, pkgDir, pkgName string) {
	pkgSplit := strings.Split(binPath, "/")
	blazeOut = strings.Join(pkgSplit[:3], "/")
	pkgDir = strings.Join(pkgSplit[3:len(pkgSplit)-1], "/")
	pkgName = pkgSplit[len(pkgSplit)-1]
	pkgName = pkgName[:len(pkgName)-len(path.Ext(pkgName))]
	return
}

// newPackage reads a library binary file and returns it as a Go package.
func newPackage(name string) (generator.Pkg, error) {
	pkg := generator.Pkg{}
	var blazeOut string
	blazeOut, pkg.Dir, pkg.Name = splitPath(name)
	importPath := "google3/" + pkg.Dir + "/" + pkg.Name
	imp, err := importer.NewImporter(blazeOut, "", "", "")
	if err != nil {
		return pkg, err
	}
	pkg.Pkg, err = imp.Import(importPath)
	if err != nil {
		return pkg, fmt.Errorf("cannot import go library %s: %v", importPath, err)
	}
	return pkg, nil
}

func goImport(cmd *cobra.Command, args []string) (err error) {
	cmd.SetErrPrefix(fmt.Sprintf("gx %s:", cmd.Name()))
	outFolder := filepath.Dir(args[0])
	target := generator.Target{}
	target.Src, err = newPackage(args[0])
	if err != nil {
		return err
	}
	target.Name, err = cmd.Flags().GetString(sourceFilenameFlag)
	if err != nil {
		return err
	}
	if target.Name == "" {
		target.Name = target.Src.Name
	}
	generators := []generator.New{
		gengo.New,
		gengx.New,
	}
	for _, newGen := range generators {
		gen := newGen(target)
		src, err := gen.Generate()
		if err != nil {
			return err
		}
		outName := target.Name + "." + gen.FileExtension()
		out := filepath.Join(outFolder, outName)
		if err := os.WriteFile(out, []byte(src), 0644); err != nil {
			return err
		}
	}
	return nil
}
