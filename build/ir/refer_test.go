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

package ir_test

import (
	"fmt"
	"go/token"
	"math/big"
	"testing"

	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irhelper"
	"github.com/gx-org/gx/src/gx/ast"
)

func referTestFiles() (otherFile, aliasFile *ir.File) {
	otherFile = &ir.File{
		Package: &ir.Package{Name: &ast.Ident{Name: "other"}, Decls: &ir.Declarations{}},
	}
	aliasFile = &ir.File{
		Package: &ir.Package{Name: &ast.Ident{Name: "consumer"}, Decls: &ir.Declarations{}},
		Imports: []*ir.ImportDecl{{
			Src:     &ast.ImportSpec{Name: &ast.Ident{Name: "myir"}, Path: &ast.BasicLit{Kind: token.STRING, Value: `"ir_test"`}},
			Path:    "ir_test",
			Package: testFile.Package,
		}},
	}
	return otherFile, aliasFile
}

func intLit(val int64) *ir.NumberInt {
	return &ir.NumberInt{
		Src: &ast.BasicLit{Kind: token.INT, Value: fmt.Sprint(val)},
		Val: big.NewInt(val),
	}
}

func intLitAs(val int64, typ ir.Type) ir.Expr {
	return &ir.NumberCastExpr{X: intLit(val), Typ: typ}
}

func floatLit(val float64) *ir.NumberFloat {
	return &ir.NumberFloat{
		Src: &ast.BasicLit{Kind: token.FLOAT, Value: fmt.Sprint(val)},
		Val: big.NewFloat(val),
	}
}

func newGenericNamedType(name string, fields *ir.FieldList, values ...ir.GenericValue) *ir.NamedType {
	nt := newNamedType(name, ir.Float32Type())
	nt.GenParams.SetFields(fields)
	if len(values) > 0 {
		nt.GenParams.Values = values
	}
	return nt
}

