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

package gengx

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/gx-org/gx/internal/cmd/impgo/genast"
)

func (g *gen) gxTypeFromBasic(t *types.Basic) ast.Expr {
	return genast.Ident(t.Name()).X
}

func (g *gen) gxTypeFromGo(t types.Type) ast.Expr {
	switch tT := t.(type) {
	case *types.Basic:
		return g.gxTypeFromBasic(tT)
	}
	return genast.Ident(fmt.Sprintf("Unsupported(%T)", t)).X
}

func (g *gen) gxFieldsFromTuple(t *types.Tuple) *ast.FieldList {
	fields := make([]*ast.Field, t.Len())
	for i := range t.Len() {
		vr := t.At(i)
		fields[i] = &ast.Field{
			Names: []*ast.Ident{genast.Ident(vr.Name()).X},
			Type:  g.gxTypeFromGo(vr.Type()),
		}
	}
	return &ast.FieldList{List: fields}
}

func (g *gen) gxFuncTypeFromGo(sig *types.Signature) *ast.FuncType {
	return &ast.FuncType{
		Params:  g.gxFieldsFromTuple(sig.Params()),
		Results: g.gxFieldsFromTuple(sig.Results()),
	}
}
