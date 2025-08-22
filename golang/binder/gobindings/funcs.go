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

package gobindings

import (
	"fmt"
	"slices"
	"strings"

	"github.com/gx-org/gx/build/ir"
)

type function struct {
	*binder
	ir.Func
	FuncIndex int
}

func (b *binder) newFunc(f ir.Func, i int) (*function, error) {
	return &function{
		binder:    b,
		Func:      f,
		FuncIndex: i,
	}, nil
}

func (b *binder) Methods() []*function {
	var funcs []*function
	for _, namedType := range b.NamedTypes {
		funcs = append(funcs, toFuncs(namedType.Methods())...)
	}
	return funcs
}

func (b *binder) FunctionsAndMethods() []*function {
	return append(b.Methods(), b.Funcs...)
}

func (f function) RunnerType() string {
	return f.FuncType().ReceiverField().Type().(*ir.NamedType).Name()
}

func (f function) RecvTypeName() string {
	recv := f.FuncType().ReceiverField()
	if recv == nil {
		return ""
	}
	return recv.Type().(*ir.NamedType).NameDef().Name
}

func (f function) ReceiverField() (string, error) {
	recvName := nameFromRecv(f.FuncType().ReceiverField())
	if recvName == "" {
		return "", nil
	}
	return fmt.Sprintf("receiver handle%s", recvName), nil
}

func outputN(i int) string {
	return fmt.Sprintf("out%d", i)
}

func argN(i int) string {
	return fmt.Sprintf("arg%d", i)
}

func toKindSuffix(typ ir.Type) string {
	kind := typ.Kind()
	if kind == ir.IntLenKind || kind == ir.IntIdxKind {
		return "DefaultInt" // Matches platform.Host.DefaultInt(ir.Int)
	}
	kindS := kind.String()
	kindS = strings.ToUpper(kindS[:1]) + kindS[1:]
	return kindS
}

func hasStructIn(fields *ir.FieldList) bool {
	for _, grp := range fields.List {
		if grp.Type.Typ.Kind() == ir.StructKind {
			return true
		}
	}
	return false
}

func (f function) DefinePackageVariable() string {
	fType := f.Func.FuncType()
	if !hasStructIn(fType.Results) {
		return ""
	}
	if fType.ReceiverField() == nil {
		return "fty := pkg.handle.Factory\n\t"
	}
	return "fty := recv.handle.pkg.handle.Factory\n\t"
}

func (f function) ProcessDeviceOutput() (string, error) {
	fields := f.Func.FuncType().Results.Fields()
	lines := []string{""}
	for i, field := range fields {
		fieldLines, err := f.setTargetFromSourceType(outputN(i), fmt.Sprintf("outputs[%d]", i), field.Type())
		if err != nil {
			return "", err
		}
		lines = slices.Concat(lines, fieldLines)
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n\t"), nil
}

func (f function) BackendArguments() (string, error) {
	fields := f.Func.FuncType().Params.Fields()
	if len(fields) == 0 {
		return "nil", nil
	}
	args := []string{
		"[]values.Value{",
	}
	for i, field := range fields {
		gxValue := fmt.Sprintf("%s.Bridge().GXValue()", argN(i))
		args = append(args,
			fmt.Sprintf("\t\t%s, // %s %s", gxValue, field.Name, field.Type()),
		)
	}
	args = append(args, "\t}")
	return strings.Join(args, "\n"), nil

}

func (f function) Parameters() (string, error) {
	fields := f.Func.FuncType().Params.Fields()
	params := make([]string, len(fields))
	for i, field := range fields {
		typeS, err := f.bridgerType(field.Group.Type.Typ)
		if err != nil {
			return "", err
		}
		params[i] = argN(i) + " " + typeS
	}
	return strings.Join(params, ", "), nil
}

func (f function) Returns() (string, error) {
	fields := f.Func.FuncType().Results.Fields()
	returns := make([]string, len(fields))
	for i := range fields {
		returns[i] = outputN(i)
	}
	return strings.Join(returns, ", "), nil
}

func (f function) Results() (string, error) {
	fields := f.Func.FuncType().Results.Fields()
	results := make([]string, len(fields))
	for i, field := range fields {
		typeS, err := f.binder.bridgerType(field.Group.Type.Typ)
		if err != nil {
			return "", err
		}
		results[i] = "_ " + typeS
	}
	return strings.Join(results, ", "), nil
}
