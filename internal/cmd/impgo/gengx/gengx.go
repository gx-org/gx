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

// Package gengx provides the generation of the GX code.
package gengx

import (
	"go/types"

	"github.com/gx-org/gx/internal/cmd/impgo/genast"
	"github.com/gx-org/gx/internal/cmd/impgo/generator"
	"github.com/gx-org/gx/internal/cmd/impgo/parser"
)

type gxGenerator struct {
	cfg    generator.Config
	target generator.Target

	file *genast.File
}

// New GX code generator.
func New(cfg generator.Config, target generator.Target) generator.Generator {
	return &gxGenerator{cfg: cfg, target: target}
}

func (g *gxGenerator) Generate() (string, error) {
	g.file = genast.NewFile(g.target.Name)
	if err := parser.Walk(g.target, g); err != nil {
		return "", err
	}
	return g.file.Write()
}

func (*gxGenerator) FileExtension() string {
	return "gx"
}

func (g *gxGenerator) Func(f *types.Func) error {
	g.file.FuncDecl(
		f.Name(),
		g.gxFuncTypeFromGo(f.Signature()),
		nil)
	return nil
}

func (*gxGenerator) Struct(name *types.TypeName, s *types.Struct) error {
	return nil
}
func (*gxGenerator) Interface(name *types.TypeName, s *types.Interface) error {
	return nil
}