func TestNamedTypeRefer(t *testing.T) {
	otherFile, aliasFile := referTestFiles()
	plain := newNamedType("Plain", ir.Int32Type())

	sizeFields := irhelper.Fields("Size", ir.IntType())
	sizeParam := ir.NewGenericNonTypeParam(sizeFields.Fields()[0])
	mParam := ir.NewGenericNonTypeParam(irhelper.Fields("M", ir.IntType()).Fields()[0])

	uninstantiated := newGenericNamedType("Linear", sizeFields)
	singleLit := newGenericNamedType("Linear", sizeFields, ir.NewNonTypeGenericValue(sizeParam, intLitAs(3, ir.IntType())))
	singleIdent := newGenericNamedType("Linear", sizeFields, ir.NewNonTypeGenericValue(sizeParam, ir.NewIdent(sizeParam)))
	singleBinary := newGenericNamedType("Linear", sizeFields, ir.NewNonTypeGenericValue(sizeParam, &ir.BinaryExpr{
		Src: &ast.BinaryExpr{Op: token.ADD, X: &ast.Ident{Name: "StaleSize"}, Y: &ast.BasicLit{Kind: token.INT, Value: "1"}},
		X:   ir.NewIdent(mParam),
		Y:   intLitAs(1, ir.IntType()),
		Typ: ir.IntType(),
	}))

	shapeFields := irhelper.Fields("Shape", ir.IntSliceType())
	sliceShape := newGenericNamedType("Tensor", shapeFields, ir.NewNonTypeGenericValue(
		ir.NewGenericNonTypeParam(shapeFields.Fields()[0]),
		&ir.SliceLitExpr{Src: &ast.ArrayType{}, Typ: ir.IntSliceType(), Elts: []ir.Expr{intLitAs(2, ir.IntType()), intLitAs(3, ir.IntType())}},
	))

	multiFields := irhelper.Fields("Rows", ir.IntType(), "Cols", ir.IntType())
	multiArg := newGenericNamedType("Matrix", multiFields,
		ir.NewNonTypeGenericValue(ir.NewGenericNonTypeParam(multiFields.Fields()[0]), intLitAs(2, ir.IntType())),
		ir.NewNonTypeGenericValue(ir.NewGenericNonTypeParam(multiFields.Fields()[1]), intLitAs(4, ir.IntType())),
	)

	boxFields := irhelper.Fields("T", ir.AnyType())
	boxParam := ir.NewGenericTypeParam(boxFields.Fields()[0])
	// Simulate (*TypeValExpr).Specialise, where val is updated to float64 while src stays U.
	typeParamConcrete := newGenericNamedType("Box", boxFields, ir.NewTypeExprGenericValue(boxParam, ir.TypeExpr(&ast.Ident{Name: "U"}, ir.Float64Type())))
	typeParamGeneric := newGenericNamedType("Box", boxFields, ir.NewTypeGenericValue(boxParam, boxParam))

	outerFields := irhelper.Fields("T", ir.AnyType(), "N", ir.IntType())
	nestedGeneric := newGenericNamedType("Outer", outerFields,
		ir.NewTypeGenericValue(ir.NewGenericTypeParam(outerFields.Fields()[0]), singleLit),
		ir.NewNonTypeGenericValue(ir.NewGenericNonTypeParam(outerFields.Fields()[1]), intLitAs(5, ir.IntType())),
	)

	for _, tc := range []struct {
		name     string
		named    *ir.NamedType
		file     *ir.File
		want     string
		wantType string
	}{
		{"NonGenericSameFile", plain, testFile, "Plain", "*ast.Ident"},
		{"NonGenericNilFile", plain, nil, "Plain", "*ast.Ident"},
		{"NonGenericCrossPackage", plain, otherFile, "ir_test.Plain", "*ast.SelectorExpr"},
		{"NonGenericCrossPackageAlias", plain, aliasFile, "myir.Plain", "*ast.SelectorExpr"},
		{"GenericUninstantiated", uninstantiated, testFile, "Linear", "*ast.Ident"},
		{"SingleArgLiteralSameFile", singleLit, testFile, "Linear[3]", "*ast.IndexExpr"},
		{"SingleArgLiteralNilFile", singleLit, nil, "Linear[3]", "*ast.IndexExpr"},
		{"SingleArgIdent", singleIdent, testFile, "Linear[Size]", "*ast.IndexExpr"},
		{"SingleArgSpecializedBinaryExpr", singleBinary, testFile, "Linear[M+1]", "*ast.IndexExpr"},
		{"SingleArgSliceShape", sliceShape, testFile, "Tensor[[]int{2, 3}]", "*ast.IndexExpr"},
		{"MultiArg", multiArg, testFile, "Matrix[2, 4]", "*ast.IndexListExpr"},
		{"TypeParamSpecializedPrefersDefinedType", typeParamConcrete, testFile, "Box[float64]", "*ast.IndexExpr"},
		{"TypeParamGeneric", typeParamGeneric, testFile, "Box[T]", "*ast.IndexExpr"},
		{"NestedGenericSameFile", nestedGeneric, testFile, "Outer[Linear[3], 5]", "*ast.IndexListExpr"},
		{"CrossPackageSingleArg", singleLit, otherFile, "ir_test.Linear[3]", "*ast.IndexExpr"},
		{"CrossPackageAliasSingleArg", singleLit, aliasFile, "myir.Linear[3]", "*ast.IndexExpr"},
		{"CrossPackageNestedMultiArg", nestedGeneric, otherFile, "ir_test.Outer[ir_test.Linear[3], 5]", "*ast.IndexListExpr"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.named.Refer(tc.file)
			if gotType := fmt.Sprintf("%T", got); gotType != tc.wantType {
				t.Errorf("NamedType.Refer(%v) returned %s, want %s", tc.file, gotType, tc.wantType)
			}
			if gotStr := gxfmt.Format(got); gotStr != tc.want {
				t.Errorf("gxfmt.Format(NamedType.Refer(%v)) = %q, want %q", tc.file, gotStr, tc.want)
			}
		})
	}
}

