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

package builtin

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/impl"
)

type registerMacro struct {
	name string
	impl ir.MacroImpl
}

// RegisterMacro registers the implementation of a meta function.
func RegisterMacro(name string, impl ir.MacroImpl) Builder {
	return &registerMacro{
		name: name,
		impl: impl,
	}
}

func (b *registerMacro) Build(bld *builder.Builder, _ *impl.Stdlib, pkg *builder.FilePackage) (err error) {
	pkgIR := pkg.IR()
	defer func() {
		if err != nil {
			err = fmterr.Internal(errors.Errorf("cannot set the implementation of %s.%s: %v", pkgIR.Name, b.name, err))
		}
	}()
	fun := pkgIR.FindFunc(b.name)
	if fun == nil {
		return errors.Errorf("cannot find the function in the package IR")
	}
	macro, ok := fun.(*ir.Macro)
	if !ok {
		return errors.Errorf("type %T is not %s", fun, reflect.TypeFor[*ir.Macro]())
	}
	macro.BuildSynthetic = b.impl
	return nil
}

func (b *registerMacro) Name() string {
	return fmt.Sprintf("%T:%s", b, b.name)
}

type registerAnnotator struct {
	name string
	impl ir.AnnotatorImpl
}

// RegisterAnnotator registers the implementation of a annotator.
func RegisterAnnotator(name string, impl ir.AnnotatorImpl) Builder {
	return &registerAnnotator{
		name: name,
		impl: impl,
	}
}

func (b *registerAnnotator) Build(bld *builder.Builder, _ *impl.Stdlib, pkg *builder.FilePackage) (err error) {
	pkgIR := pkg.IR()
	defer func() {
		if err != nil {
			err = fmterr.Internal(errors.Errorf("cannot set the implementation of %s.%s: %v", pkgIR.Name, b.name, err))
		}
	}()
	fun := pkgIR.FindFunc(b.name)
	if fun == nil {
		return errors.Errorf("cannot find the function in the package IR")
	}
	macro, ok := fun.(*ir.Annotator)
	if !ok {
		return errors.Errorf("type %T is not %s", fun, reflect.TypeFor[*ir.Macro]())
	}
	macro.Annotate = b.impl
	return nil
}

func (b *registerAnnotator) Name() string {
	return fmt.Sprintf("%T:%s", b, b.name)
}
