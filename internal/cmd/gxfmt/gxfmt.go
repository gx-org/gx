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

// Package gxfmt implements the command to format GX source files.
package gxfmt

import (
	"bytes"
	"errors"
	"fmt"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/gx-org/gx/src/gx/format"
	"github.com/gx-org/gx/src/gx/parser"
)

const (
	commandName = "fmt"
	writeFlag   = "write"

	parserMode = parser.ParseComments | parser.SkipObjectResolution
)

// Cmd returns the implementation of the fmt command.
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   commandName + " [flags] [path ...]",
		Short: "Format GX source files",
		Long: `Format GX source files.

Without an explicit path, fmt processes the standard input. Given a file,
it operates on that file; given a directory, it operates on all .gx files
in that directory, recursively. (Files starting with a period are ignored.)
By default, fmt prints the reformatted sources to standard output.`,
		Args: cobra.ArbitraryArgs,
		RunE: gxFmt,
	}
	cmd.Flags().BoolP(writeFlag, "w", false, "write result to (source) file instead of stdout")
	return cmd
}

type formatter struct {
	cmd   *cobra.Command
	write bool
}

func gxFmt(cmd *cobra.Command, args []string) error {
	cmd.SetErrPrefix(fmt.Sprintf("gx %s:", cmd.Name()))
	write, err := cmd.Flags().GetBool(writeFlag)
	if err != nil {
		return err
	}
	f := &formatter{cmd: cmd, write: write}
	if len(args) == 0 {
		return f.processStdin()
	}
	var errs []error
	for _, arg := range args {
		if err := f.processPath(arg); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// processStdin formats the standard input and prints the result on the standard output.
func (f *formatter) processStdin() error {
	if f.write {
		return errors.New("cannot use -w with standard input")
	}
	src, err := io.ReadAll(f.cmd.InOrStdin())
	if err != nil {
		return err
	}
	// Accept a program fragment in lieu of a complete source file.
	res, err := format.Source(src)
	if err != nil {
		return err
	}
	_, err = f.cmd.OutOrStdout().Write(res)
	return err
}

func isGXFilename(name string) bool {
	return !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".gx")
}

// processPath formats root if root is a file.
// If root is a directory, it formats all the GX files in root, recursively.
func (f *formatter) processPath(root string) error {
	var errs []error
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir():
			return nil // Recurse into directories.
		case path != root && !isGXFilename(d.Name()):
			return nil // Skip walked non-GX files. Files given as arguments are always formatted.
		}
		if err := f.processFile(path); err != nil {
			errs = append(errs, err)
		}
		return nil
	})
	return errors.Join(append(errs, walkErr)...)
}

// processFile formats a GX source file. The result is either printed on the
// standard output or written back to the file.
func (f *formatter) processFile(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parserMode)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	res := buf.Bytes()
	if !f.write {
		_, err := f.cmd.OutOrStdout().Write(res)
		return err
	}
	if bytes.Equal(src, res) {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	return os.WriteFile(path, res, info.Mode().Perm())
}
