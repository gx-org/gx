// Copyright 2025 Google LLC
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

// Package irstring builds a string representation of an IR tree.
package irstring

import (
	"fmt"
	"go/ast"
	"math/big"
	"reflect"
	"strings"

	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/ir"
)

type rStringType int

const (
	// TestStringType is the type used by default.
	TestStringType rStringType = iota
	// VerboseStringType is the type used for debugging.
	// It helps the user to identify differences or what to build.
	VerboseStringType
	// SilentStringType is used to silent the diffs.
	SilentStringType
)

// ReflectStringOutput sets the mode with which the string is built.
// SilentStringType means that the string is built like a test string
// but may not be reported in the test error.
//
// This value needs to be set to testStringType or silentStringType for the test to pass.
var ReflectStringOutput rStringType = TestStringType

func valueToString(done map[any]bool, val reflect.Value, proc processor) string {
	if !val.CanInterface() {
		return val.String()
	}
	return val.Interface().(fmt.Stringer).String()
}

func fieldList(done map[any]bool, val reflect.Value, proc processor) string {
	list := val.Interface().(*ir.FieldList)
	var all []string
	for _, group := range list.List {
		var names []string
		for _, field := range group.Fields {
			if field.Name != nil {
				names = append(names, field.Name.Name)
			}
		}
		allNames := ""
		if len(names) > 0 {
			allNames = strings.Join(names, ",") + " "
		}
		if group.Type == nil {
			return "<type:nil>"
		}
		typeS := reflectString(done, reflect.ValueOf(group.Type), proc)
		all = append(all, allNames+typeS)
	}
	return fmt.Sprintf("FieldList{%s}", strings.Join(all, ","))
}

func typeParam(done map[any]bool, val reflect.Value, proc processor) string {
	tParam := val.Interface().(*ir.TypeParam)
	typ := reflect.ValueOf(tParam.Field.Type())
	return fmt.Sprintf("%s:%s", tParam.Field.Name.Name, reflectString(done, typ, proc))
}

func constExpr(done map[any]bool, val reflect.Value, proc processor) string {
	cExpr := val.Interface().(*ir.ConstExpr)
	return fmt.Sprintf("const %s %s", cExpr.VName.Name, reflectString(done, reflect.ValueOf(cExpr.Val), proc))
}

func arrayType(done map[any]bool, val reflect.Value, proc processor) string {
	array := val.Interface().(ir.ArrayType)
	return fmt.Sprintf("%s%s", array.DataType().String(), reflectString(done, reflect.ValueOf(array.Rank()), proc))
}

func skip(map[any]bool, reflect.Value, processor) string {
	return ""
}

func rank(done map[any]bool, val reflect.Value, proc processor) string {
	rnk := val.Interface().(*ir.Rank)
	axes := make([]string, len(rnk.Ax))
	for i, ax := range rnk.Ax {
		switch ax.Type().Kind() {
		case ir.SliceKind:
			axes[i] = fmt.Sprintf("[group<%s>]", ax.String())
		case ir.IntLenKind:
			axes[i] = ax.String()
		default:
			axes[i] = "invalid"
		}
	}
	return strings.Join(axes, "")
}

func typeValExpr(done map[any]bool, val reflect.Value, proc processor) string {
	ref := val.Interface().(*ir.TypeValExpr)
	switch typT := ref.Typ.(type) {
	case *ir.FuncType, ir.ArrayType, *ir.TypeParam:
		return reflectString(done, reflect.ValueOf(typT), proc)
	default:
		return typT.String()
	}
}

func funcValExpr(done map[any]bool, val reflect.Value, proc processor) string {
	ref := val.Interface().(*ir.FuncValExpr)
	switch fT := ref.F.(type) {
	case *ir.FuncLit:
		return reflectString(done, reflect.ValueOf(fT), proc)
	case ir.PkgFunc:
		return fT.Name()
	case *ir.SpecialisedFunc:
		return fT.X.String()
	default:
		return ref.T.Type().String()
	}
}