func TestExprRefer(t *testing.T) {
	otherFile, aliasFile := referTestFiles()
	plain := newNamedType("Plain", ir.Int32Type())
	mParam := ir.NewGenericNonTypeParam(irhelper.Fields("M", ir.IntType()).Fields()[0])
	sParam := ir.NewGenericNonTypeParam(irhelper.Fields("S", ir.IntSliceType()).Fields()[0])
	pkgConst := &ir.ConstExpr{Decl: &ir.ConstSpec{FFile: testFile}, VName: &ast.Ident{Name: "N"}}
	pkgFunc := &ir.FuncDecl{
		FFile: testFile,
		Src:   &ast.FuncDecl{Name: &ast.Ident{Name: "F"}},
		FType: irhelper.FuncType(nil, nil, irhelper.Fields("x", ir.IntType()), irhelper.Fields(ir.IntType())),
	}
	lenKeyword := &ir.FuncKeyword{ID: &ast.Ident{Name: "len"}}
	origImport := &ir.ImportDecl{
		Src:     &ast.ImportSpec{Name: &ast.Ident{Name: "orig"}, Path: &ast.BasicLit{Kind: token.STRING, Value: `"ir_test"`}},
		Path:    "ir_test",
		Package: testFile.Package,
	}
	importIdent := &ir.Ident{Src: &ast.Ident{Name: "orig"}, Stor: origImport}
	nilPkgImport := &ir.ImportDecl{
		Src:  &ast.ImportSpec{Name: &ast.Ident{Name: "unresolved"}, Path: &ast.BasicLit{Kind: token.STRING, Value: `"unresolved"`}},
		Path: "unresolved",
	}
	nilPkgIdent := &ir.Ident{Src: &ast.Ident{Name: "unresolved"}, Stor: nilPkgImport}
	pkgRef := &ir.PackageRef{X: importIdent, Decl: origImport}
	embeddedImport := &ir.ImportEmbedded{Src: &ast.Ident{Name: "orig"}, Package: testFile.Package}
	embeddedSel := &ir.SelectorExpr{
		Src:  &ast.SelectorExpr{X: &ast.Ident{Name: "orig"}, Sel: &ast.Ident{Name: "N"}},
		X:    embeddedImport,
		Stor: pkgConst,
	}
	pkgRefSel := &ir.SelectorExpr{
		Src:  &ast.SelectorExpr{X: &ast.Ident{Name: "orig"}, Sel: &ast.Ident{Name: "N"}},
		X:    pkgRef,
		Stor: pkgConst,
	}
	importIdentSel := &ir.SelectorExpr{
		Src:  &ast.SelectorExpr{X: &ast.Ident{Name: "orig"}, Sel: &ast.Ident{Name: "N"}},
		X:    importIdent,
		Stor: pkgConst,
	}
	cfgFields := irhelper.Fields("N", ir.IntType(), "Zero", ir.IntType())
	cfgNamed := newNamedType("Cfg", &ir.StructType{Fields: cfgFields})
	fieldSel := &ir.SelectorExpr{
		Src:  &ast.SelectorExpr{X: &ast.Ident{Name: "cfg"}, Sel: &ast.Ident{Name: "N"}},
		X:    ir.NewIdent(irhelper.LocalVar("cfg", cfgNamed)),
		Stor: cfgFields.Fields()[0].Storage(),
	}
	nilPkgFile := &ir.File{}
	arrType := ir.NewArrayType(
		&ast.ArrayType{Len: &ast.BasicLit{Kind: token.INT, Value: "2"}, Elt: &ast.Ident{Name: "int"}},
		ir.IntType(),
		&ir.Rank{Ax: []ir.AxisLengths{&ir.AxisExpr{X: intLitAs(2, ir.IntType())}}},
	)

	for _, tc := range []struct {
		name string
		expr ir.Expr
		file *ir.File
		want string
	}{
		{"NumberInt", intLit(42), testFile, "42"},
		{"NumberFloat", floatLit(1.5), testFile, "1.5"},
		{"NumberCast", intLitAs(9, ir.IntType()), testFile, "9"},
		{"BoolValueTrue", ir.True(), testFile, "true"},
		{"BoolValueFalse", ir.False(), testFile, "false"},
		{"StringLiteral", &ir.StringLiteral{Src: &ast.BasicLit{Kind: token.STRING, Value: `"hi"`}}, testFile, `"hi"`},
		{"ArrayLitExpr", &ir.ArrayLitExpr{Typ: arrType, Elts: []ir.Expr{intLit(1), intLit(2)}}, testFile, "[2]int{1, 2}"},
		{"TypeValExprCrossPackage", ir.TypeExpr(&ast.Ident{Name: "Stale"}, plain), otherFile, "ir_test.Plain"},
		{"IdentConstSamePackage", ir.NewIdent(pkgConst), testFile, "N"},
		{"IdentConstCrossPackage", ir.NewIdent(pkgConst), otherFile, "ir_test.N"},
		{"IdentConstCrossPackageAlias", ir.NewIdent(pkgConst), aliasFile, "myir.N"},
		{"IdentFuncKeywordSamePackage", ir.NewIdent(lenKeyword), testFile, "len"},
		{"IdentFuncKeywordCrossPackage", ir.NewIdent(lenKeyword), otherFile, "len"},
		{"IdentImportDeclNilFile", importIdent, nil, "orig"},
		{"IdentImportDeclNilPackage", nilPkgIdent, otherFile, "unresolved"},
		{"IdentImportDeclOtherPackage", importIdent, otherFile, "ir_test"},
		{"IdentImportDeclAliasPackage", importIdent, aliasFile, "myir"},
		{"PackageRefNilFile", pkgRef, nil, "orig"},
		{"PackageRefOtherPackage", pkgRef, otherFile, "ir_test"},
		{"PackageRefAliasPackage", pkgRef, aliasFile, "myir"},
		{"ImportEmbeddedNilFile", embeddedImport, nil, "orig"},
		{"ImportEmbeddedOtherPackage", embeddedImport, otherFile, "ir_test"},
		{"ImportEmbeddedAliasPackage", embeddedImport, aliasFile, "myir"},
		{"SelectorImportEmbeddedNilFile", embeddedSel, nil, "orig.N"},
		{"SelectorImportEmbeddedSamePackage", embeddedSel, testFile, "N"},
		{"SelectorImportEmbeddedCrossPackage", embeddedSel, otherFile, "ir_test.N"},
		{"SelectorImportEmbeddedCrossPackageAlias", embeddedSel, aliasFile, "myir.N"},
		{"SelectorPackageRefNilFile", pkgRefSel, nil, "orig.N"},
		{"SelectorPackageRefSamePackage", pkgRefSel, testFile, "N"},
		{"SelectorPackageRefCrossPackage", pkgRefSel, otherFile, "ir_test.N"},
		{"SelectorPackageRefCrossPackageAlias", pkgRefSel, aliasFile, "myir.N"},
		{"SelectorImportIdentNilFile", importIdentSel, nil, "orig.N"},
		{"SelectorImportIdentSamePackage", importIdentSel, testFile, "N"},
		{"SelectorImportIdentCrossPackage", importIdentSel, otherFile, "ir_test.N"},
		{"SelectorImportIdentCrossPackageAlias", importIdentSel, aliasFile, "myir.N"},
		{"SelectorFieldSamePackage", fieldSel, testFile, "cfg.N"},
		{"SelectorFieldNilPackageFile", fieldSel, nilPkgFile, "cfg.N"},
		{"UnaryExprSynthetic", &ir.UnaryExpr{Src: &ast.UnaryExpr{Op: token.SUB}, X: ir.NewIdent(mParam)}, testFile, "-M"},
		{"ParenAndBinaryExprSynthetic", &ir.ParenExpr{Src: &ast.ParenExpr{}, X: &ir.BinaryExpr{Src: &ast.BinaryExpr{Op: token.ADD}, X: ir.NewIdent(mParam), Y: intLit(1), Typ: ir.IntType()}}, testFile, "(M + 1)"},
		{"StructLitExprSyntheticCrossPackage", &ir.StructLitExpr{Typ: cfgNamed, Elts: []*ir.FieldLit{{FieldStorage: cfgFields.Fields()[0].Storage(), X: intLit(4)}, {FieldStorage: cfgFields.Fields()[1].Storage()}}}, otherFile, "ir_test.Cfg{N: 4}"},
		{"SliceExprLowerBoundOnly", &ir.SliceExpr{Src: &ast.SliceExpr{}, X: ir.NewIdent(sParam), Low: intLit(1)}, testFile, "S[1:]"},
		{"IndexExprSpecialized", &ir.IndexExpr{Src: &ast.IndexExpr{}, X: ir.NewIdent(sParam), Index: intLit(0), Typ: ir.IntType()}, testFile, "S[0]"},
		{"CastExprSyntheticCrossPackage", &ir.CastExpr{Typ: plain, X: intLit(3)}, otherFile, "ir_test.Plain(3)"},
		{"FuncCallExprSpecializedCrossPackage", &ir.FuncCallExpr{Src: &ast.CallExpr{}, Callee: ir.NewFuncValExpr(ir.NewIdent(pkgFunc), pkgFunc), Args: []ir.Expr{ir.NewIdent(mParam)}}, otherFile, "ir_test.F(M)"},
		{"UnpackExpr", &ir.UnpackExpr{X: ir.NewIdent(sParam), EltTyp: ir.IntType()}, testFile, "S..."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.expr.Refer(tc.file)
			if gotStr := gxfmt.Format(got); gotStr != tc.want {
				t.Errorf("gxfmt.Format(%T.Refer(%v)) = %q, want %q", tc.expr, tc.file, gotStr, tc.want)
			}
		})
	}
}

