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

package testrtm

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
)

const (
	setStatic = "setStatic"
)

func findVarDecl(pkg *ir.Package, name string) (*ir.VarExpr, error) {
	for _, decl := range pkg.Decls.Vars {
		for _, vr := range decl.Exprs {
			if vr.VName.Name == name {
				return vr, nil
			}
		}
	}
	return nil, errors.Errorf("cannot find variable %s in package %s", name, pkg.Path())
}

func buildSetStaticOption(rtm *api.Runtime, pkg *ir.Package, cmdS []string) (options.PackageOption, error) {
	cmdS = cmdS[1:]
	const numArgs = 3
	if len(cmdS) != numArgs {
		return nil, errors.Errorf("%s: invalid parameters: got %d but want %d", setStatic, len(cmdS), numArgs)
	}
	valName, valType, valS := cmdS[0], cmdS[1], cmdS[2]
	varDecl, err := findVarDecl(pkg, valName)
	if err != nil {
		return nil, err
	}
	typeWant := varDecl.Type().Kind().String()
	if valType != typeWant {
		return nil, errors.Errorf("cannot use a value of type %s to set variable %s (type %s)", valType, valName, typeWant)
	}
	var val types.Bridger
	switch valType {
	case "int32":
		valInt, err := strconv.Atoi(valS)
		if err != nil {
			return nil, err
		}
		val = types.Int32(int32(valInt))
	case "int64":
		valInt, err := strconv.Atoi(valS)
		if err != nil {
			return nil, err
		}
		val = types.Int64(int64(valInt))
	case "intlen":
		valInt, err := strconv.Atoi(valS)
		if err != nil {
			return nil, err
		}
		val = types.DefaultInt(ir.Int(valInt))
	default:
		return nil, errors.Errorf("type %q not supported", valType)
	}
	return options.PackageVarSetValue{
		Pkg:   pkg.Path(),
		Var:   valName,
		Value: val.Bridge().GXValue(),
	}, nil
}

func buildOption(rtm *api.Runtime, pkg *ir.Package, cmd string) (options.PackageOption, error) {
	cmdS := strings.Split(cmd, " ")
	if len(cmdS) == 0 {
		return nil, nil
	}
	switch cmdS[0] {
	case setStatic:
		return buildSetStaticOption(rtm, pkg, cmdS)
	default:
		return nil, errors.Errorf("compiler option command %q not supported", cmdS[0])
	}
}

// BuildCompileOptions from the source code of the package.
func BuildCompileOptions(rtm *api.Runtime, pkg *ir.Package) ([]options.PackageOption, error) {
	var options []options.PackageOption
	for _, file := range pkg.Files {
		for _, grp := range file.Src.Comments {
			if !strings.HasPrefix(grp.Text(), "Test options:") {
				continue
			}
			for _, cmt := range grp.List[1:] {
				text := strings.TrimSpace(cmt.Text)
				text = strings.TrimPrefix(text, "//")
				text = strings.TrimSpace(text)
				opt, err := buildOption(rtm, pkg, text)
				if err != nil {
					return nil, fmterr.Errorf(file.Package.FSet, cmt, "cannot build option %q: %v", text, err)
				}
				if opt == nil {
					continue
				}
				options = append(options, opt)
			}
		}
	}
	return options, nil
}