func stringLiteral(done map[any]bool, val reflect.Value, proc processor) string {
	ref := val.Interface().(*ir.StringLiteral)
	return fmt.Sprintf("StringLiteral{%s}", ref.Src.Value)
}

func valueRefToString(done map[any]bool, val reflect.Value, proc processor) string {
	ref := val.Interface().(*ir.ValueRef)
	typeOf := reflect.TypeOf(ref.Stor).Elem()
	typeS := reflectString(done, reflect.ValueOf(ref.Type()), proc)
	return fmt.Sprintf("%s->%s[%s]", ref.Stor.NameDef().Name, typeOf.Name(), typeS)
}

type (
	stringerFunc func(done map[any]bool, val reflect.Value, proc processor) string
	debugFunc    struct {
		stringer stringerFunc
		debug    bool
	}
)

func debugOk(stringer stringerFunc) debugFunc {
	return debugFunc{
		stringer: stringer,
		debug:    true,
	}
}

func notOnDebug(stringer stringerFunc) debugFunc {
	return debugFunc{
		stringer: stringer,
		debug:    false,
	}
}

var typeToProcess = map[string]debugFunc{}

func init() {
	// Not in the map initialisation to prevent a cycle.
	typeToProcess["github.com/gx-org/gx/build/ir.FieldList"] = notOnDebug(fieldList)
	typeToProcess["github.com/gx-org/gx/build/ir.arrayType"] = notOnDebug(arrayType)
	typeToProcess["github.com/gx-org/gx/build/ir.TypeParam"] = notOnDebug(typeParam)
	typeToProcess["github.com/gx-org/gx/build/ir.ConstExpr"] = debugOk(constExpr)
	typeToProcess["github.com/gx-org/gx/build/ir.TypeValExpr"] = debugOk(typeValExpr)
	typeToProcess["go/ast.Ident"] = debugOk(func(done map[any]bool, val reflect.Value, proc processor) string {
		return val.Interface().(*ast.Ident).Name
	})
	typeToProcess["math/big.Int"] = debugOk(func(done map[any]bool, val reflect.Value, proc processor) string {
		return val.Interface().(*big.Int).String()
	})
	typeToProcess["github.com/gx-org/gx/build/ir.atomicType"] = debugOk(valueToString)
	typeToProcess["github.com/gx-org/gx/build/ir.int32Type"] = debugOk(valueToString)
	typeToProcess["github.com/gx-org/gx/build/ir.int64Type"] = debugOk(valueToString)
	typeToProcess["github.com/gx-org/gx/build/ir.intlenType"] = debugOk(valueToString)
	typeToProcess["github.com/gx-org/gx/build/ir.float32Type"] = debugOk(valueToString)
	typeToProcess["github.com/gx-org/gx/build/ir.float64Type"] = debugOk(valueToString)
	typeToProcess["github.com/gx-org/gx/build/ir.boolType"] = debugOk(valueToString)
	typeToProcess["github.com/gx-org/gx/build/ir.stringType"] = debugOk(valueToString)
	typeToProcess["github.com/gx-org/gx/build/ir.numberIntType"] = debugOk(valueToString)
	typeToProcess["github.com/gx-org/gx/build/ir.numberFloatType"] = debugOk(valueToString)
	typeToProcess["github.com/gx-org/gx/build/ir.ValueRef"] = debugOk(valueRefToString)
	typeToProcess["github.com/gx-org/gx/build/ir.Annotations"] = debugOk(valueToString)
	typeToProcess["github.com/gx-org/gx/build/ir.StringLiteral"] = debugOk(stringLiteral)
	typeToProcess["github.com/gx-org/gx/build/ir.TypeSet"] = notOnDebug(valueToString)
	typeToProcess["github.com/gx-org/gx/build/ir.Rank"] = notOnDebug(rank)
	typeToProcess["github.com/gx-org/gx/build/ir.FuncValExpr"] = notOnDebug(funcValExpr)
	typeToProcess["github.com/gx-org/gx/build/ir.File"] = debugOk(func(map[any]bool, reflect.Value, processor) string {
		return ""
	})
}