func TestArrayTypeRefer(t *testing.T) {
	otherFile, aliasFile := referTestFiles()
	plain := newNamedType("Plain", ir.Int32Type())
	mParam := ir.NewGenericNonTypeParam(irhelper.Fields("M", ir.IntType()).Fields()[0])
	sParam := ir.NewGenericNonTypeParam(irhelper.Fields("S", ir.IntSliceType()).Fields()[0])
	tParam := ir.NewGenericTypeParam(irhelper.Fields("T", ir.AnyType()).Fields()[0])
	pkgConst := &ir.ConstExpr{Decl: &ir.ConstSpec{FFile: testFile}, VName: &ast.Ident{Name: "N"}}
	staleSrc := &ast.ArrayType{Len: &ast.BasicLit{Kind: token.INT, Value: "9"}, Elt: &ast.Ident{Name: "Stale"}}
	multiRank := &ir.Rank{
		Ax: []ir.AxisLengths{
			&ir.AxisExpr{X: ir.NewIdent(mParam)},
			&ir.AxisExpr{X: ir.NewIdent(pkgConst)},
		},
	}
	multiArray := ir.NewArrayType(staleSrc, plain, multiRank)
	lenKeyword := &ir.FuncKeyword{ID: &ast.Ident{Name: "len"}}
	lenCallRank := &ir.Rank{
		Ax: []ir.AxisLengths{
			&ir.AxisExpr{X: &ir.FuncCallExpr{
				Src:    &ast.CallExpr{},
				Callee: ir.NewFuncValExpr(ir.NewIdent(lenKeyword), lenKeyword),
				Args:   []ir.Expr{ir.NewIdent(sParam)},
			}},
		},
	}
	unpackRank := &ir.Rank{
		Ax: []ir.AxisLengths{
			&ir.AxisExpr{X: &ir.UnpackExpr{X: ir.NewIdent(sParam), EltTyp: ir.IntType()}},
			&ir.AxisExpr{X: ir.NewIdent(pkgConst)},
		},
	}
	axisInferRank := &ir.Rank{
		Ax: []ir.AxisLengths{
			&ir.AxisInfer{Src: &ast.Ident{Name: "_"}},
			&ir.AxisInfer{Src: &ast.Ident{Name: "_"}, X: &ir.AxisExpr{X: intLit(4)}},
		},
	}
	singleArray := ir.NewArrayType(staleSrc, plain, &ir.Rank{Ax: []ir.AxisLengths{&ir.AxisExpr{X: intLit(2)}}})
	sliceOfArray := &ir.SliceType{
		BaseType: ir.BaseType[ast.Expr]{Src: staleSrc},
		DType:    ir.TypeExpr(staleSrc, singleArray),
		Rank:     1,
	}

	for _, tc := range []struct {
		name string
		typ  ir.Type
		file *ir.File
		want string
	}{
		{"ScalarRankBuiltin", ir.NewArrayType(staleSrc, ir.Float32Type(), &ir.Rank{}), testFile, "float32"},
		{"ScalarRankCrossPackage", ir.NewArrayType(staleSrc, plain, &ir.Rank{}), otherFile, "ir_test.Plain"},
		{"SingleAxisLiteral", ir.NewArrayType(staleSrc, ir.Float32Type(), &ir.Rank{Ax: []ir.AxisLengths{&ir.AxisExpr{X: intLit(2)}}}), testFile, "[2]float32"},
		{"MultiAxisSamePackage", multiArray, testFile, "[M][N]Plain"},
		{"MultiAxisCrossPackage", multiArray, otherFile, "[M][ir_test.N]ir_test.Plain"},
		{"MultiAxisCrossPackageAlias", multiArray, aliasFile, "[M][myir.N]myir.Plain"},
		{"AxisGroupUnpack", ir.NewArrayType(staleSrc, ir.Float32Type(), unpackRank), otherFile, "[S...][ir_test.N]float32"},
		{"AxisInferMixed", ir.NewArrayType(staleSrc, ir.Float32Type(), axisInferRank), testFile, "[_][4]float32"},
		{"AxisFuncKeyword", ir.NewArrayType(staleSrc, ir.Float32Type(), lenCallRank), otherFile, "[len(S)]float32"},
		{"RankInferUnresolved", ir.NewArrayType(staleSrc, plain, ir.NewRankInfer(&ast.ArrayType{Len: &ast.Ellipsis{}}, nil)), otherFile, "[...]ir_test.Plain"},
		{"RankInferResolved", ir.NewArrayType(staleSrc, plain, ir.NewRankInfer(&ast.ArrayType{Len: &ast.Ellipsis{}}, multiRank)), otherFile, "[M][ir_test.N]ir_test.Plain"},
		{"RankInferResolvedScalar", ir.NewArrayType(staleSrc, ir.Float32Type(), ir.NewRankInfer(&ast.ArrayType{Len: &ast.Ellipsis{}}, &ir.Rank{})), testFile, "float32"},
		{"GenericTypeParamDType", ir.NewArrayType(staleSrc, tParam, &ir.Rank{Ax: []ir.AxisLengths{&ir.AxisExpr{X: ir.NewIdent(mParam)}}}), testFile, "[M]T"},
		{"SliceOfArrayCrossPackage", sliceOfArray, otherFile, "[][2]ir_test.Plain"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := gxfmt.Format(tc.typ.Refer(tc.file)); got != tc.want {
				t.Errorf("gxfmt.Format(%T.Refer(%v)) = %q, want %q", tc.typ, tc.file, got, tc.want)
			}
		})
	}
}

