// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package gxrun implements the command to run the Main function of a GX source file.
package gxrun

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/gx-org/gx/api/gxmain"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/golang/backend"
	"github.com/gx-org/gx/stdlib"
)

const commandName = "run"

// Cmd returns the implementation of the run command.
func Cmd() *cobra.Command {
	return &cobra.Command{
		Use:   commandName + " [flags] path",
		Short: "Run the Main function of a GX source file",
		Long: `Run the Main function of a GX source file.

The file is built as a package which can import the GX standard library.
Its Main function, which must have no parameters, is then traced and run
with the native Go backend. The values returned by Main, followed by a
"Trace:" section with the values passed to the trace builtin, are printed
on the standard output.

If Main is a compeval function (i.e. marked with //gx:compeval), it is
evaluated at compile time instead. Its parameters, if any, are symbolic:
the values it returns are then printed as expressions of these parameters.`,
		Args: cobra.ExactArgs(1),
		RunE: gxRun,
	}
}

func gxRun(cmd *cobra.Command, args []string) error {
	cmd.SetErrPrefix(fmt.Sprintf("gx %s:", cmd.Name()))
	bld := builder.New(stdlib.Importer())
	path := args[0]
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	vfs := os.DirFS(filepath.Dir(path))
	srcs := []string{filepath.Base(path)}
	rtm, err := backend.New(bld)
	if err != nil {
		return fmt.Errorf("cannot create the native Go backend: %w", err)
	}
	fn, outs, traces, err := gxmain.FromFile(rtm, vfs, srcs)
	if err != nil {
		return err
	}
	defer hostio.Finalize(outs...)
	return gxmain.Printf(cmd.OutOrStdout(), fn, outs, traces)
}