func reflectStructString(done map[any]bool, val reflect.Value, proc processor) string {
	// Filter structures we care about.
	typ := val.Type()
	process, processOk := typeToProcess[typ.PkgPath()+"."+typ.Name()]
	if processOk && (process.debug || ReflectStringOutput != VerboseStringType) {
		return process.stringer(done, val.Addr(), proc)
	}
	if val.CanAddr() && val.Addr().CanInterface() && proc != nil {
		if s := proc(done, val.Addr()); s != "" {
			return s
		}
	}
	if !strings.Contains(typ.PkgPath(), "gx/build/ir") {
		return ""
	}

	s := strings.Builder{}
	fmt.Fprintf(&s, "%s {", typ.Name())
	for i := range typ.NumField() {
		fieldVal := val.Field(i)
		if fieldVal.IsZero() {
			continue
		}
		fieldName := typ.Field(i).Name
		valS := reflectString(done, fieldVal, proc)
		if valS == "" {
			continue
		}
		s.WriteString("\n\t")
		s.WriteString(fieldName)
		s.WriteString(": ")
		s.WriteString(gxfmt.IndentSkip(1, valS))
	}
	s.WriteString("\n}")
	return s.String()
}

func reflectSliceString(done map[any]bool, val reflect.Value, proc processor) string {
	num := val.Len()
	if num == 0 {
		return ""
	}
	var s strings.Builder
	s.WriteString("[\n")
	for i := range num {
		valS := reflectString(done, val.Index(i), proc)
		s.WriteString(gxfmt.Indent(valS))
		s.WriteString("\n")
	}
	s.WriteString("]")
	return s.String()
}

func stmtProc(done map[any]bool, val reflect.Value) string {
	a := val.Interface()
	switch aT := a.(type) {
	case *ir.FuncType:
		return reflectString(done, val, nil)
	case ir.Type:
		return aT.String()
	default:
		return ""
	}
}

func toPointer(val reflect.Value) any {
	if !val.IsValid() || val.IsZero() {
		return nil
	}
	switch val.Kind() {
	case reflect.Struct:
		return val
	case reflect.Slice:
		return val
	case reflect.Interface:
		return val.Elem().Interface()
	default:
		return val.Interface()
	}
}

func reflectString(done map[any]bool, val reflect.Value, proc processor) string {
	if done[val] {
		return fmt.Sprintf("<ref:%s>", val.Type().Name())
	}
	ptr := toPointer(val)
	if ptr != nil {
		done[ptr] = true
		defer delete(done, ptr)
	} else {
		return ""
	}
	// Register the value in the stack.
	// Build a string representation of the value.
	kind := val.Type().Kind()
	switch kind {
	case reflect.Bool:
		return fmt.Sprint(val.Bool())
	case reflect.Int:
		return fmt.Sprint(val.Int())
	case reflect.Ptr:
		if val.IsNil() {
			return ""
		}
		return reflectString(done, val.Elem(), proc)
	case reflect.Interface:
		if val.IsNil() {
			return ""
		}
		if _, isStmt := val.Interface().(ir.Stmt); isStmt && ReflectStringOutput != VerboseStringType {
			proc = stmtProc
		}
		return reflectString(done, val.Elem(), proc)
	case reflect.Slice:
		return reflectSliceString(done, val, proc)
	case reflect.String:
		return val.String()
	case reflect.Struct:
		return reflectStructString(done, val, proc)
	case reflect.Uint:
		return fmt.Sprint(val.Uint())
	default:
		return fmt.Sprintf("%v not supported", kind)
	}
}

type processor func(done map[any]bool, val reflect.Value) string

// ReflectString builds a human readable string representation of an IR using reflection.
// Files are skipped.
func ReflectString(done map[any]bool, val any) string {
	if val == nil {
		return "nil"
	}
	return reflectString(done, reflect.ValueOf(val), nil)
}
