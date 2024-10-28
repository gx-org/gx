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
	"reflect"

	"github.com/gx-org/gx/build/ir"
)

func importNamedTypes(scope *scopeFile, types []*ir.NamedType) {
	for _, typ := range types {
		var nType *namedType
		identNode := scope.pkg().ns.fetchIdentNode(typ.NameT)
		if identNode == nil {
			var ok bool
			nType, ok = importType(scope, typ)
			if !ok {
				continue
			}
		} else {
			_, typeNode, ok := identNode.typeF(scope)
			if !ok {
				continue
			}
			nType, ok = typeNode.(*namedType)
			if !ok {
				scope.err().Appendf(typ.Source(), "type previously registered as %T cannot be casted as %T", typeNode, reflect.TypeFor[*namedType]())
				continue
			}
		}
		nType.importMethods(scope, typ.Methods)
	}
}

func importFuncs(scope *scopeFile, funcs []ir.Func) {
	for _, irFunc := range funcs {
		fn := importFuncBuiltin(scope, irFunc.(*ir.FuncBuiltin))
		scope.file().declareFunc(scope, fn)
	}
}

func importConstDecls(scope *scopeFile, cstDecls []*ir.ConstDecl) bool {
	ok := true
	for _, cstDecl := range cstDecls {
		declOk := importConstDecl(scope, cstDecl)
		ok = ok && declOk
	}
	return ok
}