func formatRankAST(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	arr, ok := expr.(*ast.ArrayType)
	if !ok {
		return gxfmt.Format(expr)
	}
	cur := arr
	for cur.Elt != nil {
		cur = cur.Elt.(*ast.ArrayType)
	}
	cur.Elt = &ast.Ident{Name: "T"}
	return gxfmt.Format(arr)
}

func TestRankRefer(t *testing.T) {
	otherFile, aliasFile := referTestFiles()
	mParam := ir.NewGenericNonTypeParam(irhelper.Fields("M", ir.IntType()).Fields()[0])
	pkgConst := &ir.ConstExpr{Decl: &ir.ConstSpec{FFile: testFile}, VName: &ast.Ident{Name: "N"}}
	multiRank := &ir.Rank{
		Ax: []ir.AxisLengths{
			&ir.AxisExpr{X: ir.NewIdent(mParam)},
			&ir.AxisExpr{X: ir.NewIdent(pkgConst)},
		},
	}
	inferUnresolved := ir.NewRankInfer(&ast.ArrayType{Len: &ast.Ellipsis{}}, nil)
	inferResolved := ir.NewRankInfer(&ast.ArrayType{Len: &ast.Ellipsis{}}, multiRank)
	axisInferRank := &ir.Rank{
		Ax: []ir.AxisLengths{
			&ir.AxisInfer{Src: &ast.Ident{Name: "_"}},
			&ir.AxisInfer{Src: &ast.Ident{Name: "_"}, X: &ir.AxisExpr{X: intLit(4)}},
		},
	}

	for _, tc := range []struct {
		name string
		rank ir.ArrayRank
		file *ir.File
		want string
	}{
		{"ScalarRank", &ir.Rank{}, testFile, ""},
		{"SingleAxis", &ir.Rank{Ax: []ir.AxisLengths{&ir.AxisExpr{X: intLit(3)}}}, testFile, "[3]T"},
		{"MultiAxisSamePackage", multiRank, testFile, "[M][N]T"},
		{"MultiAxisCrossPackage", multiRank, otherFile, "[M][ir_test.N]T"},
		{"MultiAxisCrossPackageAlias", multiRank, aliasFile, "[M][myir.N]T"},
		{"AxisInferMixed", axisInferRank, testFile, "[_][4]T"},
		{"RankInferUnresolved", inferUnresolved, testFile, "..."},
		{"RankInferResolved", inferResolved, otherFile, "[M][ir_test.N]T"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.rank.Refer(tc.file)
			if gotStr := formatRankAST(got); gotStr != tc.want {
				t.Errorf("formatRankAST(%T.Refer(%v)) = %q, want %q", tc.rank, tc.file, gotStr, tc.want)
			}
		})
	}
}

