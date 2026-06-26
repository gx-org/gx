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

package returnatom_importgo

import (
	"embed"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/importers/embedpkg"
	"github.com/gx-org/gx/build/ir"
	pkg0 "github.com/gx-org/gx/internal/cmd/impgo/tests/returnatom"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/stdlib/builtin"
)

//go:embed importgo.gx
var sources embed.FS

func init() {
	embedpkg.RegisterPackage("github.com/gx-org/gx/internal/cmd/impgo/tests/returnatom", Build)
}

func evalFloat32(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	res0 := pkg0.Float32()
	res0GX, err := values.AtomFloatValue[float32](ir.Float32Type(), res0)
	if err != nil {
		return nil, err
	}
	return []ir.Element{
		res0GX,
	}, nil
}

var buildPackage = builtin.PackageBuilder{
	FullPath: "github.com/gx-org/gx/internal/cmd/impgo/tests/returnatom",
	Builders: []builtin.Builder{
		builtin.ParseSource("importgo.gx"),
		builtin.ImplementBuiltin("Float32", evalFloat32),
	},
}

func Build(bld importers.Builder) (importers.Package, error) {
	return builtin.BuilderParam{
		Builder: bld,
		FS:      builtin.StripPath(sources),
	}.Build(buildPackage)
}
