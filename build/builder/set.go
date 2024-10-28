// Copyright 2024 Google LLC
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

package builder

import (
	"go/ast"

	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/ir"
)

type setFunc struct {
	ext ir.FuncBuiltin
}

var (
	_ genericCallTypeNode = (*setFunc)(nil)
	_ function            = (*setFunc)(nil)
)

func (f *setFunc) resolveGenericCallType(scope scoper, src ast.Node, fetcher ir.Fetcher, call *ir.CallExpr) (*funcType, bool) {
	params, err := builtins.BuildFuncParams(fetcher, call, "set", []ir.Type{
		builtins.GenericArrayType,
		builtins.GenericArrayType,
		builtins.PositionsType,
	})
	if err != nil {
		scope.err().AppendAt(src, err)
		return nil, false
	}
	arrayParams, err := builtins.NarrowTypes[*ir.ArrayType](fetcher, call, params)
	if err != nil {
		scope.err().Appendf(src, "cannot fetch array type: %v", err)
		return nil, false
	}
	sameDType, err := arrayParams[0].DType.Equal(fetcher, arrayParams[1].DType)
	if err != nil {
		scope.err().Appendf(src, "cannot compare datatypes: %v", err)
		return nil, false
	}
	if !sameDType {
		scope.err().Appendf(src, "cannot set a slice of a [...]%s array with a [...]%s array", arrayParams[0].DType.String(), arrayParams[1].DType.String())
		return nil, false
	}
	funcType, ok := importFuncType(scope, &ir.FuncType{
		Src:     &ast.FuncType{Func: call.Src.Pos()},
		Params:  builtins.Fields(params...),
		Results: builtins.Fields(params[0]),
	})
	if !ok {
		return funcType, ok
	}
	xResolver := arrayParams[0].Rank().(ir.ResolvedRank)
	if !ok {
		return nil, scope.err().Appendf(src, "cannot set a sub-array with an unresolved rank")
	}
	xRank := xResolver.Resolved()
	updateRank := arrayParams[1].Rank()
	posResolver, ok := arrayParams[2].Rank().(ir.ResolvedRank)
	if !ok {
		return nil, scope.err().Appendf(src, "cannot set a sub-array at a position with an unresolved rank")
	}
	posRank := posResolver.Resolved()
	if xRank == nil || updateRank == nil || posRank == nil {
		return funcType, true
	}
	if len(posRank.Axes) != 1 {
		return nil, scope.err().Appendf(src, "position has an invalid number of axes: got %d but want 1", len(posRank.Axes))
	}
	posSize, unknows, err := ir.Eval[ir.Int](scope.evalFetcher(), posRank.Axes[0].Expr())
	if err != nil {
		return nil, scope.err().AppendAt(src, err)
	}
	if unknows != nil {
		return nil, scope.err().Appendf(src, "cannot evaluate axis lengths of the position because of unknowns variables %v", unknows)
	}
	if int(posSize) > len(xRank.Axes) {
		return nil, scope.err().Appendf(src, "position (length %d) exceeds operand rank (%d)", posSize, len(xRank.Axes))
	}
	wantUpdate := &ir.ArrayType{
		DType: arrayParams[0].DType,
		RankF: &ir.Rank{
			Axes: xRank.Axes[posSize:],
		},
	}
	ok, err = arrayParams[1].Equal(fetcher, wantUpdate)
	if err != nil {
		scope.err().Appendf(src, "cannot compare rank: %v", err)
		return nil, false
	}
	if !ok {
		scope.err().Appendf(src, "cannot set array: update slice is %s but requires %s", arrayParams[1], wantUpdate)
		return nil, false
	}
	return funcType, true
}

func (f *setFunc) typeNode() typeNode {
	return f
}

func (f *setFunc) name() *ast.Ident {
	return &ast.Ident{Name: f.ext.Name()}
}

func (f *setFunc) isGeneric() bool {
	return false
}

func (f *setFunc) kind() ir.Kind {
	return ir.FuncKind
}

func (f *setFunc) buildType() ir.Type {
	return &ir.FuncType{}
}

func (f *setFunc) String() string {
	return "set"
}

func (f *setFunc) irFunc() ir.Func {
	return &f.ext
}
