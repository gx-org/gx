// Package hover computes hover information for GX source files.
package hover

import (
	"fmt"
	"go/token"
	"path"
	"slices"
	"strings"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/lsp/position"
	"github.com/gx-org/gx/src/gx/ast"
	"github.com/gx-org/gx/src/gx/format"
)

// Info holds hover information for an IR node at a source position.
type Info struct {
	IR            ir.IR
	SymbolInfo    string
	DeclaredBy    string
	Documentation string
	Definition    ir.IR
	Snippet       string
}

// At computes hover information at the given 1-based line and column in file.
func At(file *ir.File, line, column int) Info {
	node := position.At(file, line, column)
	if node == nil {
		return Info{}
	}
	info := Info{IR: node}
	populate(&info, file, node)
	return info
}

func populate(info *Info, file *ir.File, node ir.IR) {
	switch n := node.(type) {
	case *ir.Package:
		info.Definition = n
		info.SymbolInfo = "package " + n.Name.Name
		info.Documentation = strings.TrimSpace(file.Src.Doc.Text())

	case *ir.ImportDecl:
		populateImport(info, file, n)
	case *ir.ConstExpr:
		populateConst(info, file, n)
	case *ir.VarExpr:
		populateVar(info, file, n)
	case *ir.AssignExpr:
		populateAssign(info, file, n)
	case *ir.AssignCallResult:
		if lvs, ok := n.Storage.(*ir.LocalVarStorage); !ok || isRangeVar(file, lvs.NameDef()) {
			populate(info, file, n.Storage)
			return
		}
		populateLocal(info, file, n)
	case *ir.LocalVarStorage:
		populateLocal(info, file, n)
	case *ir.NamedType:
		populateNamedType(info, file, n)
	case *ir.TypeValExpr:
		populate(info, file, n.Val())
	case *ir.FuncValExpr:
		populate(info, file, n.Func())
	case *ir.FuncDecl:
		populateFuncDecl(info, file, n)
	case *ir.FuncBuiltin:
		populateFuncBuiltin(info, file, n)
	case *ir.FuncKeyword:
		info.Definition = n
		info.SymbolInfo = "func " + n.Name()
	case *ir.FuncLit:
		info.Definition = n
		info.SymbolInfo = n.FType.SourceSignature(file, nil)
		info.Snippet = formatNode(n.FFile, n.Src)
	case *ir.IMethod:
		populateIMethod(info, file, n)
	case *ir.SpecTypeExpr:
		populate(info, file, n.X)
		if ft, ok := n.Typ.(*ir.FuncType); ok {
			if fn, ok := info.Definition.(ir.PkgFunc); ok {
				info.SymbolInfo = ft.SourceSignature(file, fn.NameDef())
			}
		}
	case *ir.SelectorExpr:
		populate(info, file, n.Stor)
	case *ir.StructFieldStorage:
		populate(info, file, n.Sel)
	case *ir.FieldLit:
		populate(info, file, n.FieldStorage)
	case *ir.FieldStorage:
		populate(info, file, n.Field)
	case *ir.Field:
		populateField(info, file, n)
	case ir.GenericParam:
		populateGenericParam(info, file, n)
	case *ir.Ident:
		populate(info, file, n.Stor)
	default:
		if t, ok := node.(ir.Type); ok && isBuiltinOrSliceType(t) {
			info.SymbolInfo = "type " + t.ReferString(file)
		}
	}
}

func isBuiltinOrSliceType(t ir.Type) bool {
	switch t.(type) {
	case *ir.BuiltinType, *ir.SliceType:
		return true
	}
	switch t {
	case ir.BoolType(),
		ir.Bfloat16Type(),
		ir.Float32Type(),
		ir.Float64Type(),
		ir.IntType(),
		ir.Int32Type(),
		ir.Int64Type(),
		ir.Uint32Type(),
		ir.Uint64Type(),
		ir.StringType(),
		ir.AnyType():
		return true
	}
	return false
}

func populateImport(info *Info, file *ir.File, imp *ir.ImportDecl) {
	info.Definition = imp
	if imp.Src.Name != nil && imp.Src.Name.Name != path.Base(imp.Path) {
		info.SymbolInfo = fmt.Sprintf("import %s %q", imp.Src.Name.Name, imp.Path)
	} else {
		info.SymbolInfo = fmt.Sprintf("import %q", imp.Path)
	}
	info.Documentation = specDoc(file, imp.Src, imp.Src.Doc)
}

func populateConst(info *Info, file *ir.File, ce *ir.ConstExpr) {
	info.Definition = ce
	typePart := ""
	if ce.Decl.Type != nil {
		typePart = " " + ce.Decl.Type.SourceString(file)
	}
	info.SymbolInfo = fmt.Sprintf("const %s%s = %s", ce.VName.Name, typePart, ce.Val.SourceString(file))
	info.DeclaredBy = ce.File().Package.Path()
	info.Documentation = specDoc(ce.File(), ce.Decl.Src, ce.Decl.Src.Doc)
}

