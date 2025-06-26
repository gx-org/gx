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

package graph

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/golang/backend/kernels"
	goplatform "github.com/gx-org/gx/golang/backend/platform"
)

type argument struct {
	node
	name  string
	index int
}

var _ execNode = (*argument)(nil)

// Argument returns a node set by a caller when calling the function.
func (g *Graph) Argument(name string, sh *shape.Shape, index int) (ops.Node, error) {
	factory, err := kernels.FactoryFor(sh.DType)
	if err != nil {
		return nil, err
	}
	return &argument{
		node:  g.node(sh, factory),
		name:  name,
		index: index,
	}, nil
}

func (n *argument) exec(exec *executor) (kernels.Array, error) {
	arg := exec.args[n.index]
	handle, ok := arg.(*goplatform.Handle)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to %s", arg, reflect.TypeFor[*goplatform.Handle]())
	}
	return handle.Array(), nil
}
