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

package astbuilder_test

import (
	"bytes"
	"errors"
	"fmt"
	"go/token"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/internal/astbuilder"
	"github.com/gx-org/gx/src/gx/ast"
	"github.com/gx-org/gx/src/gx/format"
	"github.com/gx-org/gx/src/gx/parser"
)

func formatNode(t *testing.T, n ast.Node) string {
	t.Helper()
	var buf bytes.Buffer
	if err := format.Node(&buf, token.NewFileSet(), n); err != nil {
		t.Fatalf("format.Node(%T) failed: %v", n, err)
	}
	return buf.String()
}

func assertDisjointNodes(t *testing.T, orig, cloned ast.Node) {
	t.Helper()
	origNodes := make(map[ast.Node]bool)
	for n := range ast.Preorder(orig) {
		origNodes[n] = true
	}
	for n := range ast.Preorder(cloned) {
		if origNodes[n] {
			t.Errorf("cloned node %T (%p) shares pointer with original", n, n)
		}
	}
}

func mutateClone(root ast.Node) {
	var containers []ast.Node
	for n := range ast.Preorder(root) {
		containers = append(containers, n)
		switch nT := n.(type) {
		case *ast.Ident:
			nT.Name += "_mut"
		case *ast.BasicLit:
			nT.Value = "999"
		case *ast.BinaryExpr:
			nT.Op = token.MUL
		case *ast.UnaryExpr:
			nT.Op = token.ADD
		}
	}
	for _, n := range containers {
		switch nT := n.(type) {
		case *ast.IndexListExpr:
			if len(nT.Indices) > 0 {
				nT.Indices[0] = ast.NewIdent("mutated_idx")
			}
		case *ast.CompositeLit:
			if len(nT.Elts) > 0 {
				nT.Elts[0] = ast.NewIdent("mutated_elt")
			}
		}
	}
}

func TestCloneGenericTypeExpressions(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{name: "index_literal", expr: "Linear[3]"},
		{name: "index_ident", expr: "Linear[Size]"},
		{name: "index_list", expr: "Layer[In, Out]"},
		{name: "selector", expr: "pkg.S"},
		{name: "selector_index", expr: "pkg.Linear[3]"},
		{name: "binary_expr", expr: "Linear[1+2]"},
		{name: "unary_expr", expr: "Linear[-Size]"},
		{name: "paren_expr", expr: "S[(N + 1)]"},
		{name: "slice_array_type", expr: "[]int"},
		{name: "array_type_with_len", expr: "[3]float32"},
		{name: "composite_lit_slice", expr: "SShape[[]int{2, 3}]"},
		{name: "composite_lit_key_value", expr: "T[Size{A: 2, B: 3}]"},
		{name: "composite_lit_nested_nil_type", expr: "M[[][]int{{1, 2}, {3, 4}}]"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			orig, err := parser.ParseExpr(tc.expr)
			if err != nil {
				t.Fatalf("parser.ParseExpr(%q) failed: %v", tc.expr, err)
			}
			wantSrc := formatNode(t, orig)

			cloned, err := astbuilder.Clone(orig, nil)
			if err != nil {
				t.Fatalf("astbuilder.Clone(%q) unexpected error: %v", tc.expr, err)
			}
			if diff := cmp.Diff(orig, cloned); diff != "" {
				t.Errorf("astbuilder.Clone(%q) mismatch (-orig +cloned):\n%s", tc.expr, diff)
			}
			if gotSrc := formatNode(t, cloned); gotSrc != wantSrc {
				t.Errorf("formatNode(cloned) = %q, want %q", gotSrc, wantSrc)
			}

			assertDisjointNodes(t, orig, cloned)
			mutateClone(cloned)
			if afterSrc := formatNode(t, orig); afterSrc != wantSrc {
				t.Errorf("mutating clone modified original: got %q, want %q", afterSrc, wantSrc)
			}
		})
	}
}

