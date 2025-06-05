package ir

import (
	"go/ast"
)

type builtinStorage struct {
	name *ast.Ident
	val  AssignableExpr
}

var builtinStore builtinStorage

func (*builtinStorage) node()         {}
func (*builtinStorage) storage()      {}
func (*builtinStorage) storageValue() {}

func (s *builtinStorage) Source() ast.Node {
	return s.name
}

func (s *builtinStorage) NameDef() *ast.Ident {
	return s.name
}

func (s *builtinStorage) Type() Type {
	return s.val.Type()
}

func (s *builtinStorage) Value(Expr) AssignableExpr {
	return s.val
}

// BuiltinStorage stores a value in a builtin storage.
func BuiltinStorage(name string, val AssignableExpr) StorageWithValue {
	return &builtinStorage{
		name: &ast.Ident{Name: name},
		val:  val,
	}
}

var (
	falseStorage = BuiltinStorage("false", False())
	trueStorage  = BuiltinStorage("true", True())
)

// FalseStorage returns the storage for the value of true.
func FalseStorage() StorageWithValue {
	return falseStorage
}

// TrueStorage returns the storage for the value of true.
func TrueStorage() StorageWithValue {
	return trueStorage
}