func populateVar(info *Info, file *ir.File, ve *ir.VarExpr) {
	info.Definition = ve
	info.SymbolInfo = fmt.Sprintf("var %s %s", ve.VName.Name, ve.Type().ReferString(file))
	if slices.Contains(ve.Decl.FFile.Package.Decls.Vars, ve.Decl) {
		info.DeclaredBy = ve.Decl.FFile.Package.Path()
	}
	info.Documentation = specDoc(ve.Decl.FFile, ve.Decl.Src, ve.Decl.Src.Doc)
}

func populateAssign(info *Info, file *ir.File, ae *ir.AssignExpr) {
	lvs, ok := ae.Storage.(*ir.LocalVarStorage)
	if !ok || isRangeVar(file, lvs.NameDef()) {
		populate(info, file, ae.Storage)
		return
	}
	if bin, ok := ae.X.(*ir.BinaryExpr); ok {
		if id, ok := bin.X.(*ir.Ident); ok && id.Src == lvs.NameDef() {
			populate(info, file, id.Stor)
			return
		}
	}
	populateLocal(info, file, ae)
	if fv, ok := ae.X.(*ir.FuncValExpr); ok {
		if fnLit, ok := fv.Func().(*ir.FuncLit); ok {
			info.Snippet = fmt.Sprintf("%s := %s", ae.NameDef().Name, fnLit.DefineString(file))
		}
	}
}

func isRangeVar(file *ir.File, ident *ast.Ident) bool {
	for n := range ast.Preorder(file.Src) {
		if rs, ok := n.(*ast.RangeStmt); ok && (rs.Key == ident || rs.Value == ident) {
			return true
		}
	}
	return false
}

func populateLocal(info *Info, file *ir.File, stor ir.Storage) {
	info.Definition = stor
	info.SymbolInfo = fmt.Sprintf("var %s %s", stor.NameDef().Name, stor.Type().ReferString(file))
}

func populateField(info *Info, file *ir.File, field *ir.Field) {
	origin := field.Origin()
	info.Definition = origin
	if nt := enclosingStruct(file, origin); nt != nil {
		info.SymbolInfo = fmt.Sprintf("%s %s", field.Name.Name, field.Type().ReferString(file))
		info.DeclaredBy = nt.FullName()
		info.Documentation = strings.TrimSpace(origin.Group.Src.Doc.Text())
		info.Snippet = typeSnippet(nt)
		return
	}
	if isGenericParamField(file, origin) && !ir.IsNonTypeGeneric(field.Type()) {
		info.SymbolInfo = fmt.Sprintf("type %s %s", field.Name.Name, field.Type().ReferString(file))
		return
	}
	info.SymbolInfo = fmt.Sprintf("var %s %s", field.Name.Name, field.Type().ReferString(file))
}

func populateGenericParam(info *Info, file *ir.File, gp ir.GenericParam) {
	origin := gp.OrigField()
	info.Definition = origin
	if gp.IsTypeParam() {
		info.SymbolInfo = fmt.Sprintf("type %s %s", gp.NameDef().Name, origin.Type().ReferString(file))
		return
	}
	info.SymbolInfo = fmt.Sprintf("var %s %s", gp.NameDef().Name, gp.Type().ReferString(file))
}

func isGenericParamField(file *ir.File, origin *ir.Field) bool {
	hasGenParam := func(fn ir.PkgFunc) bool {
		ft := fn.FuncType()
		return ft != nil && slices.Contains(ft.GenParams.Fields.Fields(), origin)
	}
	for _, nt := range file.Package.Decls.Types {
		if slices.Contains(nt.GenParams.Fields.Fields(), origin) || slices.ContainsFunc(nt.Meths, hasGenParam) {
			return true
		}
	}
	return slices.ContainsFunc(file.Package.Decls.Funcs, hasGenParam)
}

func enclosingStruct(file *ir.File, origin *ir.Field) *ir.NamedType {
	packages := make([]*ir.Package, 0, 1+len(file.Imports))
	packages = append(packages, file.Package)
	for _, imp := range file.Imports {
		packages = append(packages, imp.Package)
	}
	for _, pkg := range packages {
		if pkg == nil || pkg.Decls == nil {
			continue
		}
		for _, nt := range pkg.Decls.Types {
			if st, ok := nt.Underlying.Val().(*ir.StructType); ok && slices.Contains(st.Fields.Fields(), origin) {
				return nt
			}
		}
	}
	return nil
}

func populateNamedType(info *Info, file *ir.File, nt *ir.NamedType) {
	origin := nt.OriginNamed()
	info.Definition = origin
	info.DeclaredBy = origin.File.Package.Path()
	info.Documentation = specDoc(origin.File, origin.Src, origin.Src.Doc)

	sep := " "
	if origin.Src.Assign.IsValid() {
		sep = " = "
	}
	summary, hasBody := namedTypeUnderlying(file, nt)
	info.SymbolInfo = namedTypeHeader(file, nt) + sep + summary

	var parts []string
	if hasBody {
		if decl := typeSnippet(origin); decl != "" {
			parts = append(parts, decl)
		}
	}
	if meths := methodSnippets(origin); meths != "" {
		parts = append(parts, meths)
	}
	info.Snippet = strings.Join(parts, "\n\n")
}