func TestFieldStructAndVarArgsRefer(t *testing.T) {
	otherFile, aliasFile := referTestFiles()
	plain := newNamedType("Plain", ir.Int32Type())

	var nilFields *ir.FieldList
	if got := nilFields.Refer(testFile); got != nil {
		t.Errorf("(*ir.FieldList)(nil).Refer() = %v, want nil", got)
	}

	anonField := (&ir.FieldGroup{Type: ir.TypeExpr(nil, plain)}).Refer(otherFile)
	if anonField.Names != nil {
		t.Errorf("anon FieldGroup.Refer(otherFile).Names = %v, want nil", anonField.Names)
	}
	if got := gxfmt.Format(&ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{anonField}}}); got != "func() ir_test.Plain" {
		t.Errorf("anon FieldGroup.Refer(otherFile) formatted as %q, want %q", got, "func() ir_test.Plain")
	}

	namedFields := irhelper.Fields("a", "b", plain, "c", ir.IntType())
	if got := gxfmt.Format(&ast.FuncType{Params: namedFields.Refer(aliasFile)}); got != "func(a, b myir.Plain, c int)" {
		t.Errorf("FieldList.Refer(aliasFile) formatted as %q, want %q", got, "func(a, b myir.Plain, c int)")
	}

	st := &ir.StructType{Fields: namedFields}
	varArgsPlain := &ir.VarArgsType{
		Src: &ast.Ellipsis{Ellipsis: 1, Elt: &ast.Ident{Name: "Stale"}},
		Typ: &ir.SliceType{DType: ir.TypeExpr(nil, plain), Rank: 1},
	}
	varArgsInt := &ir.VarArgsType{Typ: ir.IntSliceType()}

	for _, tc := range []struct {
		name string
		typ  ir.Type
		file *ir.File
		want string
	}{
		{"EmptyStructNilFields", &ir.StructType{}, testFile, "struct{}"},
		{"EmptyStructEmptyFields", &ir.StructType{Fields: &ir.FieldList{}}, testFile, "struct{}"},
		{"StructSamePackage", st, testFile, "struct {\n\ta, b Plain\n\tc    int\n}"},
		{"StructCrossPackage", st, otherFile, "struct {\n\ta, b ir_test.Plain\n\tc    int\n}"},
		{"StructCrossPackageAlias", st, aliasFile, "struct {\n\ta, b myir.Plain\n\tc    int\n}"},
		{"VarArgsBuiltin", varArgsInt, testFile, "...int"},
		{"VarArgsSamePackage", varArgsPlain, testFile, "...Plain"},
		{"VarArgsCrossPackage", varArgsPlain, otherFile, "...ir_test.Plain"},
		{"VarArgsCrossPackageAlias", varArgsPlain, aliasFile, "...myir.Plain"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := gxfmt.Format(tc.typ.Refer(tc.file)); got != tc.want {
				t.Errorf("gxfmt.Format(%T.Refer(%v)) = %q, want %q", tc.typ, tc.file, got, tc.want)
			}
		})
	}
}