func TestCloneTransform(t *testing.T) {
	tests := []struct {
		name          string
		expr          string
		wantTypes     []string
		wantFormatted string
	}{
		{
			name:          "index_expr",
			expr:          "Linear[Size]",
			wantTypes:     []string{"*ast.Ident", "*ast.Ident", "*ast.IndexExpr"},
			wantFormatted: "Linear_t[Size_t]",
		},
		{
			name:          "index_list_expr",
			expr:          "Layer[In, Out]",
			wantTypes:     []string{"*ast.Ident", "*ast.Ident", "*ast.Ident", "*ast.IndexListExpr"},
			wantFormatted: "Layer_t[In_t, Out_t]",
		},
		{
			name:          "selector_expr",
			expr:          "pkg.Linear[Size]",
			wantTypes:     []string{"*ast.Ident", "*ast.Ident", "*ast.SelectorExpr", "*ast.Ident", "*ast.IndexExpr"},
			wantFormatted: "pkg_t.Linear_t[Size_t]",
		},
		{
			name:          "binary_unary_paren_expr",
			expr:          "S[-(N + 1)]",
			wantTypes:     []string{"*ast.Ident", "*ast.Ident", "*ast.BasicLit", "*ast.BinaryExpr", "*ast.ParenExpr", "*ast.UnaryExpr", "*ast.IndexExpr"},
			wantFormatted: "S_t[-(N_t + 1)]",
		},
		{
			name:          "composite_lit_and_slice_array_type",
			expr:          "SShape[[]int{2, 3}]",
			wantTypes:     []string{"*ast.Ident", "*ast.Ident", "*ast.ArrayType", "*ast.BasicLit", "*ast.BasicLit", "*ast.CompositeLit", "*ast.IndexExpr"},
			wantFormatted: "SShape_t[[]int_t{2, 3}]",
		},
		{
			name:          "composite_lit_key_value_expr",
			expr:          "T[Size{A: 2, B: N}]",
			wantTypes:     []string{"*ast.Ident", "*ast.Ident", "*ast.Ident", "*ast.BasicLit", "*ast.KeyValueExpr", "*ast.Ident", "*ast.Ident", "*ast.KeyValueExpr", "*ast.CompositeLit", "*ast.IndexExpr"},
			wantFormatted: "T_t[Size_t{A_t: 2, B_t: N_t}]",
		},
		{
			name:          "nested_composite_lit_nil_type",
			expr:          "M[[][]int{{1, N}}]",
			wantTypes:     []string{"*ast.Ident", "*ast.Ident", "*ast.ArrayType", "*ast.ArrayType", "*ast.BasicLit", "*ast.Ident", "*ast.CompositeLit", "*ast.CompositeLit", "*ast.IndexExpr"},
			wantFormatted: "M_t[[][]int_t{{1, N_t}}]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			orig, err := parser.ParseExpr(tc.expr)
			if err != nil {
				t.Fatalf("parser.ParseExpr(%q) failed: %v", tc.expr, err)
			}
			origSrc := formatNode(t, orig)

			var visited []string
			cloned, err := astbuilder.Clone(orig, func(n ast.Node) error {
				visited = append(visited, fmt.Sprintf("%T", n))
				if id, ok := n.(*ast.Ident); ok {
					id.Name += "_t"
				}
				return nil
			})
			if err != nil {
				t.Fatalf("astbuilder.Clone(%q) unexpected error: %v", tc.expr, err)
			}
			if diff := cmp.Diff(tc.wantTypes, visited); diff != "" {
				t.Errorf("Transform visited node types mismatch (-want +got):\n%s", diff)
			}
			if got := formatNode(t, cloned); got != tc.wantFormatted {
				t.Errorf("transformed clone = %q, want %q", got, tc.wantFormatted)
			}
			if afterOrig := formatNode(t, orig); afterOrig != origSrc {
				t.Errorf("Transform mutated original: got %q, want %q", afterOrig, origSrc)
			}
		})
	}
}

func TestCloneTransformError(t *testing.T) {
	orig, err := parser.ParseExpr("Linear[Size]")
	if err != nil {
		t.Fatalf("parser.ParseExpr failed: %v", err)
	}
	wantErr := errors.New("boom")
	_, err = astbuilder.Clone(orig, func(n ast.Node) error {
		if _, ok := n.(*ast.IndexExpr); ok {
			return wantErr
		}
		return nil
	})
	if err == nil {
		t.Fatalf("astbuilder.Clone returned nil error, want error containing %q", wantErr)
	}
	if !strings.Contains(err.Error(), wantErr.Error()) {
		t.Errorf("astbuilder.Clone error = %q, want substring %q", err.Error(), wantErr.Error())
	}
}

func TestCloneUnsupportedNodes(t *testing.T) {
	tests := []struct {
		name          string
		node          ast.Node
		wantErrSubstr string
	}{
		{
			name:          "star_expr",
			node:          &ast.StarExpr{X: ast.NewIdent("T")},
			wantErrSubstr: "*ast.StarExpr not supported",
		},
		{
			name:          "map_type",
			node:          &ast.MapType{Key: ast.NewIdent("string"), Value: ast.NewIdent("int")},
			wantErrSubstr: "*ast.MapType not supported",
		},
		{
			name:          "chan_type",
			node:          &ast.ChanType{Value: ast.NewIdent("int")},
			wantErrSubstr: "*ast.ChanType not supported",
		},
		{
			name: "nested_in_index_expr",
			node: &ast.IndexExpr{
				X:     ast.NewIdent("Linear"),
				Index: &ast.StarExpr{X: ast.NewIdent("T")},
			},
			wantErrSubstr: "*ast.StarExpr not supported",
		},
		{
			name: "nested_in_index_list_expr",
			node: &ast.IndexListExpr{
				X:       ast.NewIdent("Layer"),
				Indices: []ast.Expr{ast.NewIdent("In"), &ast.MapType{Key: ast.NewIdent("int"), Value: ast.NewIdent("int")}},
			},
			wantErrSubstr: "*ast.MapType not supported",
		},
		{
			name: "nested_in_composite_lit",
			node: &ast.CompositeLit{
				Type: ast.NewIdent("S"),
				Elts: []ast.Expr{&ast.ChanType{Value: ast.NewIdent("int")}},
			},
			wantErrSubstr: "*ast.ChanType not supported",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := astbuilder.Clone(tc.node, nil); err == nil {
				t.Errorf("Clone(%s, nil): got nil error, want %q", tc.name, tc.wantErrSubstr)
			} else if !strings.Contains(err.Error(), tc.wantErrSubstr) {
				t.Errorf("Clone(%s, nil): error = %q, want substring %q", tc.name, err.Error(), tc.wantErrSubstr)
			}

			_, err := astbuilder.Clone(tc.node, func(n ast.Node) error {
				if n == nil {
					t.Errorf("Clone(%s, transform): transform called with nil node", tc.name)
				}
				return nil
			})
			if err == nil {
				t.Errorf("Clone(%s, transform): got nil error, want %q", tc.name, tc.wantErrSubstr)
			} else if !strings.Contains(err.Error(), tc.wantErrSubstr) {
				t.Errorf("Clone(%s, transform): error = %q, want substring %q", tc.name, err.Error(), tc.wantErrSubstr)
			}
		})
	}
}
