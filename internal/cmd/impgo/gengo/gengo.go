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

// Package gengo provides the generation of the Go code.
package gengo

import (
	"go/ast"
	"path"

	"github.com/gx-org/gx/internal/cmd/impgo/genast"
	"github.com/gx-org/gx/internal/cmd/impgo/generator"
	"github.com/gx-org/gx/internal/cmd/impgo/parser"
)

type goGenerator struct {
	target generator.Target

	file    *genast.File
	builtin genast.IdentExpr
	engine  genast.IdentExpr
	ir      genast.IdentExpr
	values  genast.IdentExpr
	gopkg   genast.IdentExpr
}

// New Go code generator.
func New(target generator.Target) generator.Generator {
	return &goGenerator{target: target}
}

func (g goGenerator) Generate() (string, error) {
	g.file = genast.NewFile(g.target.Name)
	gxSource := g.target.Name + ".gx"

	// Imports
	embedPkg := g.file.Import("github.com/gx-org/gx/build/importers/embedpkg", false)
	importers := g.file.Import("github.com/gx-org/gx/build/importers", false)
	g.builtin = g.file.Import("github.com/gx-org/gx/stdlib/builtin", false)
	g.engine = g.file.Import("github.com/gx-org/gx/interp/engine", false)
	g.ir = g.file.Import("github.com/gx-org/gx/build/ir", false)
	g.values = g.file.Import("github.com/gx-org/gx/api/values", false)
	g.gopkg = g.file.Import(path.Join("google3", g.target.Src.Path()), true)

	// Embed the GX source code
	sources := g.file.EmbedFile("sources", gxSource)

	// Register the package.
	g.file.FuncDecl("init", nil, genast.NewBlock(genast.CallStmt(
		embedPkg.Select("RegisterPackage").X,
		genast.StringLit(path.Join("google3", g.target.Src.Path())),
		genast.Ident("Build").X,
	)))

	// Generate all the directives to complement the GX source code.
	parseSource := genast.CallExpr(g.builtin.Select("ParseSource").X, genast.StringLit(gxSource))
	buildDirectives := g.newDirectives(parseSource)
	if err := parser.Walk(g.target, buildDirectives); err != nil {
		return "", err
	}

	// Build the package.
	builders := genast.SliceLit(
		g.builtin.Select("Builder").X,
		buildDirectives.xs...)
	buildPackage := g.file.Var("buildPackage", nil, genast.StructLit(
		g.builtin.Select("PackageBuilder").X,
		genast.KeyVal("FullPath", genast.StringLit(g.target.Path())),
		genast.KeyVal("Builders", builders),
	).X)
	bld := genast.Ident("bld")
	g.file.FuncDecl("Build",
		&ast.FuncType{
			Params: genast.Fields(
				genast.Field(bld, importers.Select("Builder").X),
			),
			Results: genast.Fields(
				genast.Field(nil, importers.Select("Package").X),
				genast.Field(nil, genast.Error),
			),
		},
		genast.NewBlock().Return(genast.CallExpr(
			genast.StructLit(
				g.builtin.Select("BuilderParam").X,
				genast.KeyVal("Builder", bld.X),
				genast.KeyVal("FS", genast.CallExpr(
					g.builtin.Select("StripPath").X,
					sources.X,
				)),
			).Select("Build").X,
			buildPackage.X,
		)),
	)
	return g.file.Write()
}

func (goGenerator) FileExtension() string {
	return "go"
}