func TestInterfaceRefer(t *testing.T) {
	otherFile, aliasFile := referTestFiles()
	plain := newNamedType("Plain", ir.Int32Type())

	emptyWithSrc := ir.NewInterface(&ast.InterfaceType{}, nil, nil)
	singleType := ir.NewInterface(&ast.InterfaceType{}, []ir.Type{ir.Int32Type()}, nil)
	unionTypes := ir.NewInterface(&ast.InterfaceType{}, []ir.Type{ir.Int32Type(), ir.Int64Type(), plain}, nil)

	evalMeth := &ir.IMethod{
		Src: &ast.Field{Names: []*ast.Ident{{Name: "Eval"}}},
		FType: &ir.FuncType{
			BaseType: ir.BaseType[*ast.FuncType]{
				Src: &ast.FuncType{
					Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "x"}}, Type: &ast.Ident{Name: "Plain"}}}},
					Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "Plain"}}}},
				},
			},
			Params:  irhelper.Fields("x", plain),
			Results: irhelper.Fields(plain),
		},
	}
	resetMeth := &ir.IMethod{
		Src:   &ast.Field{Names: []*ast.Ident{{Name: "Reset"}}},
		FType: irhelper.FuncType(nil, nil, nil, nil),
	}
	methodsOnly := ir.NewInterface(&ast.InterfaceType{}, nil, []*ir.IMethod{evalMeth, resetMeth})
	combined := ir.NewInterface(&ast.InterfaceType{}, []ir.Type{ir.Int32Type(), plain}, []*ir.IMethod{resetMeth})

	for _, tc := range []struct {
		name     string
		typ      ir.Type
		file     *ir.File
		want     string
		wantType string
	}{
		{"AnyType", ir.AnyType(), testFile, "any", "*ast.Ident"},
		{"EmptyInterfaceWithSrc", emptyWithSrc, testFile, "interface{}", "*ast.InterfaceType"},
		{"SingleType", singleType, testFile, "interface {\n\tint32\n}", "*ast.InterfaceType"},
		{"TypeUnionSamePackage", unionTypes, testFile, "interface {\n\tint32 | int64 | Plain\n}", "*ast.InterfaceType"},
		{"TypeUnionCrossPackage", unionTypes, otherFile, "interface {\n\tint32 | int64 | ir_test.Plain\n}", "*ast.InterfaceType"},
		{"TypeUnionCrossPackageAlias", unionTypes, aliasFile, "interface {\n\tint32 | int64 | myir.Plain\n}", "*ast.InterfaceType"},
		{"MethodsSamePackage", methodsOnly, testFile, "interface {\n\tEval(x Plain) Plain\n\tReset()\n}", "*ast.InterfaceType"},
		{"TypesAndMethodsCrossPackage", combined, otherFile, "interface {\n\tint32 | ir_test.Plain\n\tReset()\n}", "*ast.InterfaceType"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.typ.Refer(tc.file)
			if gotType := fmt.Sprintf("%T", got); gotType != tc.wantType {
				t.Errorf("Interface.Refer(%v) returned %s, want %s", tc.file, gotType, tc.wantType)
			}
			if gotStr := gxfmt.Format(got); gotStr != tc.want {
				t.Errorf("gxfmt.Format(Interface.Refer(%v)) = %q, want %q", tc.file, gotStr, tc.want)
			}
		})
	}
}