func namedTypeUnderlying(file *ir.File, nt *ir.NamedType) (string, bool) {
	var hasBody bool
	switch u := nt.Underlying.Val().(type) {
	case *ir.StructType:
		hasBody = u.Fields.Len() > 0
	case *ir.Interface:
		if u == ir.AnyType() {
			return u.ReferString(file), false
		}
		hasBody = len(u.Methods()) > 0 || u.ReferString(file) != "any"
	default:
		return u.ReferString(file), false
	}
	if !hasBody {
		return nt.Kind().String() + "{}", false
	}
	return nt.Kind().String(), true
}

func methodSnippets(origin *ir.NamedType) string {
	if len(origin.Meths) == 0 {
		return ""
	}
	lines := make([]string, 0, len(origin.Meths))
	for _, meth := range origin.Meths {
		lines = append(lines, meth.FuncType().SourceSignature(origin.File, meth.NameDef()))
	}
	return strings.Join(lines, "\n")
}

func namedTypeHeader(file *ir.File, nt *ir.NamedType) string {
	name := nt.Name()
	var args []string
	for _, gv := range nt.GenParams.Values {
		if gv != nil {
			args = append(args, gv.Value().SourceString(file))
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf("type %s[%s]", name, strings.Join(args, ", "))
	}
	if nt.GenParams.Fields.Len() > 0 {
		return fmt.Sprintf("type %s[%s]", name, nt.GenParams.Fields.SourceString(file))
	}
	return "type " + name
}

func populateFuncDecl(info *Info, file *ir.File, decl *ir.FuncDecl) {
	info.Definition = decl
	info.SymbolInfo = decl.FType.SourceSignature(file, decl.NameDef())
	info.Documentation = funcDoc(decl.Src.Doc)
	if recv := receiverNamedType(decl); recv != nil {
		info.DeclaredBy = recv.FullName()
	} else {
		info.DeclaredBy = decl.FFile.Package.Path()
	}
	if decl.Body != nil {
		info.Snippet = funcSnippet(decl)
	}
}

func populateFuncBuiltin(info *Info, file *ir.File, fn *ir.FuncBuiltin) {
	info.Definition = fn
	info.Documentation = funcDoc(fn.Src.Doc)
	if fn.FType != nil {
		info.SymbolInfo = fn.FType.SourceSignature(file, fn.NameDef())
	} else {
		info.SymbolInfo = "func " + fn.Name()
	}
	info.DeclaredBy = fn.FFile.Package.Path()
}

func populateIMethod(info *Info, file *ir.File, m *ir.IMethod) {
	info.Definition = m
	info.SymbolInfo = m.SourceSignature(file)
	info.Documentation = funcDoc(m.Doc())
	mFile := m.File()
	if mFile == nil || mFile.Package == nil {
		return
	}
	if nt := enclosingNamedInterface(mFile.Package, m.IFace); nt != nil {
		info.DeclaredBy = nt.FullName()
		info.Snippet = typeSnippet(nt)
		return
	}
	info.DeclaredBy = mFile.Package.Path()
}

func enclosingNamedInterface(pkg *ir.Package, iface *ir.Interface) *ir.NamedType {
	for _, nt := range pkg.Decls.Types {
		if nt.Underlying.Val() == iface {
			return nt.OriginNamed()
		}
	}
	return nil
}

func receiverNamedType(fn *ir.FuncDecl) *ir.NamedType {
	recv := fn.FuncType().ReceiverField()
	if recv == nil {
		return nil
	}
	nt, ok := recv.Type().(*ir.NamedType)
	if !ok {
		return nil
	}
	return nt.OriginNamed()
}

func funcDoc(doc *ast.CommentGroup) string {
	var lines []string
	for line := range strings.SplitSeq(doc.Text(), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "gx:") {
			lines = append(lines, line)
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func funcSnippet(fn *ir.FuncDecl) string {
	decl := *fn.Src
	decl.Doc = nil
	return formatNode(fn.FFile, &decl)
}

func formatNode(file *ir.File, node ast.Node) string {
	var b strings.Builder
	if err := format.Node(&b, file.FileSet(), node); err != nil {
		return ""
	}
	return b.String()
}

func typeSnippet(origin *ir.NamedType) string {
	spec := *origin.Src
	spec.Doc = nil
	spec.Comment = nil
	var b strings.Builder
	if err := format.Node(&b, origin.File.FileSet(), &ast.GenDecl{TokPos: spec.Pos(), Tok: token.TYPE, Specs: []ast.Spec{&spec}}); err != nil {
		return ""
	}
	return b.String()
}

func specDoc(file *ir.File, spec ast.Spec, doc *ast.CommentGroup) string {
	if text := strings.TrimSpace(doc.Text()); text != "" {
		return text
	}
	for n := range ast.Preorder(file.Src) {
		gd, ok := n.(*ast.GenDecl)
		if ok && len(gd.Specs) == 1 && gd.Specs[0] == spec {
			return strings.TrimSpace(gd.Doc.Text())
		}
	}
	return ""
}
