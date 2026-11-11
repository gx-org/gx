package ir

import (
	"go/token"

	"github.com/gx-org/gx/src/gx/ast"
)

func considerNode(f Finder, node IR, astNode ast.Node) bool {
	if astNode == nil {
		return false
	}
	return f.Consider(node, astNode.Pos(), astNode.End())
}

func considerPtr[T any, P interface {
	*T
	ast.Node
}](f Finder, node IR, src P) bool {
	if src == nil {
		return false
	}
	return f.Consider(node, src.Pos(), src.End())
}

func specBounds(spec ast.Spec, singleSpecDecls map[ast.Spec]*ast.GenDecl) (token.Pos, token.Pos) {
	if gen, ok := singleSpecDecls[spec]; ok {
		return gen.Pos(), gen.End()
	}
	return spec.Pos(), spec.End()
}

func visitAssignTargetStorage(f Finder, stor Storage) {
	switch st := stor.(type) {
	case *StructFieldStorage:
		st.Visit(f)
	case *IndexStorage:
		st.Visit(f)
	}
}

func visitTypeRef(f Finder, typ Type, typeAST ast.Expr) {
	if typ == nil || !considerNode(f, typ, typeAST) {
		return
	}
	typ.Visit(f)
}

func arrayElementAST(expr ast.Expr) ast.Expr {
	for {
		arr, ok := expr.(*ast.ArrayType)
		if !ok {
			return expr
		}
		expr = arr.Elt
	}
}
