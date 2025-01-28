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

package values

import (
	"fmt"

	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/build/ir"
)

var _ Value = (*IRNode)(nil)

// IRNode is a GX string value.
type IRNode struct {
	node ir.Node
}

// NewIRNode returns a GX string value from its type and its Go value.
func NewIRNode(node ir.Node) (*IRNode, error) {
	return &IRNode{node: node}, nil
}

func (s *IRNode) value() {}

// Type returns the type of the value.
func (s *IRNode) Type() ir.Type {
	return ir.TypeFromKind(ir.IRKind)
}

// ToHost transfers the value to host given an allocator.
func (s *IRNode) ToHost(platform.Allocator) (Value, error) {
	return s, nil
}

// IR representation of the value.
// The returned string is a string reported to the user.
func (s *IRNode) String() string {
	return fmt.Sprint(s.node)
}
