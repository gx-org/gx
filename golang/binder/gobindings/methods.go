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
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/binder/bindings"
)

type (
	receiver interface {
		named() *ir.NamedType
		Index() int
	}

	method struct {
		function
		Receiver   receiver
		NameDevice string
	}
)

func (b *binder) buildMethods(r receiver) []method {
	var methods []method
	for i, fun := range r.named().Methods {
		if !ir.IsExported(fun.Name()) {
			continue
		}
		if fun.Type() == nil {
			continue
		}
		if !bindings.CanBeOnDeviceFunc(fun) {
			continue
		}
		methods = append(methods, method{
			function: function{
				binder:    b,
				PkgFunc:   fun,
				FuncIndex: i,
			},
			Receiver:   r,
			NameDevice: fun.Name(),
		})
	}
	return methods
}

func toFuncs(methods []method) []*function {
	funcs := make([]*function, len(methods))
	for i, method := range methods {
		funcs[i] = &method.function
	}
	return funcs
}
